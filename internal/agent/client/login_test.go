package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/agent/login", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var req LoginRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "agent@x.com", req.Email)
		_ = json.NewEncoder(w).Encode(LoginResponse{Token: "jwt-abc"})
	}))
	defer srv.Close()

	tok, err := Login(context.Background(), srv.URL, "agent@x.com", "secret")
	require.NoError(t, err)
	require.Equal(t, "jwt-abc", tok)
}

func TestResolveTokenPrefersPreissued(t *testing.T) {
	tok, err := ResolveToken(context.Background(), "", "", "", "ready-token")
	require.NoError(t, err)
	require.Equal(t, "ready-token", tok)
}

func TestLoginMissingFields(t *testing.T) {
	_, err := Login(context.Background(), "http://localhost", "", "")
	require.Error(t, err)
}
