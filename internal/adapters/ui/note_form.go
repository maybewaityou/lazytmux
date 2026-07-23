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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// noteModalWidth matches the width of the multi-field session form so a note
// reads identically whether edited standalone here or inside the New/Edit form.
const noteModalWidth = 62

// noteModalHeight is the centered column's height: the bordered text area
// (noteVisibleRows content rows + 2 border rows) plus one row for the key hint.
// There is no tview.Form here, so no border-padding trick is needed — the
// single TextArea owns the whole inner area and scrolls only within itself.
const noteModalHeight = noteVisibleRows + 3 // 5 content + 2 border + 1 hint

// NoteForm is a single multi-line text-area modal used to edit a session's
// note. It wraps *tview.TextArea directly — the same component the Note field
// of MultiFieldSessionForm uses — so editing feels identical standalone and
// inside the New/Edit form. Enter submits; Shift+Enter inserts a newline; Esc
// cancels. An empty submit clears the note (handled by the caller's SaveNote
// path).
type NoteForm struct {
	area     *tview.TextArea
	hint     *tview.TextView
	onSubmit func(string)
	onCancel func()
}

// NewNoteForm builds the form with the given title and placeholder. The text
// area is noteVisibleRows rows tall (reused from the session form's Note
// field) and full-width; prefill via InitialValue.
func NewNoteForm(title, placeholder string) *NoteForm {
	f := &NoteForm{
		area: tview.NewTextArea(),
		hint: tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter),
	}
	// TextArea-specific setters all return *TextArea.
	f.area.SetSize(noteVisibleRows, 0).
		SetPlaceholder(placeholder).
		SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.GetColor(colorSecondary))).
		SetTextStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.GetColor(colorPrimary)))
	// Box setters return *Box (TextArea embeds *Box); keep them on their own
	// chain so the *TextArea chain above is not broken.
	f.area.SetBorder(true).
		SetTitle(" " + title + " ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder)).
		SetBackgroundColor(tcell.ColorDefault)
	f.hint.SetText("[" + colorSecondary + "]Enter(save) · Shift+Enter(newline) · Esc(cancel)[-]")

	// Enter submits from the text area — it never sees a plain Enter (so no
	// stray newline is inserted on save), making "type and press Enter" save
	// immediately. Shift+Enter is passed through so the area inserts a newline
	// (tview's TextArea newlines on KeyEnter regardless of modifiers, so only
	// the shifted variant reaches it). Esc cancels. ↑/↓ keep moving the cursor
	// — a single field has nothing to switch, mirroring the Note field in the
	// session form.
	f.area.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyESC:
			if f.onCancel != nil {
				f.onCancel()
			}
			return nil
		case tcell.KeyEnter:
			if e.Modifiers()&tcell.ModShift != 0 {
				return e // let the area insert a newline
			}
			if f.onSubmit != nil {
				f.onSubmit(f.area.GetText())
			}
			return nil
		}
		return e
	})
	return f
}

func (f *NoteForm) OnSubmit(fn func(string)) *NoteForm { f.onSubmit = fn; return f }

func (f *NoteForm) OnCancel(fn func()) *NoteForm { f.onCancel = fn; return f }

// InitialValue prefills the text area. Unlike SessionForm.InitialValue, an
// empty value is NOT a no-op: editing a session whose note was just cleared
// must show an empty editor, so SetText runs unconditionally. cursorAtTheEnd
// (true) leaves the cursor at the end of the text, matching the Note field
// prefill in MultiFieldSessionForm.InitialValues.
func (f *NoteForm) InitialValue(v string) *NoteForm {
	f.area.SetText(v, true)
	return f
}

// Primitive returns a centered modal sized for the noteVisibleRows text area
// plus a one-row key hint below it. The hint never takes focus (Enter/Esc are
// captured on the text area itself).
func (f *NoteForm) Primitive() tview.Primitive {
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(f.area, noteVisibleRows+2, 0, true).
		AddItem(f.hint, 1, 0, false)
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(column, noteModalHeight, 0, true).
			AddItem(nil, 0, 1, false), noteModalWidth, 0, true).
		AddItem(nil, 0, 1, false)
}

// Area returns the underlying text area (for SetFocus).
func (f *NoteForm) Area() *tview.TextArea { return f.area }
