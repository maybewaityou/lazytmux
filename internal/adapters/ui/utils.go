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
	"fmt"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// formatSessionLine renders one list row with fixed-width columns so that the
// Name and Last Attached columns stay aligned across rows of different length:
//
//	📌(if pinned) ⚡/💤(attached)  Name__________  N win___  Last Attached: <rel>.
//
// Color tags sit OUTSIDE the %-N width specifiers so fmt pads only the visible
// text (otherwise the tag bytes would corrupt the column width).
func formatSessionLine(s domain.Session) string {
	// pin column: fixed 3 cells so pinned/unpinned rows stay aligned.
	pin := "   "
	if s.Pinned {
		pin = "[" + colorGreen + "]📌[-] "
	}
	icon := "[" + colorSecondary + "]💤[-]"
	if s.Attached {
		icon = "[" + colorGreen + "]⚡[-]"
	}
	name := "[" + colorPrimary + "::b]" + fmt.Sprintf("%-20s", s.Name) + "[-]"
	wins := "[" + colorSecondary + "]" + fmt.Sprintf("%-8s", fmt.Sprintf("%d win", s.WindowsCount)) + "[-]"
	attach := "[" + colorDim + "]Last Attached: " + humanizeDuration(s.LastAttached) + "[-]"
	return fmt.Sprintf("%s%s %s  %s  %s", pin, icon, name, wins, attach)
}

// humanizeDuration renders a timestamp as a relative, human-readable duration
// (e.g. "5m ago", "2h ago"). A zero time yields "never".
func humanizeDuration(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	case d < 365*24*time.Hour:
		months := int(d.Hours()) / (24 * 30)
		if months < 1 {
			months = 1
		}
		return fmt.Sprintf("%dmo ago", months)
	default:
		years := int(d.Hours()) / (24 * 365)
		if years < 1 {
			years = 1
		}
		return fmt.Sprintf("%dy ago", years)
	}
}
