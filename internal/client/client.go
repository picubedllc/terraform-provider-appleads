// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

// Package client is a standalone Apple Ads API HTTP client.
// It has no dependency on Terraform types or the Plugin Framework.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the Apple Ads Campaign Management API v5 base URL.
const DefaultBaseURL = "https://api.searchads.apple.com/api/v5"

// TokenSource provides bearer access tokens for authenticated requests.
// Auth implementations (OAuth) satisfy this interface independently of the client.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Client is a reusable Apple Ads API HTTP client.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	tokens     TokenSource
	userAgent  string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the API base URL (useful for httptest.Server in tests).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		u, err := url.Parse(strings.TrimRight(baseURL, "/"))
		if err == nil {
			c.baseURL = u
		}
	}
}

// WithHTTPClient sets the underlying HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithTokenSource sets the bearer token source used for Authorization headers.
func WithTokenSource(tokens TokenSource) Option {
	return func(c *Client) {
		c.tokens = tokens
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// New creates an Apple Ads API client.
func New(opts ...Option) (*Client, error) {
	base, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse default base URL: %w", err)
	}

	c := &Client{
		baseURL:    base,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		userAgent:  "terraform-provider-appleads",
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.baseURL == nil {
		return nil, fmt.Errorf("invalid base URL")
	}
	return c, nil
}

// DoJSON performs an HTTP request with JSON request/response handling.
// body may be nil. out may be nil when no response body is expected.
func (c *Client) DoJSON(ctx context.Context, method, path string, body any, out any) error {
	return c.do(ctx, method, path, nil, body, out)
}

// DoJSONWithQuery is like DoJSON but appends query parameters.
func (c *Client) DoJSONWithQuery(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	return c.do(ctx, method, path, query, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}
	u := c.baseURL.ResolveReference(rel)
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.tokens != nil {
		token, err := c.tokens.Token(ctx)
		if err != nil {
			return fmt.Errorf("get access token: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("unmarshal response body: %w", err)
	}
	return nil
}

// PageParams describes offset pagination used by Apple Ads list endpoints.
type PageParams struct {
	Limit  int
	Offset int
}

// PageResult is a single page of items plus pagination metadata.
type PageResult[T any] struct {
	Data       []T
	TotalCount int
	StartIndex int
	ItemsPerPage int
}

// appleListEnvelope matches Apple Ads paginated list responses.
type appleListEnvelope[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		TotalResults int `json:"totalResults"`
		StartIndex   int `json:"startIndex"`
		ItemsPerPage int `json:"itemsPerPage"`
	} `json:"pagination"`
}

// FetchPage retrieves one page of a list endpoint.
// path should not include query parameters; limit/offset are applied as query values.
func FetchPage[T any](ctx context.Context, c *Client, method, path string, page PageParams, body any) (*PageResult[T], error) {
	query := url.Values{}
	if page.Limit > 0 {
		query.Set("limit", strconv.Itoa(page.Limit))
	}
	if page.Offset > 0 {
		query.Set("offset", strconv.Itoa(page.Offset))
	}

	var env appleListEnvelope[T]
	if err := c.do(ctx, method, path, query, body, &env); err != nil {
		return nil, err
	}
	return &PageResult[T]{
		Data:         env.Data,
		TotalCount:   env.Pagination.TotalResults,
		StartIndex:   env.Pagination.StartIndex,
		ItemsPerPage: env.Pagination.ItemsPerPage,
	}, nil
}

// FetchAllPages retrieves every page from a list endpoint.
// pageSize defaults to 1000 when <= 0.
func FetchAllPages[T any](ctx context.Context, c *Client, method, path string, pageSize int, body any) ([]T, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	var all []T
	offset := 0
	for {
		page, err := FetchPage[T](ctx, c, method, path, PageParams{Limit: pageSize, Offset: offset}, body)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Data...)

		if len(page.Data) == 0 {
			break
		}
		offset += len(page.Data)
		if page.TotalCount > 0 && offset >= page.TotalCount {
			break
		}
		if page.TotalCount == 0 && len(page.Data) < pageSize {
			break
		}
	}
	return all, nil
}
