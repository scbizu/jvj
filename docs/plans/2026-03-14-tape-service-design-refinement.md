# Tape Service 细化设计（状态契约版）

## 1. 背景

当前 `docs/architecture.md` 中的 Tape Service 仍停留在“只追加会话历史”的日志模型，足以满足回放，但不足以支撑以下运行时需求：

- 在长会话中快速恢复阶段状态，而不是每次从头扫描。
- 在多阶段执行中明确“交接事实”和“新的执行起点”。
- 为不同任务动态组装上下文，而不是直接继承整段历史。
- 在压缩、纠错、摘要之后，仍保持原始事实可追溯、可审计。

参考 `https://tape.systems/` 中对 `tape / entry / anchor / handoff / view` 的定义，本设计将 Tape Service 明确为一个内部事实服务，而不是单纯的日志容器。

## 2. 设计目标

1. **Append-only**：事实只能追加，不能原地覆盖。
2. **Recoverable**：系统可以从最近相关 Anchor 恢复，而不是依赖全量扫描。
3. **Explicit-handoff**：阶段切换必须通过显式 handoff 完成。
4. **Assembled-context**：上下文由 View 在读取时组装，而不是直接截取历史。
5. **Traceable-derivation**：摘要、纠错、缓存等衍生结果必须保留 provenance。

## 3. 核心语义

### 3.1 Tape

`Tape` 是某个 session 的 append-only 事实流，负责维护顺序与存储边界，不负责保存“继承状态”。

```go
type TapeID string

type Tape struct {
    ID        TapeID
    SessionID string
    HeadSeq   uint64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

约束：

- 一个活跃 session 对应一条主 Tape。
- 写入必须通过原子递增的 `HeadSeq` 分配 `Seq`。
- `Tape` 是事实容器，不直接暴露“当前状态”字段。

### 3.2 Entry

`Entry` 是最小事实单元，只表达“发生了什么”，不表达“后续如何继承状态”。

```go
type EntryKind string

const (
    EntryUser         EntryKind = "user"
    EntryAssistant    EntryKind = "assistant"
    EntryToolCall     EntryKind = "tool_call"
    EntryToolResult   EntryKind = "tool_result"
    EntrySystem       EntryKind = "system"
    EntryCorrection   EntryKind = "correction"
    EntrySummary      EntryKind = "summary"
    EntryHandoff      EntryKind = "handoff"
)

type Entry struct {
    TapeID      TapeID
    Seq         uint64
    Kind        EntryKind
    Content     string
    Metadata    map[string]any
    SourceSeqs  []uint64
    CorrectsSeq *uint64
    CreatedAt   time.Time
    Actor       string // user | agent | system | tool:<name>
}
```

约束：

- `Entry` 一旦写入即不可变。
- `correction` 必须引用 `CorrectsSeq`，且不能删除被纠错事实。
- `summary`、`handoff` 等衍生事实必须带 `SourceSeqs`。
- `Metadata` 允许存放结构化辅助字段，但不替代显式领域字段。

### 3.3 Anchor

`Anchor` 表示一个阶段性可恢复检查点，用来回答“下一次构造上下文时，应该从哪里开始”。

```go
type Anchor struct {
    ID              string
    TapeID          TapeID
    AtSeq           uint64
    ParentAnchorIDs []string
    Phase           string
    Summary         string
    State           map[string]any
    SourceSeqs      []uint64
    CreatedAt       time.Time
    Owner           string
}
```

约束：

- `Anchor` 是独立资源，不等同于 `Entry`。
- `AtSeq` 表示恢复上下文时的逻辑起点；它不要求覆盖之前历史，而是允许 View 回溯引用。
- `ParentAnchorIDs` 支持 fork-merge 形式的锚点图，但默认链式使用。
- `State` 只保存“恢复下一阶段必须知道的最小状态”，不保存整段历史。

### 3.4 Handoff

`Handoff` 是一次强约束的阶段切换操作，不只是“写一条说明消息”。

它的职责是：

- 把当前阶段收束成可追溯的交接事实。
- 为下一阶段创建新的恢复原点。
- 明确写出来源事实、下一步、owner 和失败边界。

推荐输入：

```go
type HandoffInput struct {
    FromPhase  string
    ToPhase    string
    Summary    string
    NextSteps  []string
    SourceSeqs []uint64
    Owner      string
    StateDelta map[string]any
}
```

推荐最小状态契约：

| 字段 | 含义 | 是否必填 |
| --- | --- | --- |
| `phase` | 下一阶段名称，如 `implement` | 是 |
| `summary` | 上一阶段收束摘要 | 是 |
| `next_steps` | 下一阶段明确动作列表 | 是 |
| `source_seqs` | 支撑交接结论的事实引用 | 是 |
| `owner` | 当前负责执行者 | 是 |
| `open_items` | 仍待处理的问题/阻塞项 | 否 |
| `constraints` | 下一阶段必须继续遵守的约束 | 否 |

示例：

```json
{
  "phase": "implement",
  "summary": "Discovery complete.",
  "next_steps": ["Run migration", "Integration tests"],
  "source_seqs": [128, 130, 131],
  "owner": "agent",
  "open_items": ["confirm prod config"]
}
```

### 3.5 View

`View` 是读取时构造的上下文包，面向某个具体任务，而不是“最近 N 条消息”的直接切片。

```go
type ViewRequest struct {
    TapeID        TapeID
    Task          string
    BudgetTokens  int
    PreferredFrom []string
    Policy        ViewPolicy
}

