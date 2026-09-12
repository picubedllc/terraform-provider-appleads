// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package acctest

import (
	"testing"
)

func TestLiveTestsEnabled(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{val: "", want: false},
		{val: "0", want: false},
		{val: "1", want: true},
		{val: "true", want: true},
		{val: "TRUE", want: true},
		{val: "yes", want: true},
		{val: "no", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			t.Setenv(EnvLiveTest, tt.val)
			if got := liveTestsEnabled(); got != tt.want {
				t.Fatalf("liveTestsEnabled(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestPreCheck_SkipsUnlessLiveFlag(t *testing.T) {
	t.Setenv(EnvLiveTest, "")
	t.Run("skip", func(t *testing.T) {
		PreCheck(t)
		t.Fatal("PreCheck should skip when APPLEADS_LIVE_TEST is unset")
	})
}
