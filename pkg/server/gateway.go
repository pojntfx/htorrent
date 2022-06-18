package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
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

	onDownloadProgress func(peers int, total, completed int64, path string)

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

	onDownloadProgress func(peers int, total, completed int64, path string),

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

		files := []v1.File{}
		for _, f := range t.Files() {
			log.Debug().
				Str("magnet", magnetLink).
				Str("path", f.Path()).
				Msg("Got info")

			files = append(files, v1.File{
				Path:         f.Path(),
				Length:       f.Length(),
				CreationDate: f.Torrent().Metainfo().CreationDate,
			})
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(files); err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()

			switch err {
			case nil:
				fallthrough
			case http.StatusUnauthorized:
				fallthrough
			case http.StatusUnprocessableEntity:
				fallthrough
			case http.StatusNotFound:
				fallthrough
			default:
				w.WriteHeader(http.StatusInternalServerError)

				log.Debug().
					Err(err.(error)).
					Msg("Closed connection for client")
			}
		}()

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
								Int64("total", total).
								Int64("completed", completed).
								Str("path", f.Path()).
								Msg("Streaming")

							g.onDownloadProgress(
								len(f.Torrent().PeerConns()),
								total,
								completed,
								f.Path(),
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

		c.WaitAll()
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
