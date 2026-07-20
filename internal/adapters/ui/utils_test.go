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
	if !strings.Contains(line, "⚡") {
		t.Errorf("missing attached icon: %q", line)
	}
	if strings.Contains(line, "💤") {
		t.Errorf("should not show unattached icon: %q", line)
	}
	if !strings.Contains(line, "Last Attached: never") {
		t.Errorf("missing last attached time for zero value: %q", line)
	}
}

func TestFormatSessionLineUnattached(t *testing.T) {
	s := domain.Session{Name: "dev", Attached: false, WindowsCount: 1, Pinned: false}
	line := formatSessionLine(s)
	if !strings.Contains(line, "💤") {
		t.Errorf("missing unattached icon: %q", line)
	}
	if strings.Contains(line, "⚡") {
		t.Errorf("should not show attached icon: %q", line)
	}
	if strings.Contains(line, "📌") {
		t.Errorf("should not show pin marker: %q", line)
	}
}

// TestFormatSessionLineAlignment verifies that the Name and Last Attached
// columns line up across rows of different name lengths: the fixed-width
// padding must place "Last Attached:" at the same byte offset in every line.
func TestFormatSessionLineAlignment(t *testing.T) {
	short := formatSessionLine(domain.Session{Name: "api", WindowsCount: 1})
	long := formatSessionLine(domain.Session{Name: "a-much-longer-name", WindowsCount: 99})
	iShort := strings.Index(short, "Last Attached:")
	iLong := strings.Index(long, "Last Attached:")
	if iShort < 0 || iLong < 0 {
		t.Fatalf("missing Last Attached column:\n short=%q\n long=%q", short, long)
	}
	if iShort != iLong {
		t.Errorf("Last Attached column not aligned: short@%d long@%d\n short=%q\n long=%q", iShort, iLong, short, long)
	}
}
