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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/maybewaityou/lazytmux/internal/adapters/data/metadata"
	"github.com/maybewaityou/lazytmux/internal/adapters/tmuxcli"
	"github.com/maybewaityou/lazytmux/internal/adapters/ui"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
	"github.com/maybewaityou/lazytmux/internal/core/services"
	"github.com/maybewaityou/lazytmux/internal/logger"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func main() {
	log, err := logger.New("LAZYTMUX")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	//nolint:errcheck // log.Sync may return an error which is safe to ignore here
	defer log.Sync()

	runner := tmuxcli.NewRunner()
	if err := runner.LookPath(); err != nil {
		fmt.Fprintln(os.Stderr, "lazytmux requires `tmux` on your PATH.")
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorw("failed to get user home directory", "error", err)
		//nolint:gocritic // exitAfterDefer: ensure immediate exit on unrecoverable error
		os.Exit(1)
	}
	metaPath := filepath.Join(home, ".lazytmux", "metadata.json")

	repo := tmuxcli.NewRepository(runner)
	metaStore, err := metadata.NewStore(metaPath)
	if err != nil {
		log.Errorw("metadata store", "error", err)
		os.Exit(1)
	}

	svc := services.NewSessionService(repo, metaStore, nil)
	t := ui.NewTUI(log, svc, version, gitCommit)

	// Break the tui<->service cycle: hand the TUI's suspend function to the
	// service (used for out-of-tmux interactive attach).
	if suspender, ok := t.(interface{ Suspend(func() error) error }); ok {
		if setter, ok := svc.(interface{ SetSuspend(ports.SuspendFunc) }); ok {
			setter.SetSuspend(suspender.Suspend)
		}
	}

	root := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy tmux session picker TUI",
		RunE: func(*cobra.Command, []string) error {
			return t.Run()
		},
	}
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
