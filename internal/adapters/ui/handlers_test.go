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

	"github.com/maybewaityou/lazytmux/internal/core/domain"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
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

// staleTestServe satisfies ports.SessionService for details-rendering tests,
// overriding only LoadWindows. Other methods stay nil via embedding — the tests
// never call them.
type staleTestServe struct {
	ports.SessionService
	loadWindows func(*domain.Session) error
}

func (s *staleTestServe) LoadWindows(sess *domain.Session) error {
	return s.loadWindows(sess)
}

// TestLoadWindowsAndRenderGuardsStale reproduces the bug where an in-flight async
// window load (captured for an earlier selection) finishes after the selection
// has moved on and overwrites the details pane with stale data. The stale render
// must be dropped; only the render matching the current generation may apply.
func TestLoadWindowsAndRenderGuardsStale(t *testing.T) {
	details := NewSessionDetails()
	tt := &tui{
		details: details,
		serve: &staleTestServe{loadWindows: func(s *domain.Session) error {
			s.Windows = []domain.Window{{Index: 1, Name: s.Name + "-win"}}
			return nil
		}},
		queueDraw: func(f func()) { f() }, // drive async renders synchronously
	}

	// Selection "alpha" at generation 1 renders it.
	tt.selectionGen = 1
	tt.loadWindowsAndRender(domain.Session{Name: "alpha"}, 1)

	// A newer selection bumps generation to 2 and renders "beta" as current.
	tt.selectionGen = 2
	tt.loadWindowsAndRender(domain.Session{Name: "beta"}, 2)
	if got := details.GetText(true); !strings.Contains(got, "beta") {
		t.Fatalf("expected details to show beta, got: %q", got)
	}

	// The stale "alpha" render (captured at gen 1) fires now. It must NOT overwrite
	// beta — without the generation guard the pane flips back to alpha.
	tt.loadWindowsAndRender(domain.Session{Name: "alpha"}, 1)
	if got := details.GetText(true); strings.Contains(got, "alpha") {
		t.Errorf("stale alpha render should be dropped, details still show alpha: %q", got)
	}
	if got := details.GetText(true); !strings.Contains(got, "beta") {
		t.Errorf("details should still show beta after the stale render, got: %q", got)
	}
}

// TestClearedTagsMessage verifies the post-clear footer toast names the
// session and uses the green status color, mirroring detach's success toast.
func TestClearedTagsMessage(t *testing.T) {
	got := clearedTagsMessage("work")
	if !strings.Contains(got, "Cleared tags: work") {
		t.Errorf("clearedTagsMessage(%q) = %q, want it to name the session", "work", got)
	}
	if !strings.Contains(got, colorGreen) {
		t.Errorf("clearedTagsMessage should use colorGreen, got %q", got)
	}
}

// TestParseTags verifies the tag input is split on commas — ASCII "," and the
// fullwidth Chinese "，" alike — while spaces are preserved inside a tag and
// empty tokens are dropped. The "space inside a tag is kept" case is the point
// of the comma-separator change: it is why "release v2" survives as one tag.
func TestParseTags(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  []string
	}{
		{"ascii comma", "work,personal,urgent", []string{"work", "personal", "urgent"}},
		{"chinese fullwidth comma", "工作，生活，紧急", []string{"工作", "生活", "紧急"}},
		{"mixed comma styles", "work，personal,紧急", []string{"work", "personal", "紧急"}},
		{"comma plus surrounding spaces", "a, b , c", []string{"a", "b", "c"}},
		{"space inside a tag is kept", "release v2, bug", []string{"release v2", "bug"}},
		{"empty input", "   ", nil},
		{"only separators", ",，,", nil},
		{"blank tokens dropped", "a,,b", []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTags(tc.input)
			if !equalTags(got, tc.want) {
				t.Errorf("parseTags(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// equalTags compares two string slices by length and contents, treating a nil
// slice and an empty slice as equal (parseTags returns a non-nil empty slice,
// while the test table uses nil for clarity).
func equalTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestVisibleSessionsAppliesSearchQuery pins the unified filter pipeline: the
// sort path (applySortAndRender) now renders visibleSessions instead of allCache
// directly, so a sort cycle must preserve the active search result. It also pins
// that the tag filter and the name search compose (tag narrows first, then name).
func TestVisibleSessionsAppliesSearchQuery(t *testing.T) {
	// Query alone: allCache holds 3 sessions, only "api" fuzzy-matches "api".
	search := NewSearchBar()
	search.SetText("api")
	tt := &tui{
		allCache: []domain.Session{
			{Name: "api"},
			{Name: "notes"},
			{Name: "legacy"},
		},
		searchBar: search,
	}

	got := tt.visibleSessions()
	if len(got) != 1 || got[0].Name != "api" {
		t.Fatalf("visibleSessions(query=\"api\") = %v, want [api]", sessionNames(got))
	}

	// Tag + name composition: the tag filter narrows to "work" sessions (dropping
	// "legacy"), then the name query narrows within that set (dropping "notes") —
	// order is preserved by both stages, so the result is deterministic.
	tagged := &tui{
		allCache: []domain.Session{
			{Name: "api", Tags: []string{"work"}},
			{Name: "api-server", Tags: []string{"work"}},
			{Name: "notes", Tags: []string{"work"}},
			{Name: "legacy", Tags: []string{"personal"}},
		},
		tagFilter: []string{"work"},
		searchBar: search, // still holds query "api"
	}

	got = tagged.visibleSessions()
	if len(got) != 2 || got[0].Name != "api" || got[1].Name != "api-server" {
		t.Fatalf("visibleSessions(tag=\"work\", query=\"api\") = %v, want [api api-server]", sessionNames(got))
	}
}

// sessionNames returns the Name field of each session, in order, for terse test
// assertions and failure messages.
func sessionNames(ss []domain.Session) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Name
	}
	return out
}
