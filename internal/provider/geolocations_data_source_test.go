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

func TestGeolocationsDataSource_UnitRead(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/geo" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "New York" {
			t.Fatalf("query = %q", got)
		}
		if got := r.URL.Query().Get("entity"); got != client.GeoEntityLocality {
			t.Fatalf("entity = %q", got)
		}
		if got := r.URL.Query().Get("countrycode"); got != "US" {
			t.Fatalf("countrycode = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "US|NY|New York", "entity": "Locality", "displayName": "New York, New York, United States"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	ds, ok := NewGeolocationsDataSource().(*geolocationsDataSource)
	if !ok {
		t.Fatalf("type %T", NewGeolocationsDataSource())
	}
	ds.client = apiClient

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":           tftypes.String,
			"query":        tftypes.String,
			"entity":       tftypes.String,
			"country_code": tftypes.String,
			"geo_id":       tftypes.String,
			"limit":        tftypes.Number,
			"locations": tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"id":           tftypes.String,
					"entity":       tftypes.String,
					"display_name": tftypes.String,
				},
			}},
		},
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, nil),
			"query":        tftypes.NewValue(tftypes.String, "New York"),
			"entity":       tftypes.NewValue(tftypes.String, "Locality"),
			"country_code": tftypes.NewValue(tftypes.String, "US"),
			"geo_id":       tftypes.NewValue(tftypes.String, nil),
			"limit":        tftypes.NewValue(tftypes.Number, nil),
			"locations":    tftypes.NewValue(objType.AttributeTypes["locations"], nil),
		}),
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}

	var state geolocationsDataSourceModel
	diags := resp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("state get: %v", diags)
	}
	if len(state.Locations) != 1 || state.Locations[0].ID.ValueString() != "US|NY|New York" {
		t.Fatalf("state = %#v", state)
	}
	if state.Locations[0].DisplayName.ValueString() != "New York, New York, United States" {
		t.Fatalf("display = %q", state.Locations[0].DisplayName.ValueString())
	}
}
