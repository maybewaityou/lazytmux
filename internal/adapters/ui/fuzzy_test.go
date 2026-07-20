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

import "testing"

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		needle, hay string
		want        bool
	}{
		{"mn", "main", true},
		{"main", "main", true},
		{"", "main", true},
		{"xyz", "main", false},
		{"dev", "dev-prod", true},
		{"dp", "dev-prod", true},
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.needle, c.hay); got != c.want {
			t.Errorf("fuzzyMatch(%q,%q)=%v want %v", c.needle, c.hay, got, c.want)
		}
	}
}
