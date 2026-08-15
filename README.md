# QG - Git Commit Message 自动生成工具

基于大语言模型的 Git Commit Message 自动生成工具，支持所有 OpenAI 协议兼容的大模型服务。

## 功能特性

- 🚀 自动生成符合 Conventional Commits 规范的提交信息
- 🎯 支持精简模式和详细模式
- ✏️ 支持编辑确认后再提交
- 📤 支持一键提交并推送到远程
- 🔧 支持所有 OpenAI 协议兼容的大模型服务
- ⚡ 单二进制文件，零依赖

## 命令结构

```text
qg
├── cmt            生成 commit message(-v / -e / -p)
└── gmt <branch>   将当前分支同步到目标分支并推送
```

直接运行 `qg` 会显示命令帮助。

## 安装

### macOS / Linux

**方式一：下载预编译版本**

从 [Releases](https://github.com/zfslalala/cmt/releases) 下载对应平台压缩包（如 `qg-darwin-arm64.tar.gz`），解压后安装：

```bash
tar -xzf qg-darwin-arm64.tar.gz
chmod +x qg-darwin-arm64
sudo mv qg-darwin-arm64 /usr/local/bin/qg
```

**方式二：源码编译**

```bash
git clone https://github.com/zfslalala/cmt.git
cd cmt
make install
```

**方式三：Go install**

```bash
go install github.com/zfslalala/cmt/cmd/cmt@latest
# 安装后重命名为 qg
mv $(go env GOPATH)/bin/cmt $(go env GOPATH)/bin/qg
```

---

### Windows

**方式一：下载预编译版本**

1. 从 [Releases](https://github.com/zfslalala/cmt/releases) 下载 `qg-windows-amd64.exe`
2. 改名为 `qg.exe`
3. 放到 `C:\Windows\System32\` 或其他 PATH 目录

**方式二：源码编译**

```powershell
git clone https://github.com/zfslalala/cmt.git
cd cmt
go build -ldflags="-s -w" -o qg.exe ./cmd/cmt
# 把 qg.exe 移到 PATH 目录
```

---

### 验证安装

```bash
qg --help
```

## 配置

### 配置项

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `CMT_API_KEY` | 是 | - | 大模型服务的 API Key |
| `CMT_API_URL` | 否 | `https://api.openai.com/v1` | API Base URL，使用其他兼容服务时填对应地址 |
| `CMT_MODEL` | 否 | `gpt-4o-mini` | 模型名称 |

### 环境变量

```bash
export CMT_API_KEY="your-api-key"
export CMT_API_URL="https://api.openai.com/v1"
export CMT_MODEL="gpt-4o-mini"
```

### .env 文件

在项目目录或用户主目录创建 `.cmt.env` 文件（系统环境变量优先于文件）：

```bash
CMT_API_KEY=your-api-key
CMT_API_URL=https://api.openai.com/v1
CMT_MODEL=gpt-4o-mini
```

## 使用

### 生成 commit message

```bash
# 生成 commit message（精简模式）
qg cmt

# 生成详细模式的 commit message
qg cmt -v

# 编辑确认后再提交
qg cmt -e

# 一键提交并推送
qg cmt -p

# 编辑确认后提交并推送（完整流程）
qg cmt -e -p
```

暂存区为空时会自动暂存所有变更后再生成。

### 分支同步

```bash
# 将当前分支合并到 test 分支，推送 test 后回到当前分支
qg gmt test
```

- 如果目标分支已在其他 worktree 中打开，会直接在该 worktree 中合并并推送。
- 同步前要求当前工作区没有未提交变更；如果目标分支所在 worktree 有未提交变更，也会停止执行。
- 合并冲突时会自动回滚失败的自动合并，并提示你手动合并：
  - 目标分支不在 worktree 中时，会切到目标分支并停留在那里，提示手动执行 `git merge <源分支>`。
  - 目标分支已在 worktree 中时，会提示该 worktree 的路径，进入该目录手动执行合并。

```bash
# 安装到用户目录，避免 sudo
make install PREFIX=$HOME/.local
```

## 开发

```bash
# 安装依赖
go mod download

# 编译
make build

# 运行测试
make test

# 交叉编译
make build-all

# 打包发布
make release
```

### 项目结构

```text
cmd/cmt/         命令入口与子命令实现(root / cmt / gmt)
internal/git     Git 命令封装(状态、变更、worktree、分支同步)
internal/llm     LLM API 客户端(OpenAI 兼容 / Anthropic)
internal/prompt  Prompt 构造
pkg/config       配置加载
```

## 支持的模型服务

- Anthropic (Claude)
- OpenAI
- Azure OpenAI
- 智谱 AI (GLM)
- 通义千问
- 文心一言
- MiniMax
- 其他所有兼容 OpenAI API 格式的服务

## License

MIT
