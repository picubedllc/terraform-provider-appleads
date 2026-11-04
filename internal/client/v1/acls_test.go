// SPDX-License-Identifier: MIT

package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/client/v1"
)

func TestListAdAccounts_Query(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adaccounts/query" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{
				{"id": 111, "orgId": 9, "name": "Primary"},
			},
			"pagination": map[string]any{"offset": 0, "pageSize": 1000, "totalCount": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL), v1.WithAdAccountID("111"))
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := c.ListAdAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].ID != 111 {
		t.Fatalf("accounts = %#v", accounts)
	}
	if err := c.ValidateAdAccountID(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestGetAdAccount(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adaccounts/55" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"id": 55, "currency": "USD"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := v1.New(v1.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	acct, err := c.GetAdAccount(context.Background(), 55)
	if err != nil {
		t.Fatal(err)
	}
	if acct.Currency != "USD" {
		t.Fatalf("%#v", acct)
	}
}
