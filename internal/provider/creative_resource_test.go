// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestCreativeModelFromClient(t *testing.T) {
	t.Parallel()

	got := &client.Creative{
		ID:               573408745,
		OrgID:            39872140,
		AdamID:           899247664,
		Name:             "Trip Trek CPP",
		Type:             client.CreativeTypeCustomProductPage,
		State:            "VALID",
		StateReasons:     []string{},
		ProductPageID:    "76659d7a-d146-43d3-b6b8-b7a12f74bf6b",
		CreationTime:     "2024-10-09T20:07:19.506Z",
		ModificationTime: "2024-10-18T20:07:19.506Z",
	}
	state, diags := creativeModelFromClient(context.Background(), got)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "573408745" || state.AdamID.ValueString() != "899247664" {
		t.Fatalf("ids = %#v", state)
	}
	if state.ProductPageID.ValueString() != "76659d7a-d146-43d3-b6b8-b7a12f74bf6b" {
		t.Fatalf("product_page_id = %q", state.ProductPageID.ValueString())
	}
	if state.Type.ValueString() != client.CreativeTypeCustomProductPage || state.State.ValueString() != "VALID" {
		t.Fatalf("type/state = %#v", state)
	}
}

func TestCreativeCreateFromPlan_RequiresProductPage(t *testing.T) {
	t.Parallel()

	_, diags := creativeCreateFromPlan(creativeModel{
		AdamID: types.StringValue("1"),
		Name:   types.StringValue("n"),
		Type:   types.StringValue(client.CreativeTypeCustomProductPage),
	})
	if !diags.HasError() {
		t.Fatal("expected error")
	}
}

func TestCreativeCreate_AndStateFromResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/creatives" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body client.CreativeCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.ProductPageID == "" || body.Type != client.CreativeTypeCustomProductPage {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":            55,
				"orgId":         1,
				"adamId":        body.AdamID,
				"name":          body.Name,
				"type":          body.Type,
				"state":         "VALID",
				"productPageId": body.ProductPageID,
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	in, diags := creativeCreateFromPlan(creativeModel{
		AdamID:        types.StringValue("899247664"),
		Name:          types.StringValue("CPP"),
		Type:          types.StringValue(client.CreativeTypeCustomProductPage),
		ProductPageID: types.StringValue("45812c9b-c296-43d3-c6a0-c5a02f74bf6e"),
	})
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	created, err := apiClient.CreateCreative(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := creativeModelFromClient(context.Background(), created)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "55" {
		t.Fatalf("state = %#v", state)
	}
}

func TestCreativeUpdate_ImmutableMessages(t *testing.T) {
	t.Parallel()

	state := creativeModel{
		ID:            types.StringValue("1"),
		AdamID:        types.StringValue("10"),
		Name:          types.StringValue("a"),
		Type:          types.StringValue(client.CreativeTypeCustomProductPage),
		ProductPageID: types.StringValue("pp-1"),
	}
	plan := creativeModel{
		ID:            types.StringValue("1"),
		AdamID:        types.StringValue("11"),
		Name:          types.StringValue("a"),
		Type:          types.StringValue(client.CreativeTypeCustomProductPage),
		ProductPageID: types.StringValue("pp-1"),
	}
	diags := rejectImmutableCreativeChanges(state, plan)
	if !diags.HasError() || !strings.Contains(diags[0].Summary(), "adam_id") {
		t.Fatalf("diags = %v", diags)
	}

	plan = state
	plan.Name = types.StringValue("b")
	diags = rejectImmutableCreativeChanges(state, plan)
	if !diags.HasError() || !strings.Contains(diags[0].Summary(), "name") {
		t.Fatalf("diags = %v", diags)
	}

	plan = state
	plan.ProductPageID = types.StringValue("pp-2")
	diags = rejectImmutableCreativeChanges(state, plan)
	if !diags.HasError() || !strings.Contains(diags[0].Summary(), "product_page_id") {
		t.Fatalf("diags = %v", diags)
	}
}

func TestCreativeCreate_APIValidationDiagnostic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "productPageId is invalid",
					"field":       "productPageId",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.CreateCreative(context.Background(), &client.CreativeCreate{
		AdamID: 1, Name: "x", Type: client.CreativeTypeCustomProductPage, ProductPageID: "bad",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	diags := apiErrorDiagnostic("Unable to create Apple Ads creative", err)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "productPageId") {
		t.Fatalf("detail = %v", diags)
	}
}

func TestCreativeRead_NotFoundRemoves(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{"messageCode": "NOT_FOUND", "message": "gone"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.GetCreative(context.Background(), 99)
	if !client.IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}
