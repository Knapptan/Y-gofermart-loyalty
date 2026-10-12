package auth

import (
	"testing"
	"time"
)

func TestNewToken_Success(t *testing.T) {
	tok, err := NewToken("secret", "user-1", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == "" {
		t.Fatal("token is empty")
	}
}

func TestParseToken_Valid(t *testing.T) {
	tok, _ := NewToken("secret", "user-42", time.Hour)
	uid, err := ParseToken("secret", tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "user-42" {
		t.Fatalf("got uid %q, want %q", uid, "user-42")
	}
}

func TestParseToken_Expired(t *testing.T) {
	tok, _ := NewToken("secret", "user-1", -time.Hour)
	if _, err := ParseToken("secret", tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	tok, _ := NewToken("secret-A", "user-1", time.Hour)
	if _, err := ParseToken("secret-B", tok); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseToken_EmptyString(t *testing.T) {
	if _, err := ParseToken("secret", ""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	if _, err := ParseToken("secret", "not-a-jwt-token"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}
