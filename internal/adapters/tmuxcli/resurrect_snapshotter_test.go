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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maybewaityou/lazytmux/internal/core/services"
)

type runnerResponse struct {
	output []byte
	err    error
}

type queuedRunner struct {
	responses []runnerResponse
	calls     [][]string
}

func (r *queuedRunner) RunOutput(args ...string) ([]byte, error) {
	r.calls = append(r.calls, slices.Clone(args))
	if len(r.responses) == 0 {
		return nil, errors.New("unexpected tmux call")
	}
	response := r.responses[0]
	r.responses = r.responses[1:]
	return response.output, response.err
}

func (*queuedRunner) RunInteractive(...string) error { return nil }
func (*queuedRunner) LookPath() error                { return nil }

type executableCall struct {
	path string
	args []string
}

type fakeExecutableRunner struct {
	calls []executableCall
	err   error
}

func (r *fakeExecutableRunner) Run(path string, args ...string) error {
	r.calls = append(r.calls, executableCall{path: path, args: slices.Clone(args)})
	return r.err
}

type fakeWarningLogger struct {
	entries []warningEntry
}

type warningEntry struct {
	message string
	fields  []any
}

func (l *fakeWarningLogger) Warnw(message string, fields ...any) {
	l.entries = append(l.entries, warningEntry{message: message, fields: slices.Clone(fields)})
}

func TestResurrectSnapshotterSavesAndVerifiesTargetSession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := executableFile(t, dir)
	writeSnapshot(t, dir, "work")
	tmux := &queuedRunner{responses: []runnerResponse{
		{output: []byte(script + "\n")},
		{output: []byte(dir + "\n")},
	}}
	executable := &fakeExecutableRunner{}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, executable, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	wantTmux := [][]string{
		{"show-options", "-gqv", "@resurrect-save-script-path"},
		{"show-options", "-gqv", "@resurrect-dir"},
	}
	if !slices.EqualFunc(tmux.calls, wantTmux, slices.Equal[[]string]) {
		t.Fatalf("tmux calls = %v, want %v", tmux.calls, wantTmux)
	}
	wantExec := []executableCall{{path: script, args: []string{"quiet"}}}
	if !slices.EqualFunc(executable.calls, wantExec, equalExecutableCall) {
		t.Fatalf("executable calls = %v, want %v", executable.calls, wantExec)
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestResurrectSnapshotterSkipsUnavailablePlugin(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{{output: []byte("\n")}}}
	executable := &fakeExecutableRunner{}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, executable, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	if len(executable.calls) != 0 {
		t.Fatalf("executable calls = %v, want none", executable.calls)
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestResurrectSnapshotterSkipsWhenNoServerIsRunning(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{{err: errors.New("no server running on /tmp/tmux/default")}}}
	executable := &fakeExecutableRunner{}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, executable, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	if len(executable.calls) != 0 || len(logger.entries) != 0 {
		t.Fatalf("no-server should be a silent no-op, executable=%v warnings=%v", executable.calls, logger.entries)
	}
}

func TestResurrectSnapshotterWarnsWhenDiscoveryFails(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{{err: errors.New("permission denied")}}}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	assertWarning(t, logger, "discover save script", "work")
}

func TestResurrectSnapshotterWarnsWhenScriptPathIsRelative(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{{output: []byte("save.sh")}}}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	assertWarning(t, logger, "validate save script", "work")
}

func TestResurrectSnapshotterWarnsWhenScriptIsInvalid(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{{output: []byte(filepath.Join(t.TempDir(), "missing"))}}}
	executable := &fakeExecutableRunner{}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, executable, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	if len(executable.calls) != 0 {
		t.Fatalf("executable calls = %v, want none", executable.calls)
	}
	assertWarning(t, logger, "validate save script", "work")
}

func TestResurrectSnapshotterWarnsWhenScriptFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := executableFile(t, dir)
	tmux := &queuedRunner{responses: []runnerResponse{{output: []byte(script)}}}
	executable := &fakeExecutableRunner{err: errors.New("exit status 1: disk full")}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, executable, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	assertWarning(t, logger, "run save script", "work")
}

func TestResurrectSnapshotterWarnsWhenSuccessfulScriptMissesSession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := executableFile(t, dir)
	writeSnapshot(t, dir, "other")
	tmux := &queuedRunner{responses: []runnerResponse{
		{output: []byte(script)},
		{output: []byte(dir)},
	}}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	assertWarning(t, logger, "verify snapshot", "work")
}

