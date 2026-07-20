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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandRunner abstracts shelling out to tmux so parsing/behaviour can be
// tested without a real tmux server.
type CommandRunner interface {
	// RunOutput runs tmux with args and returns stdout. On failure the error
	// carries tmux's stderr text so callers can tell failure modes apart —
	// notably the "no server running" empty-state from a genuine fault.
	RunOutput(args ...string) ([]byte, error)
	// RunInteractive runs tmux attached to the parent's stdio (for attach).
	RunInteractive(args ...string) error
	// LookPath reports whether tmux is on PATH.
	LookPath() error
}

// ExecRunner is the production CommandRunner backed by os/exec.
type ExecRunner struct{}

// NewRunner returns the production runner.
func NewRunner() CommandRunner { return ExecRunner{} }

func (ExecRunner) RunOutput(args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("tmux", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Surface tmux's stderr. Without it every failure collapses to a useless
		// "exit status 1", and the "no server running" empty-state becomes
		// indistinguishable from a real error.
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("tmux %v: %s", args, msg)
		}
		return nil, fmt.Errorf("tmux %v: %w", args, err)
	}
	return stdout.Bytes(), nil
}

func (ExecRunner) RunInteractive(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecRunner) LookPath() error {
	_, err := exec.LookPath("tmux")
	return err
}
