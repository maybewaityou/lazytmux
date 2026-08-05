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
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// spinnerPeriod is the cadence of the loading spinner animation.
const spinnerPeriod = 100 * time.Millisecond

// spinnerFrames is the braille spinner glyph cycle. Advancing through it one
// frame per spinnerPeriod tick produces a smooth clockwise spin.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// spinnerFrame returns the spinner glyph for animation step i. It wraps around
// the cycle, so callers may keep incrementing i without bounds.
func spinnerFrame(i int) rune {
	return spinnerFrames[i%len(spinnerFrames)]
}

// loadingText composes the centered loading line: a cyan spinner glyph followed
// by the message in the primary text color. Extracted as a pure function so the
// wording and color tags are unit-testable like refreshStatusMessage.
func loadingText(msg string, frame int) string {
	return "[" + colorCyan + "]" + string(spinnerFrame(frame)) + "[-] " +
		"[" + colorPrimary + "]" + msg + "[-]"
}

// LoadingOverlay is a centered loading indicator shown while a slow session
// operation runs on a background goroutine. Create (a) and kill (k) both drive
// tmux-resurrect's save.sh, which takes a second or two; running it inline
// blocks the tview event loop and freezes the UI. The overlay's InputCapture
// swallows every key — that is the re-entrancy guard: the user cannot trigger
// a/k again (or quit) while the snapshot is in flight, so no separate busy flag
// is needed.
//
// It reuses the same centered Flex skeleton as SessionForm.Primitive / the
// confirm modals (outer Flex centers horizontally, inner Flex vertically) so
// the loading box sits dead-center in the layout and reads as one of the app's
// modal dialogs rather than free-floating text.
type LoadingOverlay struct {
	text    *tview.TextView
	root    tview.Primitive
	message string
	frame   int
}

// NewLoadingOverlay builds a loading overlay for the given status message.
func NewLoadingOverlay(message string) *LoadingOverlay {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignCenter)
	tv.SetText(loadingText(message, 0))
	tv.SetBorder(true)
	tv.SetBorderColor(tcell.GetColor(colorBorder))
	tv.SetBackgroundColor(tcell.ColorDefault)
	// Swallow every key on the widget and on the wrapping layout. Returning nil
	// from InputCapture consumes the event before any handler (including the
	// global a/k/q bindings) sees it.
	swallow := func(*tcell.EventKey) *tcell.EventKey { return nil }
	tv.SetInputCapture(swallow)

	root := tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tv, 3, 0, true).
			AddItem(nil, 0, 1, false), 44, 0, true).
		AddItem(nil, 0, 1, false)
	root.SetInputCapture(swallow)

	return &LoadingOverlay{text: tv, root: root, message: message}
}

// Primitive returns the centered layout for SetRoot.
func (o *LoadingOverlay) Primitive() tview.Primitive { return o.root }

// advance steps the spinner to the next frame and refreshes the text. It must
// run on the tview main loop (via queueDraw), like every widget mutation.
func (o *LoadingOverlay) advance() {
	o.frame++
	o.text.SetText(loadingText(o.message, o.frame))
}
