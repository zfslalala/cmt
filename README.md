# QG - Quick Git

快速 Git 操作工具：一条命令完成常见 Git 操作，可选接入大模型，用 AI 自动生成 commit message。

## 功能特性

- ⚡ **快速分支同步**：一条命令合并当前分支到目标分支并推送（`qg gmt`）
- 🤖 **AI 生成提交信息**：根据暂存区变更自动生成符合 Conventional Commits 规范的 commit message（`qg cmt`）
- 🌳 **工作树管理**：创建新分支并在工作树根目录同级创建独立工作树，可一键用 IDEA / WorkBuddy / DeepSeek Harness / Qoder / Codex Desktop 打开（`qg wt`）
- 🧠 **知识库挂载**：知识库放在仓库外，用符号链接挂进工作树，多个工作树共享同一份项目知识（`qg kb`）
- 🧹 **分支清理**：删除当前分支、对应工作树及 DSH 工作区，可选连远程分支一起删除（`qg rm`）
- 🏃 **脚本运行**：用 Go 编写一次性脚本，像本地命令一样执行（`qg run`）
- ✏️ 支持精简/详细模式、编辑确认后再提交、一键提交并推送
- 🔧 支持所有 OpenAI 协议兼容的大模型服务，以及 Anthropic Messages API
- 🪶 单二进制文件，零依赖；纯 Git 操作无需任何配置

## 环境要求

- Git（必需）
- Go 1.21+（仅 `qg run` 和源码编译需要）
- 可选：IntelliJ IDEA、WorkBuddy、DeepSeek Harness Desktop、Qoder、Codex Desktop（分别用于 `qg wt -i / -w / -d / -q / -c`；`-d` 还需要仓库中的 `dsh-plugin`，`-q` 建议安装带 CLI 的 Qoder IDE，`-c` 需要可执行的 `codex`）

## 命令结构

```text
qg
├── cmt             AI 生成 commit message（-v / -e / -p）
├── gmt <branch>    快速分支同步：合并当前分支到目标分支并推送（-b 触发构建）
├── from            查看当前分支来源信息（来源 / 切出时间）
├── p <备注>         提交所有修改并推送
├── wt <branch>     创建分支和工作树（--from 指定起点 / -i IDEA / -w WorkBuddy / -d DSH Desktop / -q Qoder / -c Codex）
├── run <脚本>       执行 Go 脚本（--timeout 秒数；脚本目录 ~/.qg/scripts/）
├── kb              知识库挂载与状态
│   ├── attach [路径] 挂载知识库（缺省当前目录）
│   └── status      查看各工作树的挂载状态
└── rm              删除当前分支、工作树及 DSH 工作区（-o 连远程分支一起删 / --force 强制）
```

直接运行 `qg` 或 `qg --help` 会显示命令帮助；`qg <命令> --help` 查看单个命令的旗标。

## 安装

### macOS / Linux

**方式一：下载预编译版本**

