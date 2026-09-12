// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultTokenURL is Apple's OAuth 2 token endpoint.
	DefaultTokenURL = "https://appleid.apple.com/auth/oauth2/token"

	// OAuthAudience is the JWT audience for Apple Ads client assertions.
	OAuthAudience = "https://appleid.apple.com"

	// OAuthScope is the client-credentials scope for Apple Ads org access.
	OAuthScope = "searchadsorg"

	// clientAssertionTTL is how long we mint client-secret JWTs for.
	// Apple allows up to 180 days; we use a short lifetime and remint as needed.
	clientAssertionTTL = 24 * time.Hour

	// refreshSkew refreshes the access token this long before expires_in.
	refreshSkew = 5 * time.Minute
)

// Credentials holds Apple Ads OAuth credentials.
type Credentials struct {
	ClientID   string
	TeamID     string
	KeyID      string
	PrivateKey string // PEM-encoded EC private key
	OrgID      string
}

// OAuthConfig configures an OAuthTokenSource.
type OAuthConfig struct {
	Credentials Credentials
	TokenURL    string
	HTTPClient  *http.Client
	// Now is an injectable clock for tests. Defaults to time.Now.
	Now func() time.Time
}

// OAuthTokenSource implements TokenSource using Apple's client-assertion OAuth flow.
type OAuthTokenSource struct {
	creds      Credentials
	tokenURL   string
	httpClient *http.Client
	now        func() time.Time

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewOAuthTokenSource creates a TokenSource for Apple Ads OAuth.
func NewOAuthTokenSource(cfg OAuthConfig) (*OAuthTokenSource, error) {
	if cfg.Credentials.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if cfg.Credentials.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if cfg.Credentials.KeyID == "" {
		return nil, fmt.Errorf("key_id is required")
	}
	if strings.TrimSpace(cfg.Credentials.PrivateKey) == "" {
		return nil, fmt.Errorf("private_key is required")
	}
	if _, err := parseECPrivateKey(cfg.Credentials.PrivateKey); err != nil {
		return nil, fmt.Errorf("parse private_key: %w", err)
	}

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &OAuthTokenSource{
		creds:      cfg.Credentials,
		tokenURL:   tokenURL,
		httpClient: httpClient,
		now:        now,
	}, nil
}

// Token returns a cached bearer access token, refreshing before expiration.
func (s *OAuthTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && s.now().Before(s.expiresAt.Add(-refreshSkew)) {
		return s.accessToken, nil
	}

	token, expiresIn, err := s.exchange(ctx)
	if err != nil {
		return "", err
	}
	s.accessToken = token
	s.expiresAt = s.now().Add(time.Duration(expiresIn) * time.Second)
	return s.accessToken, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func (s *OAuthTokenSource) exchange(ctx context.Context) (string, int, error) {
	assertion, err := s.clientAssertion()
	if err != nil {
		return "", 0, err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.creds.ClientID)
	form.Set("client_secret", assertion)
	form.Set("scope", OAuthScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Host", "appleid.apple.com")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read token response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", 0, fmt.Errorf("decode token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || tr.AccessToken == "" {
		msg := tr.Error
		if tr.ErrorDesc != "" {
			msg = tr.Error + ": " + tr.ErrorDesc
		}
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return "", 0, fmt.Errorf("token exchange failed: status=%d %s", resp.StatusCode, msg)
	}
	if tr.ExpiresIn <= 0 {
		tr.ExpiresIn = 3600
	}
	return tr.AccessToken, tr.ExpiresIn, nil
}

// ClientAssertion builds an ES256-signed JWT used as the OAuth client_secret.
// Exported for unit tests that verify claims without performing a token exchange.
func (s *OAuthTokenSource) ClientAssertion() (string, error) {
	return s.clientAssertion()
}

func (s *OAuthTokenSource) clientAssertion() (string, error) {
	key, err := parseECPrivateKey(s.creds.PrivateKey)
	if err != nil {
		return "", err
	}

	now := s.now()
	claims := jwt.MapClaims{
		"sub": s.creds.ClientID,
		"aud": OAuthAudience,
		"iss": s.creds.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(clientAssertionTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = s.creds.KeyID
	// Apple's documented examples omit typ; keep header minimal.
	delete(token.Header, "typ")

	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign client assertion: %w", err)
	}
	return signed, nil
}

func parseECPrivateKey(pemData string) (*ecdsa.PrivateKey, error) {
	// Env vars often store PEM with literal \n sequences instead of real newlines.
	pemData = strings.ReplaceAll(strings.TrimSpace(pemData), `\n`, "\n")
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	switch block.Type {
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		key, ok := parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not ECDSA")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported PEM type %q", block.Type)
	}
}
