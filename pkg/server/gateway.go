package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/phayes/freeport"
	"github.com/pojntfx/go-auth-utils/pkg/authn"
	"github.com/pojntfx/go-auth-utils/pkg/authn/basic"
	"github.com/pojntfx/go-auth-utils/pkg/authn/oidc"
	v1 "github.com/pojntfx/htorrent/pkg/api/http/v1"
	"github.com/rs/zerolog/log"
)

var (
	ErrEmptyMagnetLink  = errors.New("could not work with empty magnet link")
	ErrEmptyPath        = errors.New("could not work with empty path")
	ErrCouldNotFindPath = errors.New("could not find path in torrent")
)

type Gateway struct {
	laddr        string
	storage      string
	apiUsername  string
	apiPassword  string
	oidcIssuer   string
	oidcClientID string
	debug        bool

	onDownloadProgress func(torrentMetrics v1.TorrentMetrics, fileMetrics v1.FileMetrics)

	torrentClient *torrent.Client
	srv           *http.Server

	errs chan error

	ctx context.Context
}

func NewGateway(
	laddr string,
	storage string,
	apiUsername string,
	apiPassword string,
	oidcIssuer string,
	oidcClientID string,
	debug bool,

	onDownloadProgress func(torrentMetrics v1.TorrentMetrics, fileMetrics v1.FileMetrics),

	ctx context.Context,
) *Gateway {
	return &Gateway{
		laddr:        laddr,
		storage:      storage,
		apiUsername:  apiUsername,
		apiPassword:  apiPassword,
		oidcIssuer:   oidcIssuer,
		oidcClientID: oidcClientID,
		debug:        debug,

		onDownloadProgress: onDownloadProgress,

		errs: make(chan error),

		ctx: ctx,
	}
}

