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

import "testing"

// TestNoteFormAreaIsTextArea pins that the editor is backed by a usable
// *tview.TextArea — the same component the session form's Note field uses. The
// Area return type already fixes this at compile time; this test guards against
// a future regression to a single-line InputField by exercising GetText at
// construction time (it must be a live text area that starts empty).
func TestNoteFormAreaIsTextArea(t *testing.T) {
	area := NewNoteForm("Note", "freeform note").Area()
	if area == nil {
		t.Fatal("NoteForm.Area must return a non-nil *tview.TextArea")
	}
	if got := area.GetText(); got != "" {
		t.Fatalf("new NoteForm text area should start empty, got %q", got)
	}
}

// TestNoteFormInitialValuePrefill verifies a multi-line note round-trips through
// the text area's GetText, matching the session form's Note prefill behavior.
func TestNoteFormInitialValuePrefill(t *testing.T) {
	f := NewNoteForm("Note", "freeform note").InitialValue("line one\nline two")
	if got := f.Area().GetText(); got != "line one\nline two" {
		t.Fatalf("multi-line prefill: got %q, want %q", got, "line one\nline two")
	}
}

// TestNoteFormInitialValueEmptyIsNotNoOp is the key behavioral difference from
// SessionForm.InitialValue: an empty value must clear the area, not be a no-op.
// Editing a session whose note was just cleared has to show an empty editor, so
// InitialValue sets unconditionally. Prefill text first, then pass "", and
// assert the area is empty.
func TestNoteFormInitialValueEmptyIsNotNoOp(t *testing.T) {
	f := NewNoteForm("Note", "freeform note").InitialValue("leftover note")
	f.InitialValue("") // editing a session whose note was cleared must show empty
	if got := f.Area().GetText(); got != "" {
		t.Fatalf("empty InitialValue must clear the area (not a no-op), got %q", got)
	}
}
