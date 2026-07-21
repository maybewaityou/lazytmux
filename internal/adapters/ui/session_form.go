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

// SessionForm is a single-field modal used for creating and renaming sessions.
type SessionForm struct {
	input    *tview.InputField
	onSubmit func(string)
	onCancel func()
}

func NewSessionForm(title, placeholder string) *SessionForm {
	f := &SessionForm{input: tview.NewInputField()}
	f.input.SetLabel(title + ": ").
		SetLabelColor(tcell.GetColor(colorAccent)).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.GetColor(colorPrimary)).
		SetPlaceholder(placeholder).
		SetPlaceholderTextColor(tcell.GetColor(colorSecondary))
	f.input.SetBorder(true).SetTitle(" " + title + " ").
		SetTitleColor(tcell.GetColor(colorTitle)).SetBorderColor(tcell.GetColor(colorBorder))

	f.input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:
			if f.onSubmit != nil {
				f.onSubmit(f.input.GetText())
			}
			return nil
		case tcell.KeyESC:
			if f.onCancel != nil {
				f.onCancel()
			}
			return nil
		}
		return e
	})
	return f
}

func (f *SessionForm) OnSubmit(fn func(string)) *SessionForm { f.onSubmit = fn; return f }
func (f *SessionForm) OnCancel(fn func()) *SessionForm       { f.onCancel = fn; return f }

// InitialValue 预填输入框文本(重命名时显示原名)。空串为 no-op,这样
// 新建会话流程可以无副作用地传 ""。tview SetText 把光标置于文本末尾。
func (f *SessionForm) InitialValue(v string) *SessionForm {
	if v != "" {
		f.input.SetText(v)
	}
	return f
}

// Primitive returns a centered, fixed-size modal wrapping the input.
func (f *SessionForm) Primitive() tview.Primitive {
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(f.input, 3, 0, true).
			AddItem(nil, 0, 1, false), 48, 0, true).
		AddItem(nil, 0, 1, false)
}

// Input returns the underlying input field (for SetFocus).
func (f *SessionForm) Input() *tview.InputField { return f.input }