func TestResurrectSnapshotterAcceptsExistingValidLastSnapshot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := executableFile(t, dir)
	writeSnapshot(t, dir, "work")
	last := filepath.Join(dir, "last")
	before, err := os.Stat(last)
	if err != nil {
		t.Fatalf("stat last snapshot: %v", err)
	}
	tmux := &queuedRunner{responses: []runnerResponse{
		{output: []byte(script)},
		{output: []byte(dir)},
	}}
	logger := &fakeWarningLogger{}
	snapshotter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), logger)

	snapshotter.SaveSession("work")

	after, err := os.Stat(last)
	if err != nil {
		t.Fatalf("stat last snapshot after save: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("test fixture unexpectedly changed mtime: before=%v after=%v", before.ModTime(), after.ModTime())
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestDefaultResurrectDir(t *testing.T) {
	home := t.TempDir()

	t.Run("legacy directory takes precedence", func(t *testing.T) {
		legacy := filepath.Join(home, ".tmux", "resurrect")
		if err := os.MkdirAll(legacy, 0o755); err != nil {
			t.Fatalf("create legacy directory: %v", err)
		}
		if got := defaultResurrectDir(home); got != legacy {
			t.Fatalf("defaultResurrectDir = %q, want %q", got, legacy)
		}
	})

	if err := os.RemoveAll(filepath.Join(home, ".tmux")); err != nil {
		t.Fatalf("remove legacy directory: %v", err)
	}
	t.Run("XDG data home", func(t *testing.T) {
		dataHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", dataHome)
		want := filepath.Join(dataHome, "tmux", "resurrect")
		if got := defaultResurrectDir(home); got != want {
			t.Fatalf("defaultResurrectDir = %q, want %q", got, want)
		}
	})

	t.Run("home local share fallback", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		want := filepath.Join(home, ".local", "share", "tmux", "resurrect")
		if got := defaultResurrectDir(home); got != want {
			t.Fatalf("defaultResurrectDir = %q, want %q", got, want)
		}
	})
}

func TestResurrectSnapshotterWithRealPlugin(t *testing.T) {
	script := os.Getenv("LAZYTMUX_RESURRECT_SAVE_SCRIPT")
	if script == "" {
		t.Skip("set LAZYTMUX_RESURRECT_SAVE_SCRIPT to run the real-plugin integration test")
	}

	tmuxDir, err := os.MkdirTemp("/tmp", "lt-")
	if err != nil {
		t.Fatalf("create short tmux socket directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxDir) })
	snapshotDir := t.TempDir()
	socket := "lazytmux-resurrect-integration"
	session := "lazytmux-save-test"
	runner := socketRunner{socket: socket, env: append(os.Environ(), "TMUX_TMPDIR="+tmuxDir)}
	t.Cleanup(func() { _, _ = runner.RunOutput("kill-server") })
	if _, err := runner.RunOutput("new-session", "-d", "-s", "bootstrap"); err != nil {
		t.Fatalf("start isolated tmux server: %v", err)
	}
	if _, err := runner.RunOutput("set-option", "-g", resurrectSavePathOption, script); err != nil {
		t.Fatalf("set resurrect script option: %v", err)
	}
	if _, err := runner.RunOutput("set-option", "-g", resurrectDirOption, snapshotDir); err != nil {
		t.Fatalf("set resurrect directory option: %v", err)
	}
	pid, err := runner.RunOutput("display-message", "-p", "#{pid}")
	if err != nil {
		t.Fatalf("resolve isolated tmux server PID: %v", err)
	}
	socketPath := filepath.Join(tmuxDir, fmt.Sprintf("tmux-%d", os.Getuid()), socket)
	t.Setenv("TMUX", socketPath+","+strings.TrimSpace(string(pid))+",0")
	logger := &fakeWarningLogger{}
	snapshotter := NewResurrectSnapshotter(runner, t.TempDir(), logger)
	svc := services.NewSessionService(NewRepository(runner), nil, snapshotter, nil)

	if err := svc.CreateSession(session); err != nil {
		t.Fatalf("create session through service: %v", err)
	}

	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
	if err := verifySnapshot(filepath.Join(snapshotDir, "last"), session); err != nil {
		t.Fatalf("real resurrect snapshot: %v", err)
	}
}

type socketRunner struct {
	socket string
	env    []string
}

func (r socketRunner) RunOutput(args ...string) ([]byte, error) {
	allArgs := append([]string{"-f", "/dev/null", "-L", r.socket}, args...)
	cmd := exec.Command("tmux", allArgs...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tmux %v: %s: %w", allArgs, strings.TrimSpace(string(out)), err)
	}
	return out, nil
}

func (socketRunner) RunInteractive(...string) error { return nil }
func (socketRunner) LookPath() error                { return nil }

func TestExecutableRunnerSurfacesStderr(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "save.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf 'disk full' >&2\nexit 7\n"), 0o755); err != nil {
		t.Fatalf("write failing executable: %v", err)
	}

	err := (osExecutableRunner{}).Run(script, "quiet")
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("Run error = %v, want stderr", err)
	}
}

func executableFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "save.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}

func writeSnapshot(t *testing.T, dir, session string) {
	t.Helper()
	path := filepath.Join(dir, "snapshot.txt")
	content := "pane\t" + session + "\t1\t0\nwindow\t" + session + "\t1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if err := os.Symlink(filepath.Base(path), filepath.Join(dir, "last")); err != nil {
		t.Fatalf("link last snapshot: %v", err)
	}
}

func equalExecutableCall(left, right executableCall) bool {
	return left.path == right.path && slices.Equal(left.args, right.args)
}

func assertWarning(t *testing.T, logger *fakeWarningLogger, stage, session string) {
	t.Helper()
	if len(logger.entries) != 1 {
		t.Fatalf("warnings = %v, want one", logger.entries)
	}
	fields := logger.entries[0].fields
	if !containsField(fields, "stage", stage) || !containsField(fields, "session", session) {
		t.Fatalf("warning fields = %v, want stage=%q session=%q", fields, stage, session)
	}
}

func containsField(fields []any, key string, value any) bool {
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] == key && fields[i+1] == value {
			return true
		}
	}
	return false
}
