// SPDX-License-Identifier: MIT

package v1_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
)

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

func TestDoJSON_ResultEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/campaigns/1" {
			t.Errorf("path = %s, want /campaigns/1", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"id": 1, "name": "demo"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL), v1.WithTokenSource(staticToken("tok")))
	if err != nil {
		t.Fatal(err)
	}

	var env v1.Response[map[string]any]
	if err := c.DoJSON(context.Background(), http.MethodGet, "campaigns/1", nil, &env); err != nil {
		t.Fatal(err)
	}
	if env.Result["name"] != "demo" {
		t.Fatalf("result = %#v", env.Result)
	}
}

func TestQueryPage_FiltersAndPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/query" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"filters"`) {
			t.Fatalf("body missing filters: %s", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{{"id": 9}},
			"pagination": map[string]any{
				"offset": 0, "pageSize": 100, "totalCount": 1,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	page, err := v1.QueryPage[map[string]any](context.Background(), c, "campaigns/query", v1.QueryRequest{
		Filters: []v1.QueryFilter{v1.EqualsFilter("id", 9)},
		Pagination: &v1.QueryPagination{Offset: 0, PageSize: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.TotalCount != 1 {
		t.Fatalf("page = %#v", page)
	}
}

func TestAuthTransport_AdAccountContext(t *testing.T) {
	t.Parallel()

	var gotContext string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContext = r.Header.Get("X-AP-Context")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	httpClient := v1.NewAuthenticatedHTTPClient(staticToken("tok"), "4242", srv.Client().Transport)
	c, err := v1.New(v1.WithBaseURL(srv.URL), v1.WithHTTPClient(httpClient), v1.WithAdAccountID("4242"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, "campaigns/1", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotContext != "adAccountId=4242" {
		t.Fatalf("X-AP-Context = %q", gotContext)
	}
}

func TestAuthTransport_SkipsContextOnACLs(t *testing.T) {
	t.Parallel()

	var gotContext string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContext = r.Header.Get("X-AP-Context")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{}})
	}))
	t.Cleanup(srv.Close)

	httpClient := v1.NewAuthenticatedHTTPClient(staticToken("tok"), "4242", srv.Client().Transport)
	c, err := v1.New(v1.WithBaseURL(srv.URL), v1.WithHTTPClient(httpClient), v1.WithAdAccountID("4242"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListUserACLs(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotContext != "" {
		t.Fatalf("expected empty context on /acls, got %q", gotContext)
	}
}

func TestParseAPIError_PlatformDetails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "Validation errors found",
				"details": []map[string]any{{
					"code":    "DUPLICATE_NAME",
					"message": "name already exists",
					"info":    map[string]any{"field": "name"},
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(context.Background(), http.MethodPost, "campaigns", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*v1.APIError)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if apiErr.Code != "DUPLICATE_NAME" {
		t.Fatalf("code = %s", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "name") {
		t.Fatalf("message = %s", apiErr.Message)
	}
}

func TestCheckAdAccountID(t *testing.T) {
	t.Parallel()
	accounts := []v1.AdAccount{{ID: 10, Name: "A"}, {ID: 20, DisplayName: "B"}}
	if err := v1.CheckAdAccountID("20", accounts); err != nil {
		t.Fatal(err)
	}
	err := v1.CheckAdAccountID("99", accounts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Do not reuse org_id") {
		t.Fatalf("message = %s", err.Error())
	}
}

func TestDefaultBaseURL(t *testing.T) {
	t.Parallel()
	if v1.DefaultBaseURL != "https://api.ads.apple.com/v1/" {
		t.Fatalf("DefaultBaseURL = %s", v1.DefaultBaseURL)
	}
}
