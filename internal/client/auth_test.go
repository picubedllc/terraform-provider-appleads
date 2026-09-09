// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func testECPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func TestClientAssertionJWTClaims(t *testing.T) {
	t.Parallel()

	pemKey := testECPrivateKeyPEM(t)
	fixed := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	src, err := client.NewOAuthTokenSource(client.OAuthConfig{
		Credentials: client.Credentials{
			ClientID:   "SEARCHADS.client",
			TeamID:     "SEARCHADS.team",
			KeyID:      "key-123",
			PrivateKey: pemKey,
		},
		Now: func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatal(err)
	}

	assertion, err := src.ClientAssertion()
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := jwt.Parse(assertion, func(token *jwt.Token) (any, error) {
		block, _ := pem.Decode([]byte(pemKey))
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return &key.PublicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}

	if alg, _ := parsed.Header["alg"].(string); alg != "ES256" {
		t.Errorf("alg = %v", parsed.Header["alg"])
	}
	if kid, _ := parsed.Header["kid"].(string); kid != "key-123" {
		t.Errorf("kid = %v", parsed.Header["kid"])
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims type")
	}
	if claims["sub"] != "SEARCHADS.client" {
		t.Errorf("sub = %v", claims["sub"])
	}
	if claims["iss"] != "SEARCHADS.team" {
		t.Errorf("iss = %v", claims["iss"])
	}
	if claims["aud"] != client.OAuthAudience {
		t.Errorf("aud = %v", claims["aud"])
	}
	if int64(claims["iat"].(float64)) != fixed.Unix() {
		t.Errorf("iat = %v", claims["iat"])
	}
	if int64(claims["exp"].(float64)) != fixed.Add(24*time.Hour).Unix() {
		t.Errorf("exp = %v", claims["exp"])
	}
}

func TestOAuthTokenExchangeAndCache(t *testing.T) {
	t.Parallel()

	var exchanges atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exchanges.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type = %q", ct)
		}
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("scope") != client.OAuthScope {
			t.Errorf("scope = %q", r.Form.Get("scope"))
		}
		if r.Form.Get("client_id") != "SEARCHADS.client" {
			t.Errorf("client_id = %q", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") == "" {
			t.Error("missing client_secret")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-abc",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"scope":        "searchadsorg",
		})
	}))
	t.Cleanup(srv.Close)

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	src, err := client.NewOAuthTokenSource(client.OAuthConfig{
		Credentials: client.Credentials{
			ClientID:   "SEARCHADS.client",
			TeamID:     "SEARCHADS.team",
			KeyID:      "key-123",
			PrivateKey: testECPrivateKeyPEM(t),
		},
		TokenURL:   srv.URL,
		HTTPClient: srv.Client(),
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}

	tok1, err := src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok1 != "access-abc" {
		t.Fatalf("token = %q", tok1)
	}

	tok2, err := src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok2 != tok1 {
		t.Fatalf("cached token mismatch")
	}
	if exchanges.Load() != 1 {
		t.Fatalf("exchanges = %d, want 1 (cache hit)", exchanges.Load())
	}

	// Advance past refresh skew window (expires_in 3600 - 5m skew).
	now = now.Add(56 * time.Minute)
	_, err = src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if exchanges.Load() != 2 {
		t.Fatalf("exchanges = %d, want 2 after proactive refresh", exchanges.Load())
	}
}
