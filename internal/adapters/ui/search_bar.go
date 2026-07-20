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

// SearchBar is the fuzzy-search input above the session list.
type SearchBar struct {
	*tview.InputField
	onSearch   func(string)
	onEscape   func()
	onNavigate func()
}

func NewSearchBar() *SearchBar {
	sb := &SearchBar{InputField: tview.NewInputField()}
	sb.build()
	return sb
}

func (sb *SearchBar) build() {
	sb.SetLabel("🔍 ").
		SetLabelColor(tcell.GetColor(colorAccent)).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.GetColor(colorPrimary)).
		SetPlaceholder("fuzzy search sessions…").
		SetPlaceholderTextColor(tcell.GetColor(colorSecondary)).
		SetBorder(true).
		SetTitle(" Search ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder))

	sb.SetChangedFunc(func(text string) {
		if sb.onSearch != nil {
			sb.onSearch(text)
		}
	})

	sb.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyESC:
			if sb.onEscape != nil {
				sb.onEscape()
			}
			return nil
		case tcell.KeyDown, tcell.KeyUp, tcell.KeyEnter:
			if sb.onNavigate != nil {
				sb.onNavigate()
			}
			return nil
		}
		return e
	})
}

func (sb *SearchBar) OnSearch(fn func(string)) *SearchBar { sb.onSearch = fn; return sb }
func (sb *SearchBar) OnEscape(fn func()) *SearchBar       { sb.onEscape = fn; return sb }
func (sb *SearchBar) OnNavigate(fn func()) *SearchBar     { sb.onNavigate = fn; return sb }
