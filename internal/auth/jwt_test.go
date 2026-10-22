package auth

import (
	"testing"
	"time"

	"icecream-api/internal/models"
)

func TestGenerateAndParse(t *testing.T) {
	tm := NewTokenManager("a-very-secret-test-key", time.Hour)
	tok, _, err := tm.Generate(&models.User{ID: 7, Role: models.RoleManager})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != 7 || claims.Role != models.RoleManager || claims.ID == "" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseRejectsWrongSecretAndExpired(t *testing.T) {
	a := NewTokenManager("secret-number-one-123", time.Hour)
	b := NewTokenManager("secret-number-two-456", time.Hour)
	tok, _, _ := a.Generate(&models.User{ID: 1, Role: models.RoleOwner})
	if _, err := b.Parse(tok); err == nil {
		t.Fatal("expected error for wrong secret")
	}

	expired := NewTokenManager("secret-number-one-123", -time.Minute)
	tok, _, _ = expired.Generate(&models.User{ID: 1, Role: models.RoleOwner})
	if _, err := a.Parse(tok); err == nil {
		t.Fatal("expected error for expired token")
	}
	if _, err := a.Parse("garbage"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}
