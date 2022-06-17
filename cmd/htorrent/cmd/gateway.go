package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	storageFlag = "storage"
	laddrFlag   = "laddr"
)

var (
	errEmptyMagnetLink  = errors.New("could not work with empty magnet link")
	errEmptyPath        = errors.New("could not work with empty path")
	errCouldNotFindPath = errors.New("could not find path in torrent")
)

type file struct {
	Path         string `json:"path"`
	Length       int64  `json:"length"`
	CreationDate int64  `json:"creationTime"`
}

var gatewayCmd = &cobra.Command{
	Use:     "gateway",
	Aliases: []string{"g"},
	Short:   "Start a gateway",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		viper.SetEnvPrefix("htorrent")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		switch viper.GetInt(verboseFlag) {
		case 0:
			zerolog.SetGlobalLevel(zerolog.Disabled)
		case 1:
			zerolog.SetGlobalLevel(zerolog.PanicLevel)
		case 2:
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		case 3:
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case 4:
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case 5:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case 6:
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := torrent.NewDefaultClientConfig()

		if viper.GetInt(verboseFlag) > 5 {
			cfg.Debug = true
		}

		cfg.DefaultStorage = storage.NewFileByInfoHash(viper.GetString(storageFlag))

		c, err := torrent.NewClient(cfg)
		if err != nil {
			return err
		}
		defer c.Close()

		log.Info().
			Str("address", viper.GetString(laddrFlag)).
			Msg("Listening")

		mux := http.NewServeMux()

		mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
			magnetLink := r.URL.Query().Get("magnet")
			if magnetLink == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)

				panic(errEmptyMagnetLink)
			}

			log.Info().
				Str("magnet", magnetLink).
				Msg("Getting info")

			t, err := c.AddMagnet(magnetLink)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				panic(err)
			}
			<-t.GotInfo()

			files := []file{}
			for _, f := range t.Files() {
				log.Info().
					Str("magnet", magnetLink).
					Str("path", f.Path()).
					Msg("Got info")

				files = append(files, file{
					Path:         f.Path(),
					Length:       f.Length(),
					CreationDate: f.Torrent().Metainfo().CreationDate,
				})
			}

			enc := json.NewEncoder(w)
			if err := enc.Encode(files); err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				panic(err)
			}
		})

		mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
			magnetLink := r.URL.Query().Get("magnet")
			if magnetLink == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)

				panic(errEmptyMagnetLink)
			}

			path := r.URL.Query().Get("path")
			if path == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)

				panic(errEmptyPath)
			}

			log.Info().
				Str("magnet", magnetLink).
				Str("path", path).
				Msg("Getting stream")

			t, err := c.AddMagnet(magnetLink)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				panic(err)
			}
			<-t.GotInfo()

			found := false
			for _, f := range t.Files() {
				if f.Path() != path {
					continue
				}

				found = true

				go func() {
					tick := time.NewTicker(time.Millisecond * 100)
					defer tick.Stop()

					lastCompleted := int64(0)
					for range tick.C {
						if completed, total := f.BytesCompleted(), f.Length(); completed < total {
							if completed != lastCompleted {
								log.Debug().
									Int("peers", len(f.Torrent().PeerConns())).
									Int64("completed", completed).
									Int64("total", total).
									Str("path", f.Path()).
									Msg("Streaming")
							}

							lastCompleted = completed
						} else {
							return
						}
					}
				}()

				log.Info().
					Str("magnet", magnetLink).
					Str("path", path).
					Msg("Got stream")

				http.ServeContent(w, r, f.DisplayPath(), time.Unix(f.Torrent().Metainfo().CreationDate, 0), f.NewReader())
			}

			if !found {
				w.WriteHeader(http.StatusNotFound)

				panic(errCouldNotFindPath)
			}

			c.WaitAll()
		})

		return http.ListenAndServe(viper.GetString(laddrFlag), mux)
	},
}

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().StringP(storageFlag, "s", filepath.Join(home, ".local", "share", "htorrent", "var", "lib", "htorrent", "data"), "Path to store downloaded torrents in")
	rootCmd.PersistentFlags().StringP(laddrFlag, "l", ":1337", "Listening address")

	viper.AutomaticEnv()

	rootCmd.AddCommand(gatewayCmd)
}
