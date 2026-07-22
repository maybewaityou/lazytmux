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
	"reflect"
	"testing"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

func TestFilterByTags(t *testing.T) {
	sessions := []domain.Session{
		{Name: "api", Tags: []string{"work"}},
		{Name: "notes", Tags: []string{"personal"}},
		{Name: "workbench", Tags: []string{"work", "personal"}},
		{Name: "legacy", Tags: nil},
	}
	for _, tc := range []struct {
		name string
		tags []string
		want []string // expected session Names, in input order
	}{
		{"no filter returns all", nil, []string{"api", "notes", "workbench", "legacy"}},
		{"single tag (OR within session)", []string{"work"}, []string{"api", "workbench"}},
		{"multiple tags OR", []string{"work", "personal"}, []string{"api", "notes", "workbench"}},
		{"no tag match drops everything", []string{"nope"}, []string{}},
		{"session without tags is filtered out", []string{"work"}, []string{"api", "workbench"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := filterByTags(sessions, tc.tags)
			gotNames := namesOf(got)
			if !reflect.DeepEqual(gotNames, tc.want) {
				t.Errorf("filterByTags(%v) = %v, want %v", tc.tags, gotNames, tc.want)
			}
		})
	}
}

func TestCollectTags(t *testing.T) {
	sessions := []domain.Session{
		{Name: "a", Tags: []string{"work", "urgent"}},
		{Name: "b", Tags: []string{"work"}},
		{Name: "c", Tags: nil},
		{Name: "d", Tags: []string{"personal"}},
	}
	got := collectTags(sessions)
	// Sorted, de-duplicated.
	want := []string{"personal", "urgent", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectTags = %v, want %v", got, want)
	}
}

func TestCollectTagsEmpty(t *testing.T) {
	if got := collectTags(nil); len(got) != 0 {
		t.Errorf("collectTags(nil) = %v, want empty", got)
	}
}

func TestFilterDescription(t *testing.T) {
	if got := filterDescription(nil); got != "" {
		t.Errorf("filterDescription(nil) = %q, want empty", got)
	}
	if got := filterDescription([]string{"work", "personal"}); got != "work, personal" {
		t.Errorf("filterDescription = %q, want %q", got, "work, personal")
	}
}

func TestFormatTagItem(t *testing.T) {
	unchecked := formatTagItem("work", false)
	if want := "  [" + colorDim + "]○[-]  work"; unchecked != want {
		t.Errorf("unchecked = %q, want %q", unchecked, want)
	}
	checked := formatTagItem("work", true)
	if want := "  [" + colorGreen + "]●[-]  work"; checked != want {
		t.Errorf("checked = %q, want %q", checked, want)
	}
}

func namesOf(ss []domain.Session) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.Name)
	}
	return out
}

func TestFilterByName(t *testing.T) {
	sessions := []domain.Session{{Name: "api"}, {Name: "notes"}, {Name: "legacy"}}
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"", []string{"api", "notes", "legacy"}}, // empty query = pass-through
		{"api", []string{"api"}},
		{"n", []string{"notes"}}, // subsequence match
		{"zzz", []string{}},
	} {
		got := namesOf(filterByName(sessions, tc.query))
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("filterByName(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestApplyFilters(t *testing.T) {
	sessions := []domain.Session{
		{Name: "api", Tags: []string{"work"}},
		{Name: "notes", Tags: []string{"personal"}},
		{Name: "legacy", Tags: nil},
	}
	// tag only
	got := namesOf(applyFilters(sessions, []string{"work"}, ""))
	want := []string{"api"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tag-only = %v, want %v", got, want)
	}
	// tag OR + name subsequence: tags {work,personal} keep api+notes; query "not" keeps notes
	got = namesOf(applyFilters(sessions, []string{"work", "personal"}, "not"))
	want = []string{"notes"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tag+name = %v, want %v", got, want)
	}
	// no filters = pass-through (order preserved)
	got = namesOf(applyFilters(sessions, nil, ""))
	want = []string{"api", "notes", "legacy"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("no-filter = %v, want %v", got, want)
	}
}
