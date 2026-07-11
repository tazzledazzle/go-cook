package peer

import (
	"bytes"
	"testing"
)

func TestWriteMessage_roundtrip(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantID  uint8
		wantLen int
	}{
		{
			name:    "keepalive",
			msg:     Message{ID: MsgKeepalive},
			wantID:  MsgKeepalive,
			wantLen: 0,
		},
		{
			name:    "choke",
			msg:     Message{ID: MsgChoke},
			wantID:  MsgChoke,
			wantLen: 1,
		},
		{
			name:    "unchoke",
			msg:     Message{ID: MsgUnchoke},
			wantID:  MsgUnchoke,
			wantLen: 1,
		},
		{
			name:    "interested",
			msg:     Message{ID: MsgInterested},
			wantID:  MsgInterested,
			wantLen: 1,
		},
		{
			name:    "have",
			msg:     Message{ID: MsgHave, Payload: []byte{0, 0, 0, 5}},
			wantID:  MsgHave,
			wantLen: 5,
		},
		{
			name:    "bitfield",
			msg:     Message{ID: MsgBitfield, Payload: []byte{0xff, 0x00}},
			wantID:  MsgBitfield,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteMessage(&buf, tt.msg); err != nil {
				t.Fatalf("WriteMessage() error = %v", err)
			}

			got, err := ReadMessage(&buf)
			if err != nil {
				t.Fatalf("ReadMessage() error = %v", err)
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", got.ID, tt.wantID)
			}
			if !bytes.Equal(got.Payload, tt.msg.Payload) {
				t.Errorf("Payload = %v, want %v", got.Payload, tt.msg.Payload)
			}
		})
	}
}

func TestReadMessage_incompleteData(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0, 0, 0, 5, MsgHave})
	_, err := ReadMessage(buf)
	if err == nil {
		t.Fatal("ReadMessage() expected error for truncated stream, got nil")
	}
}
