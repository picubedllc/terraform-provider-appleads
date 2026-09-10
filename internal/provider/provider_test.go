// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"testing"
)

func TestProviderMetadata(t *testing.T) {
	t.Parallel()

	p := New("test")()
	if p == nil {
		t.Fatal("expected provider instance")
	}
}
