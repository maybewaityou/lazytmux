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
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverResurrectCapability(t *testing.T) {
	t.Parallel()

	executable := executableFile(t, t.TempDir())
	nonExecutable := filepath.Join(t.TempDir(), "save.sh")
	if err := os.WriteFile(nonExecutable, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write non-executable script: %v", err)
	}
	directory := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing.sh")

	tests := []struct {
		name       string
		response   runnerResponse
		wantStatus resurrectCapabilityStatus
		wantStage  string
		wantScript string
	}{
		{name: "option unavailable", response: runnerResponse{output: []byte("\n")}, wantStatus: resurrectUnavailable},
		{name: "linux no server", response: runnerResponse{err: errors.New("no server running on /tmp/tmux/default")}, wantStatus: resurrectUnavailable},
		{name: "macOS no server", response: runnerResponse{err: errors.New("error connecting to /tmp/tmux/default")}, wantStatus: resurrectUnavailable},
		{name: "query failure", response: runnerResponse{err: errors.New("permission denied")}, wantStatus: resurrectBroken, wantStage: "discover save script"},
		{name: "relative script", response: runnerResponse{output: []byte("save.sh")}, wantStatus: resurrectBroken, wantStage: "validate save script"},
		{name: "missing script", response: runnerResponse{output: []byte(missing)}, wantStatus: resurrectBroken, wantStage: "validate save script"},
		{name: "directory instead of script", response: runnerResponse{output: []byte(directory)}, wantStatus: resurrectBroken, wantStage: "validate save script"},
		{name: "non-executable script", response: runnerResponse{output: []byte(nonExecutable)}, wantStatus: resurrectBroken, wantStage: "validate save script"},
		{name: "ready", response: runnerResponse{output: []byte("  " + executable + "\n")}, wantStatus: resurrectReady, wantScript: executable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			runner := &queuedRunner{responses: []runnerResponse{test.response}}
			capability := discoverResurrectCapability(runner)

			if capability.status != test.wantStatus {
				t.Fatalf("status = %v, want %v", capability.status, test.wantStatus)
			}
			if capability.stage != test.wantStage {
				t.Fatalf("stage = %q, want %q", capability.stage, test.wantStage)
			}
			if capability.saveScript != test.wantScript {
				t.Fatalf("save script = %q, want %q", capability.saveScript, test.wantScript)
			}
			if capability.status == resurrectBroken && capability.err == nil {
				t.Fatal("broken capability error = nil")
			}
			if capability.status != resurrectBroken && capability.err != nil {
				t.Fatalf("capability error = %v, want nil", capability.err)
			}
		})
	}
}
