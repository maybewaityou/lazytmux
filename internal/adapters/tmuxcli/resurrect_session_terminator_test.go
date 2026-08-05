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
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/services"
)

type deletionTmux struct {
	pid        string
	before     []string
	after      []string
	killErr    error
	killed     bool
	final      bool
	calls      [][]string
	socketPath string
}

func (t *deletionTmux) Run(socketPath string, args ...string) ([]byte, error) {
	t.socketPath = socketPath
	t.calls = append(t.calls, slices.Clone(args))
	switch args[0] {
	case "display-message":
		if t.killed && t.final {
			return nil, errors.New("no server running on " + socketPath)
		}
		return []byte(t.pid + "\n"), nil
	case "list-sessions":
		sessions := t.before
		if t.killed {
			sessions = t.after
		}
		return []byte(strings.Join(sessions, "\n") + "\n"), nil
	case "kill-session":
		if t.killErr != nil {
			return nil, t.killErr
		}
		t.killed = true
		return nil, nil
	default:
		return nil, errors.New("unexpected tmux command")
	}
}

type snapshotRewriter struct {
	dir      string
	sessions []string
	calls    int
}

func (r *snapshotRewriter) RunEnv(string, []string, ...string) error {
	r.calls++
	writeSnapshotSessionsRaw(r.dir, r.sessions)
	return nil
}

type fakeHelperLauncher struct {
	attempted bool
	err       error
	configs   []killHelperConfig
}

func (l *fakeHelperLauncher) Run(config killHelperConfig) (bool, error) {
	l.configs = append(l.configs, config)
	return l.attempted, l.err
}

