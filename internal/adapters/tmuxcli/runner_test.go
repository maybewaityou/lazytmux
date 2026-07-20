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

import "github.com/maybewaityou/lazytmux/internal/core/domain"

// FakeRunner is a test double for CommandRunner. Tests in later tasks embed it.
type FakeRunner struct {
	Output      []byte
	Err         error
	Interactive func(args []string) error
	LastArgs    []string
	AllCalls    [][]string
	LookPathErr error
}

func (f *FakeRunner) RunOutput(args ...string) ([]byte, error) {
	f.LastArgs = args
	f.AllCalls = append(f.AllCalls, args)
	return f.Output, f.Err
}

func (f *FakeRunner) RunInteractive(args ...string) error {
	f.LastArgs = args
	f.AllCalls = append(f.AllCalls, args)
	if f.Interactive != nil {
		return f.Interactive(args)
	}
	return f.Err
}

func (f *FakeRunner) LookPath() error { return f.LookPathErr }

// keep domain referenced (used by repo tests in task 4).
var _ = domain.Session{}
