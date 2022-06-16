package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

type File struct {
	Path         string `json:"path"`
	Length       int64  `json:"length"`
	CreationDate int64  `json:"creationTime"`
}

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

	mux := http.NewServeMux()

	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		magnetLink := r.URL.Query().Get("magnet")
		if magnetLink == "" {
			panic("could not work with empty magnet link")
		}

		t, err := c.AddMagnet(magnetLink)
		if err != nil {
			panic(err)
		}
		<-t.GotInfo()

		files := []File{}
		for _, file := range t.Files() {
			files = append(files, File{
				Path:         file.Path(),
				Length:       file.Length(),
				CreationDate: file.Torrent().Metainfo().CreationDate,
			})
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(files); err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
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
	})

	panic(http.ListenAndServe(*laddr, mux))
}
