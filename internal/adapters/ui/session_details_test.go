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
)

func TestDetailsRenderMarksCurrent(t *testing.T) {
	d := NewSessionDetails()
	d.SetCurrent("main")
	d.Render(domain.Session{Name: "main", WindowsCount: 1})
	if got := d.GetText(true); !strings.Contains(got, "(current)") {
		t.Errorf("current session details must contain (current), got: %q", got)
	}
}

func TestDetailsRenderNoMarkWhenNotCurrent(t *testing.T) {
	d := NewSessionDetails()
	d.SetCurrent("main")
	d.Render(domain.Session{Name: "other", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "(current)") {
		t.Errorf("non-current details must not contain (current), got: %q", got)
	}
}

func TestDetailsRenderNoMarkWhenCurrentUnset(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "main", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "(current)") {
		t.Errorf("unset-current details must not contain (current), got: %q", got)
	}
}

func TestDetailsRenderNote(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1, Note: "primary box"})
	got := d.GetText(true)
	if !strings.Contains(got, "note:") {
		t.Errorf("details must render a note label, got: %q", got)
	}
	if !strings.Contains(got, "primary box") {
		t.Errorf("details must render the note text, got: %q", got)
	}
}

func TestDetailsRenderNoNoteLineWhenEmpty(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "note:") {
		t.Errorf("empty note must not render a note line, got: %q", got)
	}
}

func TestDetailsRenderPinnedTrue(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1, Pinned: true})
	got := d.GetText(false) // GetText(false) keeps color tags; value's tag is the assertion
	if !strings.Contains(got, "pinned:") {
		t.Errorf("pinned label missing: %q", got)
	}
	if !strings.Contains(got, colorPrimary+"]true[-]") {
		t.Errorf("pinned=true value missing: %q", got)
	}
}

func TestDetailsRenderPinnedFalse(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1, Pinned: false})
	got := d.GetText(false) // GetText(false) keeps color tags; value's tag is the assertion
	if !strings.Contains(got, colorPrimary+"]false[-]") {
		t.Errorf("pinned=false value missing: %q", got)
	}
}

func TestDetailsRenderTagsAsChips(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1, Tags: []string{"work", "urgent"}})
	got := d.GetText(false) // GetText(false) keeps color tags; chips carry their own tags
	if !strings.Contains(got, "tags:") {
		t.Errorf("tags label missing: %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] work [-:-:-]") {
		t.Errorf("first tag chip missing: %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] urgent [-:-:-]") {
		t.Errorf("second tag chip missing: %q", got)
	}
}

func TestDetailsRenderNoTagsLineWhenEmpty(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "api", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "tags:") {
		t.Errorf("empty tags must not render a tags line: %q", got)
	}
}

// TestDetailsRenderFieldAlignment verifies every detail value lines up at the
// same visible column: padTagged must pad labels to detailLabelWidth using
// visible width, so the new "last attached:" label (longest) does not push its
// value right of the others.
func TestDetailsRenderFieldAlignment(t *testing.T) {
	d := NewSessionDetails()
	// Path + Note set so short labels ("path:", "note:") render alongside the
	// longest "last attached:" — all must pad to the same value column.
	d.Render(domain.Session{Name: "api", WindowsCount: 1, Path: "/x", Note: "n"})
	plain := d.GetText(true) // GetText(true) strips color tags → visible text
	lines := strings.Split(plain, "\n")
	var values []int
	for _, ln := range lines {
		// A detail line is "  label:   value". The value starts right after the
		// first colon and its trailing space-padding, so its column is
		// colon+1 + leading-space-count. Every detail value must share it.
		colon := strings.Index(ln, ":")
		if colon < 0 {
			continue
		}
		rest := ln[colon+1:]
		values = append(values, colon+1+len(rest)-len(strings.TrimLeft(rest, " ")))
	}
	if len(values) < 2 {
		t.Fatalf("expected at least 2 detail lines, got %d in %q", len(values), plain)
	}
	first := values[0]
	for _, v := range values[1:] {
		if v != first {
			t.Errorf("detail values not aligned at column %d: %v", first, values)
			break
		}
	}
}