func TestResurrectPersistenceWithoutPluginPreservesPlainLifecycle(t *testing.T) {
	t.Parallel()

	tmux := &queuedRunner{responses: []runnerResponse{
		{},
		{output: []byte("\n")},
		{output: []byte("\n")},
		{},
	}}
	logger := &fakeWarningLogger{}
	persistence := NewResurrectPersistence(tmux, t.TempDir(), logger)
	service := services.NewSessionService(NewRepository(tmux), nil, persistence, persistence, nil)

	if err := service.CreateSession("work"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := service.KillSession("work"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	want := [][]string{
		{"new-session", "-d", "-s", "work"},
		{"show-options", "-gqv", resurrectSavePathOption},
		{"show-options", "-gqv", resurrectSavePathOption},
		{"kill-session", "-t", "work"},
	}
	assertRunnerCalls(t, tmux, want)
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestResurrectTerminatorFallsBackWhenPluginIsUnavailable(t *testing.T) {
	tmux := &queuedRunner{responses: []runnerResponse{
		{output: []byte("\n")},
		{},
	}}
	launcher := &fakeHelperLauncher{}
	adapter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), &fakeWarningLogger{})
	adapter.launcher = launcher

	if err := adapter.KillSession("doomed"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	want := [][]string{
		{"show-options", "-gqv", resurrectSavePathOption},
		{"kill-session", "-t", "doomed"},
	}
	assertRunnerCalls(t, tmux, want)
	if len(launcher.configs) != 0 {
		t.Fatalf("helper configs = %v, want none", launcher.configs)
	}
}

func TestResurrectTerminatorFallsBackWhenPluginIsBroken(t *testing.T) {
	t.Parallel()

	killErr := errors.New("kill failed")
	tmux := &queuedRunner{responses: []runnerResponse{
		{output: []byte("relative-save.sh")},
		{err: killErr},
	}}
	logger := &fakeWarningLogger{}
	launcher := &fakeHelperLauncher{}
	adapter := newResurrectSnapshotter(tmux, &fakeExecutableRunner{}, t.TempDir(), logger)
	adapter.launcher = launcher

	if err := adapter.KillSession("doomed"); !errors.Is(err, killErr) {
		t.Fatalf("KillSession error = %v, want %v", err, killErr)
	}
	want := [][]string{
		{"show-options", "-gqv", resurrectSavePathOption},
		{"kill-session", "-t", "doomed"},
	}
	assertRunnerCalls(t, tmux, want)
	if len(launcher.configs) != 0 {
		t.Fatalf("helper configs = %v, want none", launcher.configs)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("warnings = %v, want one", logger.entries)
	}
	if logger.entries[0].message != "tmux-resurrect deletion reconciliation failed" {
		t.Fatalf("warning message = %q, want deletion reconciliation warning", logger.entries[0].message)
	}
	assertWarning(t, logger, "validate save script", "doomed")
}

func TestResurrectTerminatorReturnsHelperPrimaryKillError(t *testing.T) {
	killErr := errors.New("kill failed")
	launcher := &fakeHelperLauncher{attempted: true, err: killErr}
	adapter := &resurrectSnapshotter{launcher: launcher}
	adapterConfig := deletionConfig(t, t.TempDir(), "doomed")
	adapter.tmux = &queuedRunner{responses: []runnerResponse{
		{output: []byte(adapterConfig.SaveScript)},
		{output: []byte(adapterConfig.SnapshotDir)},
		{output: []byte(adapterConfig.SocketPath + "\t" + adapterConfig.ServerPID)},
	}}
	adapter.home = adapterConfig.Home

	if err := adapter.KillSession("doomed"); !errors.Is(err, killErr) {
		t.Fatalf("KillSession error = %v, want %v", err, killErr)
	}
	if len(launcher.configs) != 1 {
		t.Fatalf("helper calls = %d, want 1", len(launcher.configs))
	}
}

func TestDeletionRuntimeSavesSnapshotWithoutDeletedSession(t *testing.T) {
	dir := t.TempDir()
	writeSnapshotSessions(t, dir, []string{"doomed", "keeper"})
	tmux := &deletionTmux{pid: "42", before: []string{"doomed", "keeper"}, after: []string{"keeper"}}
	executable := &snapshotRewriter{dir: dir, sessions: []string{"keeper"}}
	logger := &fakeWarningLogger{}
	runtime := deletionRuntime{
		config: deletionConfig(t, dir, "doomed"), tmux: tmux, executable: executable, logger: logger,
	}

	last, before, err := runtime.prepare()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := runtime.kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	runtime.reconcile(last, before)

	if executable.calls != 1 {
		t.Fatalf("save calls = %d, want 1", executable.calls)
	}
	summary, err := parseSnapshot(filepath.Join(dir, "last"))
	if err != nil {
		t.Fatalf("parse last: %v", err)
	}
	if got := summary.sessions(); !sameSessionSet(got, []string{"keeper"}) {
		t.Fatalf("snapshot sessions = %v, want [keeper]", got)
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestDeletionRuntimeUnlinksLastButPreservesHistoryAfterFinalSession(t *testing.T) {
	dir := t.TempDir()
	target := writeSnapshotSessions(t, dir, []string{"doomed"})
	tmux := &deletionTmux{pid: "42", before: []string{"doomed"}, final: true}
	logger := &fakeWarningLogger{}
	runtime := deletionRuntime{
		config: deletionConfig(t, dir, "doomed"), tmux: tmux, executable: &snapshotRewriter{}, logger: logger,
	}

	last, before, err := runtime.prepare()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := runtime.kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	runtime.reconcile(last, before)

	if _, err := os.Lstat(filepath.Join(dir, "last")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("last should be unlinked, err=%v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("history target should remain: %v", err)
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestDeletionRuntimePreservesLastWhenTargetChangesBeforeFinalDelete(t *testing.T) {
	dir := t.TempDir()
	target := writeSnapshotSessions(t, dir, []string{"doomed"})
	tmux := &deletionTmux{pid: "42", before: []string{"doomed"}, final: true}
	logger := &fakeWarningLogger{}
	runtime := deletionRuntime{
		config: deletionConfig(t, dir, "doomed"), tmux: tmux, executable: &snapshotRewriter{}, logger: logger,
	}

	last, before, err := runtime.prepare()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := runtime.kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if err := os.WriteFile(target, []byte("pane\tother\t1\t0\nwindow\tother\t1\n"), 0o600); err != nil {
		t.Fatalf("rewrite target: %v", err)
	}
	runtime.reconcile(last, before)

	if _, err := os.Lstat(filepath.Join(dir, "last")); err != nil {
		t.Fatalf("last should be preserved after target change: %v", err)
	}
	assertWarning(t, logger, "unlink-last", "doomed")
}

func TestDeletionRuntimeLeavesLastUntouchedWhenKillFails(t *testing.T) {
	dir := t.TempDir()
	target := writeSnapshotSessions(t, dir, []string{"doomed", "keeper"})
	killErr := errors.New("kill failed")
	tmux := &deletionTmux{pid: "42", before: []string{"doomed", "keeper"}, killErr: killErr}
	executable := &snapshotRewriter{dir: dir, sessions: []string{"keeper"}}
	runtime := deletionRuntime{config: deletionConfig(t, dir, "doomed"), tmux: tmux, executable: executable}

	_, _, err := runtime.prepare()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := runtime.kill(); !errors.Is(err, killErr) {
		t.Fatalf("kill error = %v, want %v", err, killErr)
	}

	link, err := os.Readlink(filepath.Join(dir, "last"))
	if err != nil {
		t.Fatalf("read last: %v", err)
	}
	if link != filepath.Base(target) || executable.calls != 0 {
		t.Fatalf("last=%q save calls=%d, want unchanged and zero", link, executable.calls)
	}
}

func TestResurrectDeleteWithRealPlugin(t *testing.T) {
	script := os.Getenv("LAZYTMUX_RESURRECT_SAVE_SCRIPT")
	binary := os.Getenv("LAZYTMUX_TEST_BINARY")
	if script == "" || binary == "" {
		t.Skip("set LAZYTMUX_RESURRECT_SAVE_SCRIPT and LAZYTMUX_TEST_BINARY to run integration")
	}
	tmuxDir, err := os.MkdirTemp("/tmp", "lt-del-")
	if err != nil {
		t.Fatalf("create short tmux directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxDir) })
	snapshotDir := t.TempDir()
	home := t.TempDir()
	socket := "lazytmux-delete-integration"
	runner := socketRunner{socket: socket, env: append(os.Environ(), "TMUX_TMPDIR="+tmuxDir, "HOME="+home)}
	t.Cleanup(func() { _, _ = runner.RunOutput("kill-server") })
	for _, session := range []string{"keeper", "doomed"} {
		if _, err := runner.RunOutput("new-session", "-d", "-s", session); err != nil {
			t.Fatalf("create %s: %v", session, err)
		}
	}
	if _, err := runner.RunOutput("set-option", "-g", resurrectSavePathOption, script); err != nil {
		t.Fatalf("set save script: %v", err)
	}
	if _, err := runner.RunOutput("set-option", "-g", resurrectDirOption, snapshotDir); err != nil {
		t.Fatalf("set snapshot dir: %v", err)
	}
	pid, err := runner.RunOutput("display-message", "-p", "#{pid}")
	if err != nil {
		t.Fatalf("resolve PID: %v", err)
	}
	socketPath := filepath.Join(tmuxDir, fmt.Sprintf("tmux-%d", os.Getuid()), socket)
	t.Setenv("TMUX_TMPDIR", tmuxDir)
	t.Setenv("HOME", home)
	t.Setenv("TMUX", socketPath+","+strings.TrimSpace(string(pid))+",0")
	env := os.Environ()
	pluginRunner := socketRunner{socket: socket, env: env}
	logger := &fakeWarningLogger{}
	adapter := newResurrectSnapshotter(pluginRunner, osExecutableRunner{}, home, logger)
	adapter.launcher = processHelperLauncher{executable: binary}
	adapter.SaveSession("doomed")

	if err := adapter.KillSession("doomed"); err != nil {
		t.Fatalf("kill doomed: %v", err)
	}
	if err := verifyDeletedSnapshot(filepath.Join(snapshotDir, "last"), "doomed", []string{"keeper"}); err != nil {
		t.Fatalf("snapshot after deleting doomed: %v", err)
	}
	lastTarget, err := os.Readlink(filepath.Join(snapshotDir, "last"))
	if err != nil {
		t.Fatalf("read last before final delete: %v", err)
	}
	historyPath := filepath.Join(snapshotDir, lastTarget)

	if err := adapter.KillSession("keeper"); err != nil {
		t.Fatalf("kill keeper: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(snapshotDir, "last")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("last should be absent after final delete: %v", err)
	}
	if _, err := os.Stat(historyPath); err != nil {
		t.Fatalf("history should remain after final delete: %v", err)
	}
	if len(logger.entries) != 0 {
		t.Fatalf("warnings = %v, want none", logger.entries)
	}
}

func TestResurrectSelfDeleteWithRealPlugin(t *testing.T) {
	if os.Getenv("LAZYTMUX_SELF_DELETE_CHILD") == "1" {
		runner := NewRunner()
		adapter := newResurrectSnapshotter(runner, osExecutableRunner{}, os.Getenv("HOME"), nil)
		adapter.launcher = processHelperLauncher{executable: os.Getenv("LAZYTMUX_TEST_BINARY")}
		_ = adapter.KillSession("victim")
		return
	}

	script := os.Getenv("LAZYTMUX_RESURRECT_SAVE_SCRIPT")
	binary := os.Getenv("LAZYTMUX_TEST_BINARY")
	if script == "" || binary == "" {
		t.Skip("set LAZYTMUX_RESURRECT_SAVE_SCRIPT and LAZYTMUX_TEST_BINARY to run integration")
	}
	for _, final := range []bool{false, true} {
		name := "with-keeper"
		if final {
			name = "final-session"
		}
		t.Run(name, func(t *testing.T) {
			runSelfDeleteCase(t, script, binary, final)
		})
	}
}

func runSelfDeleteCase(t *testing.T, script, binary string, final bool) {
	t.Helper()
	tmuxDir, err := os.MkdirTemp("/tmp", "lt-self-")
	if err != nil {
		t.Fatalf("create short tmux directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxDir) })
	home := t.TempDir()
	snapshotDir := t.TempDir()
	socket := "lazytmux-self-delete"
	runner := socketRunner{socket: socket, env: append(os.Environ(), "TMUX_TMPDIR="+tmuxDir, "HOME="+home)}
	t.Cleanup(func() { _, _ = runner.RunOutput("kill-server") })
	if !final {
		if _, err := runner.RunOutput("new-session", "-d", "-s", "keeper"); err != nil {
			t.Fatalf("create keeper: %v", err)
		}
	}
	if _, err := runner.RunOutput("new-session", "-d", "-s", "victim", "sleep 60"); err != nil {
		t.Fatalf("create victim: %v", err)
	}
	for key, value := range map[string]string{
		resurrectSavePathOption: script,
		resurrectDirOption:      snapshotDir,
	} {
		if _, err := runner.RunOutput("set-option", "-g", key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	pid, err := runner.RunOutput("display-message", "-p", "#{pid}")
	if err != nil {
		t.Fatalf("resolve PID: %v", err)
	}
	socketPath := filepath.Join(tmuxDir, fmt.Sprintf("tmux-%d", os.Getuid()), socket)
	t.Setenv("TMUX_TMPDIR", tmuxDir)
	t.Setenv("HOME", home)
	t.Setenv("TMUX", socketPath+","+strings.TrimSpace(string(pid))+",0")
	adapter := newResurrectSnapshotter(socketRunner{socket: socket, env: os.Environ()}, osExecutableRunner{}, home, nil)
	adapter.launcher = processHelperLauncher{executable: binary}
	adapter.SaveSession("victim")
	lastTarget, err := os.Readlink(filepath.Join(snapshotDir, "last"))
	if err != nil {
		t.Fatalf("read initial last: %v", err)
	}
	historyPath := filepath.Join(snapshotDir, lastTarget)

	for key, value := range map[string]string{
		"LAZYTMUX_SELF_DELETE_CHILD": "1",
		"LAZYTMUX_TEST_BINARY":       binary,
		"HOME":                       home,
	} {
		if _, err := runner.RunOutput("set-environment", "-g", key, value); err != nil {
			t.Fatalf("set child environment %s: %v", key, err)
		}
	}
	childCommand := shellQuote(os.Args[0]) + " -test.run '^TestResurrectSelfDeleteWithRealPlugin$'"
	if _, err := runner.RunOutput("respawn-pane", "-k", "-t", "victim:0.0", childCommand); err != nil {
		t.Fatalf("start self-delete child: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if final {
			if _, err := os.Lstat(filepath.Join(snapshotDir, "last")); errors.Is(err, os.ErrNotExist) {
				if _, err := os.Stat(historyPath); err != nil {
					t.Fatalf("history should remain: %v", err)
				}
				return
			}
		} else if err := verifyDeletedSnapshot(filepath.Join(snapshotDir, "last"), "victim", []string{"keeper"}); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("self-delete reconciliation did not complete before deadline")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func TestWaitPastLastSnapshotSecond(t *testing.T) {
	dir := t.TempDir()
	stamp := time.Date(2026, 8, 5, 7, 30, 15, 250_000_000, time.Local)
	target := "tmux_resurrect_20260805T073015.txt"
	if err := os.WriteFile(filepath.Join(dir, target), nil, 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "last")); err != nil {
		t.Fatalf("link last: %v", err)
	}
	var slept time.Duration

	waitPastLastSnapshotSecond(filepath.Join(dir, "last"), func() time.Time { return stamp }, func(d time.Duration) { slept = d })

	if slept != 750*time.Millisecond {
		t.Fatalf("sleep = %v, want 750ms", slept)
	}
}

func deletionConfig(t *testing.T, dir, session string) killHelperConfig {
	t.Helper()
	script := executableFile(t, t.TempDir())
	return killHelperConfig{
		Protocol: killProtocol, Session: session, TmuxPath: "/usr/bin/tmux",
		SocketPath: "/tmp/tmux-test", ServerPID: "42", SaveScript: script,
		SnapshotDir: dir, Home: t.TempDir(), Environment: os.Environ(),
	}
}

func writeSnapshotSessions(t *testing.T, dir string, sessions []string) string {
	t.Helper()
	return writeSnapshotSessionsRaw(dir, sessions)
}

func writeSnapshotSessionsRaw(dir string, sessions []string) string {
	path := filepath.Join(dir, "snapshot.txt")
	var content strings.Builder
	for _, session := range sessions {
		content.WriteString("pane\t" + session + "\t1\t0\n")
		content.WriteString("window\t" + session + "\t1\n")
	}
	_ = os.WriteFile(path, []byte(content.String()), 0o600)
	last := filepath.Join(dir, "last")
	_ = os.Remove(last)
	_ = os.Symlink(filepath.Base(path), last)
	return path
}
