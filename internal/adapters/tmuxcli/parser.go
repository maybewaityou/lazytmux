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
	"strconv"
	"strings"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// fieldSep is the separator used in tmux -F format strings.
const fieldSep = "|"

// ParseSessions parses `tmux list-sessions -F` output into Sessions.
// Lines that do not contain exactly 7 fields are skipped.
func ParseSessions(out []byte) ([]domain.Session, error) {
	sessions := make([]domain.Session, 0)
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, fieldSep)
		if len(fields) != 7 {
			continue
		}
		attached, _ := strconv.Atoi(fields[2])
		created, _ := strconv.ParseInt(fields[3], 10, 64)
		activity, _ := strconv.ParseInt(fields[4], 10, 64)
		windows, _ := strconv.Atoi(fields[5])

		sessions = append(sessions, domain.Session{
			ID:            fields[0],
			Name:          fields[1],
			Attached:      attached > 0,
			AttachedCount: attached,
			Created:       time.Unix(created, 0),
			LastActivity:  time.Unix(activity, 0),
			WindowsCount:  windows,
			Path:          fields[6],
		})
	}
	return sessions, nil
}

// ParseWindows parses `tmux list-windows -t <name> -F` output into Windows.
func ParseWindows(out []byte) ([]domain.Window, error) {
	windows := make([]domain.Window, 0)
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, fieldSep)
		if len(fields) != 4 {
			continue
		}
		idx, _ := strconv.Atoi(fields[0])
		active, _ := strconv.Atoi(fields[2])
		windows = append(windows, domain.Window{
			Index:   idx,
			Name:    fields[1],
			Active:  active > 0,
			Command: fields[3],
		})
	}
	return windows, nil
}

// ensure bytes is referenced (used by runner.go in this same package).
var _ = bytes.TrimSpace
