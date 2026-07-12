package peer

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	MsgKeepalive     uint8 = 255 // sentinel; keepalive has no id byte on the wire
	MsgChoke         uint8 = 0
	MsgUnchoke     uint8 = 1
	MsgInterested  uint8 = 2
	MsgNotInterested uint8 = 3
	MsgHave        uint8 = 4
	MsgBitfield    uint8 = 5
	MsgRequest     uint8 = 6
	MsgPiece       uint8 = 7
	MsgCancel      uint8 = 8
)

// Message is a length-prefixed BitTorrent peer wire message.
type Message struct {
	ID      uint8
	Payload []byte
}

// WriteMessage encodes msg to w using 4-byte big-endian length prefix.
func WriteMessage(w io.Writer, msg Message) error {
	if msg.ID == MsgKeepalive {
		var header [4]byte
		_, err := w.Write(header[:])
		return err
	}

	length := 1 + len(msg.Payload)
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(length))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if _, err := w.Write([]byte{msg.ID}); err != nil {
		return err
	}
	if len(msg.Payload) > 0 {
		if _, err := w.Write(msg.Payload); err != nil {
			return err
		}
	}
	return nil
}

// ReadMessage decodes the next message from r.
func ReadMessage(r io.Reader) (Message, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Message{}, fmt.Errorf("read message length: %w", err)
	}

	length := binary.BigEndian.Uint32(header[:])
	if length == 0 {
		return Message{ID: MsgKeepalive}, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Message{}, fmt.Errorf("read message body: %w", err)
	}

	msg := Message{ID: buf[0]}
	if length > 1 {
		msg.Payload = append([]byte(nil), buf[1:]...)
	}
	return msg, nil
}

// BuildRequest constructs a request message for a piece block.
func BuildRequest(index, begin, length uint32) Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], index)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return Message{ID: MsgRequest, Payload: payload}
}

// ParseRequest extracts index, begin, and length from a request message.
func ParseRequest(msg Message) (index, begin, length uint32, err error) {
	if msg.ID != MsgRequest {
		return 0, 0, 0, fmt.Errorf("peer: expected request message, got id %d", msg.ID)
	}
	if len(msg.Payload) != 12 {
		return 0, 0, 0, fmt.Errorf("peer: request payload length %d, want 12", len(msg.Payload))
	}
	index = binary.BigEndian.Uint32(msg.Payload[0:4])
	begin = binary.BigEndian.Uint32(msg.Payload[4:8])
	length = binary.BigEndian.Uint32(msg.Payload[8:12])
	return index, begin, length, nil
}

// ParsePiece extracts index, begin, and block data from a piece message.
func ParsePiece(msg Message) (index, begin uint32, block []byte, err error) {
	if msg.ID != MsgPiece {
		return 0, 0, nil, fmt.Errorf("peer: expected piece message, got id %d", msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, 0, nil, fmt.Errorf("peer: piece payload too short")
	}
	index = binary.BigEndian.Uint32(msg.Payload[0:4])
	begin = binary.BigEndian.Uint32(msg.Payload[4:8])
	block = append([]byte(nil), msg.Payload[8:]...)
	return index, begin, block, nil
}
