package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/downloader"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/tracker"
)

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// Run executes CLI commands from args.
func Run(args []string) error {
	if len(args) < 2 || args[0] != "download" {
		return fmt.Errorf("usage: torrent download <file.torrent>")
	}

	tor, err := torrent.ParseTorrent(args[1])
	if err != nil {
		return err
	}

	var peerID [20]byte
	copy(peerID[:], []byte("-GO0001-000000"))

	client := tracker.NewClient(peerID, nil)
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	d := &downloader.Downloader{
		PeerID: peerID,
		ListPeers: func(t *torrent.Torrent) ([]downloader.PeerAddress, error) {
			peers, err := client.Announce(t)
			if err != nil {
				return nil, err
			}
			addrs := make([]downloader.PeerAddress, len(peers))
			for i, p := range peers {
				addrs[i] = downloader.PeerAddress{IP: p.IP, Port: p.Port}
			}
			return addrs, nil
		},
		OnProgress: func(p torrent.Progress) {
			fmt.Printf("\r%s", p.String())
		},
	}

	if err := d.Download(context.Background(), tor, dir); err != nil {
		return err
	}
	fmt.Printf("\nDownloaded %s to %s\n", tor.Info.Name, filepath.Join(dir, tor.Info.Name))
	return nil
}
