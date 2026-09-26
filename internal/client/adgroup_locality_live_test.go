// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// Live smoke for PUT locality targeting. Set APPLEADS_TEST_AD_GROUP_ID to an
// existing ad group id (plus APPLEADS_* and APPLEADS_LIVE_TEST=1).
func TestLiveUpdateAdGroup_LocalityNewYork(t *testing.T) {
	c, _ := acctest.LiveClient(t)

	raw := os.Getenv("APPLEADS_TEST_AD_GROUP_ID")
	if raw == "" {
		t.Skip("set APPLEADS_TEST_AD_GROUP_ID to run locality update smoke")
	}
	adGroupID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("APPLEADS_TEST_AD_GROUP_ID: %v", err)
	}

	ag, err := c.FindAdGroupByID(context.Background(), adGroupID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	t.Logf("before: campaign=%d name=%q td=%#v", ag.CampaignID, ag.Name, ag.TargetingDimensions)

	updated, err := c.UpdateAdGroup(context.Background(), ag.CampaignID, ag.ID, &client.AdGroupUpdate{
		TargetingDimensions: &client.TargetingDimensions{
			Locality: &client.LocalityTarget{Included: []string{"US|NY|New York"}},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.TargetingDimensions == nil || updated.TargetingDimensions.Locality == nil {
		t.Fatalf("expected locality after update, got %#v", updated.TargetingDimensions)
	}
	got := updated.TargetingDimensions.Locality.Included
	if len(got) != 1 || got[0] != "US|NY|New York" {
		t.Fatalf("locality = %#v", got)
	}
	t.Logf("after: name=%q locality=%v", updated.Name, got)
}
