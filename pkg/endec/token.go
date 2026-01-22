package endec

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// Token prefixes (3 chars)
const (
	PrefixSession = "ses" // session token
	PrefixAPI     = "api" // api key
	PrefixRefresh = "ref" // refresh token
	PrefixInvite  = "inv" // invite token
	PrefixReset   = "rst" // password reset
	PrefixVerify  = "vfy" // email verification
	PrefixWebhook = "whk" // webhook secret
)

// CreateToken generates a nah token with the given prefix and random payload
func CreateToken(prefix string, payloadBytes int) (string, error) {
	if len(prefix) != 3 {
		return "", fmt.Errorf("prefix must be 3 characters")
	}
	if payloadBytes <= 0 {
		return "", fmt.Errorf("payload must be at least 1 byte")
	}

	payload := make([]byte, payloadBytes)
	if _, err := rand.Read(payload); err != nil {
		return "", fmt.Errorf("failed to generate random payload: %w", err)
	}

	return fmt.Sprintf("nah_%s_%s", prefix, Encode(payload)), nil
}

// CreateTokenFromData creates a token from existing data
func CreateTokenFromData(prefix string, data []byte) (string, error) {
	if len(prefix) != 3 {
		return "", fmt.Errorf("prefix must be 3 characters")
	}
	return fmt.Sprintf("nah_%s_%s", prefix, Encode(data)), nil
}

// ParseToken parses a nah token and returns its prefix and decoded payload
func ParseToken(token string) (prefix string, data []byte, err error) {
	if !strings.HasPrefix(token, "nah_") {
		return "", nil, fmt.Errorf("invalid token prefix")
	}

	parts := strings.SplitN(token[4:], "_", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid token format")
	}

	prefix = parts[0]
	if len(prefix) != 3 {
		return "", nil, fmt.Errorf("invalid token prefix length")
	}

	data, err = Decode(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("invalid token payload: %w", err)
	}

	return prefix, data, nil
}

// ValidateToken checks if a token is valid and matches the expected prefix
func ValidateToken(token string, expectedPrefix string) ([]byte, error) {
	prefix, data, err := ParseToken(token)
	if err != nil {
		return nil, err
	}

	if prefix != expectedPrefix {
		return nil, fmt.Errorf("token prefix mismatch: expected %s, got %s", expectedPrefix, prefix)
	}

	return data, nil
}
