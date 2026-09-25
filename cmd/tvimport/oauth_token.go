package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func exchangeCode(ctx context.Context, tokenURL, clientID, code, redirectURI, verifier, resource string) (*tokenBundle, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("code_verifier", verifier)
	if resource != "" {
		form.Set("resource", resource)
	}
	return postToken(ctx, tokenURL, form)
}

func refreshTokens(ctx context.Context, tokenURL, clientID, refresh, resource string) (*tokenBundle, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	form.Set("client_id", clientID)
	if resource != "" {
		form.Set("resource", resource)
	}
	return postToken(ctx, tokenURL, form)
}

func postToken(ctx context.Context, tokenURL string, form url.Values) (*tokenBundle, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 400))
	}
	var b tokenBundle
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, err
	}
	if b.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token: %s", truncate(string(body), 200))
	}
	return &b, nil
}
