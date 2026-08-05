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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

const (
	killHelperCommand = "__resurrect-kill-helper"
	killProtocol      = 1
)

type killHelperConfig struct {
	Protocol    int      `json:"protocol"`
	Session     string   `json:"session"`
	TmuxPath    string   `json:"tmux_path"`
	SocketPath  string   `json:"socket_path"`
	ServerPID   string   `json:"server_pid"`
	SaveScript  string   `json:"save_script"`
	SnapshotDir string   `json:"snapshot_dir"`
	Home        string   `json:"home"`
	Environment []string `json:"environment"`
}

type killHelperFrame struct {
	Type  string `json:"type"`
	Error string `json:"error,omitempty"`
}

type helperLauncher interface {
	Run(config killHelperConfig) (attempted bool, err error)
}

type processHelperLauncher struct {
	executable string
}

type explicitTmuxRunner interface {
	Run(socketPath string, args ...string) ([]byte, error)
}

type execExplicitTmuxRunner struct {
	path string
	env  []string
}

func (r execExplicitTmuxRunner) Run(socketPath string, args ...string) ([]byte, error) {
	allArgs := append([]string{"-S", socketPath}, args...)
	cmd := exec.Command(r.path, allArgs...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tmux %v: %s: %w", allArgs, strings.TrimSpace(string(out)), err)
	}
	return out, nil
}

type envExecutableRunner interface {
	RunEnv(path string, env []string, args ...string) error
}

type osEnvExecutableRunner struct{}

func (osEnvExecutableRunner) RunEnv(path string, env []string, args ...string) error {
	cmd := exec.Command(path, args...) //nolint:gosec // validated tmux-resurrect runtime path; no shell
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %s: %w", path, args, strings.TrimSpace(string(out)), err)
	}
	return nil
}

type preparedLast struct {
	path       string
	target     string
	info       os.FileInfo
	eligible   bool
	targetPath string
}

type deletionRuntime struct {
	config     killHelperConfig
	tmux       explicitTmuxRunner
	executable envExecutableRunner
	logger     WarningLogger
}

// NewResurrectPersistence returns the shared create/delete persistence adapter.
func NewResurrectPersistence(runner CommandRunner, home string, logger WarningLogger) interface {
	ports.SessionSnapshotter
	ports.SessionTerminator
} {
	adapter := newResurrectSnapshotter(runner, osExecutableRunner{}, home, logger)
	adapter.launcher = processHelperLauncher{}
	return adapter
}

func (s *resurrectSnapshotter) KillSession(name string) error {
	config, available := s.discoverKillConfig(name)
	if !available {
		_, err := s.tmux.RunOutput("kill-session", "-t", name)
		return err
	}
	attempted, err := s.launcher.Run(config)
	if attempted {
		return err
	}
	if err != nil {
		s.warnDeletion(name, "start-helper", err)
	}
	_, directErr := s.tmux.RunOutput("kill-session", "-t", name)
	return directErr
}

func (s *resurrectSnapshotter) discoverKillConfig(name string) (killHelperConfig, bool) {
	script, available := s.saveScript(name)
	if !available {
		return killHelperConfig{}, false
	}
	dir, err := s.snapshotDir()
	if err != nil {
		s.warnDeletion(name, "resolve-directory", err)
		return killHelperConfig{}, false
	}
	if !filepath.IsAbs(dir) {
		s.warnDeletion(name, "resolve-directory", fmt.Errorf("resurrect directory is not absolute: %s", dir))
		return killHelperConfig{}, false
	}
	identity, err := s.tmux.RunOutput("display-message", "-p", "#{socket_path}\t#{pid}")
	if err != nil {
		s.warnDeletion(name, "discover-server", err)
		return killHelperConfig{}, false
	}
	fields := strings.Split(strings.TrimSpace(string(identity)), "\t")
	if len(fields) != 2 || !filepath.IsAbs(fields[0]) || fields[1] == "" {
		s.warnDeletion(name, "discover-server", fmt.Errorf("invalid tmux server identity: %q", identity))
		return killHelperConfig{}, false
	}
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		s.warnDeletion(name, "discover-server", fmt.Errorf("locate tmux: %w", err))
		return killHelperConfig{}, false
	}
	return killHelperConfig{
		Protocol:    killProtocol,
		Session:     name,
		TmuxPath:    tmuxPath,
		SocketPath:  fields[0],
		ServerPID:   fields[1],
		SaveScript:  script,
		SnapshotDir: dir,
		Home:        s.home,
		Environment: os.Environ(),
	}, true
}

