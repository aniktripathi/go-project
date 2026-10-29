package auth

import (
	"testing"
	"time"
)

const secret = "0123456789abcdef0123456789abcdef"

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("Secret123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "Secret123" {
		t.Fatal("password stored in plain text")
	}
	if !CheckPassword(hash, "Secret123") || CheckPassword(hash, "wrong") {
		t.Fatal("password check failed")
	}
}

func TestTokenValid(t *testing.T) {
	tok, err := GenerateToken(secret, 42, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseToken(secret, tok)
	if err != nil || c.UserID != 42 {
		t.Fatalf("got %v, %v", c, err)
	}
}

func TestTokenExpired(t *testing.T) {
	tok, _ := GenerateToken(secret, 1, -time.Minute)
	if _, err := ParseToken(secret, tok); err != ErrExpired {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}

func TestTokenWrongSecretAndGarbage(t *testing.T) {
	tok, _ := GenerateToken(secret, 1, time.Hour)
	if _, err := ParseToken("another-secret-another-secret-123456", tok); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
	if _, err := ParseToken(secret, "not.a.token"); err == nil {
		t.Fatal("garbage must be rejected")
	}
}
