package tracker

import (
	"net"
	"reflect"
	"testing"
)

func TestParseCompactPeers(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []Peer
		wantErr bool
	}{
		{
			name:  "single peer 127.0.0.1:6881",
			input: []byte{127, 0, 0, 1, 0x1a, 0xe1},
			want: []Peer{
				{IP: net.IP{127, 0, 0, 1}, Port: 6881},
			},
		},
		{
			name: "two peers",
			input: []byte{
				127, 0, 0, 1, 0x1a, 0xe1,
				192, 168, 0, 1, 0xc9, 0x35,
			},
			want: []Peer{
				{IP: net.IP{127, 0, 0, 1}, Port: 6881},
				{IP: net.IP{192, 168, 0, 1}, Port: 51413},
			},
		},
		{
			name:  "empty",
			input: []byte{},
			want:  nil,
		},
		{
			name:    "malformed length",
			input:   []byte{127, 0, 0, 1, 0x1a},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCompactPeers(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseCompactPeers() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCompactPeers() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCompactPeers() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
