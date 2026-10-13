package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_Success(t *testing.T) {
	h, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == "" {
		t.Fatal("hash is empty")
	}
	if !strings.HasPrefix(h, "$2a$") && !strings.HasPrefix(h, "$2b$") {
		t.Fatalf("unexpected hash prefix: %q", h)
	}
}

func TestCheckPassword_Valid(t *testing.T) {
	h, _ := HashPassword("correct")
	if !CheckPassword(h, "correct") {
		t.Fatal("expected true for matching password")
	}
}

func TestCheckPassword_Invalid(t *testing.T) {
	h, _ := HashPassword("correct")
	if CheckPassword(h, "wrong") {
		t.Fatal("expected false for wrong password")
	}
}

func TestHashPassword_DifferentPerRun(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Fatal("bcrypt hashes must differ due to salt")
	}
}
