// Copyright 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"strings"
	"testing"
)

// TestRefreshStatusMessage verifies the post-refresh footer toast: a non-empty
// list reports the count, while the empty state gets a dedicated message instead
// of the awkward "Refreshed 0 sessions".
func TestRefreshStatusMessage(t *testing.T) {
	// Empty state: dedicated wording, never "Refreshed 0 sessions".
	empty := refreshStatusMessage(0)
	if !strings.Contains(empty, "No sessions to refresh") {
		t.Errorf("refreshStatusMessage(0) = %q, want it to mention 'No sessions to refresh'", empty)
	}
	if strings.Contains(empty, "Refreshed 0") {
		t.Errorf("refreshStatusMessage(0) = %q, must not say 'Refreshed 0 sessions'", empty)
	}

	// Non-empty: the count is interpolated into the toast.
	for _, tc := range []struct {
		count int
		want  string
	}{
		{1, "Refreshed 1 sessions"},
		{3, "Refreshed 3 sessions"},
		{12, "Refreshed 12 sessions"},
	} {
		got := refreshStatusMessage(tc.count)
		if !strings.Contains(got, tc.want) {
			t.Errorf("refreshStatusMessage(%d) = %q, want it to contain %q", tc.count, got, tc.want)
		}
	}
}
