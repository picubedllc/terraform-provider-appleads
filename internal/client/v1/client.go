// SPDX-License-Identifier: MIT

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// DefaultBaseURL is the Apple Ads Platform API v1 base URL.
const DefaultBaseURL = "https://api.ads.apple.com/v1/"

// Client is a reusable Platform API v1 HTTP client.
type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	tokens      client.TokenSource
	userAgent   string
	adAccountID string
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
func WithTokenSource(tokens client.TokenSource) Option {
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

// WithAdAccountID records the ad account ID used for X-AP-Context and diagnostics.
func WithAdAccountID(adAccountID string) Option {
	return func(c *Client) {
		c.adAccountID = strings.TrimSpace(adAccountID)
	}
}

// New creates a Platform API v1 client.
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

// AdAccountID returns the configured ad account ID.
func (c *Client) AdAccountID() string {
	return c.adAccountID
}

func joinAPIPath(base *url.URL, path string) *url.URL {
	out := *base
	out.RawQuery = ""
	out.Fragment = ""
	path = strings.TrimPrefix(path, "/")
	basePath := strings.TrimSuffix(out.Path, "/")
	if path == "" {
		out.Path = basePath
		if out.Path == "" {
			out.Path = "/"
		}
		return &out
	}
	if basePath == "" {
		out.Path = "/" + path
	} else {
		out.Path = basePath + "/" + path
	}
	return &out
}

// DoJSON performs an HTTP request with JSON request/response handling.
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

	u := joinAPIPath(c.baseURL, path)
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

// QueryPage fetches one page from a /query endpoint.
func QueryPage[T any](ctx context.Context, c *Client, path string, q QueryRequest) (*PageResult[T], error) {
	var env ListResponse[T]
	if err := c.DoJSON(ctx, http.MethodPost, path, q, &env); err != nil {
		return nil, err
	}
	return &PageResult[T]{
		Data:       env.Result,
		Offset:     env.Pagination.Offset,
		PageSize:   env.Pagination.PageSize,
		TotalCount: env.Pagination.TotalCount,
	}, nil
}

// QueryAll retrieves every page from a /query endpoint.
func QueryAll[T any](ctx context.Context, c *Client, path string, q QueryRequest, pageSize int) ([]T, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}
	if q.Pagination == nil {
		q.Pagination = &QueryPagination{}
	}
	q.Pagination.PageSize = pageSize
	q.Pagination.Offset = 0

	var all []T
	for {
		page, err := QueryPage[T](ctx, c, path, q)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if len(page.Data) == 0 {
			break
		}
		q.Pagination.Offset += len(page.Data)
		if page.TotalCount > 0 && q.Pagination.Offset >= page.TotalCount {
			break
		}
		if page.TotalCount == 0 && len(page.Data) < pageSize {
			break
		}
	}
	return all, nil
}
