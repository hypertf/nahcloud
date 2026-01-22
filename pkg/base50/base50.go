package base40

import (
	"fmt"
	"math/big"
)

// Alphabet: digits + consonants (no vowels, no l/L)
// 0-9 + bcdfghjkmnpqrstvwxyz + BCDFGHJKMNPQRSTVWXYZ = 50 chars
const alphabet = "29bcdfghjkmnpqrstwxyzBCDFGHJKMNPQRSTWXYZ"

var base = big.NewInt(int64(len(alphabet)))

var decodeMap [256]int

func init() {
	for i := range decodeMap {
		decodeMap[i] = -1
	}
	for i, c := range alphabet {
		decodeMap[c] = i
	}
}

// Encode converts bytes to a base50 string
func Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	num := new(big.Int).SetBytes(data)
	if num.Sign() == 0 {
		// Handle leading zeros
		result := make([]byte, len(data))
		for i := range result {
			result[i] = alphabet[0]
		}
		return string(result)
	}

	var result []byte
	zero := big.NewInt(0)
	mod := new(big.Int)

	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		result = append(result, alphabet[mod.Int64()])
	}

	// Add leading zero chars for leading zero bytes
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append(result, alphabet[0])
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// Decode converts a base50 string back to bytes
func Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	num := big.NewInt(0)
	for _, c := range s {
		if c > 255 || decodeMap[c] == -1 {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
		num.Mul(num, base)
		num.Add(num, big.NewInt(int64(decodeMap[c])))
	}

	result := num.Bytes()

	// Add leading zeros
	leadingZeros := 0
	for _, c := range s {
		if c != rune(alphabet[0]) {
			break
		}
		leadingZeros++
	}

	if leadingZeros > 0 {
		result = append(make([]byte, leadingZeros), result...)
	}

	return result, nil
}
