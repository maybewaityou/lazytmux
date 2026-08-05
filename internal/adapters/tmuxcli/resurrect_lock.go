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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"golang.org/x/sys/unix"
)

var resurrectTimestampPattern = regexp.MustCompile(`^tmux_resurrect_(\d{8}T\d{6})\.txt$`)

type resurrectLock struct {
	file *os.File
}

func acquireResurrectLock(home, snapshotDir string) (*resurrectLock, error) {
	canonical, err := filepath.Abs(snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("resolve resurrect directory: %w", err)
	}
	hash := sha256.Sum256([]byte(filepath.Clean(canonical)))
	lockDir := filepath.Join(home, ".lazytmux", "resurrect-locks")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return nil, fmt.Errorf("create resurrect lock directory: %w", err)
	}
	path := filepath.Join(lockDir, hex.EncodeToString(hash[:])+".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open resurrect lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock resurrect directory: %w", err)
	}
	return &resurrectLock{file: file}, nil
}

func (l *resurrectLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlock resurrect directory: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close resurrect lock: %w", closeErr)
	}
	return nil
}

func waitPastLastSnapshotSecond(lastPath string, now func() time.Time, sleep func(time.Duration)) {
	target, err := os.Readlink(lastPath)
	if err != nil {
		return
	}
	match := resurrectTimestampPattern.FindStringSubmatch(filepath.Base(target))
	if len(match) != 2 {
		return
	}
	stamp, err := time.ParseInLocation("20060102T150405", match[1], now().Location())
	if err != nil {
		return
	}
	current := now()
	if current.Truncate(time.Second).Equal(stamp) {
		sleep(current.Truncate(time.Second).Add(time.Second).Sub(current))
	}
}
