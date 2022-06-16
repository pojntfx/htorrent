package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	storagePath := flag.String("storage", filepath.Join(home, ".local", "share", "htorrent", "var", "lib", "htorrent", "data"), "Path to store downloaded torrents in")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	laddr := flag.String("laddr", ":1337", "Listen address")

	flag.Parse()

	cfg := torrent.NewDefaultClientConfig()
	cfg.DefaultStorage = storage.NewFileByInfoHash(*storagePath)
	cfg.Debug = *verbose

	c, err := torrent.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	log.Println("Listening on", *laddr)

	panic(
		http.ListenAndServe(
			*laddr,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				magnetLink := r.URL.Query().Get("magnet")
				if magnetLink == "" {
					panic("could not work with empty magnet link")
				}

				path := r.URL.Query().Get("path")
				if path == "" {
					panic("could not work with empty path")
				}

				t, err := c.AddMagnet(magnetLink)
				if err != nil {
					panic(err)
				}
				<-t.GotInfo()

				found := false
				for _, file := range t.Files() {
					if file.Path() != path {
						continue
					}

					found = true

					go func() {
						tick := time.NewTicker(time.Millisecond * 100)
						defer tick.Stop()

						for range tick.C {
							if completed, total := file.BytesCompleted(), file.Length(); completed < total {
								log.Printf("%v/%v bytes downloaded", completed, total)
							} else {
								return
							}
						}
					}()

					http.ServeContent(w, r, file.DisplayPath(), time.Unix(file.Torrent().Metainfo().CreationDate, 0), file.NewReader())
				}

				if !found {
					panic("could not find path in torrent")
				}

				c.WaitAll()
			}),
		),
	)
}
