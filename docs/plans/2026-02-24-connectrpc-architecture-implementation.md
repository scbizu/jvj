# ConnectRPC Architecture Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 完成 `docs/architecture.md` 的 ConnectRPC 设计改造（仅 3.1 小节），统一协议语义并修正示例代码表达。

**Architecture:** 采用最小改动策略，仅修改 3.1.1~3.1.4，保持服务接口不扩展、不影响 3.2+ 模块。通过统一 Connect/gRPC/gRPC-Web 的术语边界，修正文档中“WebSocket 等同 Connect 流”的歧义。示例代码聚焦 connect-go 官方使用模式与 gRPC 兼容选项。

**Tech Stack:** Markdown, Protocol Buffers, Go, connect-go（文档示例）

---

### Task 1: 锁定改造范围与验收标准

**Files:**
- Modify: `docs/architecture.md:84-347`
- Test: `docs/architecture.md`（人工校验）

**Step 1: 写下失败校验（当前不满足）**

```text
FAIL 条件：
1) 3.1 中出现 Connect 与 WebSocket 等同表述
2) 服务端示例存在语义不一致（如变量遮蔽、说明不准确）
3) 客户端示例未清晰区分 Connect 与 gRPC 兼容调用
```

**Step 2: 运行校验并确认失败**

Run: `rg -n "WebSocket|connect-go|WithGRPC|NewAgentServiceHandler" docs/architecture.md`
Expected: 命中旧描述，显示需要重写的段落。

**Step 3: 最小实现（文档级）**

```markdown
- 仅重写 3.1（3.1.1~3.1.4）
- 不改动 3.2+ 章节
- 保留 proto 接口，调整注释与示例语义
```

**Step 4: 再次校验**

Run: `rg -n "WebSocket（通过 connect-go 的 bidi streaming）|connect.WithGRPC" docs/architecture.md`
Expected: 旧错误表述消失，新增 `connect.WithGRPC` 命中。

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: refine connectrpc transport architecture section"
```

### Task 2: 重写 3.1.1 与 3.1.2（协议语义 + 服务端示例）

**Files:**
- Modify: `docs/architecture.md:86-315`
- Test: `docs/architecture.md`（人工校验）

**Step 1: 写下失败校验**

```text
FAIL 条件：
1) 3.1 总述未明确 Connect/gRPC/gRPC-Web 关系
2) Chat 注释未说明流式在 Connect/gRPC 均可用
3) SetupServer 示例未体现“Connect 主路径 + gRPC 兼容语义”
```

**Step 2: 运行校验并确认失败**

Run: `rg -n "统一传输层|Bidirectional Streaming|SetupServer|h2c" docs/architecture.md`
Expected: 发现旧描述或缺失说明。

**Step 3: 写最小实现**

```go
// SetupServer 配置并返回 HTTP 服务器（支持 Connect，可选兼容 gRPC/gRPC-Web）
func SetupServer(agentServer *AgentServer) *http.Server {
    mux := http.NewServeMux()
    path, handler := agentv1connect.NewAgentServiceHandler(agentServer)
    mux.Handle(path, handler)
    h2s := &http2.Server{}
    rootHandler := h2c.NewHandler(mux, h2s)
    return &http.Server{Addr: ":8080", Handler: rootHandler}
}
```

**Step 4: 再次校验**

Run: `rg -n "Connect、gRPC 与 gRPC-Web|rootHandler|Bidirectional Streaming，Connect/gRPC 均可用" docs/architecture.md`
Expected: 三类关键表述均命中。

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: align connectrpc server semantics and examples"
```

### Task 3: 重写 3.1.3 与 3.1.4（客户端示例 + 优势）

**Files:**
- Modify: `docs/architecture.md:317-347`
- Test: `docs/architecture.md`（人工校验）

**Step 1: 写下失败校验**

```text
FAIL 条件：
1) 客户端示例未区分 Connect 默认调用与 gRPC 兼容调用
2) 优势表仍含“WebSocket”误导性传输描述
3) 互操作性描述未覆盖 Connect 与标准 gRPC 客户端并存
```

**Step 2: 运行校验并确认失败**

Run: `rg -n "WebSocket 风格|connectgrpc.NewClient|传输透明|互操作性" docs/architecture.md`
Expected: 命中旧描述，确认需修改。

**Step 3: 写最小实现**

```go
grpcClient := agentv1connect.NewAgentServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
    connect.WithGRPC(),
)
```

**Step 4: 再次校验**

Run: `rg -n "connect.WithGRPC|传输统一|同时兼容 Connect 客户端与标准 gRPC 客户端" docs/architecture.md`
Expected: 新术语与示例全部命中。

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: update connectrpc client examples and benefits table"
```

### Task 4: 设计文档沉淀与交付核验

**Files:**
- Create: `docs/plans/2026-02-24-connectrpc-design.md`
- Test: `docs/plans/2026-02-24-connectrpc-design.md`（内容核对）

**Step 1: 写下失败校验**

```text
FAIL 条件：
1) 缺少正式设计记录
2) 改造目标/范围/验收标准未显式沉淀
```

**Step 2: 运行校验并确认失败**

Run: `test -f docs/plans/2026-02-24-connectrpc-design.md; echo $?`
Expected: 输出 `1`（文件不存在）。

**Step 3: 写最小实现**

```markdown
## 目标与范围
- 仅改造 3.1
- 统一 Connect 语义
- 不改动 3.2+
```

**Step 4: 再次校验**

Run: `rg -n "目标与范围|验收标准|ConnectRPC" docs/plans/2026-02-24-connectrpc-design.md`
Expected: 关键段落命中，设计文档可追溯。

**Step 5: Commit**

```bash
git add docs/plans/2026-02-24-connectrpc-design.md
git commit -m "docs: add connectrpc architecture redesign record"
```
