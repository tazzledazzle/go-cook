package tracker

import (
	"fmt"
	"io"
	"net/http"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/bencode"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

// Client announces to HTTP trackers and parses peer lists.
type Client struct {
	PeerID     [20]byte
	HTTPClient *http.Client
}

// NewClient creates a tracker client with the given 20-byte peer ID.
func NewClient(peerID [20]byte, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{PeerID: peerID, HTTPClient: httpClient}
}

// Announce contacts the torrent's tracker and returns discovered peers.
func (c *Client) Announce(t *torrent.Torrent) ([]Peer, error) {
	req := AnnounceRequest{
		InfoHash:   t.InfoHash,
		PeerID:     c.PeerID,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       t.Info.Length,
	}

	announceURL, err := BuildAnnounceURL(t.Announce, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Get(announceURL)
	if err != nil {
		return nil, fmt.Errorf("tracker GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tracker response: %w", err)
	}

	return decodePeersResponse(body)
}

func decodePeersResponse(body []byte) ([]Peer, error) {
	dec := bencode.NewDecoder(body)
	root, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("decode tracker response: %w", err)
	}
	top, ok := root.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tracker response root is not a dictionary")
	}
	peersRaw, ok := top["peers"].(string)
	if !ok {
		return nil, fmt.Errorf("tracker response missing compact peers string")
	}
	return ParseCompactPeers([]byte(peersRaw))
}
