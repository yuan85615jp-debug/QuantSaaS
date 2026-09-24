package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// Login posts credentials to SaaS and returns a JWT.
// Broker API keys must never be sent here.
func Login(ctx context.Context, baseURL, email, password string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return "", fmt.Errorf("saas base_url required")
	}
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password required")
	}
	body, err := json.Marshal(LoginRequest{Email: email, Password: password})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/agent/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("login failed status=%d body=%s", resp.StatusCode, truncate(string(raw), 200))
	}
	var out LoginResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("empty token in login response")
	}
	return out.Token, nil
}

func ResolveToken(ctx context.Context, baseURL, email, password, preissued string) (string, error) {
	if preissued != "" {
		return preissued, nil
	}
	return Login(ctx, baseURL, email, password)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
