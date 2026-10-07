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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func mustProductPagesDataSource(t *testing.T) *productPagesDataSource {
	t.Helper()
	ds, ok := NewProductPagesDataSource().(*productPagesDataSource)
	if !ok {
		t.Fatalf("expected *productPagesDataSource, got %T", NewProductPagesDataSource())
	}
	return ds
}

func mustProductPageDataSource(t *testing.T) *productPageDataSource {
	t.Helper()
	ds, ok := NewProductPageDataSource().(*productPageDataSource)
	if !ok {
		t.Fatalf("expected *productPageDataSource, got %T", NewProductPageDataSource())
	}
	return ds
}

func productPagesObjType() tftypes.Object {
	pageType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"name":              tftypes.String,
			"state":             tftypes.String,
			"adam_id":           tftypes.String,
			"deep_link":         tftypes.String,
			"creation_time":     tftypes.String,
			"modification_time": tftypes.String,
		},
	}
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":      tftypes.String,
			"adam_id": tftypes.String,
			"name":    tftypes.String,
			"states":  tftypes.List{ElementType: tftypes.String},
			"product_pages": tftypes.List{
				ElementType: pageType,
			},
		},
	}
}

func TestProductPagesDataSource_UnitReadEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps/899247964/product-pages" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("states"); got != "VISIBLE" {
			t.Fatalf("states = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPagesDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	objType := productPagesObjType()
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":      tftypes.NewValue(tftypes.String, nil),
			"adam_id": tftypes.NewValue(tftypes.String, "899247964"),
			"name":    tftypes.NewValue(tftypes.String, nil),
			"states": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "VISIBLE"),
			}),
			"product_pages": tftypes.NewValue(objType.AttributeTypes["product_pages"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}

	var state productPagesDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.ProductPages) != 0 {
		t.Fatalf("product_pages = %#v", state.ProductPages)
	}
	if state.ID.ValueString() != "899247964/states=VISIBLE" {
		t.Fatalf("id = %q", state.ID.ValueString())
	}
}

func TestProductPagesDataSource_UnitReadWithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "CPP" {
			t.Fatalf("name = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":     "45812c9b-c296-43d3-c6a0-c5a02f74bf6e",
					"name":   "CPP",
					"state":  "VISIBLE",
					"adamId": 899247964,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPagesDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := productPagesObjType()
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, nil),
			"adam_id":       tftypes.NewValue(tftypes.String, "899247964"),
			"name":          tftypes.NewValue(tftypes.String, "CPP"),
			"states":        tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"product_pages": tftypes.NewValue(objType.AttributeTypes["product_pages"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state productPagesDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.ProductPages) != 1 || state.ProductPages[0].ID.ValueString() != "45812c9b-c296-43d3-c6a0-c5a02f74bf6e" {
		t.Fatalf("state = %#v", state)
	}
}

func TestProductPagesDataSource_UnitInvalidAdamID(t *testing.T) {
	t.Parallel()

	ds := mustProductPagesDataSource(t)
	ds.client = &client.Client{}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := productPagesObjType()
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, nil),
			"adam_id":       tftypes.NewValue(tftypes.String, "not-a-number"),
			"name":          tftypes.NewValue(tftypes.String, nil),
			"states":        tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"product_pages": tftypes.NewValue(objType.AttributeTypes["product_pages"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
	if !regexp.MustCompile(`Invalid adam_id|positive numeric`).MatchString(resp.Diagnostics.Errors()[0].Summary() + resp.Diagnostics.Errors()[0].Detail()) {
		t.Fatalf("diags = %v", resp.Diagnostics)
	}
}

func TestProductPageDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	const pageID = "45812c9b-c296-43d3-c6a0-c5a02f74bf6e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps/899247964/product-pages/"+pageID {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     pageID,
				"name":   "Trip Trek CPP",
				"state":  "VISIBLE",
				"adamId": 899247964,
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPageDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"adam_id":           tftypes.String,
			"product_page_id":   tftypes.String,
			"name":              tftypes.String,
			"state":             tftypes.String,
			"deep_link":         tftypes.String,
			"creation_time":     tftypes.String,
			"modification_time": tftypes.String,
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, "899247964"),
			"product_page_id":   tftypes.NewValue(tftypes.String, pageID),
			"name":              tftypes.NewValue(tftypes.String, nil),
			"state":             tftypes.NewValue(tftypes.String, nil),
			"deep_link":         tftypes.NewValue(tftypes.String, nil),
			"creation_time":     tftypes.NewValue(tftypes.String, nil),
			"modification_time": tftypes.NewValue(tftypes.String, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var state productPageDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if state.ID.ValueString() != pageID || state.Name.ValueString() != "Trip Trek CPP" {
		t.Fatalf("state = %#v", state)
	}
}

func TestProductPageDataSource_UnitNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]any{{"messageCode": "NOT_FOUND", "message": "missing"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds := mustProductPageDataSource(t)
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"adam_id":           tftypes.String,
			"product_page_id":   tftypes.String,
			"name":              tftypes.String,
			"state":             tftypes.String,
			"deep_link":         tftypes.String,
			"creation_time":     tftypes.String,
			"modification_time": tftypes.String,
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, "1"),
			"product_page_id":   tftypes.NewValue(tftypes.String, "missing"),
			"name":              tftypes.NewValue(tftypes.String, nil),
			"state":             tftypes.NewValue(tftypes.String, nil),
			"deep_link":         tftypes.NewValue(tftypes.String, nil),
			"creation_time":     tftypes.NewValue(tftypes.String, nil),
			"modification_time": tftypes.NewValue(tftypes.String, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestProductPageDataSource_UnitInvalidAdamID(t *testing.T) {
	t.Parallel()

	ds := mustProductPageDataSource(t)
	ds.client = &client.Client{}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"adam_id":           tftypes.String,
			"product_page_id":   tftypes.String,
			"name":              tftypes.String,
			"state":             tftypes.String,
			"deep_link":         tftypes.String,
			"creation_time":     tftypes.String,
			"modification_time": tftypes.String,
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                tftypes.NewValue(tftypes.String, nil),
			"adam_id":           tftypes.NewValue(tftypes.String, "0"),
			"product_page_id":   tftypes.NewValue(tftypes.String, "abc"),
			"name":              tftypes.NewValue(tftypes.String, nil),
			"state":             tftypes.NewValue(tftypes.String, nil),
			"deep_link":         tftypes.NewValue(tftypes.String, nil),
			"creation_time":     tftypes.NewValue(tftypes.String, nil),
			"modification_time": tftypes.NewValue(tftypes.String, nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
