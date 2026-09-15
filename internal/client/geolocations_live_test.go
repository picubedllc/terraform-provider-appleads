// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

func TestLiveSearchGeoLocations_NewYork(t *testing.T) {
	c, _ := acctest.LiveClient(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	got, err := c.SearchGeoLocations(ctx, client.GeoSearchParams{
		Query:       "New York",
		Entity:      client.GeoEntityLocality,
		CountryCode: "US",
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("SearchGeoLocations: %v", err)
	}
	found := false
	for _, loc := range got {
		t.Logf("geo id=%s entity=%s display=%s", loc.ID, loc.Entity, loc.DisplayName)
		if loc.ID == "US|NY|New York" && loc.Entity == client.GeoEntityLocality {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected locality US|NY|New York in %d results", len(got))
	}
}
