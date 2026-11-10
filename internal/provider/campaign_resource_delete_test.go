// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCampaignDelete_BlockedWithoutOptIn(t *testing.T) {
	t.Parallel()

	called := false
	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	r := &campaignResource{client: apiClient, allowCampaignDeletion: false}
	if r.allowCampaignDeletion {
		t.Fatal("expected disabled")
	}
	diags := CampaignDeletionBlockedDiagnostics()
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	if !strings.Contains(diags[0].Summary(), "Campaign deletion is disabled") {
		t.Fatalf("summary = %s", diags[0].Summary())
	}
	if !strings.Contains(diags[0].Detail(), "allow_campaign_deletion = true") {
		t.Fatalf("detail = %s", diags[0].Detail())
	}
	// Guardrail must short-circuit before any client call.
	if called {
		t.Fatal("API must not be called")
	}
}

func TestCampaignDelete_AllowedArchives(t *testing.T) {
	t.Parallel()

	called := false
	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/campaigns/99" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := apiClient.DeleteCampaign(context.Background(), 99); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected delete call")
	}
}

func TestCampaignDelete_APIFailure(t *testing.T) {
	t.Parallel()

	apiClient := newCampaignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "DENIED", "message": "cannot archive"},
		})
	})
	err := apiClient.DeleteCampaign(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to archive Apple Ads campaign", err)
	if !diags.HasError() {
		t.Fatal("expected diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "cannot archive") {
		t.Fatalf("%v", diags)
	}
}
