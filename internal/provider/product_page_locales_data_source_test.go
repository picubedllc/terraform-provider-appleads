// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func mustProductPageLocalesDataSource(t *testing.T) *productPageLocalesDataSource {
	t.Helper()
	ds, ok := NewProductPageLocalesDataSource().(*productPageLocalesDataSource)
	if !ok {
		t.Fatalf("expected *productPageLocalesDataSource, got %T", NewProductPageLocalesDataSource())
	}
	return ds
}

func mustCountriesOrRegionsDataSource(t *testing.T) *countriesOrRegionsDataSource {
	t.Helper()
	ds, ok := NewCountriesOrRegionsDataSource().(*countriesOrRegionsDataSource)
	if !ok {
		t.Fatalf("expected *countriesOrRegionsDataSource, got %T", NewCountriesOrRegionsDataSource())
	}
	return ds
}

func TestProductPageLocalesDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	const pageID = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps/42/product-pages/"+pageID+"/locale-details" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("deviceClasses"); got != "IPHONE" {
			t.Fatalf("deviceClasses = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"adamId":        42,
					"appName":       "OrbitNote",
					"deviceClasses": "IPHONE",
					"language":      "English",
					"languageCode":  "en-US",
					"productPageId": pageID,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPageLocalesDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	localeType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"adam_id":           tftypes.String,
			"app_name":          tftypes.String,
			"device_classes":    tftypes.String,
			"language":          tftypes.String,
			"language_code":     tftypes.String,
			"product_page_id":   tftypes.String,
			"promotional_text":  tftypes.String,
			"short_description": tftypes.String,
			"sub_title":         tftypes.String,
		},
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"adam_id":         tftypes.String,
			"product_page_id": tftypes.String,
			"device_classes":  tftypes.List{ElementType: tftypes.String},
			"language_codes":  tftypes.List{ElementType: tftypes.String},
			"languages":       tftypes.List{ElementType: tftypes.String},
			"expand":          tftypes.Bool,
			"locales":         tftypes.List{ElementType: localeType},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, nil),
			"adam_id":         tftypes.NewValue(tftypes.String, "42"),
			"product_page_id": tftypes.NewValue(tftypes.String, pageID),
			"device_classes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "IPHONE"),
			}),
			"language_codes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"languages":      tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"expand":         tftypes.NewValue(tftypes.Bool, nil),
			"locales":        tftypes.NewValue(tftypes.List{ElementType: localeType}, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state productPageLocalesDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Locales) != 1 || state.Locales[0].LanguageCode.ValueString() != "en-US" {
		t.Fatalf("state = %#v", state)
	}
}

func TestProductPageLocalesDataSource_UnitEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPageLocalesDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	localeType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"adam_id":           tftypes.String,
			"app_name":          tftypes.String,
			"device_classes":    tftypes.String,
			"language":          tftypes.String,
			"language_code":     tftypes.String,
			"product_page_id":   tftypes.String,
			"promotional_text":  tftypes.String,
			"short_description": tftypes.String,
			"sub_title":         tftypes.String,
		},
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"adam_id":         tftypes.String,
			"product_page_id": tftypes.String,
			"device_classes":  tftypes.List{ElementType: tftypes.String},
			"language_codes":  tftypes.List{ElementType: tftypes.String},
			"languages":       tftypes.List{ElementType: tftypes.String},
			"expand":          tftypes.Bool,
			"locales":         tftypes.List{ElementType: localeType},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, nil),
			"adam_id":         tftypes.NewValue(tftypes.String, "1"),
			"product_page_id": tftypes.NewValue(tftypes.String, "page"),
			"device_classes":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"language_codes":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"languages":       tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"expand":          tftypes.NewValue(tftypes.Bool, nil),
			"locales":         tftypes.NewValue(tftypes.List{ElementType: localeType}, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state productPageLocalesDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Locales) != 0 {
		t.Fatalf("locales = %#v", state.Locales)
	}
}

func TestCountriesOrRegionsDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/countries-or-regions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("countriesOrRegions"); got != "US" {
			t.Fatalf("countriesOrRegions = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"countryOrRegion": "US",
					"defaultLanguages": []map[string]any{
						{"language": "English", "languageCode": "en-US"},
					},
					"supportedLanguages": []map[string]any{
						{"language": "English", "languageCode": "en-US"},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCountriesOrRegionsDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	localeType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"language":      tftypes.String,
			"language_code": tftypes.String,
		},
	}
	rowType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"country_or_region":   tftypes.String,
			"default_languages":   tftypes.List{ElementType: localeType},
			"supported_languages": tftypes.List{ElementType: localeType},
		},
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                          tftypes.String,
			"countries_or_regions_filter": tftypes.List{ElementType: tftypes.String},
			"countries_or_regions":        tftypes.List{ElementType: rowType},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, nil),
			"countries_or_regions_filter": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "US"),
			}),
			"countries_or_regions": tftypes.NewValue(tftypes.List{ElementType: rowType}, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state countriesOrRegionsDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Results) != 1 || state.Results[0].CountryOrRegion.ValueString() != "US" {
		t.Fatalf("state = %#v", state)
	}
	if state.ID.ValueString() != "US" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
}

func TestCountriesOrRegionsDataSource_UnitEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustCountriesOrRegionsDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	localeType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"language":      tftypes.String,
			"language_code": tftypes.String,
		},
	}
	rowType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"country_or_region":   tftypes.String,
			"default_languages":   tftypes.List{ElementType: localeType},
			"supported_languages": tftypes.List{ElementType: localeType},
		},
	}
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                          tftypes.String,
			"countries_or_regions_filter": tftypes.List{ElementType: tftypes.String},
			"countries_or_regions":        tftypes.List{ElementType: rowType},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                          tftypes.NewValue(tftypes.String, nil),
			"countries_or_regions_filter": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"countries_or_regions":        tftypes.NewValue(tftypes.List{ElementType: rowType}, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state countriesOrRegionsDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != "all" || len(state.Results) != 0 {
		t.Fatalf("state = %#v", state)
	}
}
