package downloader

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/file"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/peer"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/pieces"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

// PeerAddress is a TCP peer endpoint for downloading.
type PeerAddress struct {
	IP   net.IP
	Port uint16
}

// Downloader coordinates peers to download a full torrent.
type Downloader struct {
	PeerID      [20]byte
	ListPeers   func(t *torrent.Torrent) ([]PeerAddress, error)
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
	OnProgress  func(torrent.Progress)
}

// Download downloads all pieces of t into dir.
func (d *Downloader) Download(ctx context.Context, t *torrent.Torrent, dir string) error {
	if d.ListPeers == nil {
		return fmt.Errorf("download: ListPeers is required")
	}
	if d.DialContext == nil {
		d.DialContext = (&net.Dialer{}).DialContext
	}

	peers, err := d.ListPeers(t)
	if err != nil {
		return fmt.Errorf("download: list peers: %w", err)
	}
	if len(peers) == 0 {
		return fmt.Errorf("download: no peers returned")
	}

	writer, err := file.NewWriter(dir, t.Info)
	if err != nil {
		return fmt.Errorf("download: create writer: %w", err)
	}

	manager := pieces.NewManager(t.Info.PieceCount())
	start := time.Now()
	var active int32

	var wg sync.WaitGroup
	errCh := make(chan error, len(peers))

	for _, p := range peers {
		wg.Add(1)
		go func(p PeerAddress) {
			defer wg.Done()
			atomic.AddInt32(&active, 1)
			d.reportProgress(t, manager, start, &active)
			defer func() {
				atomic.AddInt32(&active, -1)
				d.reportProgress(t, manager, start, &active)
			}()
			if err := d.peerWorker(ctx, t, p, writer, manager, start, &active); err != nil {
				errCh <- err
			}
		}(p)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if !manager.Complete() {
			return fmt.Errorf("download: incomplete after peers finished")
		}
		return writer.Close()
	}
}

func (d *Downloader) peerWorker(ctx context.Context, t *torrent.Torrent, p PeerAddress, writer *file.Writer, manager *pieces.Manager, start time.Time, active *int32) error {
	addr := fmt.Sprintf("%s:%d", p.IP, p.Port)
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("download: dial %s: %w", addr, err)
	}
	defer conn.Close()

	ours := peer.Handshake{InfoHash: t.InfoHash, PeerID: d.PeerID}
	pc := peer.NewConnection(conn, t.InfoHash, ours)
	if err := pc.PerformHandshake(); err != nil {
		return fmt.Errorf("download: handshake: %w", err)
	}
	if err := pieces.PrepareForDownload(pc); err != nil {
		return fmt.Errorf("download: prepare peer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		index, ok := manager.NextMissing()
		if !ok {
			return nil
		}

		if err := pieces.DownloadPieceData(pc, t, index, writer); err != nil {
			return fmt.Errorf("download: piece %d: %w", index, err)
		}
		manager.MarkComplete(index)

		d.reportProgress(t, manager, start, active)
	}
}

func (d *Downloader) reportProgress(t *torrent.Torrent, manager *pieces.Manager, start time.Time, active *int32) {
	if d.OnProgress == nil {
		return
	}
	d.OnProgress(torrent.Progress{
		CompletedPieces: manager.CompletedCount(),
		TotalPieces:     t.Info.PieceCount(),
		DownloadedBytes: int64(manager.CompletedCount()) * t.Info.PieceLength,
		TotalBytes:      t.Info.Length,
		ActivePeers:     int(atomic.LoadInt32(active)),
		Elapsed:         time.Since(start),
	})
}
