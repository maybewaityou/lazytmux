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

import "strings"

// fuzzyMatch reports whether every char of needle appears in haystack, in order.
func fuzzyMatch(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	needle = strings.ToLower(needle)
	haystack = strings.ToLower(haystack)
	hi := 0
	for ni := 0; ni < len(needle); ni++ {
		found := false
		for ; hi < len(haystack); hi++ {
			if haystack[hi] == needle[ni] {
				found = true
				hi++
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
