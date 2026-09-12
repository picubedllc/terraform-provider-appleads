// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func mustAppDataSource(t *testing.T) *appDataSource {
	t.Helper()
	ds, ok := NewAppDataSource().(*appDataSource)
	if !ok {
		t.Fatalf("expected *appDataSource, got %T", NewAppDataSource())
	}
	return ds
}

func TestAppDataSource_UnitReadByName(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 111, "appName": "OrbitNote", "developerName": "Northwind Labs", "countryOrRegion": "US"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ds := mustAppDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"name":              tftypes.String,
				"id":                tftypes.String,
				"adam_id":           tftypes.String,
				"developer_name":    tftypes.String,
				"country_or_region": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"name":              tftypes.NewValue(tftypes.String, "OrbitNote"),
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, nil),
			"developer_name":    tftypes.NewValue(tftypes.String, nil),
			"country_or_region": tftypes.NewValue(tftypes.String, nil),
		}),
	}

	req := datasource.ReadRequest{Config: config}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(config.Raw.Type(), nil)},
	}
	// State needs a typed empty object for Set to work.
	resp.State = tfsdk.State{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"name":              tftypes.String,
				"id":                tftypes.String,
				"adam_id":           tftypes.String,
				"developer_name":    tftypes.String,
				"country_or_region": tftypes.String,
			},
		}, nil),
	}

	ds.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}

	var state appDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != "111" || state.AdamID.ValueString() != "111" {
		t.Fatalf("state = %#v", state)
	}
	if state.Name.ValueString() != "OrbitNote" || state.DeveloperName.ValueString() != "Northwind Labs" {
		t.Fatalf("state = %#v", state)
	}
	if state.CountryOrRegion.ValueString() != "US" {
		t.Fatalf("country = %q", state.CountryOrRegion.ValueString())
	}
}

func TestAppDataSource_UnitNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ds := mustAppDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":              tftypes.String,
			"id":                tftypes.String,
			"adam_id":           tftypes.String,
			"developer_name":    tftypes.String,
			"country_or_region": tftypes.String,
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"name":              tftypes.NewValue(tftypes.String, "Missing"),
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, nil),
			"developer_name":    tftypes.NewValue(tftypes.String, nil),
			"country_or_region": tftypes.NewValue(tftypes.String, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
	if !regexp.MustCompile(`no app found|NOT_FOUND`).MatchString(resp.Diagnostics.Errors()[0].Detail()) &&
		!regexp.MustCompile(`no app found|NOT_FOUND`).MatchString(resp.Diagnostics.Errors()[0].Summary()) {
		t.Fatalf("diags = %v", resp.Diagnostics)
	}
}

func TestAppDataSource_UnitMultipleMatches(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"adamId": 1, "appName": "Dup", "developerName": "A"},
				{"adamId": 2, "appName": "dup", "developerName": "B"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ds := mustAppDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":              tftypes.String,
			"id":                tftypes.String,
			"adam_id":           tftypes.String,
			"developer_name":    tftypes.String,
			"country_or_region": tftypes.String,
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"name":              tftypes.NewValue(tftypes.String, "Dup"),
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, nil),
			"developer_name":    tftypes.NewValue(tftypes.String, nil),
			"country_or_region": tftypes.NewValue(tftypes.String, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !regexp.MustCompile(`multiple apps matched|MULTIPLE_MATCHES`).MatchString(detail) &&
		!regexp.MustCompile(`multiple apps matched`).MatchString(resp.Diagnostics.Errors()[0].Summary()) {
		t.Fatalf("diags = %v", resp.Diagnostics)
	}
}

// Ensure terraform-plugin-testing remains a module dependency for later acceptance tests.
var _ = resource.TestCase{}
var _ = tfprotov6.ProviderServer(nil)
var _ = providerserver.NewProtocol6