从 [Releases](https://github.com/zfslalala/cmt/releases) 下载对应平台压缩包（如 `qg-darwin-arm64.tar.gz`），解压后安装：

```bash
tar -xzf qg-darwin-arm64.tar.gz
chmod +x qg-darwin-arm64
sudo mv qg-darwin-arm64 /usr/local/bin/qg
```

可选平台包：`qg-darwin-amd64.tar.gz`、`qg-darwin-arm64.tar.gz`、`qg-linux-amd64.tar.gz`、`qg-linux-arm64.tar.gz`。

**方式二：源码编译**

```bash
git clone https://github.com/zfslalala/cmt.git
cd cmt
make install                 # 安装到 /usr/local/bin

# 或安装到用户目录，避免 sudo（确保 $HOME/.local/bin 在 PATH 中）
make install PREFIX=$HOME/.local
```

**方式三：Go install**

```bash
go install github.com/zfslalala/cmt/cmd@latest
# go install 以目录名命名产物，即 $GOPATH/bin/cmd，重命名为 qg
mv "$(go env GOPATH)/bin/cmd" "$(go env GOPATH)/bin/qg"
```

### Windows

**方式一：下载预编译版本**

1. 从 [Releases](https://github.com/zfslalala/cmt/releases) 下载 `qg-windows-amd64.zip`
2. 解压得到 `qg-windows-amd64.exe`，改名为 `qg.exe`
3. 放到 `C:\Windows\System32\` 或其他 PATH 目录

**方式二：源码编译**

```powershell
git clone https://github.com/zfslalala/cmt.git
cd cmt
go build -ldflags="-s -w" -o qg.exe ./cmd
# 把 qg.exe 移到 PATH 目录
```

### 验证安装

```bash
qg --help
```

## 使用

### 创建工作树（`qg wt`）

```bash
# 基于当前 HEAD 创建分支 feature/login，并在仓库同级目录创建工作树
qg wt feature/login

# 从线上最新 release 切出（先 fetch origin/release，再以 origin/release 为起点）
qg wt feature/login --from release

# 创建后在 IntelliJ IDEA 中打开
qg wt feature/login -i

# 创建后在 WorkBuddy 中打开（新任务草稿）
qg wt feature/login -w

# 创建后在 DSH Desktop 中创建工作区并打开
qg wt feature/login -d

# 创建后在 Qoder 中打开工作区
qg wt feature/login -q

# 创建后在 Codex Desktop 中打开工作区
qg wt feature/login -c
```

#### 起点选择

优先级从高到低：

1. `--from <分支>`：确认 `origin` 上存在该分支并 `git fetch origin <分支>`，再以最新的 `origin/<分支>` 为起点。远程分支不存在、断网或鉴权失败都会直接报错，**不会回退本地同名分支**。
2. `qg.hjson` 中的 `wt.base`：适合把团队的集成分支（如 `release`）设为默认起点。
3. 当前 `HEAD`。

```hjson
{
  wt: {
    base: release
  }
}
```

`--from` 也可以直接写 `origin/release`，效果相同。

新分支**不会继承起点分支的 upstream**。从 `release` 切出的 `feature/login` 不会跟踪 `origin/release`；首次 `qg p` / `qg cmt -p` 会自动以 `origin/feature/login` 建立跟踪，避免误推到起点分支。

#### 目录与分支规则

- 工作树创建在当前工作树根目录的**同级目录**，从仓库任意子目录执行结果一致。目录名由分支名转换而来：`/`、`\`、空格以及 Windows 非法字符都会替换为 `-`。例如 `feature/login` → 仓库根同级的 `feature-login`。
- 分支已存在时拒绝创建，避免误操作已有分支；可用 `git worktree add <路径> <已有分支>` 为已有分支补工作树。
- 分支已在任意工作树中检出时拒绝创建。
- 目标目录已存在时拒绝创建。

#### 打开方式

| 旗标 | 行为 | 依赖 |
|---|---|---|
| `-i, --idea` | 用 IntelliJ IDEA 打开工作树 | 本机安装 IDEA（PATH 中的 `idea`、Toolbox 脚本或 `/Applications/IntelliJ IDEA*.app`） |
| `-w, --workbuddy` | 通过 `workbuddy://task?action=start&cwd=...` 新建 WorkBuddy 任务草稿 | 安装 WorkBuddy 并注册 URL 协议 |
| `-d, --dsh` | 在整合版 DSH Desktop 中创建/复用该工作树的工作区并打开 | 安装仓库 `dsh-plugin/` 中的 `qg-dsh-open` 插件；DSH Desktop 已启动 |
| `-q, --qoder` | 通过 `qoder --new-window <目录>` 在 Qoder 中打开工作树 | Qoder IDE 自带的 CLI（`/Applications/Qoder IDE.app/Contents/Resources/app/bin/qoder` 或 PATH 中的 `qoder`）；找不到 CLI 时回退系统应用打开 |
| `-c, --codex` | 通过 `codex app <目录>` 在 Codex Desktop 中打开工作区 | Codex Desktop 安装后自带的 `codex` CLI（PATH 或 ChatGPT/Codex 应用包内）；可用 `QG_CODEX_CLI` 指定路径 |

说明：

- `-d` 会启动或聚焦 **DSH Desktop**（不是浏览器 Web UI），通过 `qg-dsh-open` 插件的本地桥接调用 `workspaceRegistry` 创建工作区，再由 Client 半打开它。
- 工作区已存在时直接复用并打开，保留原名称；新工作区以目录名作为标题。
- DSH 未运行时 qg 会先尝试启动 DeepSeek Harness，并等待桥接就绪（最多约 15 秒）。
- 插件未安装、未加载或令牌失效时只打印警告，**工作树仍会保留**，不会回滚已创建的分支。
- `-q` 优先调用 Qoder IDE 的 CLI，以 `--new-window` 打开工作树并创建工作区；CLI 不存在时回退到系统应用打开。当前 Qoder Desktop（`com.qoder.app`）不保证响应命令行传入的目录，若未自动创建工作区，按提示在 Qoder 中手动打开即可。可用 `QG_QODER_CLI` 指定 CLI 路径。
- `-c` 调用 Codex 官方的 `codex app <目录>` 启动器打开工作区；未检测到 `codex`、CLI 版本过旧或桌面端启动失败时只警告，工作树仍会保留，可手动执行 `codex app <目录>`。
- 工作树创建后自动挂载知识库（等价 `qg kb attach`）并生成 `AGENTS.local.md`；挂载失败只警告，工作树保留。
- 多个旗标可以同时使用，例如 `qg wt feature/login -i -d -q -c`。

#### DSH 插件（`qg-dsh-open`）

`-d` 依赖仓库中的 [dsh-plugin](dsh-plugin/)：

1. 在 DSH Desktop 中打开 **设置 → 插件 → 添加插件**，选择本地目录 `<仓库>/dsh-plugin`。
2. 安装后重启 DSH Desktop（若插件 HMR 未自动加载）。
3. 验证桥接已启动：

```bash
ls -l ~/.qg/dsh-bridge.json
```

插件由两部分组成：

- Host 半在 `127.0.0.1` 的随机端口启动桥接服务，把 `{port, token}` 写入 `~/.qg/dsh-bridge.json`（权限 `0600`）。
- Client 半监听工作区列表，看到标记后调用 `uiWorkspace.openWorkspace()` 打开工作区，并恢复原标题。

可用 `QG_DSH_BRIDGE` 覆盖桥接文件路径（测试或自定义安装时使用）。

### 知识库（`qg kb`）

知识库放在仓库之外（默认仓库同级 `_ai-kb`，其次 `~/.ai-kb`），并投影成 Codex、DSH、Qoder 都能自动读取的同一套工作区入口：

- 根 `AGENTS.md`：Codex、DSH、Qoder 都会自动加载。
- `.dsh/`：DSH 的技能、项目基座与分支知识视图。
- `.agents/skills`：Codex 的项目技能发现入口。
- `.qoder/skills`：Qoder 的项目技能发现入口。

主工作树的 `AGENTS.md` 直接链接知识库基座；特性分支的 `AGENTS.md` 由 qg 合并“共享基座 + `AGENTS.local.md`”，因此三个 agent 会优先读到同一份项目知识，同时保留当前分支的需求与约束。挂载物写入 git 的 `info/exclude`，对 git 不可见。

```bash
qg kb attach                 # 挂载当前工作树
qg kb attach ../feature-login # 挂载指定工作树
qg kb status                 # 列出各工作树的挂载状态，不写文件
```

- 特性分支的根 `AGENTS.md` 是 qg 生成文件，不要直接编辑；共享内容写到 `_ai-kb` 基座，本分支内容写到 `AGENTS.local.md`，然后执行 `qg kb attach` 刷新。
- 已存在的用户 `AGENTS.md`、非 qg 管理的 `.dsh`、外部 `.agents/skills` 或 `.qoder/skills` 不会被覆盖，只会告警或报错保留。
- **缺失自动初始化**：全局层（`AGENTS.md`、`skills/`）或项目骨架（`AGENTS.md`、`skills/`、`knowledge/{journal,glossary}.md`）缺失时自动创建；整个知识库被删后，下次 `qg kb attach` 或 `qg wt` 会重建。
- 全局层会接入 `$DSH_HOME`、`$CODEX_HOME` 与共享 `$AGENTS_HOME`：缺失、空文件或悬空链接指向 `_ai-kb`；已有非空用户配置不动。Codex 全局技能按单个技能名补进 `$CODEX_HOME/skills`，不会替换系统内置技能。
- `qg kb status` 会分别显示 Codex、DSH、Qoder 的接入状态，并给出缺失的入口。
- `qg rm` 删除分支时，若分支知识目录从未写入过，会一并清理。

### 快速分支同步（`qg gmt`）

```bash
# 将当前分支合并到 test 分支，推送 test 后回到当前分支
qg gmt test

# 合并推送后触发构建（执行 qg.hjson 中配置的 curl 命令）
qg gmt test -b
```

行为说明：

- 目标分支必须在本地存在；当前分支不能与目标分支相同。
- 当前工作区必须没有未提交变更。
- 如果目标分支已在其他 worktree 中打开，会直接在该 worktree 中执行同步；该 worktree 也必须干净。
- 同步流程：目标分支 `pull` 最新（未设置 upstream 时跳过）→ 合并当前分支 → 推送目标分支 → 回到当前分支。合并或拉取产生冲突时会自动回滚冲突现场，并提示手动处理。
- 目标分支不在 worktree 中时，冲突后会停留在目标分支并提示手动执行 `git merge <源分支>`；目标分支在 worktree 中时，会提示该 worktree 的路径。

#### 构建触发（`-b`）

加 `-b` 后，推送成功会依次执行 `qg.hjson` 中该分支配置的命令（通常是 curl 调用 CI）。
配置文件优先取**当前工作树根目录**；没有时回退到**知识库中该项目基座**的 `projects/<项目>/base/qg.hjson`（默认仓库同级 `_ai-kb`，可用 `AI_KB_ROOT` 覆盖）：

```json
{
  "builds": {
    "test": {
      "curl": [
        "curl -X POST 'https://jenkins.example.com/job/test-build/build?token=xxx'"
      ]
    },
    "release": {
      "curl": [
        "curl -H 'Authorization: Bearer xxx' -d '{\"branch\": \"test\"}' https://ci.example.com/api/build",
        "curl -X POST 'https://hook.example.com/notify'"
      ]
    }
  }
}
```

- `curl` 数组中的命令会**依次执行**；单条失败会明确告知（含命令与错误），并继续执行后续命令。
- 成功判定为**命令退出码 0**；`curl` 命令会自动注入 `-f`，HTTP 4xx/5xx 判为失败（2xx/3xx 算成功）。非 curl 命令仍按退出码判定。
- 每条命令超时 30 秒，超时会终止整个进程组，避免留下孤儿进程。
- 构建触发失败只警告，不影响已完成的合并推送，退出码仍为 0。
- 分支未配置时提示跳过；`qg.hjson` 已加入 `.gitignore`，token 不会入库。
- 两处 qg.hjson 都不存在时视为未配置，只提示跳过，不报错。

`qg.hjson` 采用 HJSON 解析（JSON 超集），标准 JSON 写法完全兼容，同时支持 `'''` 多行字符串——浏览器「复制为 cURL」的内容可以**原样粘贴，无需转义**：

```hjson
{
  builds: {
    main: {
      curl: ['''
      curl --url 'https://ci.example.com/build?token=xxx' \
        -H 'Accept: */*' \
        -b 'SESSION=abc123' \
        -H 'User-Agent: Mozilla/5.0 ...'
      ''']
    }
  }
}
```

### AI 生成 commit message（`qg cmt`，需配置）

```bash
# 生成 commit message（精简模式，只输出，不提交）
qg cmt

# 生成详细模式的 commit message
qg cmt -v

# 编辑确认后提交
qg cmt -e

# 提交并推送
qg cmt -p

# 编辑确认后提交并推送（完整流程）
qg cmt -e -p
```

- 暂存区为空时会自动 `git add .`（从仓库根暂存全部变更）后再生成。
- 不带 `-e` / `-p` 时只打印 message，不创建提交。
- `-e` 会在打印后等待输入：直接回车沿用生成的 message；输入新内容后以空行结束。加了 `-e` 就会提交。
- `-p` 会提交并推送当前分支；推送失败时本地提交已保留，可修复后手动 push。
- 提交 diff 超过 500 行或 64 KiB 时会被截断后再发送给模型，并提示可用 `git diff --cached` 查看完整内容。

### 提交所有修改并推送（`qg p`）

```bash
# 提交信息为 "fix: 登录超时"
qg p "fix: 登录超时"

# 多个参数会以空格拼接为提交信息
qg p fix login timeout
```

- 从仓库根执行 `git add .`，暂存整个仓库的变更。
- 没有未提交变更时报错退出。
- 提交后推送当前分支；分支没有 upstream 时自动 `git push -u origin <分支>`。

### 删除当前分支及工作树（`qg rm`）

```bash
# 删除当前分支；若分支检出于工作树，一并删除工作树
qg rm

# 同时删除远程同名分支（会先要求确认）
qg rm -o

# 强制删除：脏工作树、未合并分支也删
qg rm --force
```

- 固定删除**当前所在分支**，不接受分支名参数。
- 若分支检出于主仓库：先切换到本地默认分支（优先 `main`，其次 `master`）再删除；主仓库有未提交变更时拒绝切换。
- 若分支检出于链接工作树：先 `git worktree remove` 再删分支；工作树有未提交变更时默认拒绝（`--force` 可强制）。若分支因未合并被拒绝，qg 会尝试恢复刚移除的工作树。
- 删除链接工作树后，会同步删除 DSH 中路径对应的 workspace（`qg-dsh-open` 桥接的 `/remove`）。DSH 未运行、插件未安装或没有对应 workspace 时静默跳过；删除失败只警告，本地删除结果不受影响。
- 分支检出于主仓库时只删除分支，不会删除主仓库路径对应的 DSH workspace。
- 本地删除默认使用 `git branch -d`，未合并的分支会被拒绝；`--force` 改用 `-D`。
- `-o` 会先打印待删除的本地/远程分支并等待 `y` 确认；远程删除失败只警告，本地删除结果不受影响。
- 删除工作树后，如果当前终端仍位于被删目录内，请先 `cd` 出来。

### 执行 Go 脚本（`qg run`）

```bash
# 执行 ~/.qg/scripts/deploy.go
qg run deploy

# 执行显式路径的脚本，并透传参数
qg run ./scripts/check.go --env test

# 限制运行时长为 300 秒（选项必须写在脚本名之前）
qg run --timeout 300 deploy

# 用 -- 结束 qg 选项，后续参数全部透传给脚本
qg run -- deploy --timeout 10
```

- 脚本按路径解析：名字含 `/`、`\` 或以 `.go` 结尾时视为显式路径（自动补 `.go`）；否则到脚本目录找 `<名字>.go`。
- 脚本目录默认 `~/.qg/scripts`，可用 `QG_SCRIPTS_DIR` 覆盖。
- 执行方式：先 `go build` 到临时目录，再运行产物——真实透传脚本的 stdout/stderr/stdin 和退出码（`go run` 会把退出码压成 1）。
- `--timeout` 只限制脚本运行阶段，不限制编译；超时会终止整个进程组。默认不限时。
- 选项必须放在脚本名之前；脚本名之后的参数原样透传。

### 查看分支来源（`qg from`）

```bash
# 查看当前分支的来源分支与切出时间
qg from
```

示例输出：

```text
分支: feature/1024
来源: main
切出: 2026-08-10 14:00:00
```

- **来源 / 切出**：基于 reflog 最早一条记录推断（reflog 默认保留 90 天，超出后无法查到）；从初始状态切出的分支来源显示为 `HEAD`。
- Git 无法可靠记录分支的真实父分支，此命令是推断而非权威信息。

## 配置

### `qg cmt` 配置

`qg cmt` 的配置写在 `qg.hjson` 的 `cmt` 段，**不再读取环境变量或 `.env`**。文件查找顺序见下一节（仓库根 → 知识库项目基座）：

```hjson
{
  cmt: {
    apiKey: your-api-key                  // 必填
    apiUrl: https://api.openai.com/v1     // 可选，默认该值
    model: gpt-4o-mini                    // 可选，默认该值
  }
}
```

| 字段 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `cmt.apiKey` | 是 | - | 大模型服务的 API Key |
| `cmt.apiUrl` | 否 | `https://api.openai.com/v1` | API Base URL；使用其他兼容服务时填对应地址 |
| `cmt.model` | 否 | `gpt-4o-mini` | 模型名称 |

- 只需写需要的字段，`apiUrl` / `model` 缺省时用默认值。
- 没有 `qg.hjson`、或其中缺少 `cmt.apiKey` 时，`qg cmt` 直接报错并提示该改哪个文件。
- `qg.hjson` 已加入 `.gitignore`，token 不会入库。

#### Anthropic

`cmt.apiUrl` 中包含 `anthropic` 时，客户端改用 Anthropic Messages API（`/v1/messages`）：

```hjson
{
  cmt: {
    apiKey: sk-ant-...
    apiUrl: https://api.anthropic.com
    model: claude-sonnet-4-5
  }
}
```

其他服务只要兼容 OpenAI 的 `/chat/completions` 协议即可直接使用。注意：Anthropic 兼容代理如果 URL 里不含 `anthropic`，会被当作 OpenAI 协议调用。

### `qg.hjson`（大模型配置 / 构建触发 / 工作树默认起点）

`qg.hjson` 采用 HJSON 解析，支持标准 JSON 写法。该文件已被 `.gitignore` 忽略，适合放 CI token。

查找顺序（`qg cmt` 的大模型配置、`qg gmt -b` 的构建触发、`qg wt` 的默认起点共用）：

1. **当前工作树根目录**的 `qg.hjson`——项目自己的配置，优先。
2. **知识库中该项目基座**的 `<知识库>/projects/<项目>/base/qg.hjson`——工作树里没有配置时回退。
   知识库根默认是仓库（主工作树）同级的 `_ai-kb`，可用 `AI_KB_ROOT` 覆盖；项目键默认取主工作树目录名。
3. 两处都没有 → 视为未配置，构建触发提示跳过。

这样特性分支工作树（`qg wt` 建出来的目录）不必再复制一份带 token 的 `qg.hjson`，直接读主项目的共享配置。

```hjson
{
  // qg cmt 调用大模型所需；apiUrl / model 可省略
  cmt: {
    apiKey: your-api-key
    model: gpt-4o-mini
  }

  // qg gmt <branch> -b 时按分支执行
  builds: {
    test: {
      curl: [
        "curl -X POST 'https://jenkins.example.com/job/test/build?token=xxx'"
      ]
    }
  }

  // qg wt 不带 --from 时的默认起点
  wt: {
    base: release
  }
}
```

- `cmt.apiKey` 是 `qg cmt` 唯一必填项；`apiUrl` / `model` 缺省时用默认值。
- `builds` 的 key 是目标分支名，`curl` 是要执行的命令数组。
- `wt.base` 所在的 `qg.hjson` 格式错误时，`qg wt` 会直接报错，不会静默退回到当前 `HEAD`，避免从错误起点创建分支。
- 文件不存在时：`qg cmt` 报缺少 `cmt.apiKey`，其余功能跳过，不报错。

### 环境变量汇总

| 变量 | 作用 |
|---|---|
| `QG_SCRIPTS_DIR` | `qg run` 的脚本目录，默认 `~/.qg/scripts` |
| `QG_DSH_BRIDGE` | `qg wt -d` 读取的 DSH 桥接文件，默认 `~/.qg/dsh-bridge.json` |
| `QG_QODER_CLI` | `qg wt -q` 使用的 Qoder CLI 路径；缺省时查找 PATH 与常见安装目录 |
| `QG_CODEX_CLI` | `qg wt -c` 使用的 Codex CLI 路径；缺省时查找 PATH、ChatGPT/Codex 应用内置路径 |
| `AI_KB_ROOT` | 知识库根目录，默认仓库同级 `_ai-kb`，其次 `~/.ai-kb` |
| `DSH_HOME` | DSH 配置目录，默认 `~/.dsh`；`qg kb` 把全局 `AGENTS.md` / `skills` 链接到这里 |
| `CODEX_HOME` | Codex 配置目录，默认 `~/.codex`；qg 会接管空的 `AGENTS.md` 并补充全局技能 |
| `AGENTS_HOME` / `DSH_AGENTS_HOME` | 跨 agent 共享配置目录，默认 `~/.agents`；用于共享 `AGENTS.md` 与技能 |

## 开发

```bash
# 安装依赖
go mod download

# 编译到 bin/qg
make build

# 运行测试
make test

# 直接运行（不产出二进制）
make dev

# 交叉编译所有平台
make build-all

# 打包发布（在 bin/ 生成压缩包）
make release
```

### 项目结构

```text
cmd/main.go              程序入口
cmd/cmt/                 命令实现（package cmt）
  root.go                命令树与旗标注册
  cmt.go                 AI 生成 commit message
  gmt.go                 分支同步与构建触发
  from.go                分支来源
  p.go                   暂存、提交并推送
  wt.go                  创建工作树
  run.go                 运行 Go 脚本
  kb.go                 知识库挂载与状态
  rm.go                  删除分支与工作树
internal/git/            Git 命令封装（状态、变更、worktree、分支同步、删除、reflog）
internal/kb/             知识库挂载（符号链接、合成视图、骨架创建）
internal/projectconfig/  qg.hjson 定位与解析（cmt / builds / wt）
internal/build/          构建命令执行
internal/runner/         qg run 的脚本解析、编译与执行
internal/idea/           IntelliJ IDEA 检测与启动
internal/workbuddy/      WorkBuddy 深链接
internal/dsh/            通过 DSH Desktop 桥接创建并打开工作区
internal/qoder/          通过 Qoder CLI / 应用打开工作树
internal/codex/          通过 `codex app` 在 Codex Desktop 中打开工作树
internal/llm/            LLM API 客户端（OpenAI 兼容 / Anthropic）
internal/prompt/         commit message Prompt 构造
internal/proc/           跨平台进程组管理（超时终止子进程树）
pkg/config/              大模型配置值与默认值
dsh-plugin/              qg-dsh-open：DSH Desktop 工作区桥接插件（Host + Client）
```

## 常见问题

**`qg wt` 提示目标目录已存在 / 分支已存在**

工具不会覆盖已有目录或分支。删除目标目录，或为已有分支手动执行 `git worktree add <路径> <分支>`。

**`qg wt -i` 提示未检测到 IntelliJ IDEA**

已创建的工作树不会回滚。安装 IDEA 或 JetBrains Toolbox 后重试，也可以手动用 IDE 打开输出的工作树路径。

**`qg wt -w` 提示无法打开 WorkBuddy**

确认 WorkBuddy 已安装并注册了 `workbuddy://` 协议。工作树仍然创建成功。

**`qg wt -q` 没有自动创建 Qoder 工作区**

优先安装带 CLI 的 Qoder IDE（或把 `qoder` 加入 PATH / 设置 `QG_QODER_CLI`）。仅安装 Qoder Desktop 时 qg 只能唤起应用，桌面版当前不保证响应命令行目录参数，可在 Qoder 中手动打开输出的工作树路径。

**`qg wt -c` 提示未找到 Codex CLI**

安装 Codex Desktop 后确认 `codex --version` 可执行，或把 `QG_CODEX_CLI` 指向 Codex 安装包内的 `codex` 可执行文件。也可在工作树创建后手动执行 `codex app <工作树路径>`。

**`qg wt -d` 提示未检测到 DSH 工作区桥接插件**

说明 `qg-dsh-open` 插件没有安装或没有加载。按上文“DSH 插件”安装 `dsh-plugin/`，然后重启 DSH Desktop；确认 `~/.qg/dsh-bridge.json` 已生成。

**`qg wt -d` 提示桥接令牌已失效**

DSH 宿主重启后会生成新令牌，而 qg 读到了旧文件。重启 DeepSeek Harness 后重试；qg 也会尝试启动并等待新的桥接文件。

**`qg wt -d` 提示桥接返回错误**

通常是工作树路径不存在或不是目录。检查 qg 输出的路径；插件只接受已存在的绝对目录。

**`qg gmt` 合并冲突**

自动合并已回滚。按终端提示进入目标分支或对应 worktree，手动执行 `git merge <源分支>`，解决冲突后 `git add`、`git commit`、`git push`。

**`qg from` 显示来源未知**

reflog 默认只保留 90 天；分支创建时间更早、reflog 被清理或分支是通过其他方式创建时无法推断。

## 支持的模型服务

- Anthropic (Claude)
- OpenAI
- Azure OpenAI
- DeepSeek
- 智谱 AI (GLM)
- 通义千问
- 文心一言
- MiniMax
- 其他所有兼容 OpenAI API 格式的服务

## License

MIT
