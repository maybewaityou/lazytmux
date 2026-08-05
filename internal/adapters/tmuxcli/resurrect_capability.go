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

import "strings"

type resurrectCapabilityStatus uint8

const (
	resurrectUnavailable resurrectCapabilityStatus = iota
	resurrectBroken
	resurrectReady
)

type resurrectCapability struct {
	status     resurrectCapabilityStatus
	saveScript string
	stage      string
	err        error
}

func discoverResurrectCapability(runner CommandRunner) resurrectCapability {
	out, err := runner.RunOutput("show-options", "-gqv", resurrectSavePathOption)
	if err != nil {
		if isNoServerError(err) {
			return resurrectCapability{status: resurrectUnavailable}
		}
		return resurrectCapability{status: resurrectBroken, stage: "discover save script", err: err}
	}

	script := strings.TrimSpace(string(out))
	if script == "" {
		return resurrectCapability{status: resurrectUnavailable}
	}
	if err := validateSaveScript(script); err != nil {
		return resurrectCapability{status: resurrectBroken, stage: "validate save script", err: err}
	}
	return resurrectCapability{status: resurrectReady, saveScript: script}
}
