# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What it is

`lazytmux` is a terminal TUI for managing local tmux sessions — the lazyssh UX reimplemented in Go, pointed at tmux instead of SSH. It is a thin wrapper around the system `tmux` binary: it never talks to the tmux server directly, never reads `~/.tmux.conf`, and writes only its own state under `~/.lazytmux/`.

## Commands

The `makefile` (lowercase) is the source of truth:

```bash
make run            # go run ./cmd  (version/commit injected via ldflags)
make build          # runs `quality` (fmt + vet), then builds bin/lazytmux
make test           # go test -race -cover ./...
make test-verbose   # same with -v
make lint           # golangci-lint run ./...   (enable-all, very strict)
make fmt            # gofumpt + go fmt
make quality        # fmt + go vet   (prerequisite of build)
make build-all      # goreleaser cross-compile snapshot (linux/darwin × amd64/arm64)
```

Run a single test: `go test -run TestParseSessions -race -v ./internal/adapters/tmuxcli/`

The binary requires `tmux` on `$PATH` (checked at startup via `exec.LookPath`). tmux is a runtime dependency, not a build dependency.

## Architecture: hexagonal (ports & adapters)

`cmd/main.go` is the **only composition root** — it hand-wires concrete adapters into core ports. The dependency rule is one-way: `core/` imports nothing from `adapters/`.

- `internal/core/domain/` — `Session`, `Window` value types. `Session` carries both tmux-sourced fields and UI-metadata fields (`Pinned`/`Tags`/`LastAttached`/`Note`) that the service injects from the metadata store at read time.
- `internal/core/ports/` — interfaces only: `SessionRepository`, `SessionService`, `MetadataStore`, plus the `SuspendFunc` type and `ErrSuspendRequired` sentinel.
- `internal/core/services/` — business logic. `ListSessions` is the merge point: it pulls tmux data, then overlays per-session pin/tag/last-attached/note metadata.
- `internal/adapters/tmuxcli/` — shells out to `tmux`. `CommandRunner` interface lets tests inject a fake instead of `os/exec`.
- `internal/adapters/data/metadata/` — JSON store at `~/.lazytmux/metadata.json`.
- `internal/adapters/ui/` — tview TUI, Tokyo Night theme (palette in `const.go`).
- `internal/logger/` — zap `SugaredLogger` → `~/.lazytmux/lazytmux.log` (JSON), logger named `LAZYTMUX`.

Every adapter proves it satisfies its port with a compile-time assertion: `var _ ports.X = (*Y)(nil)`.

## Conventions and gotchas

**Copyright header is lint-enforced.** `.golangci.yml` runs `goheader` and requires every `.go` file (including `*_test.go` — verified) to begin with the Apache 2.0 header, where the `{{ COMPANY }}` const is `2026`. A new file without it fails `make lint`. Copy the header verbatim from any existing file. Test files are exempt from `gocyclo`/`errcheck`/`gosec`/`gochecknoglobals`, but **not** from `goheader`.

**`enable-all: true`** — golangci-lint enables every linter except `depguard`. Expect revive/gocritic/staticcheck/gosec findings on any non-trivial change; fix them rather than adding `//nolint` (the few existing `//nolint` directives carry an explanatory comment — follow that pattern if one is truly needed).

**The "smart enter" flow breaks an import cycle at runtime.** `EnterSession` (`services/session_service.go`) runs `tmux switch-client` when inside tmux (`$TMUX` set), but outside tmux it must suspend the TUI and run `tmux attach` interactively. The repository signals this by returning the `ErrSuspendRequired` sentinel; `main.go` then injects the TUI's `Suspend` function into the service via runtime interface assertions (`interface{ Suspend(...) error }` ↔ `interface{ SetSuspend(...) }`). This late wiring — not constructor injection — is what avoids the ui↔service import cycle. Do not "simplify" it into a direct dependency.

**tmux output parsing is brittle and version-sensitive — keep three things in lockstep.** Session/window data comes from `tmux list-sessions -F` / `list-windows -F` with `|`-delimited format strings. The format constants (`sessionsFormat`, `windowsFormat`) live in `tmuxcli/tmux_repo.go`; the expected field counts (7 and 4) are hardcoded in `tmuxcli/parser.go`. If you touch one, update all three: the `-F` string, the parser's field count, and the struct mapping. Lines with the wrong field count are **silently skipped**, so a mismatch degrades to an empty/short list with no error.

