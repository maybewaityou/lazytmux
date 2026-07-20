<div align="center">

# lazytmux

A terminal-based, interactive **tmux session manager** inspired by [lazyssh](https://github.com/Adembc/lazyssh).

</div>

List, search, sort, pin, create, rename, kill, and enter your tmux sessions — all from a keyboard-driven TUI.

## ✨ Features
- 📜 List sessions from the local tmux server (live status, window counts).
- 🔍 Fuzzy search by name.
- ⌨️ One-key enter: `switch-client` inside tmux, `attach` outside (auto-detected).
- ➕ Create / ✏️ rename / 🗑 kill sessions.
- 📌 Pin favorites to the top.
- ↕️ Sort by name / created / activity / last-attached.
- 🧩 Details pane with per-session window list.

## 📦 Install

```bash
make build && sudo mv bin/lazytmux /usr/local/bin/
```

Or run from source: `make run`.

## ⌨️ Key bindings

| Key | Action |
| --- | --- |
| `/` | Search |
| `↑↓` / `jk` | Navigate |
| `Enter` | Enter session |
| `a` / `e` / `d` | New / rename / kill |
| `p` | Pin / unpin |
| `s` / `S` | Sort field / cycle |
| `r` | Refresh |
| `c` | Copy `tmux attach -t <name>` |
| `q` | Quit |

## 🏗 Architecture

Hexagonal (ports & adapters), mirroring lazyssh:

```
cmd/main.go                     → cobra root, wires deps + tmux presence check
internal/core/domain/           → Session / Window models
internal/core/ports/            → SessionRepository / SessionService / MetadataStore
internal/core/services/         → business logic
internal/adapters/tmuxcli/      → tmux CLI adapter (parses list-sessions -F output)
internal/adapters/data/metadata → ~/.lazytmux/metadata.json (pins/tags)
internal/adapters/ui/           → tview TUI (Tokyo Night)
internal/logger/                → zap → ~/.lazytmux/lazytmux.log
```

## 🛠 Built with

[tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell) · [cobra](https://github.com/spf13/cobra) · [zap](https://go.uber.org/zap). Theme: Tokyo Night.
