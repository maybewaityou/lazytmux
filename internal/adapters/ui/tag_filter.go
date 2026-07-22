/*
Copyright 2026 MeePwn

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package ui

import (
	"sort"
	"strings"

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

// formatTagItem renders one row of the tag-filter modal: a checkbox plus the
// tag name. The checked box uses colorGreen so the selection state is visible
// at a glance even on a dim terminal.
func formatTagItem(tag string, selected bool) string {
	if selected {
		return "[" + colorGreen + "][x][-] " + tag
	}
	return "[ ] " + tag
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
