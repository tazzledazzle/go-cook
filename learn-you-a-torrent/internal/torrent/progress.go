package torrent

import (
	"fmt"
	"time"
)

// Progress summarizes verified download progress for CLI output.
type Progress struct {
	CompletedPieces int
	TotalPieces     int
	DownloadedBytes int64
	TotalBytes      int64
	ActivePeers     int
	Elapsed         time.Duration
}

// String formats progress as percent, speed, and active peer count.
func (p Progress) String() string {
	var pct float64
	if p.TotalPieces > 0 {
		pct = float64(p.CompletedPieces) / float64(p.TotalPieces) * 100
	}

	speed := "0 B/s"
	if p.Elapsed > 0 {
		bytesPerSec := float64(p.DownloadedBytes) / p.Elapsed.Seconds()
		speed = formatSpeed(bytesPerSec)
	}

	return fmt.Sprintf("%.1f%% complete | %s | %d peers", pct, speed, p.ActivePeers)
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.0f B/s", bytesPerSec)
}
