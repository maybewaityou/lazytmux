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
	"testing"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

func TestFormatSessionLine(t *testing.T) {
	s := domain.Session{Name: "main", Attached: true, WindowsCount: 3, Pinned: true}
	line := formatSessionLine(s)
	if !strings.Contains(line, "main") {
		t.Errorf("missing name: %q", line)
	}
	if !strings.Contains(line, "3 win") {
		t.Errorf("missing windows count: %q", line)
	}
	if !strings.Contains(line, "📌") {
		t.Errorf("missing pin marker: %q", line)
	}
	if !strings.Contains(line, "●") {
		t.Errorf("missing attached dot: %q", line)
	}
}

func TestFormatSessionLineUnattached(t *testing.T) {
	s := domain.Session{Name: "dev", Attached: false, WindowsCount: 1, Pinned: false}
	line := formatSessionLine(s)
	if strings.Contains(line, "●") {
		t.Errorf("should not show attached dot: %q", line)
	}
	if strings.Contains(line, "📌") {
		t.Errorf("should not show pin marker: %q", line)
	}
}
