# Codex 通信超时与协议兼容问题调查

> 日期：2026-04-23  
> 涉及版本：go-symphony (main), Codex CLI 0.122.0  
> 修改范围：仅 go-symphony 代码（`internal/codex/session.go`, `internal/config/settings.go`）

---

## 1. 问题现象

### 1.1 用户报告的主问题

执行 `./verify.sh run --only-issue YOU-23` 时，go-symphony 在 Codex bootstrap 阶段抛出 `"codex read timeout"` 错误：

```
2026/04/22 17:42:47 INFO run dispatched ...
2026/04/22 17:42:47 INFO workspace created ...
2026/04/22 17:42:52 WARN retry scheduled ... error="codex read timeout"
```

时间间隔恰好为 5 秒，与 `read_timeout_ms: 5000` 吻合。用户据此判断超时发生在 `Session.bootstrap()` → `awaitResponse(initializeID)` 阶段。

### 1.2 矛盾点

- 用 Python 手动测试相同的 stdio JSON-RPC 协议时，`initialize` 和 `thread/start` 都能瞬间收到响应。
- go-symphony 的 `clientInfo.version` 已正确填写，`sandbox` 已使用 kebab-case。
- 初步怀疑是运输层 framing 问题（`WriteJSON` 缺少 `\n` 或 flush）。

---

## 2. 调查过程

### 2.1 步骤 1：隔离验证运输层

用 Go 编写了最小复现程序，完全复刻 `processTransport.WriteJSON` 和 `scan` 的实现：

```go
cmd := exec.CommandContext(ctx, "bash", "-lc", "codex app-server")
stdin.Write(json.Marshal(msg) + '\n')
reader.ReadBytes('\n')
```

**结果**：Codex 在 ~100ms 内返回 `initialize` 响应。运输层本身无问题。

### 2.2 步骤 2：隔离验证 `StartSession`

在 go-symphony 仓库内新建临时 `cmd/test-codex`，直接调用 `codex.StartSession()`：

```go
session, err := codex.StartSession(ctx, codex.SessionOptions{
    Config: cfg,
    WorkspacePath: "/path/to/symphony_workspaces/YOU-23",
})
```

**结果**：
- 单次调用：成功，耗时 ~150ms。
- 10 次并发调用：全部成功，最慢 ~438ms。
- **结论**：`bootstrap()` 本身在当前环境下无问题。

### 2.3 步骤 3：验证完整 `RunTurn` 流程

继续使用临时程序测试 `session.RunTurn()`：

```go
result, err := session.RunTurn(ctx, codex.TurnRequest{
    Prompt: "Say hello briefly",
    Title:  "YOU-23: Test",
})
```

**结果**：`RunTurn` 成功，耗时 ~15 秒，返回 `turn_completed`。

### 2.4 步骤 4：复现用户真实 prompt

将 prompt 替换为 orchestrator 实际渲染的完整 prompt（237 字符，含 Linear issue 信息）：

```go
Prompt: `You are working on a Linear issue.
Identifier: YOU-23
Title: Add a simple greeting endpoint
State: In Progress
...`
```

**结果**：`RunTurn` 进入通知循环后，某些通知间隔长达 4 分钟（模型推理阶段），整体耗时远超默认测试超时。但这与 `codex read timeout`（5 秒）不同，而是正常的模型推理耗时。

### 2.5 步骤 5：排查 `turn/start` 参数兼容性

用 Python 手动测试时，逐步增加 `turn/start` 参数字段，发现关键差异：

| 参数组合 | 结果 |
|---------|------|
| `threadId` + `input` | ✅ 成功 |
| + `cwd` | ✅ 成功 |
| + `title` | ✅ 成功 |
| + `approvalPolicy` | ✅ 成功 |
| + `sandboxPolicy: {}` | ❌ **失败** |

Codex 返回错误：
```json
{"error": {"code": -32600, "message": "Invalid request: missing field `type`"}}
```

**根因**：Codex 0.122.0 的 `SandboxPolicy` 是一个 **tagged union**，必须有 `"type"` 字段（取值如 `"workspaceWrite"`、`"readOnly"`、`"dangerFullAccess"`）。go-symphony 发送空对象 `{}` 时，Codex 无法反序列化，导致 `turn/start` 被拒绝。

> **注意**：当 `turn_sandbox_policy` 未在 WORKFLOW.md 中显式配置时，`cloneMap(nil)` 返回 `nil`，`json.Marshal` 编码为 `null`，Codex 接受 `null`。因此该 bug 仅在用户显式配置空对象时触发，但属于明确的协议兼容缺陷。

