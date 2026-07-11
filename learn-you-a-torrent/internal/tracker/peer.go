package tracker

import "net"

// Peer is a BitTorrent peer address from the tracker.
type Peer struct {
	IP   net.IP
	Port uint16
}
