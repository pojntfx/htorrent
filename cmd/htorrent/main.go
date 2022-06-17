package main

import "github.com/pojntfx/htorrent/cmd/htorrent/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
