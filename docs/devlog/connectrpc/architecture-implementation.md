# ConnectRPC Architecture Implementation

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete the ConnectRPC-focused rewrite of `docs/architecture.md` section `3.1`, unify protocol semantics, and fix the transport-layer examples.

**Architecture:** Use a minimal-change strategy that only updates `3.1.1` through `3.1.4` and leaves `3.2+` intact. The implementation should clarify the Connect/gRPC/gRPC-Web relationship, remove the old “Connect stream equals WebSocket” ambiguity, and keep the examples grounded in `connect-go` usage at the transport boundary.

**Tech Stack:** Markdown, Protocol Buffers, Go, connect-go (documentation examples)

---

### Task 1: Lock the transport scope and acceptance checks

**Files:**
- Modify: `docs/architecture.md:84-347`
- Test: `docs/architecture.md` (manual review)

**Step 1: Write the failed checks**

```text
FAIL conditions:
1) section 3.1 still equates Connect with WebSocket
2) the server example still contains semantic mismatches (for example misleading variable flow or inaccurate explanation)
3) the client example still fails to distinguish Connect from the gRPC compatibility path
```

**Step 2: Run the scope check**

Run: `rg -n "WebSocket|connect-go|WithGRPC|NewAgentServiceHandler" docs/architecture.md`
Expected: the old wording is still present, confirming 3.1 needs rewriting

**Step 3: Apply the transport-layer rewrite**

```markdown
- only rewrite 3.1 (3.1.1~3.1.4)
- leave 3.2+ untouched
- keep the proto interface stable while fixing comments and example semantics
```

**Step 4: Re-run the scope check**

Run: `rg -n "WebSocket（通过 connect-go 的 bidi streaming）|connect.WithGRPC" docs/architecture.md`
Expected: the old misleading wording disappears and the compatibility example appears

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: refine connectrpc transport architecture section"
```

### Task 2: Rewrite 3.1.1 and 3.1.2 around protocol semantics and server boundary

**Files:**
- Modify: `docs/architecture.md:86-315`
- Test: `docs/architecture.md` (manual review)

**Step 1: Write the failed checks**

```text
FAIL conditions:
1) the 3.1 summary still does not explain the Connect/gRPC/gRPC-Web relationship
2) the Chat comments still do not explain that streaming works across Connect and gRPC
3) the SetupServer example still does not show a clear Connect-first service boundary with gRPC compatibility
```

**Step 2: Run the targeted check**

Run: `rg -n "统一传输层|Bidirectional Streaming|SetupServer|h2c" docs/architecture.md`
Expected: the search exposes missing or outdated transport-layer wording

**Step 3: Write the minimal server-boundary implementation**

```go
// SetupServer configures and returns an HTTP server
// with Connect as the primary transport and optional gRPC/gRPC-Web compatibility.
func SetupServer(agentServer *AgentServer) *http.Server {
    mux := http.NewServeMux()
    path, handler := agentv1connect.NewAgentServiceHandler(agentServer)
    mux.Handle(path, handler)
    h2s := &http2.Server{}
    rootHandler := h2c.NewHandler(mux, h2s)
    return &http.Server{Addr: ":8080", Handler: rootHandler}
}
```

**Step 4: Re-run the transport check**

Run: `rg -n "Connect、gRPC 与 gRPC-Web|rootHandler|Bidirectional Streaming，Connect/gRPC 均可用" docs/architecture.md`
Expected: the transport summary, server boundary, and streaming wording all match

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: align connectrpc server semantics and examples"
```

### Task 3: Rewrite 3.1.3 and 3.1.4 around client protocol choices and benefits

**Files:**
- Modify: `docs/architecture.md:317-347`
- Test: `docs/architecture.md` (manual review)

**Step 1: Write the failed checks**

```text
FAIL conditions:
1) the client example still does not distinguish the default Connect client from the gRPC compatibility client
2) the benefits section still uses misleading WebSocket-centered transport language
3) interoperability still does not clearly cover both Connect and standard gRPC clients
```

**Step 2: Run the targeted check**

Run: `rg -n "WebSocket 风格|connectgrpc.NewClient|传输透明|互操作性" docs/architecture.md`
Expected: the old wording is still visible and needs replacement

**Step 3: Write the minimal client-side compatibility example**

```go
grpcClient := agentv1connect.NewAgentServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
    connect.WithGRPC(),
)
```

**Step 4: Re-run the client check**

Run: `rg -n "connect.WithGRPC|传输统一|同时兼容 Connect 客户端与标准 gRPC 客户端" docs/architecture.md`
Expected: the updated protocol and interoperability wording is present

**Step 5: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: update connectrpc client examples and benefits table"
```

### Task 4: Preserve the design record and verify delivery

**Files:**
- Create: `docs/devlog/connectrpc/design.md`
- Test: `docs/devlog/connectrpc/design.md` (content review)

**Step 1: Write the failed checks**

```text
FAIL conditions:
1) no formal design record exists for the ConnectRPC rewrite
2) the goal, scope, and acceptance criteria are not explicitly captured
```

**Step 2: Run the existence check**

Run: `test -f docs/devlog/connectrpc/design.md; echo $?`
Expected: output `1` before the file is created

**Step 3: Write the design record**

```markdown
## Goal and Scope
- only refine 3.1
- normalize ConnectRPC semantics
- leave 3.2+ untouched
```

**Step 4: Re-run the content check**

Run: `rg -n "Goal and Scope|Acceptance Criteria|ConnectRPC" docs/devlog/connectrpc/design.md`
Expected: the key sections are present and the design is traceable

**Step 5: Commit**

```bash
git add docs/devlog/connectrpc/design.md
git commit -m "docs: add connectrpc architecture redesign record"
```
