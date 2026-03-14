# Agent Runtime 架构设计文档

## 1. 概述

本文档描述一个基于 Go 实现的 Agent Runtime 服务架构。该服务以独立进程形式部署，支持通过多种协议（WebSocket、Discord Bot、Telegram Bot 等）与外部客户端通信。

### 1.1 设计目标

- **模块化**：各组件职责清晰，可独立测试和替换
- **可扩展**：支持多种通信协议和模型后端
- **可观测**：全链路日志和状态追踪
- **高性能**：Go 原生并发模型，低延迟响应

### 1.2 核心概念

| 概念 | 说明 |
|------|------|
| Session | 一次完整的对话上下文，对应一个用户连接 |
| Turn | 单次交互轮次（用户输入 → Agent 处理 → 响应输出）|
| Tape | 只追加的会话历史记录，用于回放和审计 |
| Anchor | 会话阶段标记点，用于状态恢复和上下文切换 |
| Tool | 可执行的能力单元（函数调用、外部 API 等）|
| Skill | 启动时加载的高层语义能力模块，可编排 Tool、Tape 与运行时状态 |
| ChannelAdapter | 外部平台消息适配器，负责平台协议与内部事件格式转换 |
| BusEvent | Message Bus 内部标准事件（message/command/callback） |

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Agent Runtime Service                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              ConnectRPC Transport Layer                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │   gRPC      │  │  WebSocket  │  │  HTTP/REST      │   │  │
│  │  │  (HTTP/2)   │  │(Bidir Stream│  │  (connect-go)   │   │  │
│  │  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘   │  │
│  │         └─────────────────┴──────────────────┘             │  │
│  │                         │                                  │  │
│  │              ┌──────────┴──────────┐                       │  │
│  │              │   AgentService      │                       │  │
│  │              │   (Connect Handler) │                       │  │
│  │              └──────────┬──────────┘                       │  │
│  └─────────────────────────┼──────────────────────────────────┘  │
│                            │                                     │
│  ┌─────────────────────────┼───────────────────────────────────┐ │
│  │              ┌──────────┴──────────┐                        │ │
│  │              │   Session Manager   │                        │ │
│  │              │   (Connection Hub)  │                        │ │
│  │              └──────────┬──────────┘                        │ │
│  │                         │                                   │ │
│  │  ┌─────────────┐    ┌───┴───────────┐    ┌─────────────┐   │ │
│  │  │   Router    │───▶│  AgentLoop    │───▶│Model Runner │   │ │
│  │  │             │◀───│               │◀───│             │   │ │
│  │  └─────────────┘    └───────┬───────┘    └─────────────┘   │ │
│  │                             │                               │ │
│  │                             ▼                               │ │
│  │  ┌─────────────┐    ┌───────────────┐    ┌─────────────┐   │ │
│  │  │  Tool Exec  │◀───│  Tape Service │───▶│   Store     │   │ │
│  │  │   Engine    │    │               │    │  (Badger)   │   │ │
│  │  └─────────────┘    └───────────────┘    └─────────────┘   │ │
│  │                                                             │ │
│  │                    ┌─────────────┐                          │ │
│  │                    │  LLM Client │                          │ │
│  │                    └─────────────┘                          │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                          Core Engine                             │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                 Message Bus Ingress (Primary)                    │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │ Discord Bot │───▶│ Bus Router  │───▶│  AgentService       │  │
│  └─────────────┘    │ (normalize) │    │  (Connect Handler)  │  │
│  ┌─────────────┐    └─────────────┘    └─────────────────────┘  │
│  │Telegram Bot │───────────────────────────────────────────────▶│
│  └─────────────┘                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 模块详细设计

### 3.1 Transport Layer（传输层 - ConnectRPC）

使用 [ConnectRPC](https://connectrpc.com/) 作为统一传输层，通过单一 Protocol Buffer 定义同时支持 Connect、gRPC 与 gRPC-Web。底层统一基于 HTTP 语义（HTTP/1.1 或 HTTP/2），流式能力由 Connect 协议提供。

#### 3.1.1 Protocol Buffer 定义

```protobuf
// api/proto/agent/v1/agent.proto
syntax = "proto3";
package agent.v1;

service AgentService {
  // 创建会话
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  
  // 发送消息（Unary）
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  
  // 流式对话（Bidirectional Streaming，Connect/gRPC 均可用）
  rpc Chat(stream ChatRequest) returns (stream ChatResponse);
  
  // 获取会话历史
  rpc GetSessionHistory(GetSessionHistoryRequest) returns (GetSessionHistoryResponse);
  
  // 关闭会话
  rpc CloseSession(CloseSessionRequest) returns (CloseSessionResponse);
}

message CreateSessionRequest {
  string user_id = 1;
  map<string, string> metadata = 2;
}

message CreateSessionResponse {
  string session_id = 1;
  string created_at = 2;
}

message SendMessageRequest {
  string session_id = 1;
  string content = 2;
  MessageType type = 3;
}

message SendMessageResponse {
  string message_id = 1;
  string content = 2;
  bool is_final = 3;
}

// 流式消息类型
message ChatRequest {
  oneof payload {
    StartChat start = 1;
    UserMessage message = 2;
    CommandInput command = 3;
  }
}

message ChatResponse {
  oneof payload {
    StreamChunk chunk = 1;
    ToolCall tool_call = 2;
    ToolResult tool_result = 3;
    Error error = 4;
    Anchor anchor = 5;
    Done done = 6;
  }
}

message StreamChunk {
  string content = 1;
  int32 index = 2;
}

message ToolCall {
  string tool_name = 1;
  string tool_id = 2;
  bytes params = 3;  // JSON
}

message ToolResult {
  string tool_id = 1;
  bytes result = 2;  // JSON
  bool success = 3;
}

message Error {
  string code = 1;
  string message = 2;
}

message Done {
  string summary = 1;
  repeated string next_steps = 2;
}

enum MessageType {
  MESSAGE_TYPE_UNSPECIFIED = 0;
  MESSAGE_TYPE_TEXT = 1;
  MESSAGE_TYPE_COMMAND = 2;
}
```

#### 3.1.2 ConnectRPC 服务端实现

```go
// internal/server/server.go
package server

import (
    "context"
    "errors"
    "io"
    "net/http"
    "time"
    
    "connectrpc.com/connect"
    "connectrpc.com/grpchealth"
    "connectrpc.com/grpcreflect"
    "golang.org/x/net/http2"
    "golang.org/x/net/http2/h2c"
    
    agentv1 "github.com/yourorg/agent-runtime/gen/agent/v1"
    "github.com/yourorg/agent-runtime/gen/agent/v1/agentv1connect"
)

type AgentServer struct {
    sessionMgr *session.Manager
    router     *core.Router
    tapeStore  tape.Store
}

func NewAgentServer(sm *session.Manager, router *core.Router, store tape.Store) *AgentServer {
    return &AgentServer{
        sessionMgr: sm,
        router:     router,
        tapeStore:  store,
    }
}

// 实现 AgentService 接口
func (s *AgentServer) CreateSession(
    ctx context.Context,
    req *connect.Request[agentv1.CreateSessionRequest],
) (*connect.Response[agentv1.CreateSessionResponse], error) {
    sess, err := s.sessionMgr.Create(ctx, req.Msg.UserId, req.Msg.Metadata)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&agentv1.CreateSessionResponse{
        SessionId: sess.ID,
        CreatedAt: sess.CreatedAt.Format(time.RFC3339),
    }), nil
}

// Chat 为双向流式接口，承载会话初始化与增量消息交互
func (s *AgentServer) Chat(
    ctx context.Context,
    stream *connect.BidiStream[agentv1.ChatRequest, agentv1.ChatResponse],
) error {
    // 处理流式会话
    var currentSession *session.Session
    
    for {
        req, err := stream.Receive()
        if err != nil {
            if errors.Is(err, io.EOF) {
                return nil
            }
            return err
        }
        
        switch p := req.Payload.(type) {
        case *agentv1.ChatRequest_Start:
            // 初始化或恢复会话
            currentSession, err = s.sessionMgr.GetOrCreate(ctx, p.Start.SessionId)
            if err != nil {
                return stream.Send(&agentv1.ChatResponse{
                    Payload: &agentv1.ChatResponse_Error{
                        Error: &agentv1.Error{
                            Code:    "SESSION_ERROR",
                            Message: err.Error(),
                        },
                    },
                })
            }
            
        case *agentv1.ChatRequest_Message:
            // 处理用户消息，启动 AgentLoop
            if currentSession == nil {
                return connect.NewError(connect.CodeFailedPrecondition, 
                    errors.New("session not initialized"))
            }
            
            // 发送给 AgentLoop 处理，流式返回结果
            resultCh := s.router.HandleStreaming(ctx, currentSession, p.Message.Content)
            
            for chunk := range resultCh {
                if err := stream.Send(chunk); err != nil {
                    return err
                }
            }
        }
    }
}

// SetupServer 配置并返回 HTTP 服务器（支持 Connect，可选兼容 gRPC/gRPC-Web）
func SetupServer(agentServer *AgentServer) *http.Server {
    mux := http.NewServeMux()
    
    // AgentService（Connect 协议）
    path, handler := agentv1connect.NewAgentServiceHandler(agentServer)
    mux.Handle(path, handler)
    
    // 健康检查（供探活与服务发现）
    mux.Handle(grpchealth.NewHandler(
        grpchealth.NewStaticChecker(agentv1connect.AgentServiceName),
    ))
    
    // Reflection（开发/调试）
    mux.Handle(grpcreflect.NewHandlerV1(
        grpcreflect.NewStaticReflector(agentv1connect.AgentServiceName),
    ))
    
    // h2c：允许明文 HTTP/2，便于本地或内网 gRPC 兼容访问
    h2s := &http2.Server{}
    rootHandler := h2c.NewHandler(mux, h2s)
    
    return &http.Server{
        Addr:    ":8080",
        Handler: rootHandler,
    }
}
```

#### 3.1.3 客户端支持

```go
// Connect 客户端（默认 Connect 协议，可走 HTTP/1.1 或 HTTP/2）
client := agentv1connect.NewAgentServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
)

// gRPC 兼容客户端（同一套 proto 与生成代码）
grpcClient := agentv1connect.NewAgentServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
    connect.WithGRPC(),
)

// 双向流式调用
stream := client.Chat(context.Background())
stream.Send(&agentv1.ChatRequest{...})
for stream.Receive() { ... }
```

#### 3.1.4 优势

| 特性 | 说明 |
|------|------|
| **单一协议定义** | 一个 `.proto` 文件，多端共享 |
| **传输统一** | Connect/gRPC/gRPC-Web 统一映射到 HTTP 语义 |
| **类型安全** | 编译期类型检查，无反射开销 |
| **流式支持** | 原生支持 Server/Client/Bidi Streaming（基于 HTTP） |
| **互操作性** | 同时兼容 Connect 客户端与标准 gRPC 客户端 |

### 3.2 Message Bus Ingress（消息总线接入层）

负责承接 Discord/Telegram 平台事件，并将其标准化为内部 `BusEvent` 后路由到核心引擎。

```go
type BusEventType string

const (
    BusEventMessage  BusEventType = "message"   // 普通消息
    BusEventCommand  BusEventType = "command"   // 命令请求
    BusEventCallback BusEventType = "callback"  // 按钮/交互回调
    BusEventSystem   BusEventType = "system"    // 系统准备动作（如依赖安装）
)

type BusEvent struct {
    ID            string
    Type          BusEventType
    Platform      string // discord | telegram
    SessionID     string
    UserID        string
    Content       string
    CorrelationID string
}
```

**Command Policy（禁止直接 eval）**
- 不执行原始 shell 字符串，不使用 `sh -c` 作为主路径
- 命令流程：`BusEvent(command)` → `Router.Parse` → `ToolPlanner` → `ScriptBuilder` → `Sandbox Script Executor`
- 执行约束：固定 cwd、非 root、超时、资源配额、输出大小上限、审计必填
- 命令执行类 tool call 不直接下发 shell 字符串，而是先分析目标并生成一次性临时脚本再执行

**Dependency Install Policy（黑名单优先）**
- 提供 `deps.install` 系统事件，作为“系统准备动作”，与用户命令执行分流
- 允许常见依赖安装流程（go/npm/pip），但禁止高风险模式：
  - `curl|bash`、远程脚本直执行
  - `sudo`、全局安装（如 `npm -g`）
  - 未经约束的自定义 shell 拼接参数
- 失败必须显式回传，不允许静默降级

### 3.3 Session Manager（会话管理器）

当前运行时按 **single session** 模式设计：单个 runtime 实例在任意时刻只维护一个活动会话，并为该会话提供恢复、重连和状态持久化能力。

```go
type SessionManager struct {
    mu      sync.RWMutex
    current *Session
    store   SessionStore
}

type Session struct {
    ID        string
    Conn      Conn
    Tape      *Tape
    State     SessionState
    Attached  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**职责：**
- 创建或恢复当前会话
- 保证同一时刻只有一个活动流附着到当前会话
- 心跳检测、超时清理和会话状态持久化
- 为当前会话维护唯一的 Tape 引用

### 3.4 Router（路由层）

统一处理输入解析和命令分发。

```go
type Router struct {
    commands map[string]CommandHandler
    tools    *ToolRegistry
}

func (r *Router) Route(ctx context.Context, input string, tape *Tape) (*RouteResult, error)
```

**设计原则（参考 bub）：**
- 用户输入和 Assistant 输出走**同一路由规则**
- 命令前缀 `,` 标识需要直接执行
- 失败时生成 `<command error="...">` 块给模型上下文

### 3.5 Agent Loop（代理循环）

编排单次交互的完整流程。

```go
type AgentLoop struct {
    router      *Router
    modelRunner *ModelRunner
    tape        *Tape
    maxSteps    int
}

func (al *AgentLoop) Run(ctx context.Context, userInput string) (*TurnResult, error) {
    // 1. 路由用户输入
    // 2. 如果是命令，直接执行返回
    // 3. 否则进入模型推理循环
    // 4. 处理模型输出的工具调用
    // 5. 直到纯文本输出或达到 maxSteps
}
```

### 3.6 Model Runner（模型执行器）

管理 LLM 调用和工具激活。

```go
type ModelRunner struct {
    client     LLMClient
    tools      *ToolRegistry
    promptTmpl *PromptTemplate
}

type LLMClient interface {
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
}
```

**特性：**
- 支持流式和非流式输出
- 工具提示渐进式展开（`$toolname` 语法）
- 有界循环防止无限工具调用

### 3.7 Tape Service

Tape Service 不再只是“会话日志容器”，而是一个面向 **single session** 运行时的内部事实服务，采用“**事实层（Entry）+ 线性检查点层（Handoff/Anchor）+ 读取组装层（View）**”三层模型。

```go
type Tape struct {
    SessionID string
    HeadSeq   uint64
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Entry struct {
    Seq         uint64
    Kind        EntryKind
    Content     string
    Metadata    map[string]any
    CorrectsSeq *uint64
    CreatedAt   time.Time
    Actor       string
}

type Anchor struct {
    ID           string
    SessionID    string
    AtSeq        uint64
    PrevAnchorID string
    PhaseTag     string
    Summary      string
    State        map[string]any
    SourceSeqs   []uint64
    CreatedAt    time.Time
    Owner        string
}

type ViewRequest struct {
    SessionID    string
    Task         string
    BudgetTokens int
}

type View struct {
    SessionID      string
    AnchorID       string
    IncludedSeqs   []uint64
    OmittedRanges  [][2]uint64
    DerivedSummary string
    Provenance     []uint64
}

func (s *TapeService) Append(ctx context.Context, sessionID string, in AppendInput) (*Entry, error)
func (s *TapeService) AppendCorrection(ctx context.Context, sessionID string, correctsSeq uint64, in AppendInput) (*Entry, error)
func (s *TapeService) CreateAnchor(ctx context.Context, sessionID string, in CreateAnchorInput) (*Anchor, error)
func (s *TapeService) Handoff(ctx context.Context, sessionID string, in HandoffInput) (*Anchor, error)
func (s *TapeService) BuildView(ctx context.Context, req ViewRequest) (*View, error)
```

**细化约束：**
- **Tape**：一个 session 对应一条 Tape；当前设计不引入独立的 `TapeID` 命名空间。
- **Entry**：最小不可变事实；纠错通过追加 `correction`，不能覆盖原事实。
- **Anchor**：线性检查点；通过 `PrevAnchorID` 串联，不支持 fork-merge 图。
- **Handoff**：顺序化阶段交接；先写 handoff 事实，再生成新的 anchor，不为多写者并发设计事务回滚语义。
- **View**：总是从当前 session 的最新 anchor（若无则从头）组装上下文，不提供自定义恢复路径或缓存层。

详细设计见：`docs/plans/2026-03-14-tape-service-design-refinement.md`。

### 3.8 Tool Engine（工具引擎）

工具注册、执行规划和脚本化执行。

```go
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

type Tool interface {
    Name() string
    Description() string
    Schema() *ToolSchema
    Execute(ctx context.Context, params map[string]any) (any, error)
}

type ToolSchema struct {
    Input  jsonschema.Schema
    Output jsonschema.Schema
}

type ExecutionPlan struct {
    Goal         string
    Preconditions []CheckSpec
    Steps        []PlanStep
    Expected     []ArtifactSpec
    Cleanup      []CleanupStep
}

type ScriptArtifact struct {
    Path    string
    Hash    string
    Content string
}

type ExecutionResult struct {
    ExitCode   int
    Stdout     string
    Stderr     string
    Retryable  bool
    Artifacts  []ArtifactSummary
}
```

**执行原则：**
- 命令执行类 tool call 先生成 `ExecutionPlan`，再编译为一次性临时脚本。
- `ScriptBuilder` 统一注入安全头、日志、cwd、超时和清理逻辑。
- `Sandbox Script Executor` 只执行脚本并回传结果，不负责重新分析问题。
- 失败优先基于已有 plan 与执行结果做局部修订，而不是每次重新从 bash 级别重试。

### 3.9 Skill Engine（技能引擎）

Skill Engine 负责内建 skills 的注册、初始化加载、匹配与调用。第一版只支持内建 skills，不支持运行时动态发现或目录扫描加载。

```go
type SkillSpec struct {
    Name         string
    Description  string
    License      string
    Compatibility string
    Metadata     map[string]string
    AllowedTools []string
}

type Skill interface {
    Spec() SkillSpec
    Match(ctx context.Context, input SkillInvocation) bool
    Execute(ctx context.Context, input SkillInvocation) (*SkillResult, error)
}

type SkillRegistry struct {
    skills map[string]Skill
    order  []string
}
```

**设计原则：**
- Agent 在初始化阶段加载全部内建 skills，并建立 `name -> skill` 索引。
- `SkillSpec` 在结构上对齐 `agentskills.io` 的元数据字段，但第一版不要求从 `SKILL.md` 文件系统加载。
- Skill 属于高层语义能力，不等同于普通 Tool；它可以调用 Tool Engine、Tape Service 或其他运行时模块。
- 第一版内建 `handoff` skill，负责生成标准化 handoff payload，并调用 Tape Service 完成交接。

详细设计见：`docs/plans/2026-03-14-agent-skills-handoff-design.md`。

---

## 4. 数据流

### 4.1 单次交互（Turn）流程

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌────────────┐     ┌──────────┐
│  User   │────▶│ Router  │────▶│ Planner  │────▶│ Script     │────▶│ Script   │
│ Input   │     │ .route  │     │ / Check  │     │ Builder    │     │ Executor │
└─────────┘     └────┬────┘     └────┬─────┘     └────┬───────┘     └────┬─────┘
                     │               │                  │                  │
                     │               ▼ (not command)    │                  │
                     │          ┌─────────┐             │                  │
                     │          │ Agent   │             │                  │
                     │          │ Loop    │             │                  │
                     │          └────┬────┘             │                  │
                     │               │                  │                  │
                     │               ▼                  │                  │
                     │          ┌─────────┐             │                  │
                     │          │ Model   │             │                  │
                     │          │ Runner  │             │                  │
                     │          └────┬────┘             │                  │
                     │               │                  │                  │
                     │               ▼ (has command-like tool call)        │
                     │          ┌──────────────┐        │                  │
                     └──────────│ Tool Planner │────────┘                  │
                                └──────┬───────┘                           │
                                       │                                   │
                                       ▼ (final text / exec result)        ▼
                                  ┌─────────┐                        ┌─────────┐
                                  │ Response│                        │ Tape    │
                                  └─────────┘                        └─────────┘
```

### 4.2 消息格式

```go
// Message 客户端通信消息
type Message struct {
    ID        string          `json:"id"`
    SessionID string          `json:"session_id"`
    Type      MessageType     `json:"type"` // text | command | tool_call | tool_result | error | stream
    Content   string          `json:"content"`
    Payload   json.RawMessage `json:"payload,omitempty"`
    Timestamp time.Time       `json:"timestamp"`
}

// MessageType 枚举
type MessageType string

const (
    MsgTypeText       MessageType = "text"
    MsgTypeCommand    MessageType = "command"
    MsgTypeToolCall   MessageType = "tool_call"
    MsgTypeToolResult MessageType = "tool_result"
    MsgTypeStream     MessageType = "stream"
    MsgTypeError      MessageType = "error"
    MsgTypeAnchor     MessageType = "anchor"
)
```

---

## 5. 存储设计

### 5.1 会话存储（BadgerDB）

```
Key Schema:
  session:{id}        → Session metadata
  session:{id}:tape   → Tape entries (JSONL)
  session:{id}:state  → Current session state
  
  index:user:{userID} → []sessionIDs (用户的所有会话)
  index:time:{ts}     → []sessionIDs (按时间索引)
```

### 5.2 配置存储

```
Key Schema:
  config:global       → 全局配置
  config:user:{id}    → 用户级配置
  config:skill:{name} → Skill 配置
```

---

## 6. 通信协议

### 6.1 WebSocket 协议

```
Client                          Server
  │                               │
  ├────── CONNECT /ws/{session} ─▶│
  │                               │
  │◀──────── WELCOME {id} ────────┤
  │                               │
  ├────── MESSAGE {type,text} ───▶│
  │                               │
  │◀──────── STREAM {chunk} ──────┤
  │◀──────── STREAM {chunk} ──────┤
  │◀──────── DONE {final} ────────┤
  │                               │
  ├────── COMMAND ,tool args ────▶│
  │◀──────── RESULT {...} ────────┤
  │                               │
  ├────── CLOSE ────────────────▶│
```

### 6.2 Message Bus 协议（Discord/Telegram）

```text
Event Types: message | command | callback | system

Discord/Telegram Event
    │
    ▼
ChannelAdapter (platform SDK/Webhook/Polling)
    │
    ├─ validate signature / replay window
    └─ normalize -> BusEvent{type, session_id, user_id, content, correlation_id}
    ▼
Bus Router (state: received -> validated -> routed -> executing -> replied/failed)
    │
    ├─ message  -> AgentLoop
    ├─ command  -> Command Policy (no eval) -> Tool Engine
    ├─ callback -> Session Manager
    └─ system   -> Dependency Install Policy (blacklist-first)
    ▼
Response Adapter -> Discord/Telegram API

Error Layers:
1) Platform Error (signature/API/rate-limit)
2) Policy Reject (command/dependency blocked)
3) Execution Error (tool/runtime failure)
All errors must include correlation_id and append to Tape.
```

---

## 7. 配置结构

```toml
# config.toml
[server]
host = "0.0.0.0"
port = 8080

[transport.websocket]
enabled = true
path = "/ws"

[transport.grpc]
enabled = true
port = 50051

[message_bus]
enabled = true
provider = "inproc" # inproc | nats | redis

[message_bus.discord]
enabled = true
bot_token = "${DISCORD_BOT_TOKEN}"
application_id = "${DISCORD_APP_ID}"
webhook_secret = "${DISCORD_WEBHOOK_SECRET}"

[message_bus.telegram]
enabled = true
bot_token = "${TELEGRAM_BOT_TOKEN}"
webhook_secret = "${TELEGRAM_WEBHOOK_SECRET}"
polling = false

[message_bus.command_policy]
reject_shell_eval = true
default_timeout_seconds = 60
max_output_kb = 256

[message_bus.deps_install_policy]
enabled = true
mode = "blacklist-first"
blocked_patterns = ["curl|bash", "sudo ", "npm -g", "http://", "https://"]

[llm]
provider = "openai" # openai | anthropic | local
api_key = "${OPENAI_API_KEY}"
model = "gpt-4"
max_tokens = 4096
temperature = 0.7

[storage]
type = "badger" # badger | redis | s3
path = "./data"

[session]
timeout = "30m"
max_history = 100

[logging]
level = "info"
format = "json"
```

---

## 8. 目录结构

```
.
├── cmd/
│   └── agent-runtime/
│       └── main.go           # 服务入口
├── internal/
│   ├── server/
│   │   ├── server.go         # HTTP/gRPC 服务
│   │   └── middleware.go     # 中间件
│   ├── transport/
│   │   ├── websocket.go
│   │   ├── grpc.go
│   │   └── message_bus.go
│   ├── adapters/
│   │   ├── discord.go
│   │   └── telegram.go
│   ├── session/
│   │   ├── manager.go
│   │   └── session.go
│   ├── core/
│   │   ├── router.go
│   │   ├── agent_loop.go
│   │   └── model_runner.go
│   ├── tape/
│   │   ├── tape.go
│   │   └── store.go
│   ├── tools/
│   │   ├── registry.go
│   │   ├── executor.go
│   │   └── builtin/          # 内置工具
│   ├── llm/
│   │   ├── client.go
│   │   ├── openai.go
│   │   └── anthropic.go
│   └── storage/
│       ├── badger.go
│       └── interface.go
├── pkg/
│   └── types/
│       └── message.go        # 公共类型定义
├── api/
│   ├── proto/                # gRPC proto 文件
│   └── openapi/              # OpenAPI 规范
├── docs/
│   └── architecture.md       # 本文档
├── config/
│   └── example.toml
├── go.mod
├── go.sum
└── Makefile
```

---

## 9. 关键设计决策

### 9.1 为什么选择 Go

- **原生并发**：goroutine + channel 完美匹配 Agent 的并发需求
- **静态编译**：单二进制部署，无依赖
- **性能**：低延迟、高吞吐
- **生态**：丰富的网络与平台库（connect-go、discordgo、telegram-bot-api）

### 9.2 存储选择 Badger

- LSM-Tree 结构，写性能优异（适合只追加的 Tape）
- 纯 Go 实现，无 CGO 依赖
- 支持前缀扫描和迭代器（方便按 session 查询）

### 9.3 与 bub 的差异

| 特性 | bub | 本设计 |
|------|-----|--------|
| 部署方式 | 本地 CLI | 服务化部署 |
| 通信 | 标准输入输出 | WebSocket/gRPC/Discord/Telegram |
| 存储 | 文件系统 | BadgerDB |
| 并发 | 单会话 | 多会话并发 |
| 扩展 | Python skills | Go plugins / WASM |

---

## 11. 参考

- [bub architecture](https://bub.build/architecture/)
- [Discord Developer Docs](https://discord.com/developers/docs/intro)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
