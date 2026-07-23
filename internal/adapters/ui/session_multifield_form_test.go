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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestMultiFieldSessionFormStructure pins the shared form's shape: three items
// (Name / Tags / Note) plus Save and Cancel buttons, all starting empty. This
// follows the session_form_test.go style of asserting only constructable state
// (no event simulation); the New/Edit orchestration is covered by the store and
// service layer tests.
func TestMultiFieldSessionFormStructure(t *testing.T) {
	f := NewMultiFieldSessionForm("New session")
	if got := f.Form().GetFormItemCount(); got != 3 {
		t.Fatalf("form must have 3 items (name/tags/note), got %d", got)
	}
	if got := f.Form().GetButtonCount(); got != 0 {
		t.Fatalf("form must have no buttons (Enter saves, Esc cancels), got %d", got)
	}
	for _, i := range []int{mfFieldName, mfFieldTags, mfFieldNote} {
		if got := f.fieldText(i); got != "" {
			t.Errorf("item %d should start empty, got %q", i, got)
		}
	}
}

// TestMultiFieldSessionFormInitialValues verifies the Edit flow prefills all
// three fields (including a multi-line note), while an empty value stays empty.
func TestMultiFieldSessionFormInitialValues(t *testing.T) {
	f := NewMultiFieldSessionForm("Edit session").
		InitialValues("api", "work, prod", "line one\nline two")
	if got := f.fieldText(mfFieldName); got != "api" {
		t.Errorf("name prefill: got %q, want api", got)
	}
	if got := f.fieldText(mfFieldTags); got != "work, prod" {
		t.Errorf("tags prefill: got %q, want %q", got, "work, prod")
	}
	if got := f.fieldText(mfFieldNote); got != "line one\nline two" {
		t.Errorf("note prefill (multi-line): got %q, want %q", got, "line one\nline two")
	}
}

// TestMultiFieldFormNoteFocusKeepsNameVisible reproduces the bug where Tabbing
// into the Note text area scrolled the Name field up and out of view. Root
// cause: tview.NewForm applies a default border padding of 1 on every side,
// which made the form's inner area one row too short for the Note, so focusing
// it triggered tview.Form's scroll-to-focused-item. It draws the form at the
// size the modal allocates, focuses the Note, and asserts the Name label is
// still on screen.
func TestMultiFieldFormNoteFocusKeepsNameVisible(t *testing.T) {
	f := NewMultiFieldSessionForm("Edit session")
	form := f.Form()
	// Size the form as the modal does: full width, and the height the column
	// allocates after reserving the hint row.
	form.SetRect(0, 0, 62, modalColumnHeight-1)
	// Focus the Note like Tabbing into it would. Focus flips the item's hasFocus
	// flag, which Form.Draw reads to decide whether to scroll the focused item
	// into view.
	form.GetFormItem(mfFieldNote).(*tview.TextArea).Focus(func(tview.Primitive) {})

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 40)
	form.Draw(screen)

	if !screenContains(screen, "Name") {
		t.Errorf("the Name field scrolled out of view when the Note was focused; " +
			"the form's inner area is too short for all three items")
	}
}

// screenContains reports whether needle appears anywhere on the simulation
// screen (one row's worth of cells is searched at a time).
func screenContains(screen tcell.SimulationScreen, needle string) bool {
	w, h := screen.Size()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := screen.GetContent(x, y)
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}
	return strings.Contains(b.String(), needle)
}

// TestMultiFieldSessionFormPlaceholders pins that the three empty fields each
// show a muted placeholder, mirroring the single-field forms so a field reads
// identically whether it is edited here or standalone (Tags = "comma-separated
// tags", Note = "freeform note"). Like TestMultiFieldFormNoteFocusKeepsNameVisible
// it renders the form to a simulation screen and checks each hint is on screen —
// tview paints a placeholder only while the field is empty, so this also guards
// against a regression that sets text instead of a placeholder.
func TestMultiFieldSessionFormPlaceholders(t *testing.T) {
	f := NewMultiFieldSessionForm("New session")
	form := f.Form()
	form.SetRect(0, 0, 62, modalColumnHeight-1)

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 40)
	form.Draw(screen)

	for _, want := range []string{"session name", "comma-separated tags", "freeform note"} {
		if !screenContains(screen, want) {
			t.Errorf("empty-field placeholder %q is not rendered", want)
		}
	}
}

// focusFormItem sets hasFocus on the i-th item so GetFocusedItemIndex reports
// it. This mirrors what tview.Form.Draw does to the focused item at render
// time; SetFocus alone does not focus the item when its index is 0 and nothing
// else is focused yet (a tview quirk that only surfaces without a Draw).
func focusFormItem(f *MultiFieldSessionForm, i int) {
	switch v := f.Form().GetFormItem(i).(type) {
	case *tview.InputField:
		v.Focus(func(tview.Primitive) {})
	case *tview.TextArea:
		v.Focus(func(tview.Primitive) {})
	}
}

// TestMultiFieldFormMoveField verifies ↑/↓ switch fields only on the single-line
// Name/Tags inputs: down moves forward (Name→Tags→Note), up moves backward, the
// Note TextArea is left alone (its ↑/↓ stay on the cursor), and ↑ on Name is a
// boundary no-op. Each case focuses the starting field, calls moveField, and
// checks both the "consumed" return and where focus landed.
func TestMultiFieldFormMoveField(t *testing.T) {
	for _, tc := range []struct {
		name   string
		from   int
		down   bool
		wantOK bool
		want   int // expected focused item afterwards
	}{
		{"Name down to Tags", mfFieldName, true, true, mfFieldTags},
		{"Tags down to Note", mfFieldTags, true, true, mfFieldNote},
		{"Tags up to Name", mfFieldTags, false, true, mfFieldName},
		{"Name up is a no-op (boundary)", mfFieldName, false, false, mfFieldName},
		{"Note down stays (cursor move)", mfFieldNote, true, false, mfFieldNote},
		{"Note up stays (cursor move)", mfFieldNote, false, false, mfFieldNote},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := NewMultiFieldSessionForm("Edit session")
			focusFormItem(f, tc.from)
			got := f.moveField(tc.down)
			if got != tc.wantOK {
				t.Fatalf("moveField(down=%v) from item %d = %v, want %v", tc.down, tc.from, got, tc.wantOK)
			}
			item, _ := f.Form().GetFocusedItemIndex()
			if item != tc.want {
				t.Errorf("focused item after move = %d, want %d", item, tc.want)
			}
		})
	}
}
