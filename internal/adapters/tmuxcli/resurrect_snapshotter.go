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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

const (
	resurrectSavePathOption = "@resurrect-save-script-path"
	resurrectDirOption      = "@resurrect-dir"
)

// WarningLogger is the minimal logging boundary used by optional tmux plugin
// integrations. A zap SugaredLogger satisfies it without entering core ports.
type WarningLogger interface {
	Warnw(message string, keysAndValues ...any)
}

type executableRunner interface {
	Run(path string, args ...string) error
}

type osExecutableRunner struct{}

func (osExecutableRunner) Run(path string, args ...string) error {
	var stderr bytes.Buffer
	// The path comes from tmux-resurrect's runtime option and is validated as an
	// executable regular file before this boundary; no shell is involved.
	cmd := exec.Command(path, args...) //nolint:gosec // plugin-owned executable path, validated by validateSaveScript
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s %v: %s", path, args, msg)
		}
		return fmt.Errorf("%s %v: %w", path, args, err)
	}
	return nil
}

type resurrectSnapshotter struct {
	tmux       CommandRunner
	executable executableRunner
	home       string
	logger     WarningLogger
	launcher   helperLauncher
}

// NewResurrectSnapshotter returns an optional tmux-resurrect persistence
// adapter. When the plugin is not loaded it is a no-op; detected failures are
// logged without changing the result of the primary session creation.
func NewResurrectSnapshotter(runner CommandRunner, home string, logger WarningLogger) ports.SessionSnapshotter {
	return newResurrectSnapshotter(runner, osExecutableRunner{}, home, logger)
}

func newResurrectSnapshotter(
	runner CommandRunner,
	executable executableRunner,
	home string,
	logger WarningLogger,
) *resurrectSnapshotter {
	return &resurrectSnapshotter{
		tmux:       runner,
		executable: executable,
		home:       home,
		logger:     logger,
	}
}

func (s *resurrectSnapshotter) SaveSession(name string) {
	capability := discoverResurrectCapability(s.tmux)
	switch capability.status {
	case resurrectUnavailable:
		return
	case resurrectBroken:
		s.warn(name, capability.stage, capability.err)
		return
	case resurrectReady:
	}

	dir, err := s.snapshotDir()
	if err != nil {
		s.warn(name, "resolve snapshot directory", err)
		return
	}
	lock, err := acquireResurrectLock(s.home, dir)
	if err != nil {
		s.warn(name, "lock snapshot directory", err)
		return
	}
	defer func() {
		if err := lock.Close(); err != nil {
			s.warn(name, "unlock snapshot directory", err)
		}
	}()
	waitPastLastSnapshotSecond(filepath.Join(dir, "last"), time.Now, time.Sleep)
	if err := s.executable.Run(capability.saveScript, "quiet"); err != nil {
		s.warn(name, "run save script", err)
		return
	}
	if err := verifySnapshot(filepath.Join(dir, "last"), name); err != nil {
		s.warn(name, "verify snapshot", err)
	}
}

func validateSaveScript(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("resurrect save script path is not absolute: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat resurrect save script: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("resurrect save script is not a regular file: %s", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("resurrect save script is not executable: %s", path)
	}
	return nil
}

func (s *resurrectSnapshotter) snapshotDir() (string, error) {
	out, err := s.tmux.RunOutput("show-options", "-gqv", resurrectDirOption)
	if err != nil {
		return "", fmt.Errorf("query %s: %w", resurrectDirOption, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		dir = defaultResurrectDir(s.home)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("resolve hostname: %w", err)
	}
	dir = expandResurrectDir(dir, s.home, hostname)
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("resurrect directory is not absolute: %s", dir)
	}
	if strings.Contains(dir, "$") {
		return "", fmt.Errorf("resurrect directory contains an unsupported variable: %s", dir)
	}
	return filepath.Clean(dir), nil
}

func expandResurrectDir(dir, home, hostname string) string {
	dir = strings.ReplaceAll(dir, "${HOME}", home)
	dir = strings.ReplaceAll(dir, "$HOME", home)
	dir = strings.ReplaceAll(dir, "${HOSTNAME}", hostname)
	dir = strings.ReplaceAll(dir, "$HOSTNAME", hostname)
	if dir == "~" {
		return home
	}
	if strings.HasPrefix(dir, "~/") {
		return filepath.Join(home, strings.TrimPrefix(dir, "~/"))
	}
	return dir
}

func defaultResurrectDir(home string) string {
	legacy := filepath.Join(home, ".tmux", "resurrect")
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return legacy
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "tmux", "resurrect")
}

type snapshotSummary struct {
	sessionsWithPanes   map[string]struct{}
	sessionsWithWindows map[string]struct{}
}

func parseSnapshot(path string) (snapshotSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return snapshotSummary{}, fmt.Errorf("open resurrect snapshot: %w", err)
	}
	defer func() { _ = file.Close() }()

	summary := snapshotSummary{
		sessionsWithPanes:   make(map[string]struct{}),
		sessionsWithWindows: make(map[string]struct{}),
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 2 || fields[1] == "" {
			continue
		}
		switch fields[0] {
		case "pane":
			summary.sessionsWithPanes[fields[1]] = struct{}{}
		case "window":
			summary.sessionsWithWindows[fields[1]] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return snapshotSummary{}, fmt.Errorf("read resurrect snapshot: %w", err)
	}
	return summary, nil
}

func (s snapshotSummary) sessions() []string {
	result := make([]string, 0, len(s.sessionsWithPanes)+len(s.sessionsWithWindows))
	for name := range s.sessionsWithPanes {
		result = append(result, name)
	}
	for name := range s.sessionsWithWindows {
		if !slices.Contains(result, name) {
			result = append(result, name)
		}
	}
	return result
}

func verifySnapshot(path, sessionName string) error {
	summary, err := parseSnapshot(path)
	if err != nil {
		return err
	}
	if slices.Contains(summary.sessions(), sessionName) {
		return nil
	}
	return fmt.Errorf("session %q is absent from resurrect last snapshot", sessionName)
}

func verifyDeletedSnapshot(path, deletedName string, liveSessions []string) error {
	summary, err := parseSnapshot(path)
	if err != nil {
		return err
	}
	sessions := summary.sessions()
	if slices.Contains(sessions, deletedName) {
		return fmt.Errorf("deleted session %q remains in resurrect last snapshot", deletedName)
	}
	if len(sessions) == 0 || !sameSessionSet(sessions, liveSessions) {
		return fmt.Errorf("resurrect last snapshot sessions %v do not match live sessions %v", sessions, liveSessions)
	}
	return nil
}

func (s *resurrectSnapshotter) warn(name, stage string, err error) {
	if s.logger != nil {
		s.logger.Warnw("tmux-resurrect snapshot failed", "session", name, "stage", stage, "error", err)
	}
}

var _ ports.SessionSnapshotter = (*resurrectSnapshotter)(nil)
