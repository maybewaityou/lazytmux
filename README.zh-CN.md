<div align="center">

# lazytmux

一个基于终端的交互式 **tmux 会话管理器** —— 灵感来自 [lazyssh](https://github.com/Adembc/lazyssh)。

[English](./README.md) | **[简体中文](./README.zh-CN.md)**

</div>

---

lazytmux 把 lazyssh 的体验带到了你的 tmux server 上。
<br/>
借助 lazytmux,你可以列出、搜索、排序、置顶、创建、重命名、杀死(kill)并进入 tmux 会话 —— 全部在一个清爽、键盘驱动的 TUI 中完成。再也不用在 `tmux ls` 和 `tmux attach -t <name>` 之间来回切换;它就是套在你本地 tmux server 之上的一个 Tokyo Night 主题仪表盘。

---

## ✨ 功能特性

### 会话管理
- 📜 列出本地 tmux server 上的会话,带实时状态与窗口数。
- ➕ 从 UI 创建新会话。
- ✏️ 就地重命名会话。
- 🗑️ 安全地杀死(kill)会话。
- 📌 置顶 / 取消置顶常用会话,让它们始终排在顶部。

### 快速导航
- 🔍 按会话名模糊搜索。
- ⌨️ 一键进入:在 tmux 内使用 `switch-client`,在 tmux 外使用 `attach`(自动识别)。
- ↕️ 按名称 / 创建时间 / 活跃度 / 最近挂载时间排序。

### 工作流
- 🧩 详情面板,展示每个会话的窗口列表。
- 📋 一键复制 `tmux attach -t <name>` 到剪贴板。
- 🔄 在后台刷新会话与窗口状态。

---

## 🔒 工作原理

lazytmux 不会引入任何新的风险。它仅仅是系统原生 `tmux` 二进制程序的一个 TUI 封装。

- 所有操作(列出、创建、重命名、杀死、挂载)都通过 `tmux` CLI 执行 —— lazytmux 从不直接连接 tmux server。

- 你的 `~/.tmux.conf` 和现有会话永远不会被 lazytmux 读取或修改。

- lazytmux 唯一会写入的是它自己的状态:置顶(pin)与标签(tag)存放在 `~/.lazytmux/metadata.json`,日志写入 `~/.lazytmux/lazytmux.log`。写入是原子的,因此即使进程崩溃也不会留下写了一半的元数据文件。

---

## 📦 安装

### Homebrew(macOS / Linux)

```bash
brew tap maybewaityou/tap
brew install lazytmux
```

lazytmux 会调用系统 `tmux`,Homebrew 会通过 `tmux` 依赖自动帮你装上。

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/maybewaityou/lazytmux.git
cd lazytmux

# 构建(会先执行 fmt + go vet)
make build
sudo mv bin/lazytmux /usr/local/bin/

# 或者不安装、直接运行
make run
```

### 快照二进制(可选)

`make build-all` 通过 [goreleaser](https://goreleaser.com) 生成交叉编译快照(linux/darwin × amd64/arm64):

```bash
make build-all
```

---

## ⌨️ 快捷键

| 按键       | 动作                                  |
| ---------- | ------------------------------------- |
| `/`        | 聚焦搜索框                            |
| `↑↓` / `jk` | 上下浏览会话                         |
| `Enter`    | 进入会话(`switch-client` / `attach`)  |
| `a`        | 新建会话                              |
| `e`        | 重命名会话                            |
| `d`        | 杀死会话                              |
| `p`        | 置顶 / 取消置顶                       |
| `s`        | 切换排序字段                          |
| `S`        | 切换排序字段(跳过一项)              |
| `r`        | 刷新                                  |
| `c`        | 复制 `tmux attach -t <name>`          |
| `Esc`      | 退出搜索框(保留搜索词)              |
| `q`        | 退出                                  |

**在会话表单中:**

| 按键    | 动作   |
| ------- | ------ |
| `Enter` | 提交   |
| `Esc`   | 取消   |

提示:底部的状态栏会显示你上一次操作的结果。

---

## 🏗 架构

采用六边形架构(端口与适配器),与 lazyssh 一致:

```
cmd/main.go                       → cobra 根命令,装配依赖 + tmux 存在性检查
internal/core/domain/             → Session / Window 领域模型
internal/core/ports/              → SessionRepository / SessionService / MetadataStore
internal/core/services/           → 业务逻辑
internal/adapters/tmuxcli/        → tmux CLI 适配器(解析 list-sessions -F 输出)
internal/adapters/data/metadata   → ~/.lazytmux/metadata.json(置顶 / 标签)
internal/adapters/ui/             → tview TUI(Tokyo Night)
internal/logger/                  → zap → ~/.lazytmux/lazytmux.log
```

---

## 🤝 参与贡献

欢迎贡献!

- 如果你发现了 bug 或有新功能想法,请[提一个 issue](https://github.com/maybewaityou/lazytmux/issues)。
- 如果你愿意贡献代码,fork 仓库后提交 pull request ❤️。

### 语义化提交信息

本项目遵循语义化提交。请将你的 commit / PR 标题写成:

- `type(scope): 简短描述`

常见 type:`feat`、`fix`、`improve`、`refactor`、`docs`、`test`、`ci`、`chore`、`revert`。
scope 可选(例如 `ui`、`cli`、`core`)。

示例:
- `feat(ui): keep cursor when ESC blurs search bar`
- `fix(core): handle empty session list on startup`
- `docs: expand installation instructions`

---

## 🙏 致谢

- 基于 [tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell)、[cobra](https://github.com/spf13/cobra) 与 [zap](https://go.uber.org/zap) 构建。
- 深受 [lazyssh](https://github.com/Adembc/lazyssh) 启发 —— 相同的架构,相同的交互语言,不同的目标。
- 主题:Tokyo Night。
