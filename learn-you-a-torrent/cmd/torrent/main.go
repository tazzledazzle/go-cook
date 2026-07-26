package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signalNotify(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()

	return runDownload(ctx, args, os.Stdout)
}

var signalNotify = func(c chan<- os.Signal) {
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
}

func runDownload(ctx context.Context, args []string, out io.Writer) error {
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

	var lastProgress torrent.Progress
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
			lastProgress = p
			fmt.Fprintf(out, "\r%s", p.String())
		},
	}

	downloadErr := d.Download(ctx, tor, dir)
	return handleDownloadResult(downloadErr, lastProgress, out, tor.Info.Name, dir)
}

func handleDownloadResult(err error, last torrent.Progress, out io.Writer, name, dir string) error {
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "\nShutdown: %s\n", last.String())
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nDownloaded %s to %s\n", name, filepath.Join(dir, name))
	return nil
}
