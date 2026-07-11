package tracker

import (
	"encoding/binary"
	"fmt"
	"net"
)

// ParseCompactPeers parses the tracker's compact peer format (6 bytes per peer).
func ParseCompactPeers(data []byte) ([]Peer, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if len(data)%6 != 0 {
		return nil, fmt.Errorf("compact peers length %d is not a multiple of 6", len(data))
	}

	peers := make([]Peer, 0, len(data)/6)
	for i := 0; i < len(data); i += 6 {
		ip := net.IP(append([]byte(nil), data[i:i+4]...))
		port := binary.BigEndian.Uint16(data[i+4 : i+6])
		peers = append(peers, Peer{IP: ip, Port: port})
	}
	return peers, nil
}
