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

func TestDetailsRenderMarksCurrent(t *testing.T) {
	d := NewSessionDetails()
	d.SetCurrent("main")
	d.Render(domain.Session{Name: "main", WindowsCount: 1})
	if got := d.GetText(true); !strings.Contains(got, "(current)") {
		t.Errorf("current session details must contain (current), got: %q", got)
	}
}

func TestDetailsRenderNoMarkWhenNotCurrent(t *testing.T) {
	d := NewSessionDetails()
	d.SetCurrent("main")
	d.Render(domain.Session{Name: "other", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "(current)") {
		t.Errorf("non-current details must not contain (current), got: %q", got)
	}
}

func TestDetailsRenderNoMarkWhenCurrentUnset(t *testing.T) {
	d := NewSessionDetails()
	d.Render(domain.Session{Name: "main", WindowsCount: 1})
	if got := d.GetText(true); strings.Contains(got, "(current)") {
		t.Errorf("unset-current details must not contain (current), got: %q", got)
	}
}
