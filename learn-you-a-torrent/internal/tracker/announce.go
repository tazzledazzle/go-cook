package tracker

import (
	"fmt"
	"net/url"
	"strconv"
)

// AnnounceRequest holds tracker announce query parameters.
type AnnounceRequest struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       uint16
	Uploaded   int64
	Downloaded int64
	Left       int64
}

// BuildAnnounceURL constructs an HTTP GET URL for tracker announce.
func BuildAnnounceURL(base string, req AnnounceRequest) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse announce url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid announce url: %q", base)
	}

	q := url.Values{}
	q.Set("info_hash", string(req.InfoHash[:]))
	q.Set("peer_id", string(req.PeerID[:]))
	q.Set("port", strconv.FormatUint(uint64(req.Port), 10))
	q.Set("uploaded", strconv.FormatInt(req.Uploaded, 10))
	q.Set("downloaded", strconv.FormatInt(req.Downloaded, 10))
	q.Set("left", strconv.FormatInt(req.Left, 10))
	q.Set("compact", "1")
	q.Set("event", "started")

	u.RawQuery = q.Encode()
	return u.String(), nil
}
