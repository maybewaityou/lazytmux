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

// Package tz ensures time.Local tracks the real system timezone on every
// platform lazytmux ships to — most critically Termux/Android.
//
// Why this package exists: Termux/Android stores the IANA tz database under a
// non-standard prefix ($PREFIX/share/zoneinfo, i.e. /data/data/com.termux/...)
// that time.LoadLocation never searches. Even with $TZ set, Go cannot load the
// zone and silently falls back to UTC, so every .Local() call formats UTC. The
// blank time/tzdata import embeds the database so LoadLocation is filesystem-
// agnostic; Init then rebinds time.Local from the first source that yields a
// loadable zone.
package tz

import (
	"os"
	"os/exec"
	"strings"
	"time"

	// Embed the IANA tz database into the binary. With this, LoadLocation
	// resolves any zone name without touching system files — the fix for
	// Termux/Android where zoneinfo lives outside Go's search paths.
	_ "time/tzdata"
)

// androidProp reads persist.sys.timezone via getprop (Termux/Android only).
// It is a package-level var so tests can inject a stub instead of needing the
// real getprop binary on the host.
var androidProp = readAndroidPropDefault

// readAndroidPropDefault is the production reader: `getprop` on Android returns
// the IANA name (e.g. "Asia/Shanghai") or exits non-zero on non-Android hosts.
func readAndroidPropDefault() string {
	out, err := exec.Command("getprop", "persist.sys.timezone").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Init resolves the system timezone and assigns it to time.Local, returning the
// resolved zone name ("" if none could be loaded, in which case time.Local is
// left at Go's default). Safe to call once at process start; callers should
// invoke it before any time formatting.
func Init() string {
	for _, name := range candidates() {
		loc, err := time.LoadLocation(name)
		if err != nil {
			continue
		}
		time.Local = loc
		return name
	}
	return ""
}

// candidates returns timezone-name candidates in priority order: $TZ first
// (explicit user/shell config — the OPPO-Pad/Termux case where it is already
// "Asia/Shanghai"), then the Android system property as a fallback for a fresh
// Termux that has not exported TZ.
func candidates() []string {
	var names []string
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		names = append(names, tz)
	}
	if name := androidProp(); name != "" {
		names = append(names, name)
	}
	return names
}
