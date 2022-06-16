package main

import (
	"flag"
	"fmt"
	"log"
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

	magnet := flag.String("magnet", "", "Magnet link to download")
	storagePath := flag.String("storage", filepath.Join(home, ".local", "share", "htorrent", "var", "lib", "htorrent", "data"), "Path to store downloaded torrents in")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	if *magnet == "" {
		panic("could not work with empty magnet link")
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DefaultStorage = storage.NewFileByInfoHash(*storagePath)
	cfg.Debug = *verbose

	c, err := torrent.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	t, err := c.AddMagnet(*magnet)
	if err != nil {
		panic(err)
	}
	<-t.GotInfo()

	t.DownloadAll()

	go func() {
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()

		for range tick.C {
			if completed, total := t.BytesCompleted(), t.Length(); completed < total {
				log.Printf("%v/%v bytes downloaded", completed, total)
			} else {
				return
			}
		}
	}()

	c.WaitAll()

	for _, file := range t.Files() {
		fmt.Println(filepath.Join(*storagePath, file.Torrent().InfoHash().String(), file.Path()))
	}
}
