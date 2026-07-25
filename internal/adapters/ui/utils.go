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
	"strings"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// activeThreshold is how recently session_activity must have updated for a
// session to count as "live". tmux refreshes session_activity on any pane
// output, so a session running a build/log/training loop lands well inside
// this window; an idle shell or a finished task falls outside it. 60s is
// forgiving enough that an interactive program paused for thought (vim, a
// REPL) does not flicker back to 💤 between keystrokes.
const activeThreshold = 60 * time.Second

// formatSessionLine renders one list row with fixed-width columns so that the
// Name and Last Attached columns stay aligned across rows of different length:
//
//	▶?(if current) 📌(if pinned) ⚡/⏳/💤  Name__________  N win___  Last Attached: <rel>.
//
// Color tags sit OUTSIDE the %-N width specifiers so fmt pads only the visible
// text (otherwise the tag bytes would corrupt the column width).
func formatSessionLine(s domain.Session, current string) string {
	// isCurrent drives both the row marker and the name styling so the "you are
	// here" signal has a single semantic source. The marker column is a fixed
	// 2 visible cells (▶+space, or space+space) so every row left-aligns
	// whether or not it is the current session.
	isCurrent := current != "" && s.Name == current
	marker := ""
	if isCurrent {
		marker = "[" + colorAccent + "]▶[-]"
	}
	// pin column: fixed 3 cells so pinned/unpinned rows stay aligned.
	pin := "  "
	if s.Pinned {
		pin = "[" + colorGreen + "]📌[-]"
	}
	icon := activityIcon(s.LastActivity, s.Attached)
	// The current session's name uses the accent color (matching the ▶ marker)
	// so it stands out beyond the row marker alone; other rows keep the default
	// primary + bold. Color is the reliable differentiator — the ::b flag is
	// terminal/font-dependent and may not render visibly. The tag stays outside
	// the %-20s so column alignment is unaffected.
	nameColor := colorPrimary + "::b"
	if isCurrent {
		nameColor = colorAccent + "::b"
	}
	name := "[" + nameColor + "]" + fmt.Sprintf("%-20s", s.Name) + "[-]"
	wins := "[" + colorSecondary + "]" + fmt.Sprintf("%-8s", fmt.Sprintf("%d win", s.WindowsCount)) + "[-]"
	attach := "[" + colorDim + "]Last Attached: " + humanizeDuration(s.LastAttached) + "[-]"
	line := fmt.Sprintf("%s %s%s %s  %s  %s", marker, pin, icon, name, wins, attach)
	// Tags append at the row tail (after Last Attached) so the fixed-width
	// columns and the "Last Attached:" byte offset are unaffected — the color
	// tags never enter the %-N spec, preserving TestFormatSessionLineAlignment.
	if badges := renderTagBadgesForList(s.Tags); badges != "" {
		line += "  " + badges
	}
	return line
}

// activityIcon picks the three-state status emoji. The attached flag is
// checked first so a reliable fact always wins over the 60s activity
// heuristic — a session you are watching never reads as idle just because
// you paused to read. Each icon maps to a kill decision:
//
//	⚡ attached          (you are in this session — never kill)
//	⏳ detached + live  (you left, but a task is still running — don't kill)
//	💤 detached + idle  (no recent activity — safe to clean up)
//
// "live" means session_activity is within activeThreshold of now. A zero
// LastActivity is treated as never-active: a detached session with no
// activity reads as idle (safe to clean up), while an attached one still
// shows ⚡ because the attached fact takes priority. The zero treatment
// mirrors humanizeDuration, which renders the same value as "never".
func activityIcon(last time.Time, attached bool) string {
	if attached {
		return "[" + colorGreen + "]⚡[-]"
	}
	if time.Since(last) <= activeThreshold {
		return "[" + colorYellow + "]⏳[-]"
	}
	return "[" + colorSecondary + "]💤[-]"
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

// tagChip renders one tag as a black-on-accent pill. The trailing [-:-:-]
// resets foreground/background/style, so callers must not wrap the result in
// another color tag — that inner reset would clash with an outer wrap.
func tagChip(t string) string {
	return fmt.Sprintf("[black:%s] %s [-:-:-]", colorAccent, t)
}

// renderTagChips renders every tag as a pill for the details pane (no
// truncation).
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	chips := make([]string, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, tagChip(t))
	}
	return strings.Join(chips, " ")
}

// renderTagBadgesForList renders at most two tag chips for the list row (space
// is tight) and collapses any overflow into a dim "+N" marker, matching the
// lazyssh list style. Returns "" when there are no tags so the row tail stays
// clean.
func renderTagBadgesForList(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	const maxTags = 2
	shown := tags
	if len(tags) > maxTags {
		shown = tags[:maxTags]
	}
	parts := make([]string, 0, len(shown)+1)
	for _, t := range shown {
		parts = append(parts, tagChip(t))
	}
	if extra := len(tags) - len(shown); extra > 0 {
		parts = append(parts, fmt.Sprintf("[%s]+%d[-]", colorDim, extra))
	}
	return strings.Join(parts, " ")
}