func (l processHelperLauncher) Run(config killHelperConfig) (bool, error) {
	executable := l.executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return false, fmt.Errorf("resolve lazytmux executable: %w", err)
		}
	}
	cmd := exec.Command(executable, killHelperCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	control, err := cmd.StdinPipe()
	if err != nil {
		return false, fmt.Errorf("create helper control pipe: %w", err)
	}
	resultReader, resultWriter, err := os.Pipe()
	if err != nil {
		return false, fmt.Errorf("create helper result pipe: %w", err)
	}
	defer func() { _ = resultReader.Close() }()
	cmd.ExtraFiles = []*os.File{resultWriter}
	if err := cmd.Start(); err != nil {
		_ = resultWriter.Close()
		return false, fmt.Errorf("start helper: %w", err)
	}
	_ = resultWriter.Close()
	encoder := json.NewEncoder(control)
	decoder := json.NewDecoder(resultReader)
	if err := encoder.Encode(config); err != nil {
		_ = control.Close()
		_ = cmd.Wait()
		return false, fmt.Errorf("send helper config: %w", err)
	}
	var frame killHelperFrame
	if err := decoder.Decode(&frame); err != nil || frame.Type != "ready" {
		_ = control.Close()
		_ = cmd.Wait()
		return false, fmt.Errorf("helper ready: frame=%+v error=%w", frame, err)
	}
	if err := encoder.Encode(killHelperFrame{Type: "go"}); err != nil {
		_ = control.Close()
		_ = cmd.Wait()
		return true, fmt.Errorf("start helper kill: %w", err)
	}
	if err := decoder.Decode(&frame); err != nil {
		_ = control.Close()
		return true, fmt.Errorf("read helper kill result: %w", err)
	}
	attempted := frame.Type == "kill-result"
	if frame.Error != "" {
		_ = control.Close()
		_ = cmd.Wait()
		return attempted, errors.New(frame.Error)
	}
	if err := decoder.Decode(&frame); err != nil {
		_ = control.Close()
		_ = cmd.Wait()
		return attempted, nil
	}
	_ = control.Close()
	_ = cmd.Wait()
	return attempted, nil
}

// RunResurrectKillHelper runs the hidden detached helper protocol.
func RunResurrectKillHelper(control io.Reader, result io.Writer, logger WarningLogger) error {
	decoder := json.NewDecoder(control)
	encoder := json.NewEncoder(result)
	var config killHelperConfig
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("decode helper config: %w", err)
	}
	runtime := deletionRuntime{
		config:     config,
		tmux:       execExplicitTmuxRunner{path: config.TmuxPath, env: config.Environment},
		executable: osEnvExecutableRunner{},
		logger:     logger,
	}
	if err := validateKillConfig(config); err != nil {
		return err
	}
	lock, err := acquireResurrectLock(config.Home, config.SnapshotDir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	last, beforeSessions, err := runtime.prepare()
	if err != nil {
		return err
	}
	if err := encoder.Encode(killHelperFrame{Type: "ready"}); err != nil {
		return fmt.Errorf("send helper ready: %w", err)
	}
	var goFrame killHelperFrame
	if err := decoder.Decode(&goFrame); err != nil {
		return fmt.Errorf("read helper go: %w", err)
	}
	if goFrame.Type != "go" {
		return fmt.Errorf("unexpected helper command: %q", goFrame.Type)
	}
	killErr := runtime.kill()
	resultFrame := killHelperFrame{Type: "kill-result"}
	if killErr != nil {
		resultFrame.Error = killErr.Error()
	}
	_ = encoder.Encode(resultFrame)
	if killErr == nil {
		runtime.reconcile(last, beforeSessions)
	}
	_ = encoder.Encode(killHelperFrame{Type: "done"})
	return nil
}

func (r deletionRuntime) prepare() (preparedLast, []string, error) {
	pid, err := r.tmux.Run(r.config.SocketPath, "display-message", "-p", "#{pid}")
	if err != nil || strings.TrimSpace(string(pid)) != r.config.ServerPID {
		return preparedLast{}, nil, fmt.Errorf("tmux server identity changed")
	}
	sessions, err := r.liveSessions()
	if err != nil {
		return preparedLast{}, nil, err
	}
	if !slices.Contains(sessions, r.config.Session) {
		return preparedLast{}, nil, fmt.Errorf("session %q is not live", r.config.Session)
	}
	last, err := captureLast(filepath.Join(r.config.SnapshotDir, "last"), sessions)
	if err != nil {
		r.warn("inspect-last", err)
	}
	return last, sessions, nil
}