func (g *Gateway) Open() error {
	log.Trace().Msg("Opening gateway")

	cfg := torrent.NewDefaultClientConfig()
	cfg.Debug = g.debug
	cfg.DefaultStorage = storage.NewFileByInfoHash(g.storage)

	torrentPort, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}
	cfg.ListenPort = torrentPort

	c, err := torrent.NewClient(cfg)
	if err != nil {
		return err
	}
	g.torrentClient = c

	var auth authn.Authn
	if strings.TrimSpace(g.oidcIssuer) == "" && strings.TrimSpace(g.oidcClientID) == "" {
		auth = basic.NewAuthn(g.apiUsername, g.apiPassword)
	} else {
		auth = oidc.NewAuthn(g.oidcIssuer, g.oidcClientID)
	}

	if err := auth.Open(g.ctx); err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if err := auth.Validate(u, p); !ok || err != nil {
			w.WriteHeader(http.StatusUnauthorized)

			panic(fmt.Errorf("%v", http.StatusUnauthorized))
		}

		magnetLink := r.URL.Query().Get("magnet")
		if magnetLink == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)

			panic(ErrEmptyMagnetLink)
		}

		log.Debug().
			Str("magnet", magnetLink).
			Msg("Getting info")

		t, err := c.AddMagnet(magnetLink)
		if err != nil {
			panic(err)
		}
		<-t.GotInfo()

		info := v1.Info{
			Files: []v1.File{},
		}
		info.Name = t.Info().BestName()
		info.InfoHash = t.InfoHash().HexString()
		info.CreationDate = t.Metainfo().CreationDate

		foundDescription := false
		for _, f := range t.Files() {
			log.Debug().
				Str("magnet", magnetLink).
				Str("path", f.Path()).
				Msg("Got info")

			info.Files = append(info.Files, v1.File{
				Path:   f.Path(),
				Length: f.Length(),
			})

			if path.Ext(f.Path()) == ".txt" {
				if foundDescription {
					continue
				}

				r := f.NewReader()
				defer r.Close()

				var description bytes.Buffer
				if _, err := io.Copy(&description, r); err != nil {
					panic(err)
				}

				info.Description = description.String()

				foundDescription = true
			}
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(info); err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if err := auth.Validate(u, p); !ok || err != nil {
			w.WriteHeader(http.StatusUnauthorized)

			panic(fmt.Errorf("%v", http.StatusUnauthorized))
		}

		log.Debug().
			Msg("Getting metrics")

		metrics := []v1.TorrentMetrics{}
		for _, t := range g.torrentClient.Torrents() {
			mi := t.Metainfo()

			info, err := mi.UnmarshalInfo()
			if err != nil {
				log.Error().
					Err(err).
					Msg("Could not unmarshal metainfo")

				continue
			}

			fileMetrics := []v1.FileMetrics{}
			for _, f := range t.Files() {
				fileMetrics = append(fileMetrics, v1.FileMetrics{
					Path:      f.Path(),
					Length:    f.Length(),
					Completed: f.BytesCompleted(),
				})
			}

			torrentMetrics := v1.TorrentMetrics{
				Magnet:   mi.Magnet(nil, &info).String(),
				InfoHash: mi.HashInfoBytes().HexString(),
				Peers:    len(t.PeerConns()),
				Files:    fileMetrics,
			}

			metrics = append(metrics, torrentMetrics)
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(metrics); err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()

			switch err {
			case http.StatusUnauthorized:
				fallthrough
			default:
				w.WriteHeader(http.StatusInternalServerError)

				e, ok := err.(error)
				if ok {
					log.Debug().
						Err(e).
						Msg("Closed connection for client")
				} else {
					log.Debug().Msg("Closed connection for client")
				}
			}
		}()

		u, p, ok := r.BasicAuth()
		if err := auth.Validate(u, p); !ok || err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="hTorrent"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			panic(fmt.Errorf("%v", http.StatusUnauthorized))
		}

		magnetLink := r.URL.Query().Get("magnet")
		if magnetLink == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)

			panic(ErrEmptyMagnetLink)
		}

		path := r.URL.Query().Get("path")
		if path == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)

			panic(ErrEmptyPath)
		}

		log.Debug().
			Str("magnet", magnetLink).
			Str("path", path).
			Msg("Getting stream")

		t, err := c.AddMagnet(magnetLink)
		if err != nil {
			panic(err)
		}
		<-t.GotInfo()

		found := false
		for _, l := range t.Files() {
			f := l

			if f.Path() != path {
				continue
			}

			found = true

			go func() {
				tick := time.NewTicker(time.Millisecond * 100)
				defer tick.Stop()

				lastCompleted := int64(0)
				for range tick.C {
					if completed, length := f.BytesCompleted(), f.Length(); completed < length {
						if completed != lastCompleted {
							g.onDownloadProgress(
								v1.TorrentMetrics{
									Magnet: magnetLink,
									Peers:  len(f.Torrent().PeerConns()),
									Files:  []v1.FileMetrics{},
								},
								v1.FileMetrics{
									Path:      f.Path(),
									Length:    length,
									Completed: completed,
								},
							)
						}

						lastCompleted = completed
					} else {
						return
					}
				}
			}()

			log.Debug().
				Str("magnet", magnetLink).
				Str("path", path).
				Msg("Got stream")

			http.ServeContent(w, r, f.DisplayPath(), time.Unix(f.Torrent().Metainfo().CreationDate, 0), f.NewReader())
		}

		if !found {
			w.WriteHeader(http.StatusNotFound)

			panic(ErrCouldNotFindPath)
		}
	})

	g.srv = &http.Server{Addr: g.laddr}
	g.srv.Handler = mux

	log.Debug().
		Str("address", g.laddr).
		Msg("Listening")

	go func() {
		if err := g.srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				close(g.errs)

				return
			}

			g.errs <- err

			return
		}
	}()

	return nil
}

func (g *Gateway) Close() error {
	log.Trace().Msg("Closing gateway")

	if err := g.srv.Shutdown(g.ctx); err != nil {
		if err != context.Canceled {
			return err
		}
	}

	errs := g.torrentClient.Close()
	for _, err := range errs {
		if err != nil {
			if err != context.Canceled {
				return err
			}
		}
	}

	return nil
}

func (g *Gateway) Wait() error {
	for err := range g.errs {
		if err != nil {
			return err
		}
	}

	return nil
}