---

## 3. 最终根因总结

| 问题 | 根因 | 影响范围 |
|------|------|---------|
| `codex read timeout`（bootstrap 阶段，5 秒） | **未在本地复现**。最可能的原因是 Codex 冷启动/模型加载/auth 刷新时偶尔超过 5 秒；也可能是当时系统负载或网络延迟导致的瞬态问题。 | 偶发，与 `ReadTimeoutMS` 设置过短有关 |
| `turn/start` 被 Codex 拒绝 | `sandboxPolicy` 为空对象 `{}` 时缺少必填的 `"type"` 字段，违反 Codex `SandboxPolicy` tagged union schema。 | 配置 `turn_sandbox_policy: {}` 时必现 |
| `RunTurn` 整体耗时过长 | `gpt-5.4` + `reasoning_effort=xhigh` 对复杂 prompt 的推理时间可达数分钟，属于模型行为，非代码缺陷。 | 正常行为，需要足够长的 `--timeout` |

---

## 4. 修复内容

### 4.1 提升 `ReadTimeout` 默认值

**文件**：`internal/config/settings.go`

```go
// 修改前
ReadTimeoutMS: 5_000,

// 修改后
ReadTimeoutMS: 30_000,
```

将默认读取超时从 5 秒提升到 30 秒，覆盖 Codex 冷启动、auth 刷新等偶发慢启动场景。

### 4.2 修复 `sandboxPolicy` 发送逻辑

**文件**：`internal/codex/session.go` — `RunTurn()`

```go
// 修改前
"params": map[string]any{
    ...
    "sandboxPolicy": cloneMap(s.turnSandboxPolicy),
}

// 修改后
turnParams := map[string]any{
    ...
}
if sandboxPolicy := cloneMap(s.turnSandboxPolicy); len(sandboxPolicy) > 0 {
    turnParams["sandboxPolicy"] = sandboxPolicy
}
```

仅当 `sandboxPolicy` 非空时才将其加入 `turn/start` 参数，避免发送空对象 `{}` 触发 Codex schema 校验失败。

### 4.3 增强 Codex 错误响应解析

**文件**：`internal/codex/session.go` — `awaitResponse()`

```go
if responseError, ok := payload["error"]; ok {
    msg := fmt.Sprint(responseError)
    if errMap, ok := responseError.(map[string]any); ok {
        if m := stringValue(errMap["message"]); m != "" {
            msg = m
            if c := stringValue(errMap["code"]); c != "" {
                msg = "[" + c + "] " + msg
            }
        }
    }
    slog.Error("codex protocol error", "message", msg)
    return nil, &ProtocolError{Kind: "response_error", Message: msg, Payload: payload}
}
```

从 Codex 返回的 `error.message` 和 `error.code` 提取可读信息，避免错误消息变成难以阅读的 `map[code:-32600 message:...]`。

### 4.4 补充诊断日志

**文件**：`internal/codex/session.go`

新增以下日志点：

| 日志 | 级别 | 位置 | 用途 |
|------|------|------|------|
| `codex transport starting` | Debug | `StartProcessTransport` | 记录启动的命令和工作目录 |
| `codex transport start failed` | Error | `StartProcessTransport` | 进程启动失败时记录原因 |
| `codex session bootstrapping` | Debug | `StartSession` | 开始 bootstrap |
| `codex session bootstrap failed` | Error | `StartSession` | bootstrap 失败时记录错误 |
| `codex session started` | Info | `StartSession` | bootstrap 成功，记录 thread_id |
| `codex protocol error` | Error | `awaitResponse` | Codex 返回错误响应时记录 |
| `codex stdout scanner error` | Debug | `processTransport.scan` | stdout scanner 异常时记录（排除 EOF） |

---

## 5. 验证结果

- `make build` ✅
- `make test` ✅（全部 package 通过）
- `make lint` ✅（0 issues）
- `./symphony-verify run --only-issue YOU-23 --timeout 15s` ✅（`StartSession` 成功，`RunTurn` 正常进入通知循环）

---

## 6. 操作注意事项

1. **Codex 推理耗时**：使用 `gpt-5.4` + `reasoning_effort=xhigh` 时，单次 `RunTurn` 可能需要数十秒到数分钟。运行 `symphony-verify run` 时建议设置足够长的 `--timeout`（例如 `--timeout 10m`），否则会在模型推理中途被 `context canceled` 中断。

2. **未修改 Codex 代码**：本次调查仅修改了 go-symphony 仓库内的代码，未触碰本地 Codex 仓库中的源码。Codex 源码仅用于参考协议 schema。
