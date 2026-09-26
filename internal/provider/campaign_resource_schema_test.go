// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/shopspring/decimal"
)

// Keep schema-stage model referenced until CRUD wires it into lifecycle methods.
var _ = campaignModel{}

func TestCampaignResource_SchemaMutableImmutableClassification(t *testing.T) {
	t.Parallel()

	r := NewCampaignResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	immutable := []string{"adam_id", "countries_or_regions", "supply_sources", "ad_channel_type", "billing_event", "budget_amount", "budget_currency"}
	for _, name := range immutable {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing immutable attribute %q", name)
		}
		desc := attr.GetMarkdownDescription()
		lower := strings.ToLower(desc)
		if !strings.Contains(lower, "immutable") && !strings.Contains(lower, "create-only") {
			t.Fatalf("attribute %q should document immutability/create-only; got %q", name, desc)
		}
	}

	mutable := []string{"name", "status", "daily_budget_amount", "budget_orders", "end_time", "bidding_strategy", "target_cpa_amount"}
	for _, name := range mutable {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing mutable attribute %q", name)
		}
		desc := attr.GetMarkdownDescription()
		if !strings.Contains(strings.ToLower(desc), "mutable") {
			t.Fatalf("attribute %q should document mutability; got %q", name, desc)
		}
	}

	if _, ok := resp.Schema.Attributes["target_cpa_currency"]; !ok {
		t.Fatal("missing target_cpa_currency attribute")
	}

	startTime, ok := resp.Schema.Attributes["start_time"]
	if !ok {
		t.Fatal("missing start_time attribute")
	}
	if !startTime.IsOptional() || !startTime.IsComputed() {
		t.Fatal("start_time should be Optional/Computed")
	}
	if startTime.IsRequired() {
		t.Fatal("start_time must not be required")
	}

	billingEvent, ok := resp.Schema.Attributes["billing_event"]
	if !ok {
		t.Fatal("missing billing_event attribute")
	}
	if !billingEvent.IsOptional() || !billingEvent.IsComputed() {
		t.Fatal("billing_event should be Optional/Computed")
	}

	paymentModel, ok := resp.Schema.Attributes["payment_model"]
	if !ok {
		t.Fatal("missing payment_model attribute")
	}
	if !paymentModel.IsComputed() || paymentModel.IsOptional() || paymentModel.IsRequired() {
		t.Fatal("payment_model must be Computed only (not writable)")
	}
	if strings.Contains(strings.ToLower(paymentModel.GetMarkdownDescription()), "mutable") {
		t.Fatal("payment_model must not be documented as mutable")
	}

	for _, name := range []string{"id", "serving_status", "display_status", "modification_time", "payment_model"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("missing computed attribute %q", name)
		}
	}
}

func TestMoneyAmountRegexpAndDecimal(t *testing.T) {
	t.Parallel()

	valid := []string{"0", "1", "10.00", "0.5", "123.456"}
	for _, v := range valid {
		if !moneyAmountRegexp.MatchString(v) {
			t.Fatalf("expected valid: %q", v)
		}
		if _, err := decimal.NewFromString(v); err != nil {
			t.Fatalf("decimal parse %q: %v", v, err)
		}
	}

	invalid := []string{"", "-1", "01", "abc", "1.", ".5"}
	for _, v := range invalid {
		if moneyAmountRegexp.MatchString(v) {
			t.Fatalf("expected invalid: %q", v)
		}
	}

	// Precision: string round-trip must not introduce float noise.
	d, err := decimal.NewFromString("12.34")
	if err != nil {
		t.Fatal(err)
	}
	if d.StringFixed(2) != "12.34" {
		t.Fatalf("got %s", d.StringFixed(2))
	}
	if regexp.MustCompile(`12\.339+`).MatchString(d.String()) {
		t.Fatal("unexpected float noise")
	}
}