**"no server running" is the normal empty state, not an error.** tmux has no daemon — its server starts with the first session and exits with the last, so a 0-session system surfaces as a `list-sessions` *failure*, not empty stdout. `isNoServerError` (`tmux_repo.go`) swallows it so the UI renders its empty state. The wording differs by platform: Linux says `"no server running"`, macOS says `"error connecting to"` — both substrings are matched. Supporting a new platform means extending this matcher. The same lesson applies to `runner.go`: it always surfaces tmux's stderr so failure modes stay distinguishable instead of collapsing to `exit status 1`.

**Metadata writes are atomic + backup-once.** `metadata.Store.save()` writes `metadata.json.tmp` then `os.Rename`. Before the *first* write it stashes `metadata.json.original.backup` (never overwritten) as a recovery net. Preserve this pattern when extending the store. State is `sync.RWMutex`-guarded and loaded once at startup into memory; reads (`IsPinned`/`Tags`/`LastAttached`/`Note`) keep hitting that in-memory cache, so a concurrent instance's change is invisible until this process restarts or mutates. **Every mutation is a read-modify-write, not a blind overwrite.** `mutate()` re-reads the freshest on-disk state under the write lock *before* applying the change and saving — this is what stops several lazytmux processes (common inside tmux, where multiple panes/instances share the file) from silently clobbering each other's edits. Decode the reload into a **fresh** `fileModel` (see `reloadLocked`): `json.Unmarshal` *merges* into existing maps and never drops keys absent from the payload, so reusing the live `s.data` would leave stale entries behind. There remains a narrow cross-process TOCTOU window between the read and the rename; writes are low-frequency and manual, so a `flock` would be over-engineering for this single-user tool.

**Version is injected via ldflags.** `main.version` and `main.gitCommit` are set by `-ldflags -X` in both the `makefile` and `.goreleaser.yaml`; they default to `"develop"` / `"unknown"`. The TUI header reads them.

**Current-session detection is best-effort.** The `▶` in the list and the `(current)` suffix in the details pane mark the tmux session the lazytmux process is currently running inside, resolved by `SessionRepository.CurrentSession()` via `tmux display-message -p '#S'` (returns `("", false)` when `$TMUX` is unset). It is a decorative enhancement: any failure degrades silently to "no marker" rather than surfacing an error — the same philosophy as `isNoServerError` translating "no server" into an empty list.

**Tag filter is view-only; keybindings are a single source.** The `f`-key tag filter (`ui/tag_filter.go`) writes only to an in-memory `tui.tagFilter` slice — never to `metadata.json` — so a fresh launch always starts unfiltered (a user's last filter must not pollute the next session). The session list flows through one filter pipeline, `visibleSessions()` (`ui/handlers.go`): tag filter → name search → sort, so the three compose orthogonally across every render path (search, sort, refresh). The active sort and filter are surfaced in the list border title (`SetSortTitle`/`SetFilter`), which is why the footer `defaultHints` (`ui/status_bar.go`) can stay short — just the core keys plus `?` Help — without advertising sort/filter. Every advertised key lives in one slice, `keyBindings` (`ui/keybindings.go`); the `?` help panel (`ui/help.go`, two-column via `renderHelpColumns`) and the README Key Bindings table both derive from it, so adding a key means editing that one slice (and the `TestKeyBindingsComplete` test that locks it) — keep the README table in sync when you do.

## Release

Pushing a `v*` tag triggers `.github/workflows/release.yml` → goreleaser builds linux/darwin × amd64/arm64, publishes the GitHub Release, and pushes an auto-generated Homebrew formula to `maybewaityou/homebrew-tap`. The formula step needs `HOMEBREW_TAP_GITHUB_TOKEN` — a PAT with `contents:write` on the tap repo — because the default `GITHUB_TOKEN` cannot write cross-repo. The formula declares a `tmux` runtime dependency (lazytmux shells out to it, and `brew test` runs `lazytmux --help`). Locally, `make build-all` produces snapshot binaries without cutting a release.

## Commit / PR titles

Semantic: `type(scope): subject`. Types: `feat`, `fix`, `improve`, `refactor`, `docs`, `test`, `ci`, `chore`, `revert`. Scope is optional but conventional (`ui`, `core`, `cli`, `tmux`). Examples: `feat(ui): keep cursor when ESC blurs search bar`, `fix(core): handle empty session list on startup`.

## Local-only files (gitignored — not project conventions)

`.superpowers/`, `docs/superpowers/`, `scripts/deploy.sh`, and `docs/homebrew-release.md` exist on disk but are gitignored personal tooling/notes — do not treat them as authoritative structure.