type View struct {
    TapeID         TapeID
    AnchorPath     []string
    IncludedSeqs   []uint64
    OmittedRanges  [][2]uint64
    DerivedSummary string
    Provenance     []uint64
}
```

约束：

- `View` 默认是瞬时读模型，不要求持久化。
- 如果为了调试或审计落地 `view_snapshot`，也必须回指原始 `Anchor/Entry`。
- `View` 的来源必须可解释，不能返回“黑盒上下文”。

## 4. 设计不变量

1. **History is append-only**：原始事实不可原地修改或删除。
2. **Derivatives never replace facts**：摘要、纠错、缓存不会替代原事实。
3. **Context is constructed**：上下文通过 View 组装，而不是整段继承。
4. **Execution shifts by anchor**：阶段切换后，新的执行原点由 Anchor 表示。
5. **Provenance is mandatory**：交接、摘要、View 都必须能追溯来源。

## 5. 服务职责边界

Tape Service 只负责事实、锚点、视图组装，不负责：

- 解释用户意图。
- 选择是否调用某个工具。
- 执行模型推理。

它与现有模块的边界如下：

- `Session Manager`：负责 session 生命周期；Tape Service 只接收 `session_id -> tape_id` 绑定结果。
- `Agent Loop`：负责单轮编排；在关键阶段通过 Tape Service 读写事实和 handoff。
- `Tool Engine`：只产出工具执行事实，不负责决定如何形成 Anchor/View。

## 6. 内部接口设计

```go
type TapeService interface {
    Append(ctx context.Context, tapeID TapeID, in AppendInput) (*Entry, error)
    AppendCorrection(ctx context.Context, tapeID TapeID, correctsSeq uint64, in AppendInput) (*Entry, error)
    CreateAnchor(ctx context.Context, tapeID TapeID, in CreateAnchorInput) (*Anchor, error)
    Handoff(ctx context.Context, tapeID TapeID, in HandoffInput) (*Anchor, error)

    ListEntries(ctx context.Context, tapeID TapeID, fromSeq, limit uint64) ([]Entry, error)
    GetAnchor(ctx context.Context, tapeID TapeID, anchorID string) (*Anchor, error)
    ResolveLatestAnchor(ctx context.Context, tapeID TapeID, phase string) (*Anchor, error)
    BuildView(ctx context.Context, req ViewRequest) (*View, error)
    ExplainView(ctx context.Context, req ViewRequest) (*ViewExplanation, error)
}
```

### 6.1 写接口语义

- `Append`：追加普通事实，不推断阶段语义。
- `AppendCorrection`：追加纠错事实，不影响原事实存在性。
- `CreateAnchor`：主动创建检查点，适用于长阶段内的中途固化。
- `Handoff`：执行阶段切换，负责一次性写出 handoff 与 anchor。

### 6.2 读接口语义

- `ListEntries`：按 `Seq` 顺序读取原始事实。
- `GetAnchor`：按 ID 读取特定锚点。
- `ResolveLatestAnchor`：返回某阶段最近可用锚点。
- `BuildView`：按任务、预算、偏好锚点组装上下文。
- `ExplainView`：返回选取/省略原因，便于调试与审计。

## 7. Handoff 写入流程

`Handoff` 必须被视为一个事务性操作。

### 7.1 成功路径

1. 校验 `SourceSeqs` 非空，且全部存在于当前 Tape。
2. 校验 `Summary`、`ToPhase`、`Owner`、`NextSteps` 满足最小契约。
3. 追加一条 `EntryHandoff`。
4. 基于 `HandoffInput` 生成新的 `Anchor.State`。
5. 写入新 `Anchor`，并把其 `AtSeq` 设置为 handoff 之后的恢复起点。
6. 提交事务，对外返回新 Anchor。

### 7.2 失败路径

- `EntryHandoff` 写入成功但 `Anchor` 写入失败：整体回滚，对外表现为 `Handoff` 失败。
- `SourceSeqs` 缺失或无效：返回校验错误，不能自动降级成普通 entry。
- `StateDelta` 不可序列化：返回内部错误，不产生部分交接结果。

### 7.3 为什么必须原子

如果只写入 handoff 而没有新的 anchor，那么系统会留下“交接已发生，但下一阶段无恢复原点”的半完成状态；这会直接破坏 `execution shifts by anchor` 的语义。

## 8. View 组装策略

### 8.1 默认算法

1. **选起点**：优先选择最近且相关的 Anchor；若调用方传入 `PreferredFrom`，则优先其指定路径。
2. **拉硬约束**：纳入当前任务必须看到的事实：
   - 最近用户意图
   - 最近 handoff
   - 尚未闭合的工具结果
   - 当前阶段约束与未完成事项
3. **补上下文**：在预算允许时，纳入支撑这些结论的关键事实。
4. **压缩衍生**：预算不足时，优先保留原始关键事实，再以 summary 替代长链路细节。
5. **返回 provenance**：输出 `IncludedSeqs`、`OmittedRanges`、`AnchorPath` 和 `Provenance`。

### 8.2 组装原则

- 优先保留原始关键事实，而不是先做摘要。
- 省略是允许的，但必须显式标注省略范围。
- 缓存 View 可以提升性能，但缓存不能成为新的权威事实源。

## 9. 一致性与错误语义

| 操作 | 成功后保证 | 失败时保证 |
| --- | --- | --- |
| `Append` | 事实可按 `Seq` 重放 | 不产生部分 entry |
| `AppendCorrection` | 原事实保留，纠错可追溯 | 原事实不受影响 |
| `CreateAnchor` | 可从该 anchor 恢复 | 不改变已有事实 |
| `Handoff` | `handoff + anchor` 同时可见 | 不暴露半完成阶段切换 |
| `BuildView` | 返回明确 provenance | 不静默伪造上下文 |

额外约束：

- 不允许静默丢弃 `SourceSeqs`。
- 不允许“猜测性恢复”：没有可用锚点时，必须显式回退到全量扫描策略。
- 不允许“成功形状的降级”：如果 View 预算不足导致关键信息缺失，必须显式说明压缩发生。

## 10. 与现有运行时的对齐方式

### 10.1 Agent Loop

- 用户输入、模型输出、工具调用结果继续通过 `Append` 写入 Tape。
- 当某轮结束并需要跨阶段交接时，由 Agent Loop 调用 `Handoff`。
- 下一轮构造模型上下文时，不再直接截取最近历史，而是调用 `BuildView`。

### 10.2 Session Manager

- Session 继续持有 Tape 引用。
- Session 恢复时优先读取最近 Anchor，再由 `BuildView` 组装工作上下文。

### 10.3 Tool Engine

- 工具调用写入 `tool_call` 和 `tool_result` 事实。
- 未完成工具结果会被 `BuildView` 视为高优先级硬约束。

## 11. 存储与索引建议（Badger）

```text
tape:{session_id}:meta                    -> Tape
tape:{session_id}:seq                     -> uint64(atomic)
tape:{session_id}:entry:{seq}             -> Entry
tape:{session_id}:anchor:{anchor_id}      -> Anchor
tape:{session_id}:anchor_at:{seq}         -> anchor_id
tape:{session_id}:phase:{phase}:latest    -> anchor_id
tape:{session_id}:view_cache:{hash}       -> View (optional, ttl)
```

建议：

- `entry:{seq}` 负责顺序扫描与回放。
- `anchor_at:{seq}` 负责按时间定位恢复起点。
- `phase:{phase}:latest` 方便快速恢复某阶段最新锚点。
- `view_cache` 只能作为性能优化层，不能替代事实层。

## 12. 非目标

本轮设计不覆盖：

- 对外 `AgentService` 协议修改。
- 外部 API 中新增 `View` / `Handoff` RPC。
- 多租户跨 Tape 聚合查询。
- 向量检索或长期知识库设计。

## 13. 文档落地结论

本设计将 Tape Service 从“会话日志服务”提升为“事实写入 + 阶段切换 + 上下文组装”的内部基础设施，重点强化三件事：

1. `Handoff` 是受约束的阶段切换协议，而不是普通文本记录。
2. `Anchor` 是执行恢复原点，而不是历史摘要别名。
3. `View` 是面向任务的组装上下文，而不是最近消息切片。

这使后续实现可以直接围绕状态契约、事务边界和组装算法展开，而不必再次补一轮概念设计。
