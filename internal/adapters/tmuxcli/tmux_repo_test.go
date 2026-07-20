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

	"github.com/maybewaityou/lazytmux/internal/core/ports"
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
	if gotArgs := runner.LastArgs; len(gotArgs) < 3 || gotArgs[0] != "list-windows" || gotArgs[1] != "-t" || gotArgs[2] != "main" {
		t.Errorf("expected [list-windows -t main -F ...], got args %v", gotArgs)
	}
}

func TestCreateKillRename(t *testing.T) {
	runner := &FakeRunner{}
	repo := NewRepository(runner)

	if err := repo.CreateSession("foo"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if !equalArgs(runner.AllCalls[len(runner.AllCalls)-1], "new-session", "-d", "-s", "foo") {
		t.Errorf("create args: %v", runner.AllCalls)
	}

	if err := repo.KillSession("foo"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	if !equalArgs(runner.AllCalls[len(runner.AllCalls)-1], "kill-session", "-t", "foo") {
		t.Errorf("kill args: %v", runner.AllCalls)
	}

	if err := repo.RenameSession("foo", "bar"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}
	if !equalArgs(runner.AllCalls[len(runner.AllCalls)-1], "rename-session", "-t", "foo", "bar") {
		t.Errorf("rename args: %v", runner.AllCalls)
	}
}

func TestSwitchOrAttachInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	runner := &FakeRunner{}
	repo := NewRepository(runner)

	if err := repo.SwitchOrAttach("main"); err != nil {
		t.Fatalf("SwitchOrAttach inside tmux: %v", err)
	}
	if !equalArgs(runner.LastArgs, "switch-client", "-t", "main") {
		t.Errorf("switch-client args: %v", runner.LastArgs)
	}
}

func TestSwitchOrAttachOutsideTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	repo := NewRepository(&FakeRunner{})

	err := repo.SwitchOrAttach("main")
	if err == nil {
		t.Fatal("expected ErrSuspendRequired when outside tmux")
	}
	if !errors.Is(err, ports.ErrSuspendRequired) {
		t.Fatalf("expected ErrSuspendRequired, got %v", err)
	}
}

func equalArgs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
