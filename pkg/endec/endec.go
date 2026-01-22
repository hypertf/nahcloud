package endec

import (
	"fmt"
	"math/big"
	"strings"
)

// Standard base32 uses: 0-9 a-v (32 chars)
const stdAlphabet = "0123456789abcdefghijklmnopqrstuv"

// Our alphabet (base32): removed vowels, l33t vowels (0,1,3,4), 6, g/G, k/K, l, n/N, q/Q, x/X, y/Y
const customAlphabet = "25789bcdfhjmprstwzBCDFHJLMPRSTWZ"

var toCustom [256]byte
var toStd [256]byte

func init() {
	for i := 0; i < len(stdAlphabet); i++ {
		toCustom[stdAlphabet[i]] = customAlphabet[i]
		toStd[customAlphabet[i]] = stdAlphabet[i]
	}
}

// Encode converts bytes to base40 string with custom alphabet
func Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	num := new(big.Int).SetBytes(data)
	std := num.Text(32)

	var sb strings.Builder
	sb.Grow(len(std))
	for i := 0; i < len(std); i++ {
		sb.WriteByte(toCustom[std[i]])
	}
	return sb.String()
}

// Decode converts base40 string back to bytes
func Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	// Map back to standard alphabet
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := toStd[s[i]]
		if c == 0 && s[i] != customAlphabet[0] {
			return nil, fmt.Errorf("invalid character: %c", s[i])
		}
		sb.WriteByte(c)
	}

	num := new(big.Int)
	_, ok := num.SetString(sb.String(), 32)
	if !ok {
		return nil, fmt.Errorf("invalid string")
	}

	return num.Bytes(), nil
}
