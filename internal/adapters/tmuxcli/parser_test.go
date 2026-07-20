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

package tmuxcli

import (
	"testing"
	"time"
)

func TestParseSessions(t *testing.T) {
	// Fields: id|name|attached|created|activity|windows|path
	input := []byte("$1|main|1|1700000000|1700000500|3|/home/me/code\n" +
		"$2|dev|0|1700001000|1700001100|1|/home/me/dev\n")

	got, err := ParseSessions(input)
	if err != nil {
		t.Fatalf("ParseSessions error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got))
	}
	if got[0].Name != "main" || got[0].ID != "$1" {
		t.Errorf("first session mismatch: %+v", got[0])
	}
	if !got[0].Attached || got[1].Attached {
		t.Errorf("attached flags wrong: %v %v", got[0].Attached, got[1].Attached)
	}
	if got[0].WindowsCount != 3 {
		t.Errorf("windows count wrong: %d", got[0].WindowsCount)
	}
	if !got[0].Created.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("created time wrong: %v", got[0].Created)
	}
}

func TestParseSessionsEmpty(t *testing.T) {
	got, err := ParseSessions([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(got))
	}
}

func TestParseSessionsMalformedSkipped(t *testing.T) {
	// A line with too few fields is skipped, not fatal.
	input := []byte("badline\n$1|main|1|1700000000|1700000500|3|/p\n")
	got, err := ParseSessions(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "main" {
		t.Fatalf("expected only the valid session, got %+v", got)
	}
}

func TestParseWindows(t *testing.T) {
	// Fields: index|name|active|command
	input := []byte("0|vim|1|nvim\n1|bash|0|bash\n")

	got, err := ParseWindows(input)
	if err != nil {
		t.Fatalf("ParseWindows error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(got))
	}
	if got[0].Index != 0 || got[0].Name != "vim" || !got[0].Active {
		t.Errorf("first window mismatch: %+v", got[0])
	}
	if got[1].Active {
		t.Error("second window should not be active")
	}
}
