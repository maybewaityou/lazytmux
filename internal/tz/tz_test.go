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

package tz

import (
	"testing"
	"time"
)

// withLocal saves and restores the package-global time.Local so a test that
// rebinds it cannot leak into siblings.
func withLocal(t *testing.T) {
	t.Helper()
	prev := time.Local
	t.Cleanup(func() { time.Local = prev })
}

// TestInitFromTZEnv is the OPPO-Pad/Termux regression: $TZ is set to a valid
// IANA name but the system has no zoneinfo at Go's standard search paths. With
// the embedded tzdb, LoadLocation must still succeed and rebind time.Local to
// the resolved zone (CST = +0800, no DST).
func TestInitFromTZEnv(t *testing.T) {
	withLocal(t)
	t.Setenv("TZ", "Asia/Shanghai")
	androidProp = func() string { return "" } // ignore getprop path

	if name := Init(); name != "Asia/Shanghai" {
		t.Fatalf("Init() = %q, want %q", name, "Asia/Shanghai")
	}
	_, off := time.Now().Zone()
	if off != 8*60*60 {
		t.Fatalf("time.Local offset = %d, want 28800 (UTC+8)", off)
	}
}

// TestInitFromAndroidProp covers a Termux install with no $TZ: Init falls back
// to `getprop persist.sys.timezone`, the authoritative Android source.
func TestInitFromAndroidProp(t *testing.T) {
	withLocal(t)
	t.Setenv("TZ", "")
	androidProp = func() string { return "Asia/Shanghai" }

	if name := Init(); name != "Asia/Shanghai" {
		t.Fatalf("Init() = %q, want %q", name, "Asia/Shanghai")
	}
	_, off := time.Now().Zone()
	if off != 8*60*60 {
		t.Fatalf("time.Local offset = %d, want 28800 (UTC+8)", off)
	}
}

// TestInitEmbeddedTZDataAvailable proves the blank time/tzdata import embedded
// the IANA database: an arbitrary zone loads even though no system file is
// consulted. This is the contract that makes Termux resolution work at all.
func TestInitEmbeddedTZDataAvailable(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v (tzdata not embedded?)", err)
	}
	if loc == nil {
		t.Fatal("LoadLocation returned nil location")
	}
}

// TestInitNoCandidates ensures that when neither $TZ nor getprop yields a name,
// Init degrades gracefully — returns "" and leaves time.Local untouched rather
// than panicking.
func TestInitNoCandidates(t *testing.T) {
	withLocal(t)
	t.Setenv("TZ", "")
	androidProp = func() string { return "" }
	before := time.Local

	if name := Init(); name != "" {
		t.Fatalf("Init() = %q, want empty", name)
	}
	if time.Local != before {
		t.Fatal("Init mutated time.Local despite having no candidate")
	}
}
