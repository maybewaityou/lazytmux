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
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// filterByTags keeps sessions that carry ANY of the given tags (OR semantics).
// An empty tag slice is a pass-through (no filtering), so the no-filter state
// and the "all tags cleared" state both render the full list.
func filterByTags(sessions []domain.Session, tags []string) []domain.Session {
	if len(tags) == 0 {
		return sessions
	}
	wanted := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		wanted[t] = struct{}{}
	}
	out := make([]domain.Session, 0, len(sessions))
	for _, s := range sessions {
		for _, st := range s.Tags {
			if _, ok := wanted[st]; ok {
				out = append(out, s)
				break // OR: one matching tag is enough
			}
		}
	}
	return out
}

// collectTags returns the sorted, de-duplicated union of every session's tags.
// It feeds the tag-filter modal's candidate list.
func collectTags(sessions []domain.Session) []string {
	seen := make(map[string]struct{})
	for _, s := range sessions {
		for _, t := range s.Tags {
			seen[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// filterDescription joins the active filter tags for display in the list title
// and the help modal's status line. Empty input yields "" so callers can treat
// a non-empty result as "filter active".
func filterDescription(tags []string) string {
	return strings.Join(tags, ", ")
}

// formatTagItem renders one row of the tag-filter modal: a filled dot (●) when
// selected, a hollow dot (○) when not, padded with leading and trailing space so
// the marker breathes away from the list border and the tag name. ● uses
// colorGreen so the selection state pops at a glance even on a dim terminal; ○
// uses colorDim.
func formatTagItem(tag string, selected bool) string {
	if selected {
		return "  [" + colorGreen + "]●[-]  " + tag
	}
	return "  [" + colorDim + "]○[-]  " + tag
}

// filterByName keeps sessions whose Name is a fuzzy match for query (reuses
// fuzzyMatch). An empty query is a pass-through. Extracted from handleSearchInput
// so the unified visibleSessions pipeline can compose tag + name filtering.
func filterByName(sessions []domain.Session, query string) []domain.Session {
	if query == "" {
		return sessions
	}
	out := make([]domain.Session, 0, len(sessions))
	for _, s := range sessions {
		if fuzzyMatch(query, s.Name) {
			out = append(out, s)
		}
	}
	return out
}

// applyFilters is the ordered filter pipeline used by visibleSessions: tag
// filter first (narrow by category), then name search (find within category).
// Either stage is a pass-through when its input is empty.
func applyFilters(sessions []domain.Session, tags []string, query string) []domain.Session {
	out := filterByTags(sessions, tags)
	out = filterByName(out, query)
	return out
}

// TagFilterForm is a multi-select modal listing every known tag with a
// checkbox. Space toggles the focused tag, Enter applies the selection, ESC
// cancels (keeping the prior filter). It mirrors the existing modal language
// (centered, Tokyo Night) while adding checkbox navigation on top of tview.List.
type TagFilterForm struct {
	*tview.List
	candidates []string
	selected   map[string]bool
	onApply    func([]string)
	onCancel   func()
}

// NewTagFilterForm builds the modal. candidates is the sorted tag union;
// initial pre-selects the currently active filter so reopening the modal shows
// the live state.
func NewTagFilterForm(candidates, initial []string) *TagFilterForm {
	f := &TagFilterForm{
		List:       tview.NewList(),
		candidates: candidates,
		selected:   make(map[string]bool, len(initial)),
	}
	for _, t := range initial {
		f.selected[t] = true
	}
	f.build()
	return f
}

func (f *TagFilterForm) build() {
	f.ShowSecondaryText(false)
	f.SetBorder(true).
		SetTitle(" Tag filter ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.GetColor(colorBorder)).
		SetTitleColor(tcell.GetColor(colorTitle))
	f.SetSelectedBackgroundColor(tcell.GetColor(colorSelected)).
		SetSelectedTextColor(tcell.GetColor(colorPrimary)).
		SetHighlightFullLine(true)

	for _, tag := range f.candidates {
		f.AddItem(f.formatRow(tag), "", 0, nil)
	}

	f.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyESC:
			if f.onCancel != nil {
				f.onCancel()
			}
			return nil
		case tcell.KeyEnter:
			if f.onApply != nil {
				f.onApply(f.selection())
			}
			return nil
		}
		// Space arrives as KeyRune with rune ' ' (tcell represents printable
		// chars as KeyRune, and tcell.KeySpace is not defined in this version).
		// Toggle the focused row's checkbox.
		if e.Rune() == ' ' {
			idx := f.GetCurrentItem()
			if idx >= 0 && idx < len(f.candidates) {
				tag := f.candidates[idx]
				f.selected[tag] = !f.selected[tag]
				f.SetItemText(idx, f.formatRow(tag), "")
			}
			return nil
		}
		return e
	})
}

// formatRow renders a candidate's checkbox row from its current selection state.
func (f *TagFilterForm) formatRow(tag string) string {
	return formatTagItem(tag, f.selected[tag])
}

// selection returns the active tags in candidate order (stable, reproducible).
func (f *TagFilterForm) selection() []string {
	out := make([]string, 0, len(f.candidates))
	for _, tag := range f.candidates {
		if f.selected[tag] {
			out = append(out, tag)
		}
	}
	return out
}

func (f *TagFilterForm) OnApply(fn func([]string)) *TagFilterForm { f.onApply = fn; return f }
func (f *TagFilterForm) OnCancel(fn func()) *TagFilterForm        { f.onCancel = fn; return f }

// Primitive wraps the list in a centered, fixed-width flex like SessionForm.
func (f *TagFilterForm) Primitive() tview.Primitive {
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(f, 0, 1, true).
			AddItem(nil, 0, 1, false), 40, 0, true).
		AddItem(nil, 0, 1, false)
}
