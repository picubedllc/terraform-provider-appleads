// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client_test

import (
	"os"
	"testing"

	"github.com/picubedllc/terraform-provider-appleads/internal/acctest"
)

func TestLiveSearchApps(t *testing.T) {
	c, _ := acctest.LiveClient(t)

	query := os.Getenv("APPLEADS_TEST_APP_NAME")
	if query == "" {
		query = "OrbitNote"
	}

	apps := acctest.RequireSearchApps(t, c, query, true)
	t.Logf("SearchApps(%q) owned matches=%d", query, len(apps))
}
