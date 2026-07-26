package torrent

import (
	"strings"
	"testing"
	"time"
)

func TestProgress_string(t *testing.T) {
	tests := []struct {
		name    string
		progress Progress
		wantContains []string
	}{
		{
			name: "zero complete one peer",
			progress: Progress{
				CompletedPieces: 0,
				TotalPieces:     1,
				DownloadedBytes: 0,
				TotalBytes:      12,
				ActivePeers:     1,
				Elapsed:         time.Second,
			},
			wantContains: []string{"0.0%", "1 peers"},
		},
		{
			name: "complete two peers",
			progress: Progress{
				CompletedPieces: 1,
				TotalPieces:     1,
				DownloadedBytes: 12,
				TotalBytes:      12,
				ActivePeers:     2,
				Elapsed:         2 * time.Second,
			},
			wantContains: []string{"100.0%", "2 peers", "6 B/s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.progress.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("String() = %q, want substring %q", got, want)
				}
			}
		})
	}
}
