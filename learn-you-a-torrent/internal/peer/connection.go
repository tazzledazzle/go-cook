package peer

import "net"

// Connection wraps a TCP (or pipe) connection after peer wire handshake.
type Connection struct {
	conn     net.Conn
	infoHash [20]byte
	handshake Handshake
}

// NewConnection creates a peer connection that has not yet completed handshake.
func NewConnection(conn net.Conn, expectedHash [20]byte, ours Handshake) *Connection {
	return &Connection{
		conn:     conn,
		infoHash: expectedHash,
		handshake: ours,
	}
}

// PerformHandshake exchanges the BitTorrent handshake and verifies info_hash.
func (c *Connection) PerformHandshake() error {
	peer, err := ExchangeHandshake(c.conn, c.infoHash, c.handshake)
	if err != nil {
		return err
	}
	c.handshake = peer
	return nil
}

// ReadMessage reads the next length-prefixed message from the peer.
func (c *Connection) ReadMessage() (Message, error) {
	return ReadMessage(c.conn)
}
