package base50

import (
	"bytes"
	"testing"
)

func TestAlphabetLength(t *testing.T) {
	if len(alphabet) != 50 {
		t.Errorf("expected alphabet length 50, got %d", len(alphabet))
	}
}

func TestAlphabetNoVowelsOrL(t *testing.T) {
	forbidden := "aeiouAEIOUlL"
	for _, c := range forbidden {
		for _, a := range alphabet {
			if c == a {
				t.Errorf("alphabet contains forbidden character: %c", c)
			}
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := [][]byte{
		{},
		{0},
		{0, 0, 0},
		{1},
		{255},
		{1, 2, 3, 4, 5},
		[]byte("hello world"),
		{0, 1, 2, 3},
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

func TestDecodeInvalidChar(t *testing.T) {
	_, err := Decode("abc!")
	if err == nil {
		t.Error("expected error for invalid character")
	}
}

func TestEncodeProducesReadableOutput(t *testing.T) {
	data := []byte("test data 123")
	encoded := Encode(data)
	t.Logf("encoded: %s", encoded)

	// Verify no vowels or l
	for _, c := range encoded {
		if c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u' ||
			c == 'A' || c == 'E' || c == 'I' || c == 'O' || c == 'U' ||
			c == 'l' || c == 'L' {
			t.Errorf("encoded contains forbidden char: %c", c)
		}
	}
}
