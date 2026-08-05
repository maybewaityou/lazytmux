<div align="center">

# lazytmux

A terminal-based, interactive **tmux session manager** — inspired by [lazyssh](https://github.com/Adembc/lazyssh).

**[English](./README.md)** | [简体中文](./README.zh-CN.md)

</div>

---

lazytmux brings the lazyssh experience to your tmux server.
<br/>
With lazytmux, you can list, search, sort, pin, tag, create, edit, kill, detach, and enter your tmux sessions — all from a clean, keyboard-driven TUI. No more juggling `tmux ls` and `tmux attach -t <name>`; just a Tokyo Night–themed dashboard over your local tmux server.

---

## ✨ Features

### Session Management
- 📜 List sessions from the local tmux server with live status and window counts.
- ➕ Create new sessions from the UI; if the current tmux server exposes a usable [tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect) save capability, lazytmux synchronously attempts a best-effort recovery snapshot.
- ✏️ Edit sessions (name, tags, note) in place.
- 🗑️ Kill sessions safely. When the resurrect save capability is available, lazytmux best-effort reconciles recovery state: remaining sessions are snapshotted, while deleting the final session removes only the active `last` pointer and preserves timestamped history.
- 🔌 Detach sessions (keep them running in the background).
- 📌 Pin / unpin favorites to keep them at the top.
- 🏷️ Tag sessions to group and find them later.

### Quick Navigation
- 🔍 Fuzzy search by session name.
- ⌨️ One‑keypress enter: `switch-client` inside tmux, `attach` outside (auto‑detected).
- ↕️ Sort by name / created / activity / last‑attached.
- 🏷️ Filter the list by tag (`f`, multi-select, OR‑matched).

### Workflow
- 🧩 Details pane with a per-session window list.
- 📋 Copy `tmux attach -t <name>` to the clipboard.
- 🔄 Background refresh of session and window state.
- ❓ In-app help (`?`) — every key binding in a two-column, grouped panel.

---

## 🔒 How it works

lazytmux does not introduce any new risks. It is simply a TUI wrapper around your system's native `tmux` binary.

- All primary session operations (list, create, rename, kill, detach, attach) are executed through the `tmux` CLI — lazytmux never talks to the tmux server directly. Optional resurrect integration also invokes the plugin-provided save script and manages its active `last` pointer.

- Your tmux configuration is never read or modified by lazytmux. Plugin integration is discovered from the current tmux server's runtime options.

- Pins, tags, and notes live in `~/.lazytmux/metadata.json`, and logs go to `~/.lazytmux/lazytmux.log`. Metadata writes are atomic, so a crashed run never leaves a half-written file.

- **Optional restart recovery:** lazytmux does not install, update, or configure tmux-resurrect or tmux-continuum. Basic session creation and deletion never require either plugin. For each create/kill operation, lazytmux queries the current server's `@resurrect-save-script-path` option and enables snapshot coordination only when it names an absolute, regular, executable file. A plugin directory downloaded by TPM is not enough unless that plugin is loaded into the current server; tmux-continuum alone is not a save capability.

  When the capability is ready, creating a session synchronously attempts a verified snapshot. tmux-resurrect has no session-deletion API, so lazytmux coordinates deletion itself: if sessions remain, it attempts to save a snapshot without the deleted session; if the final session is killed, it safely unlinks only resurrect's active `last` pointer and preserves every timestamped history file. Without the capability, create/kill use the normal tmux behavior. Configuration or snapshot failures are logged to `~/.lazytmux/lazytmux.log` and never turn a successful primary create/kill into a failure.

  Restoring after reboot remains controlled by your own resurrect/continuum configuration (for example `@continuum-restore 'on'`). Resurrect restores tmux sessions, windows, panes, layouts, working directories, and supported commands — not process memory, live network connections, or unsaved editor state. Snapshot reconciliation coordinates concurrent lazytmux instances, but external manual/continuum saves do not share lazytmux's lock, and multiple tmux servers should use separate `@resurrect-dir` values to avoid sharing one `last` pointer.

---

## 📷 Screenshots

<div align="center">

### 📋 Session Dashboard
<img src="./docs/resources/list.png" alt="Session list dashboard" width="900" />

Main dashboard listing every local tmux session with live status and window counts, a per-session details pane, and pinned favorites kept at the top.

---

### 🔎 Fuzzy Search
<img src="./docs/resources/search.png" alt="Fuzzy search sessions" width="900" />

Press `/`, type, and the list narrows to matching sessions in real time.

---

### 🏷️ Tag Filter
<img src="./docs/resources/filters.png" alt="Tag filter multi-select" width="900" />

Press `f` to open a multi-select tag picker — choose any number of tags and the list filters to sessions matching any of them (OR-matched).

<img src="./docs/resources/filtered-list.png" alt="Filtered session list" width="900" />

The active filter is surfaced in the list's border title, so it's always clear which tags are constraining the view. The filter is view-only and never persisted — a fresh launch always starts unfiltered.

---

### ➕ Create Session
<img src="./docs/resources/add.png" alt="Create a new session" width="900" />

Press `a` to spin up a new tmux session straight from the UI. Tab through the form to optionally add tags and a freeform note at the same time (re-edit anytime with `e`, or `t` / `n` for just tags / note).

---

### 🔌 Detach Session
<img src="./docs/resources/detach.png" alt="Detach a session" width="900" />

Press `d` to detach the selected session — it keeps running in the background while you drop back to the dashboard. The status bar confirms the result.

---

### 🗑️ Kill Session
<img src="./docs/resources/kill.png" alt="Kill a session" width="900" />

Press `k` to safely kill the selected session — the status bar confirms the result.

---

### 🔄 Background Refresh
<img src="./docs/resources/refresh.png" alt="Refresh session state" width="900" />

Press `r` to pull fresh session and window state from tmux in the background while keeping your selection intact.

---

### ❓ In-app Help
<img src="./docs/resources/help.png" alt="Help panel" width="900" />

Press `?` to pop up a two-column, grouped panel listing every key binding — the same single source the dashboard, this README's Key Bindings table, and the `?` panel all draw from.

</div>

---

## 📦 Installation

### Option 1: Homebrew (macOS)

```bash
brew install maybewaityou/tap/lazytmux
```

`lazytmux` drives your system `tmux`, which Homebrew pulls in automatically via the `tmux` dependency.

> **Newer Homebrew (5.1.15+/6.0):** third-party taps are untrusted by default. If install fails with `Refusing to load formula ... from untrusted tap`, trust the tap first (one-time):
>
> ```bash
> brew trust maybewaityou/tap
> ```

### Option 2: Download Binary from Releases

Download a prebuilt binary from [GitHub Releases](https://github.com/maybewaityou/lazytmux/releases). The snippet below detects the latest version and fetches the right tarball for your OS/arch (darwin/linux × amd64/arm64):

```bash
# Detect latest version
LATEST_TAG=$(curl -fsSL https://api.github.com/repos/maybewaityou/lazytmux/releases/latest | jq -r .tag_name)

# Normalize OS/arch to the release asset name (darwin/linux × amd64/arm64)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

# Download + extract + install
curl -LJO "https://github.com/maybewaityou/lazytmux/releases/download/${LATEST_TAG}/lazytmux_${OS}_${ARCH}.tar.gz"
tar -xzf lazytmux_${OS}_${ARCH}.tar.gz
sudo mv lazytmux /usr/local/bin/

# enjoy!
lazytmux
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/maybewaityou/lazytmux.git
cd lazytmux

# Build (runs fmt + go vet first)
make build
sudo mv bin/lazytmux /usr/local/bin/

# Or run it directly without installing
make run
```

### Snapshot binaries (optional)

`make build-all` produces cross-compiled snapshots via [goreleaser](https://goreleaser.com) (linux/darwin × amd64/arm64):

```bash
make build-all
```

---

## ⌨️ Key Bindings

| Key   | Action                                  |
| ----- | --------------------------------------- |
| `/`   | Focus search bar                        |
| `↑↓`  | Navigate sessions                |
| `←/→` | Focus list ↔ details             |
| `Enter` | Enter session (`switch-client` / `attach`) |
| `a`   | New session & enter                             |
| `e`   | Edit session (name, tags, note)         |
| `d`   | Detach session (keeps it running)       |
| `k`   | Kill session                            |
| `p`   | Pin / unpin                             |
| `t`   | Edit tags (comma-separated)             |
| `n`   | Edit note (multi-line)                  |
| `s`   | Cycle sort field                        |
| `S`   | Cycle sort field (skip one)             |
| `f`   | Filter sessions by tag                 |
| `r`   | Refresh                                 |
| `c`   | Copy `tmux attach -t <name>`            |
| `?`   | Help (key bindings)                     |
| `Esc` | Blur search bar (keeps the query)       |
| `q`   | Quit                                    |

**In the session form:**

| Key           | Action              |
| ------------- | ------------------- |
| `↑↓`         | Switch field (Name/Tags) |
| `Tab/Shift+Tab` | Move between fields |
| `Enter`       | Submit (save)       |
| `Shift+Enter` | Newline (in Note)   |
| `Esc`         | Cancel              |

The same keys apply when editing a note directly with `n`: Shift+Enter inserts a newline, Enter saves.

Tip: the status bar at the bottom shows the result of your last action.

---

## 🏗 Architecture

Hexagonal (ports & adapters):

```
cmd/main.go                       → cobra root, wires deps + tmux presence check
internal/core/domain/             → Session / Window models
internal/core/ports/              → SessionRepository / SessionSnapshotter / SessionTerminator / SessionService / MetadataStore
internal/core/services/           → business logic
internal/adapters/tmuxcli/        → tmux CLI + optional tmux-resurrect snapshot adapter
internal/adapters/data/metadata   → ~/.lazytmux/metadata.json (pins/tags)
internal/adapters/ui/             → tview TUI (Tokyo Night)
internal/logger/                  → zap → ~/.lazytmux/lazytmux.log
```

---

## 🤝 Contributing

Contributions are welcome!

- If you spot a bug or have a feature request, please [open an issue](https://github.com/maybewaityou/lazytmux/issues).
- If you'd like to contribute, fork the repo and submit a pull request ❤️.

### Semantic commit messages

This project follows semantic commits. Please format your commit/PR title as:

- `type(scope): short descriptive subject`

Common types: `feat`, `fix`, `improve`, `refactor`, `docs`, `test`, `ci`, `chore`, `revert`.
Scope is optional (e.g. `ui`, `cli`, `core`).

Examples:
- `feat(ui): keep cursor when ESC blurs search bar`
- `fix(core): handle empty session list on startup`
- `docs: expand installation instructions`

---

## ⭐ Support

If you find lazytmux useful, please consider giving the repo a **star** ⭐️ and joining the [stargazers](https://github.com/maybewaityou/lazytmux/stargazers).

### ☕ Sponsor

If you'd like to support development:

<a href="https://www.buymeacoffee.com/maybewaityou" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" width="200" /></a>

**WeChat Pay / Alipay**

<table>
  <tr>
    <td align="center">
      <img src="./docs/resources/donate-wechat.jpg" alt="WeChat Pay" width="180" />
      <br/>
      <b>WeChat Pay</b>
    </td>
    <td width="80"></td>
    <td align="center">
      <img src="./docs/resources/donate-alipay.jpg" alt="Alipay" width="180" />
      <br/>
      <b>Alipay</b>
    </td>
  </tr>
</table>

---

## 🙏 Acknowledgments

- Built with [tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell), [cobra](https://github.com/spf13/cobra), and [zap](https://go.uber.org/zap).
- Heavily inspired by [lazyssh](https://github.com/Adembc/lazyssh) — same architecture, same UX language, different target.
- Theme: Tokyo Night.
