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
	"go.uber.org/zap"

	"github.com/maybewaityou/lazytmux/internal/adapters/data/metadata"
	"github.com/maybewaityou/lazytmux/internal/adapters/tmuxcli"
	"github.com/maybewaityou/lazytmux/internal/adapters/ui"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
	"github.com/maybewaityou/lazytmux/internal/core/services"
	"github.com/maybewaityou/lazytmux/internal/logger"
	"github.com/maybewaityou/lazytmux/internal/tz"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:          ui.AppName,
		Short:        "Lazy tmux session picker TUI",
		SilenceUsage: true,
	}
	root.RunE = func(*cobra.Command, []string) error { return runTUI() }
	root.AddCommand(newKillHelperCommand())
	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTUI() error {
	log, err := logger.New("LAZYTMUX")
	if err != nil {
		return err
	}
	//nolint:errcheck // log.Sync may return an error which is safe to ignore here
	defer log.Sync()

	// Resolve the real local timezone before anything formats a time. Without
	// this, Termux/Android silently falls back to UTC (its zoneinfo lives under a
	// non-standard $PREFIX path Go's LoadLocation never searches), so every
	// .Local() timestamp renders 8h off on a CST device. tz.Init embeds the tzdb
	// and rebinds time.Local from $TZ / Android getprop.
	if name := tz.Init(); name != "" {
		log.Infow("timezone resolved", "zone", name)
	}

	runner := tmuxcli.NewRunner()
	if err := runner.LookPath(); err != nil {
		return fmt.Errorf("lazytmux requires `tmux` on your PATH")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}
	metaStore, err := metadata.NewStore(filepath.Join(home, ".lazytmux", "metadata.json"))
	if err != nil {
		return fmt.Errorf("metadata store: %w", err)
	}

	repo := tmuxcli.NewRepository(runner)
	persistence := tmuxcli.NewResurrectPersistence(runner, home, log)
	svc := services.NewSessionService(repo, metaStore, persistence, persistence, nil)
	t := ui.NewTUI(log, svc, version, gitCommit)

	// Break the tui<->service cycle: hand the TUI's suspend function to the
	// service (used for out-of-tmux interactive attach).
	if suspender, ok := t.(interface{ Suspend(func() error) error }); ok {
		if setter, ok := svc.(interface{ SetSuspend(ports.SuspendFunc) }); ok {
			setter.SetSuspend(suspender.Suspend)
		}
	}
	return t.Run()
}

func newKillHelperCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "__resurrect-kill-helper",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKillHelper()
		},
	}
}

func runKillHelper() error {
	result := os.NewFile(3, "lazytmux-kill-helper-result")
	if result == nil {
		return fmt.Errorf("kill helper result fd is unavailable")
	}
	defer func() { _ = result.Close() }()
	log, err := logger.New("LAZYTMUX_HELPER")
	if err != nil {
		log = zap.NewNop().Sugar()
	}
	//nolint:errcheck // helper logging is best-effort after its primary kill
	defer log.Sync()
	return tmuxcli.RunResurrectKillHelper(os.Stdin, result, log)
}
