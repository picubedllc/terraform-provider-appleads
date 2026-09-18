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

func TestBudgetOrderModelFromInfo(t *testing.T) {
	t.Parallel()

	info := &client.BudgetOrderInfo{
		OrgIDs: []int64{40669820},
		Bo: &client.BudgetOrder{
			ID:                542370539,
			Name:              "Q1 LOC order",
			StartDate:         "2026-01-01T00:00:00.000",
			EndDate:           "2026-03-31T23:59:59.999",
			Budget:            &client.Money{Amount: "300.00", Currency: "USD"},
			PrimaryBuyerName:  "Example Agency",
			PrimaryBuyerEmail: "buyer@example.com",
			BillingEmail:      "billing@example.com",
			ClientName:        "Example Client",
			OrderNumber:       "PO-1001",
			Status:            "ACTIVE",
			ParentOrgID:       27154130,
			SupplySources:     []string{"APPSTORE_SEARCH_RESULTS", "APPSTORE_SEARCH_TAB"},
		},
	}
	state, diags := budgetOrderModelFromInfo(context.Background(), info)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "542370539" || state.Name.ValueString() != "Q1 LOC order" {
		t.Fatalf("state = %#v", state)
	}
	if state.BudgetAmount.ValueString() != "300.00" || state.BudgetCurrency.ValueString() != "USD" {
		t.Fatalf("budget = %#v", state)
	}
	if state.ParentOrgID.ValueString() != "27154130" || state.Status.ValueString() != "ACTIVE" {
		t.Fatalf("computed = %#v", state)
	}
}

func TestBudgetOrderModelFromInfo_Empty(t *testing.T) {
	t.Parallel()

	_, diags := budgetOrderModelFromInfo(context.Background(), &client.BudgetOrderInfo{})
	if !diags.HasError() {
		t.Fatal("expected error")
	}
}

func TestOverlayBudgetOrderReported_MoneyScale(t *testing.T) {
	t.Parallel()

	configured := budgetOrderModel{
		BudgetAmount:   types.StringValue("300.00"),
		BudgetCurrency: types.StringValue("USD"),
	}
	reported := budgetOrderModel{
		BudgetAmount:   types.StringValue("300"),
		BudgetCurrency: types.StringValue("USD"),
		SupplySources:  types.ListNull(types.StringType),
	}
	out, diags := overlayBudgetOrderReported(context.Background(), configured, reported)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if out.BudgetAmount.ValueString() != "300.00" {
		t.Fatalf("amount = %q", out.BudgetAmount.ValueString())
	}
}

func TestBudgetOrderCreateFromPlan_AndRoundTrip(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/budgetorders" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bo := body["bo"].(map[string]any)
		budget := bo["budget"].(map[string]any)
		if budget["amount"] != "150.25" {
			t.Fatalf("amount = %#v", budget["amount"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{99},
				"bo": map[string]any{
					"id":        77,
					"name":      bo["name"],
					"startDate": bo["startDate"],
					"endDate":   bo["endDate"],
					"budget":    map[string]string{"amount": "150.25", "currency": "USD"},
					"status":    "ACTIVE",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL), client.WithOrgID("99"))
	if err != nil {
		t.Fatal(err)
	}

	plan := budgetOrderModel{
		Name:           types.StringValue("Round trip"),
		BudgetAmount:   types.StringValue("150.25"),
		BudgetCurrency: types.StringValue("USD"),
		StartDate:      types.StringValue("2026-04-01T00:00:00.000"),
		EndDate:        types.StringValue("2026-06-30T23:59:59.999"),
		SupplySources:  types.ListNull(types.StringType),
	}
	in, diags := budgetOrderCreateFromPlan(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	created, err := apiClient.CreateBudgetOrder(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := budgetOrderModelFromInfo(context.Background(), created)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	state, diags = overlayBudgetOrderReported(context.Background(), plan, state)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "77" || state.BudgetAmount.ValueString() != "150.25" {
		t.Fatalf("state = %#v", state)
	}
}

func TestBudgetOrderCreateFromPlan_InvalidMoney(t *testing.T) {
	t.Parallel()

	_, diags := budgetOrderCreateFromPlan(context.Background(), budgetOrderModel{
		Name:          types.StringValue("x"),
		BudgetAmount:  types.StringValue("not-money"),
		StartDate:     types.StringValue("a"),
		EndDate:       types.StringValue("b"),
		SupplySources: types.ListNull(types.StringType),
	})
	if !diags.HasError() {
		t.Fatal("expected error")
	}
}

func TestBudgetOrderImportIDParsing(t *testing.T) {
	t.Parallel()

	id, err := client.ParseBudgetOrderID("542370539")
	if err != nil || id != 542370539 {
		t.Fatalf("got %d %v", id, err)
	}
	_, err = client.ParseBudgetOrderID("bo/542370539")
	if err == nil {
		t.Fatal("expected error for non-numeric import id")
	}
}

func TestBudgetOrderResource_ImportViaGet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/budgetorders/42") {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgIds": []int64{1},
				"bo": map[string]any{
					"id":     42,
					"name":   "imported",
					"budget": map[string]string{"amount": "10.00", "currency": "USD"},
					"status": "ACTIVE",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL), client.WithOrgID("1"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := client.ParseBudgetOrderID("42")
	if err != nil {
		t.Fatal(err)
	}
	got, err := apiClient.GetBudgetOrder(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := budgetOrderModelFromInfo(context.Background(), got)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if state.ID.ValueString() != "42" || state.Name.ValueString() != "imported" {
		t.Fatalf("state = %#v", state)
	}
}

func TestBudgetOrderUpdateFromPlan(t *testing.T) {
	t.Parallel()

	plan := budgetOrderModel{
		ID:             types.StringValue("99"),
		Name:           types.StringValue("updated"),
		BudgetAmount:   types.StringValue("400.50"),
		BudgetCurrency: types.StringValue("USD"),
		StartDate:      types.StringValue("2026-01-01T00:00:00.000"),
		EndDate:        types.StringValue("2026-12-31T23:59:59.999"),
		BillingEmail:   types.StringValue("billing@example.com"),
		SupplySources:  types.ListNull(types.StringType),
	}
	in, diags := budgetOrderUpdateFromPlan(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags = %v", diags)
	}
	if in.Name != "updated" || in.Budget.Amount != "400.50" || in.BillingEmail != "billing@example.com" {
		t.Fatalf("in = %#v", in)
	}
}

func TestApiErrorDiagnostic_BudgetOrder(t *testing.T) {
	t.Parallel()

	err := &client.APIError{StatusCode: 400, Code: "INVALID_NAME", Message: "bad name"}
	diags := apiErrorDiagnostic("Unable to create Apple Ads budget order", err)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "INVALID_NAME") {
		t.Fatalf("detail = %q", diags[0].Detail())
	}
}
