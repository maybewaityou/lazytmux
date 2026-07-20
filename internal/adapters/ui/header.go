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
	"time"

	"github.com/rivo/tview"
)

// AppHeader is the top bar: brand left, version/commit center, repo+time right.
type AppHeader struct {
	*tview.Flex
}

func NewAppHeader(version, commit, repoURL string) *AppHeader {
	h := &AppHeader{Flex: tview.NewFlex()}
	h.build(version, commit, repoURL)
	return h
}

func (h *AppHeader) build(version, commit, repoURL string) {
	left := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	left.SetText("🚀 [" + colorPrimary + "::b]lazy[-][" + colorAccent + "::b]tmux[-]")

	center := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	center.SetText(makeTag(version, colorGreen) + "  " + makeTag(shortCommit(commit), colorPurple))

	right := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	currentTime := time.Now().Format("Mon, 02 Jan 2006 15:04")
	right.SetText("[" + colorAccent + "]🔗 " + repoURL + "[-]  [" + colorSecondary + "]• " + currentTime + "[-]")

	bar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 1, false).
		AddItem(center, 0, 1, false).
		AddItem(right, 0, 1, false)

	sep := tview.NewTextView().SetDynamicColors(true)
	sep.SetText("[" + colorBorder + "]" + strings.Repeat("─", 200) + "[-]")

	h.Flex.SetDirection(tview.FlexRow).AddItem(bar, 1, 0, false).AddItem(sep, 1, 0, false)
}

func shortCommit(c string) string {
	c = strings.TrimSpace(c)
	if c == "" || c == "unknown" {
		return ""
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// makeTag returns a rectangular-looking colored chip for the given text.
func makeTag(text, bg string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "[black:" + bg + "::b]  " + text + "  [-]"
}
