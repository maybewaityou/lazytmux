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
	"errors"
	"testing"
)

func TestListSessions(t *testing.T) {
	runner := &FakeRunner{
		Output: []byte("$1|main|1|1700000000|1700000500|3|/code\n"),
	}
	repo := NewRepository(runner)

	got, err := repo.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "main" {
		t.Fatalf("unexpected sessions: %+v", got)
	}
	wantArgs := []string{"list-sessions", "-F",
		"#{session_id}|#{session_name}|#{session_attached}|#{session_created}|#{session_activity}|#{session_windows}|#{session_path}"}
	for i, a := range wantArgs {
		if runner.LastArgs[i] != a {
			t.Errorf("arg %d: got %q want %q", i, runner.LastArgs[i], a)
		}
	}
}

func TestListSessionsError(t *testing.T) {
	repo := NewRepository(&FakeRunner{Err: errors.New("no server")})
	if _, err := repo.ListSessions(); err == nil {
		t.Fatal("expected error when tmux fails")
	}
}

func TestListWindows(t *testing.T) {
	runner := &FakeRunner{
		Output: []byte("0|vim|1|nvim\n1|sh|0|bash\n"),
	}
	repo := NewRepository(runner)

	got, err := repo.ListWindows("main")
	if err != nil {
		t.Fatalf("ListWindows error: %v", err)
	}
	if len(got) != 2 || !got[0].Active {
		t.Fatalf("unexpected windows: %+v", got)
	}
	// args = [list-windows, -t, <name>, -F, <fmt>]; name sits at index 2.
	if got := runner.LastArgs; len(got) < 3 || got[0] != "list-windows" || got[1] != "-t" || got[2] != "main" {
		t.Errorf("expected [list-windows -t main -F ...], got args %v", got)
	}
}
