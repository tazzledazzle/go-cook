package bencode

import (
	"os"
	"reflect"
	"testing"
)

func TestDecodeString_tableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "spam", input: "4:spam", want: "spam"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder([]byte(tt.input))
			got, err := d.Decode()
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			s, ok := got.(string)
			if !ok {
				t.Fatalf("Decode() type = %T, want string", got)
			}
			if s != tt.want {
				t.Errorf("Decode() = %q, want %q", s, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  interface{}
	}{
		{name: "string", input: "4:spam", want: "spam"},
		{name: "int positive", input: "i42e", want: int64(42)},
		{name: "int negative", input: "i-3e", want: int64(-3)},
		{name: "list", input: "li1ei2ei3ee", want: []interface{}{int64(1), int64(2), int64(3)}},
		{name: "dict", input: "d3:foo3:bare", want: map[string]interface{}{"foo": "bar"}},
		{name: "nested", input: "li1e4:spame", want: []interface{}{int64(1), "spam"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder([]byte(tt.input))
			got, err := d.Decode()
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecode_invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "missing e on int", input: "i42"},
		{name: "string length mismatch", input: "5:abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder([]byte(tt.input))
			if _, err := d.Decode(); err == nil {
				t.Fatal("Decode() expected error, got nil")
			}
		})
	}
}

func TestInfoDictBytes(t *testing.T) {
	data, err := os.ReadFile("../../testdata/minimal.torrent")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	d := NewDecoder(data)
	if _, err := d.Decode(); err != nil {
		t.Fatalf("Decode(): %v", err)
	}

	info := d.InfoDictBytes()
	if info == nil {
		t.Fatal("InfoDictBytes() = nil")
	}
	if info[0] != 'd' || info[len(info)-1] != 'e' {
		t.Fatalf("InfoDictBytes() = %q, want dict wrapped in d...e", info)
	}

	// Use documented offset from testdata/README.md (avoid false substring match in piece hash bytes).
	const infoOffset = 47
	const infoLength = 83
	if infoOffset+infoLength > len(data) {
		t.Fatalf("torrent file shorter than documented info dict bounds")
	}
	manual := data[infoOffset : infoOffset+infoLength]
	if string(info) != string(manual) {
		t.Errorf("InfoDictBytes() mismatch\n got:  %q\n want: %q", info, manual)
	}
}
