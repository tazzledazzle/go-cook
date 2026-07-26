package peer

import (
	"bytes"
	"encoding/binary"
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

func TestBuildRequest_roundtrip(t *testing.T) {
	msg := BuildRequest(0, 0, 12)
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	index, begin, length, err := ParseRequest(got)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if index != 0 || begin != 0 || length != 12 {
		t.Errorf("ParseRequest() = (%d, %d, %d), want (0, 0, 12)", index, begin, length)
	}
}

func TestParsePiece_extractsBlock(t *testing.T) {
	block := []byte("hello world\n")
	payload := make([]byte, 8+len(block))
	binary.BigEndian.PutUint32(payload[0:4], 0)
	binary.BigEndian.PutUint32(payload[4:8], 0)
	copy(payload[8:], block)

	index, begin, gotBlock, err := ParsePiece(Message{ID: MsgPiece, Payload: payload})
	if err != nil {
		t.Fatalf("ParsePiece() error = %v", err)
	}
	if index != 0 || begin != 0 {
		t.Errorf("index/begin = (%d, %d), want (0, 0)", index, begin)
	}
	if !bytes.Equal(gotBlock, block) {
		t.Errorf("block = %q, want %q", gotBlock, block)
	}
}
