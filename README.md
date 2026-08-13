# CMT - Git Commit Message 自动生成工具

基于大语言模型的 Git Commit Message 自动生成工具，支持所有 OpenAI 协议兼容的大模型服务。

## 功能特性

- 🚀 自动生成符合 Conventional Commits 规范的提交信息
- 🎯 支持精简模式和详细模式
- ✏️ 支持编辑确认后再提交
- 📤 支持一键提交并推送到远程
- 🔧 支持所有 OpenAI 协议兼容的大模型服务
- ⚡ 单二进制文件，零依赖

## 安装

### macOS / Linux

**方式一：下载预编译版本**

从 [Releases](https://github.com/z/cmt/releases) 下载对应版本。

```bash
# 解压并安装
chmod +x cmt
sudo mv cmt /usr/local/bin/
```

**方式二：源码编译**

```bash
git clone https://github.com/z/cmt.git
cd cmt
make install
```

**方式三：Go install**

```bash
go install github.com/z/cmt@latest
```

---

### Windows

**方式一：下载预编译版本**

1. 从 [Releases](https://github.com/z/cmt/releases) 下载 `cmt-windows-amd64.exe`
2. 改名为 `cmt.exe`
3. 放到 `C:\Windows\System32\` 或其他 PATH 目录

**方式二：源码编译**

```powershell
git clone https://github.com/z/cmt.git
cd cmt
go build -ldflags="-s -w" -o cmt.exe ./cmd/cmt
# 把 cmt.exe 移到 PATH 目录
```

**方式三：Scoop**

```powershell
scoop install cmt
```

---

### 验证安装

```bash
cmt --help
```

## 配置

### 环境变量

```bash
# 必需
export CMT_API_KEY="your-api-key"
export CMT_API_URL="https://api.openai.com/v1"
```

### .env 文件

在项目目录或用户主目录创建 `.cmt.env` 文件：

```bash
CMT_API_KEY=your-api-key
CMT_API_URL=https://api.openai.com/v1
CMT_MODEL=gpt-4o-mini
```

## 使用

```bash
# 生成 commit message（精简模式）
cmt

# 生成详细模式的 commit message
cmt -v

# 编辑确认后再提交
cmt -e

# 一键提交并推送
cmt -p

# 编辑确认后提交并推送（完整流程）
cmt -e -p
```

### 分支同步

```bash
# 将当前分支合并到 test 分支，推送 test 后回到当前分支
gmt test

# 如果 test 分支已在其他 worktree 中打开，会直接在该 worktree 中合并并推送
gmt test
gmt <target-branch>
```

同步前要求当前工作区没有未提交变更；如果目标分支所在 worktree 有未提交变更，也会停止执行。

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
