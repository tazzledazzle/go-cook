package bencode

import (
	"fmt"
	"strconv"
)

// Decoder parses bencode from a byte slice.
type Decoder struct {
	data      []byte
	pos       int
	infoStart int
	infoEnd   int
	dictDepth int
}

// NewDecoder returns a decoder for data.
func NewDecoder(data []byte) *Decoder {
	return &Decoder{data: data, infoStart: -1, infoEnd: -1}
}

// Pos returns the current read position.
func (d *Decoder) Pos() int {
	return d.pos
}

// InfoDictBytes returns the raw bytes of the top-level info dictionary, or nil.
func (d *Decoder) InfoDictBytes() []byte {
	if d.infoStart < 0 || d.infoEnd < 0 {
		return nil
	}
	return d.data[d.infoStart:d.infoEnd]
}

// Decode parses the next bencode value at the current position.
func (d *Decoder) Decode() (interface{}, error) {
	if d.pos >= len(d.data) {
		return nil, fmt.Errorf("bencode: unexpected end of input")
	}
	switch d.data[d.pos] {
	case 'i':
		return d.decodeInt()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	default:
		if d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
			return d.decodeString()
		}
		return nil, fmt.Errorf("bencode: invalid type at offset %d", d.pos)
	}
}

func (d *Decoder) decodeString() (string, error) {
	colon := -1
	for i := d.pos; i < len(d.data); i++ {
		if d.data[i] == ':' {
			colon = i
			break
		}
		if d.data[i] < '0' || d.data[i] > '9' {
			return "", fmt.Errorf("bencode: invalid string length at offset %d", d.pos)
		}
	}
	if colon < 0 {
		return "", fmt.Errorf("bencode: unterminated string length")
	}
	length, err := strconv.Atoi(string(d.data[d.pos:colon]))
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length: %w", err)
	}
	start := colon + 1
	end := start + length
	if end > len(d.data) {
		return "", fmt.Errorf("bencode: string length %d exceeds remaining input", length)
	}
	d.pos = end
	return string(d.data[start:end]), nil
}

func (d *Decoder) decodeInt() (int64, error) {
	if d.data[d.pos] != 'i' {
		return 0, fmt.Errorf("bencode: expected integer")
	}
	d.pos++
	start := d.pos
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		d.pos++
	}
	if d.pos >= len(d.data) {
		return 0, fmt.Errorf("bencode: unterminated integer")
	}
	num, err := strconv.ParseInt(string(d.data[start:d.pos]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bencode: invalid integer: %w", err)
	}
	d.pos++ // consume 'e'
	return num, nil
}

func (d *Decoder) decodeList() ([]interface{}, error) {
	if d.data[d.pos] != 'l' {
		return nil, fmt.Errorf("bencode: expected list")
	}
	d.pos++
	var items []interface{}
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		v, err := d.Decode()
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	if d.pos >= len(d.data) {
		return nil, fmt.Errorf("bencode: unterminated list")
	}
	d.pos++ // consume 'e'
	return items, nil
}

func (d *Decoder) decodeDict() (map[string]interface{}, error) {
	if d.data[d.pos] != 'd' {
		return nil, fmt.Errorf("bencode: expected dictionary")
	}
	d.pos++
	d.dictDepth++
	out := make(map[string]interface{})
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		key, err := d.decodeString()
		if err != nil {
			d.dictDepth--
			return nil, err
		}
		if d.dictDepth == 1 && key == "info" {
			d.infoStart = d.pos
		}
		val, err := d.Decode()
		if err != nil {
			d.dictDepth--
			return nil, err
		}
		if d.dictDepth == 1 && key == "info" {
			d.infoEnd = d.pos
		}
		out[key] = val
	}
	if d.pos >= len(d.data) {
		d.dictDepth--
		return nil, fmt.Errorf("bencode: unterminated dictionary")
	}
	d.pos++ // consume 'e'
	d.dictDepth--
	return out, nil
}
