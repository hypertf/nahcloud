package endec

import (
	"bytes"
	"testing"
)

func TestAlphabetLength(t *testing.T) {
	if len(customAlphabet) != 32 {
		t.Errorf("expected 32, got %d", len(customAlphabet))
	}
	if len(stdAlphabet) != 32 {
		t.Errorf("expected 32, got %d", len(stdAlphabet))
	}
}

func TestRoundTrip(t *testing.T) {
	tests := [][]byte{
		{1},
		{255},
		{1, 2, 3, 4, 5},
		[]byte("hello world"),
		{0xff, 0xfe, 0xfd},
	}

	for _, input := range tests {
		encoded := Encode(input)
		decoded, err := Decode(encoded)
		if err != nil {
			t.Errorf("decode error for %v: %v", input, err)
			continue
		}
		if !bytes.Equal(input, decoded) {
			t.Errorf("roundtrip failed: input=%v encoded=%s decoded=%v", input, encoded, decoded)
		}
	}
}

func TestEmpty(t *testing.T) {
	if Encode(nil) != "" {
		t.Error("expected empty")
	}
	d, _ := Decode("")
	if len(d) != 0 {
		t.Error("expected empty")
	}
}

func TestInvalidChar(t *testing.T) {
	_, err := Decode("abc!")
	if err == nil {
		t.Error("expected error")
	}
}

func TestOutput(t *testing.T) {
	data := []byte("test")
	encoded := Encode(data)
	t.Logf("'test' -> %s", encoded)
}
