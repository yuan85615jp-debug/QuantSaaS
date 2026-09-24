package auth

import (
	"testing"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
)

func TestSignAndParse(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{Secret: "test-secret-please-change", ExpireHour: 1},
	}
	svc := NewService(cfg)
	tok, err := svc.SignToken(42, "user")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ParseToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.Role != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestInvalidToken(t *testing.T) {
	svc := NewService(&config.Config{
		JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1},
	})
	_, err := svc.ParseToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
