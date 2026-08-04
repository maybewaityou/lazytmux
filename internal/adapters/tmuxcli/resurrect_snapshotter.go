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
	"strings"

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
) ports.SessionSnapshotter {
	return &resurrectSnapshotter{
		tmux:       runner,
		executable: executable,
		home:       home,
		logger:     logger,
	}
}

func (s *resurrectSnapshotter) SaveSession(name string) {
	script, available := s.saveScript(name)
	if !available {
		return
	}
	if err := s.executable.Run(script, "quiet"); err != nil {
		s.warn(name, "run save script", err)
		return
	}
	dir, err := s.snapshotDir()
	if err != nil {
		s.warn(name, "resolve snapshot directory", err)
		return
	}
	if err := verifySnapshot(filepath.Join(dir, "last"), name); err != nil {
		s.warn(name, "verify snapshot", err)
	}
}

func (s *resurrectSnapshotter) saveScript(name string) (string, bool) {
	out, err := s.tmux.RunOutput("show-options", "-gqv", resurrectSavePathOption)
	if err != nil {
		if !isNoServerError(err) {
			s.warn(name, "discover save script", err)
		}
		return "", false
	}
	script := strings.TrimSpace(string(out))
	if script == "" {
		return "", false
	}
	if err := validateSaveScript(script); err != nil {
		s.warn(name, "validate save script", err)
		return "", false
	}
	return script, true
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
	dir = strings.ReplaceAll(dir, "$HOME", s.home)
	dir = strings.ReplaceAll(dir, "$HOSTNAME", hostname)
	dir = strings.ReplaceAll(dir, "~", s.home)
	return filepath.Clean(dir), nil
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

func verifySnapshot(path, sessionName string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open resurrect last snapshot: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) >= 2 && (fields[0] == "pane" || fields[0] == "window") && fields[1] == sessionName {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read resurrect last snapshot: %w", err)
	}
	return fmt.Errorf("session %q is absent from resurrect last snapshot", sessionName)
}

func (s *resurrectSnapshotter) warn(name, stage string, err error) {
	if s.logger != nil {
		s.logger.Warnw("tmux-resurrect snapshot failed", "session", name, "stage", stage, "error", err)
	}
}

var _ ports.SessionSnapshotter = (*resurrectSnapshotter)(nil)
