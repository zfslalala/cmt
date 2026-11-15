# qg workspace bridge

DSH Desktop 的一个小型桥接插件，让 `qg` 可以在整合版 DSH 中创建工作区并打开工作树。

## 组成

- **Host 半**：在 `127.0.0.1` 的随机端口启动一个只接受本地请求的 HTTP 服务，并把 `{port, token}` 写入 `~/.qg/dsh-bridge.json`（权限 `0600`）。收到 `POST /open` 后调用 `workspaceRegistry` 创建/复用工作区。
- **Client 半**：监听工作区列表。Host 会把工作区标题临时改成 `qg-open:<requestId>:<encodedTitle>`；Client 看到标记后调用 `uiWorkspace.openWorkspace()` 打开工作区，再把标题恢复。

## 安装

在 DSH Desktop 中打开 **设置 → 插件**，选择“添加插件/本地目录”，指向本目录；或由 DSH 的插件管理安装本地目录。安装后需要重启 DSH Desktop（如果 HMR 没有自动加载）。

## 接口

```http
GET  /health
POST /open
POST /remove
Authorization: Bearer <token from ~/.qg/dsh-bridge.json>
Content-Type: application/json

{"path": "/absolute/worktree/path"}
```

`/open` 响应：

```json
{"ok":true,"workspaceId":"...","created":true,"requestId":"..."}
```

`/remove` 响应：

```json
{"ok":true,"removed":true,"workspaceId":"..."}
```

`/remove` 按存储的 workspace 路径匹配；工作树目录已被删除时也能工作。没有对应 workspace 时返回 `{"ok":true,"removed":false}`。

## 安全

- 只监听回环地址。
- 每次宿主启动生成新的随机 token，写入仅当前用户可读的 `0600` 文件。
- 只接受绝对路径，且路径必须是已存在的目录；不会执行请求中的命令。
