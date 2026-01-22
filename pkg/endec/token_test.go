package endec

import (
	"strings"
	"testing"
)

func TestTokenFormat(t *testing.T) {
	token, err := CreateToken(PrefixSession, 16)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	if !strings.HasPrefix(token, "nah_ses_") {
		t.Errorf("token should start with 'nah_ses_', got: %s", token)
	}

	t.Logf("token: %s", token)
}

func TestTokenRoundTrip(t *testing.T) {
	prefixes := []string{
		PrefixSession,
		PrefixAPI,
		PrefixRefresh,
		PrefixInvite,
		PrefixReset,
		PrefixVerify,
		PrefixWebhook,
	}

	for _, prefix := range prefixes {
		token, err := CreateToken(prefix, 16)
		if err != nil {
			t.Fatalf("CreateToken failed for prefix %s: %v", prefix, err)
		}

		gotPrefix, data, err := ParseToken(token)
		if err != nil {
			t.Fatalf("ParseToken failed: %v", err)
		}

		if gotPrefix != prefix {
			t.Errorf("prefix mismatch: expected %s, got %s", prefix, gotPrefix)
		}

		if len(data) != 16 {
			t.Errorf("data length mismatch: expected 16, got %d", len(data))
		}
	}
}

func TestTokenFromData(t *testing.T) {
	data := []byte("hello world test")
	token, err := CreateTokenFromData(PrefixAPI, data)
	if err != nil {
		t.Fatalf("CreateTokenFromData failed: %v", err)
	}

	prefix, decoded, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if prefix != PrefixAPI {
		t.Errorf("prefix mismatch")
	}

	if string(decoded) != string(data) {
		t.Errorf("data mismatch: expected %q, got %q", data, decoded)
	}
}

func TestValidateToken(t *testing.T) {
	token, _ := CreateToken(PrefixSession, 16)

	// Valid prefix
	_, err := ValidateToken(token, PrefixSession)
	if err != nil {
		t.Errorf("ValidateToken should succeed: %v", err)
	}

	// Wrong prefix
	_, err = ValidateToken(token, PrefixAPI)
	if err == nil {
		t.Errorf("ValidateToken should fail with wrong prefix")
	}
}

func TestInvalidTokens(t *testing.T) {
	tests := []string{
		"invalid",
		"nah_",
		"nah_ab",
		"nah_abcd_payload",
		"foo_ses_abc",
	}

	for _, token := range tests {
		_, _, err := ParseToken(token)
		if err == nil {
			t.Errorf("ParseToken should fail for: %s", token)
		}
	}
}

func TestInvalidPrefix(t *testing.T) {
	_, err := CreateToken("ab", 16)
	if err == nil {
		t.Error("should fail with 2-char prefix")
	}

	_, err = CreateToken("abcd", 16)
	if err == nil {
		t.Error("should fail with 4-char prefix")
	}
}
