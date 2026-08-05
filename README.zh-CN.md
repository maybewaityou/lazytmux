<div align="center">

# lazytmux

一个基于终端的交互式 **tmux 会话管理器** —— 灵感来自 [lazyssh](https://github.com/Adembc/lazyssh)。

[English](./README.md) | **[简体中文](./README.zh-CN.md)**

</div>

---

lazytmux 把 lazyssh 的体验带到了你的 tmux server 上。
<br/>
借助 lazytmux,你可以列出、搜索、排序、置顶、打标签、创建、编辑、杀死(kill)、分离(detach)并进入 tmux 会话 —— 全部在一个清爽、键盘驱动的 TUI 中完成。再也不用在 `tmux ls` 和 `tmux attach -t <name>` 之间来回切换;它就是套在你本地 tmux server 之上的一个 Tokyo Night 主题仪表盘。

---

## ✨ 功能特性

### 会话管理
- 📜 列出本地 tmux server 上的会话,带实时状态与窗口数。
- ➕ 从 UI 创建新会话；如果当前 tmux server 已加载 [tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect)，lazytmux 会立即保存一份恢复快照。
- ✏️ 就地编辑会话(名称、标签、备注)。
- 🗑️ 安全地杀死(kill)会话。已加载 tmux-resurrect 时，lazytmux 会同步更新恢复状态：仍有会话时立即保存新快照；删除最后一个会话时只移除活动 `last` 指针，保留所有时间戳历史。
- 🔌 分离(detach)会话,让其在后台继续运行。
- 📌 置顶 / 取消置顶常用会话,让它们始终排在顶部。
- 🏷️ 给会话打标签,方便分组与查找。

### 快速导航
- 🔍 按会话名模糊搜索。
- ⌨️ 一键进入:在 tmux 内使用 `switch-client`,在 tmux 外使用 `attach`(自动识别)。
- ↕️ 按名称 / 创建时间 / 活跃度 / 最近挂载时间排序。
- 🏷️ 按标签过滤会话列表(`f`,多选,OR 匹配)。

### 工作流
- 🧩 详情面板,展示每个会话的窗口列表。
- 📋 一键复制 `tmux attach -t <name>` 到剪贴板。
- 🔄 在后台刷新会话与窗口状态。
- ❓ 应用内帮助(`?`)——双列分组面板,显示全部快捷键。

---

## 🔒 工作原理

lazytmux 不会引入任何新的风险。它仅仅是系统原生 `tmux` 二进制程序的一个 TUI 封装。

- 所有操作(列出、创建、重命名、杀死、分离、挂载)都通过 `tmux` CLI 执行 —— lazytmux 从不直接连接 tmux server。

- lazytmux 不会读取或修改你的 tmux 配置；插件集成通过当前 tmux server 的运行时 option 自动发现。

- 置顶、标签和备注存放在 `~/.lazytmux/metadata.json`，日志写入 `~/.lazytmux/lazytmux.log`。元数据写入是原子的，因此即使进程崩溃也不会留下写了一半的文件。

- **可选的重启恢复：**当前 server 已加载 `tmux-resurrect` 时，新建会话会同步触发其配置的保存脚本。删除会话也会同步更新恢复状态：仍有会话时保存不含删除项的新快照；删除最后一个会话时仅撤销 resurrect 的活动 `last` 指针，所有时间戳历史仍可用于手动恢复。未安装插件时，创建和删除行为与之前完全一致。重启后的恢复仍由你自己的 resurrect/continuum 配置控制（例如 `@continuum-restore 'on'`）。Resurrect 能恢复 tmux 会话、窗口、窗格、布局、工作目录和受支持的命令，但不能恢复进程内存、实时网络连接或编辑器中未落盘的内容。

  快照协调采用 best-effort，并会串行化多个 lazytmux 实例。外部手动保存和 continuum 不共享 lazytmux 的锁；多个 tmux server 应分别配置不同的 `@resurrect-dir`，避免共用同一个 `last` 指针。

---

## 📷 应用截图

<div align="center">

### 📋 会话仪表盘
<img src="./docs/resources/list.png" alt="会话列表仪表盘" width="900" />

主仪表盘列出本地所有 tmux 会话,带实时状态与窗口数、每会话详情面板,置顶的常用会话始终排在顶部。

---

### 🔎 模糊搜索
<img src="./docs/resources/search.png" alt="模糊搜索会话" width="900" />

按 `/` 输入,列表实时收窄到匹配的会话。

---

### 🏷️ 标签过滤
<img src="./docs/resources/filters.png" alt="标签过滤多选面板" width="900" />

按 `f` 打开标签多选面板 —— 可任选若干标签,列表会过滤出匹配其中任意一个的会话(OR 匹配)。

<img src="./docs/resources/filtered-list.png" alt="过滤后的会话列表" width="900" />

当前激活的过滤器会显示在列表边框标题中,一眼即可看出是哪些标签在约束视图。该过滤器仅作用于当前视图、不会持久化 —— 重新启动后总是回到无过滤状态。

---

### ➕ 新建会话
<img src="./docs/resources/add.png" alt="新建会话" width="900" />

按 `a` 即可在 UI 中直接新建一个 tmux 会话。表单内可用 Tab 切换,同时可选填 tags 和自由文本备注(之后可用 `e` 整体编辑,或 `t` / `n` 单独改 tags / 备注)。

---

### 🔌 分离会话
<img src="./docs/resources/detach.png" alt="分离会话" width="900" />

按 `d` 分离选中的会话 —— 它会在后台继续运行,而你回到仪表盘。底部状态栏会确认操作结果。

---

### 🗑️ 杀死会话
<img src="./docs/resources/kill.png" alt="杀死会话" width="900" />

按 `k` 安全地杀死(kill)选中会话,底部状态栏会确认操作结果。

---

### 🔄 后台刷新
<img src="./docs/resources/refresh.png" alt="刷新会话状态" width="900" />

按 `r` 在后台从 tmux 拉取最新的会话与窗口状态,并保持当前选中不变。

---

### ❓ 应用内帮助
<img src="./docs/resources/help.png" alt="帮助面板" width="900" />

按 `?` 弹出双列分组的快捷键面板,列出全部按键 —— 它与仪表盘、本 README 的快捷键表同出一源。

</div>

---

## 📦 安装

### Option 1: Homebrew(macOS)

```bash
brew install maybewaityou/tap/lazytmux
```

lazytmux 会调用系统 `tmux`,Homebrew 会通过 `tmux` 依赖自动帮你装上。

> **较新版 Homebrew(5.1.15+/6.0):** 第三方 tap 默认不受信任。若安装时报 `Refusing to load formula ... from untrusted tap`,先信任该 tap(只需一次):
>
> ```bash
> brew trust maybewaityou/tap
> ```

### Option 2: 从 Release 下载二进制

从 [GitHub Releases](https://github.com/maybewaityou/lazytmux/releases) 下载预编译二进制。下面这段脚本会自动检测最新版本,并按你的系统拉取对应的 tarball(darwin/linux × amd64/arm64):

```bash
# 检测最新版本
LATEST_TAG=$(curl -fsSL https://api.github.com/repos/maybewaityou/lazytmux/releases/latest | jq -r .tag_name)

# 把 OS / 架构归一化成 Release 资源名(darwin/linux × amd64/arm64)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

# 下载 + 解压 + 安装
curl -LJO "https://github.com/maybewaityou/lazytmux/releases/download/${LATEST_TAG}/lazytmux_${OS}_${ARCH}.tar.gz"
tar -xzf lazytmux_${OS}_${ARCH}.tar.gz
sudo mv lazytmux /usr/local/bin/

# 享受!
lazytmux
```

### Option 3: 从源码构建

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
| `↑↓`       | 上下浏览会话                         |
| `←/→`      | 列表 ↔ 详情切换焦点                 |
| `Enter`    | 进入会话(`switch-client` / `attach`)  |
| `a`        | 新建会话                              |
| `e`        | 编辑会话(名称、标签、备注)        |
| `d`        | 分离会话(保留后台运行)                |
| `k`        | 杀死会话                              |
| `p`        | 置顶 / 取消置顶                       |
| `t`        | 编辑标签(逗号分隔)                   |
| `n`        | 编辑备注(自由文本)                   |
| `s`        | 切换排序字段                          |
| `S`        | 切换排序字段(跳过一项)              |
| `f`        | 按标签过滤会话                          |
| `r`        | 刷新                                  |
| `c`        | 复制 `tmux attach -t <name>`          |
| `?`        | 帮助(快捷键列表)                     |
| `Esc`      | 退出搜索框(保留搜索词)              |
| `q`        | 退出                                  |

**在会话表单中:**

| 按键           | 动作                |
| -------------- | ------------------- |
| `↑↓`          | 切换字段(仅 Name/Tags) |
| `Tab/Shift+Tab` | 在字段间切换        |
| `Enter`        | 提交(保存)         |
| `Shift+Enter`  | 换行(仅 Note 字段) |
| `Esc`          | 取消                |

提示:底部的状态栏会显示你上一次操作的结果。

---

## 🏗 架构

采用六边形架构(端口与适配器):

```
cmd/main.go                       → cobra 根命令,装配依赖 + tmux 存在性检查
internal/core/domain/             → Session / Window 领域模型
internal/core/ports/              → SessionRepository / SessionSnapshotter / SessionService / MetadataStore
internal/core/services/           → 业务逻辑
internal/adapters/tmuxcli/        → tmux CLI + 可选的 tmux-resurrect 快照适配器
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

## ⭐ 支持

如果你觉得 lazytmux 好用,欢迎给仓库点个 **star** ⭐️,也欢迎加入 [stargazers](https://github.com/maybewaityou/lazytmux/stargazers)。

### ☕ 赞助

如果你愿意支持开发:

<table>
  <tr>
    <td align="center">
      <img src="./docs/resources/donate-wechat.jpg" alt="微信" width="180" />
      <br/>
      <b>微信</b>
    </td>
    <td width="80"></td>
    <td align="center">
      <img src="./docs/resources/donate-alipay.jpg" alt="支付宝" width="180" />
      <br/>
      <b>支付宝</b>
    </td>
  </tr>
</table>

---

## 🙏 致谢

- 基于 [tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell)、[cobra](https://github.com/spf13/cobra) 与 [zap](https://go.uber.org/zap) 构建。
- 深受 [lazyssh](https://github.com/Adembc/lazyssh) 启发 —— 相同的架构,相同的交互语言,不同的目标。
- 主题:Tokyo Night。