func validateKillConfig(config killHelperConfig) error {
	if config.Protocol != killProtocol || config.Session == "" {
		return fmt.Errorf("invalid kill helper protocol or session")
	}
	for label, path := range map[string]string{
		"tmux": config.TmuxPath, "socket": config.SocketPath,
		"save script": config.SaveScript, "snapshot directory": config.SnapshotDir,
	} {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("%s path is not absolute: %s", label, path)
		}
	}
	if err := validateSaveScript(config.SaveScript); err != nil {
		return err
	}
	return nil
}

func (r deletionRuntime) kill() error {
	_, err := r.tmux.Run(r.config.SocketPath, "kill-session", "-t", r.config.Session)
	return err
}

func (r deletionRuntime) reconcile(last preparedLast, beforeSessions []string) {
	pid, err := r.tmux.Run(r.config.SocketPath, "display-message", "-p", "#{pid}")
	if err != nil {
		if isNoServerError(err) {
			if err := invalidateLast(last, beforeSessions); err != nil {
				r.warn("unlink-last", err)
			}
			return
		}
		r.warn("inspect-server", err)
		return
	}
	if strings.TrimSpace(string(pid)) != r.config.ServerPID {
		r.warn("inspect-server", fmt.Errorf("tmux server identity changed"))
		return
	}
	live, err := r.liveSessions()
	if err != nil {
		r.warn("inspect-server", err)
		return
	}
	if slices.Contains(live, r.config.Session) {
		r.warn("verify-snapshot", fmt.Errorf("deleted session %q is live again", r.config.Session))
		return
	}
	env := replaceEnvironment(r.config.Environment, "TMUX", r.config.SocketPath+","+r.config.ServerPID+",0")
	waitPastLastSnapshotSecond(filepath.Join(r.config.SnapshotDir, "last"), time.Now, time.Sleep)
	if err := r.executable.RunEnv(r.config.SaveScript, env, "quiet"); err != nil {
		r.warn("run-save-script", err)
		return
	}
	if err := verifyDeletedSnapshot(filepath.Join(r.config.SnapshotDir, "last"), r.config.Session, live); err != nil {
		r.warn("verify-snapshot", err)
	}
	_ = beforeSessions
}

func (r deletionRuntime) liveSessions() ([]string, error) {
	out, err := r.tmux.Run(r.config.SocketPath, "list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, err
	}
	var sessions []string
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name != "" {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

func (r deletionRuntime) warn(stage string, err error) {
	if r.logger != nil {
		r.logger.Warnw("tmux-resurrect deletion reconciliation failed", "session", r.config.Session, "stage", stage, "error", err)
	}
}

func captureLast(path string, liveSessions []string) (preparedLast, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return preparedLast{path: path}, nil
	}
	if err != nil {
		return preparedLast{path: path}, fmt.Errorf("lstat last: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return preparedLast{path: path}, fmt.Errorf("last is not a symlink")
	}
	target, err := os.Readlink(path)
	if err != nil {
		return preparedLast{path: path}, fmt.Errorf("read last link: %w", err)
	}
	targetPath := target
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(filepath.Dir(path), targetPath)
	}
	summary, err := parseSnapshot(targetPath)
	if err != nil {
		return preparedLast{path: path}, err
	}
	eligible := sameSessionSet(summary.sessions(), liveSessions)
	return preparedLast{path: path, target: target, info: info, eligible: eligible, targetPath: targetPath}, nil
}

func invalidateLast(last preparedLast, expectedSessions []string) error {
	if last.path == "" || last.info == nil || !last.eligible {
		return nil
	}
	current, err := os.Lstat(last.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("lstat current last: %w", err)
	}
	target, err := os.Readlink(last.path)
	if err != nil {
		return fmt.Errorf("read current last link: %w", err)
	}
	if target != last.target || !os.SameFile(last.info, current) {
		return fmt.Errorf("last changed during session deletion")
	}
	summary, err := parseSnapshot(last.targetPath)
	if err != nil {
		return fmt.Errorf("recheck last target: %w", err)
	}
	if !sameSessionSet(summary.sessions(), expectedSessions) {
		return fmt.Errorf("last target changed during session deletion")
	}
	if err := os.Remove(last.path); err != nil {
		return fmt.Errorf("unlink last: %w", err)
	}
	return nil
}

func replaceEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func sameSessionSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for _, value := range left {
		if !slices.Contains(right, value) {
			return false
		}
	}
	return true
}

func (s *resurrectSnapshotter) warnDeletion(name, stage string, err error) {
	if s.logger != nil {
		s.logger.Warnw("tmux-resurrect deletion reconciliation failed", "session", name, "stage", stage, "error", err)
	}
}

var _ ports.SessionTerminator = (*resurrectSnapshotter)(nil)
