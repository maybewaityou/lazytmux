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

package ui

import "testing"

// Rename 用 InitialValue 把当前 session 名预填进输入框(光标停末尾)。
func TestSessionFormInitialValue(t *testing.T) {
	f := NewSessionForm("Rename", "new name").InitialValue("mysession")
	if got := f.Input().GetText(); got != "mysession" {
		t.Fatalf("InitialValue: got %q, want mysession", got)
	}
}

// 新建会话流程传空串,必须是 no-op,输入框保持空。
func TestSessionFormInitialValueEmptyIsNoOp(t *testing.T) {
	f := NewSessionForm("New session", "session name").InitialValue("")
	if got := f.Input().GetText(); got != "" {
		t.Fatalf("empty InitialValue should be no-op, got %q", got)
	}
}
