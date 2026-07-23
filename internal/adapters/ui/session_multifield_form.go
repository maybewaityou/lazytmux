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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Indices of the form's items, in AddXxx order.
const (
	mfFieldName = iota
	mfFieldTags
	mfFieldNote
)

// noteVisibleRows is the Note text area's visible height. The modal is sized so
// Name + Tags + Note all fit at once (no scrolling of the modal itself); a note
// longer than this scrolls only within the Note field.
const noteVisibleRows = 5

// modalColumnHeight is the centered column's height: the form (border + Name +
// Tags + the noteVisibleRows Note area) plus one row for the key hint. With the
// form's vertical border padding cleared (see NewMultiFieldSessionForm), the
// form's inner area is tall enough that focusing the Note never scrolls the
// Name field out of view.
const modalColumnHeight = 13

// MultiFieldSessionForm is a Name / Tags / Note modal shared by New and Edit.
// Name and Tags are single-line inputs; Note is a multi-line text area. Enter
// submits from any field; Shift+Enter inserts a newline inside Note; Esc
// cancels. Tab and arrows move between fields via tview.Form's built-in
// navigation. (Shift+Enter requires a terminal that reports it distinctly from
// Enter — one speaking the CSI-u / kitty keyboard protocol; on a terminal that
// doesn't, Shift+Enter behaves like plain Enter and submits.)
type MultiFieldSessionForm struct {
	form     *tview.Form
	hint     *tview.TextView
	onSubmit func(name, tags, note string)
	onCancel func()
}

// NewMultiFieldSessionForm builds the form with the given title ("New session"
// or "Edit session"). Fields start empty; prefill via InitialValues.
func NewMultiFieldSessionForm(title string) *MultiFieldSessionForm {
	f := &MultiFieldSessionForm{
		form: tview.NewForm(),
		hint: tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter),
	}
	f.form.SetBorder(true).
		SetTitle(" "+title+" ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder)).
		// Clear the vertical border padding. tview.NewForm defaults to a padding
		// of 1 on every side, which steals two rows from the inner area and makes
		// it one row shorter than the Note needs — so focusing the Note makes
		// tview.Form scroll the Name field up and out of view. Horizontal padding
		// is kept so the fields sit a column inside the border.
		SetBorderPadding(0, 0, 1, 1)
	f.form.SetLabelColor(tcell.GetColor(colorAccent)).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.GetColor(colorPrimary)).
		SetButtonTextColor(tcell.GetColor(colorPrimary)).
		SetButtonBackgroundColor(tcell.ColorDefault).
		SetBackgroundColor(tcell.ColorDefault)
	// fieldWidth 0 = expand to the form's width; the Note area is noteVisibleRows
	// rows tall and unbounded in length (maxLength 0).
	f.form.AddInputField("Name", "", 0, nil, nil).
		AddInputField("Tags", "", 0, nil, nil).
		AddTextArea("Note", "", 0, noteVisibleRows, 0, nil)
	f.hint.SetText("[" + colorSecondary + "]Enter(save) · Shift+Enter(newline in Note) · Esc(cancel)[-]")

	// Enter submits from any field — the Note text area never sees a plain Enter
	// (so no stray newline is inserted), making "type a name and press Enter"
	// create/save immediately. Shift+Enter is passed through to the focused item
	// so Note can insert a newline (tview's TextArea newlines on KeyEnter
	// regardless of modifiers, so only the shifted variant reaches it). Esc
	// cancels. Everything else is tview.Form's default (it forwards unhandled
	// keys to the focused item, which handles Tab/arrows navigation).
	f.form.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyESC:
			f.cancel()
			return nil
		case tcell.KeyEnter:
			if e.Modifiers()&tcell.ModShift != 0 {
				return e // let Note insert a newline
			}
			f.submit()
			return nil
		}
		return e
	})
	return f
}

// InitialValues prefills the three fields. Edit passes the session's current
// name / tags / note; New passes "" for all. An empty value is a valid prefill
// (e.g. a session with no note), so each field is set unconditionally.
func (f *MultiFieldSessionForm) InitialValues(name, tags, note string) *MultiFieldSessionForm {
	if field, ok := f.form.GetFormItem(mfFieldName).(*tview.InputField); ok {
		field.SetText(name)
	}
	if field, ok := f.form.GetFormItem(mfFieldTags).(*tview.InputField); ok {
		field.SetText(tags)
	}
	if area, ok := f.form.GetFormItem(mfFieldNote).(*tview.TextArea); ok {
		area.SetText(note, true)
	}
	return f
}

func (f *MultiFieldSessionForm) OnSubmit(fn func(name, tags, note string)) *MultiFieldSessionForm {
	f.onSubmit = fn
	return f
}

func (f *MultiFieldSessionForm) OnCancel(fn func()) *MultiFieldSessionForm {
	f.onCancel = fn
	return f
}

// submit is the Save button handler. An empty Name is ignored (the form stays
// open) so an accidental Save can't act on a nameless session; otherwise it
// fires onSubmit with the three current field values.
func (f *MultiFieldSessionForm) submit() {
	name := strings.TrimSpace(f.fieldText(mfFieldName))
	if name == "" {
		return
	}
	if f.onSubmit != nil {
		f.onSubmit(name, f.fieldText(mfFieldTags), f.fieldText(mfFieldNote))
	}
}

func (f *MultiFieldSessionForm) cancel() {
	if f.onCancel != nil {
		f.onCancel()
	}
}

// fieldText reads the current value of the i-th item, whether it is an
// InputField (Name/Tags) or a TextArea (Note).
func (f *MultiFieldSessionForm) fieldText(i int) string {
	switch v := f.form.GetFormItem(i).(type) {
	case *tview.InputField:
		return v.GetText()
	case *tview.TextArea:
		return v.GetText()
	}
	return ""
}

// Primitive returns a centered modal tall enough to show Name + Tags + Note
// (noteVisibleRows) plus the key hint, so focusing the Note never scrolls the
// Name field out of view. The hint sits below the bordered form and never takes
// focus (Tab stays within the form's items).
func (f *MultiFieldSessionForm) Primitive() tview.Primitive {
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(f.form, 0, 1, true).
		AddItem(f.hint, 1, 0, false)
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(column, modalColumnHeight, 0, true).
			AddItem(nil, 0, 1, false), 62, 0, true).
		AddItem(nil, 0, 1, false)
}

// Form returns the underlying form for SetFocus (focuses the first field).
func (f *MultiFieldSessionForm) Form() *tview.Form { return f.form }
