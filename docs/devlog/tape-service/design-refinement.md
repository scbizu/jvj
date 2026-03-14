# Tape Service Design Refinement

## Background

当前运行时的设计前提是 **single session**：单个 runtime 实例在任意时刻只服务一个活动会话。
因此，Tape Service 的目标不是支持多 session / 多分支恢复，而是为**单会话、线性执行**提供：

- append-only 的事实记录；
- 可恢复的线性检查点；
- 明确但不过度复杂的阶段交接；
- 面向当前任务的上下文组装。

参考 `https://tape.systems/` 中 `tape / entry / anchor / handoff / view` 的思想，本设计保留其“事实不可变、上下文按需组装”的核心，但去掉多分支、多路径、多缓存等不适合 single-session 的复杂度。

## Goals

1. **Append-only**：事实只能追加，不能原地覆盖。
2. **Single-session**：一个 session 对应一条 Tape，不引入独立的多 Tape 命名空间。
3. **Linear-recovery**：总是从当前 session 的最新 Anchor 恢复，不支持分叉恢复路径。
4. **Explicit-handoff**：阶段交接用显式 handoff 表达，但保持顺序化写入语义。
5. **Assembled-context**：View 在读取时构造，不缓存、不持久化为新的权威状态。

## 3. 核心语义

### 3.1 Tape

`Tape` 是当前 session 的 append-only 事实流，负责维护顺序与回放能力。

```go
type Tape struct {
    SessionID string
    HeadSeq   uint64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

约束：

- 当前设计下 `session_id` 就是 Tape 的唯一归属标识。
- 一个活跃 session 始终绑定一条 Tape。
- `HeadSeq` 必须单调递增，用于顺序回放与审计。

### 3.2 Entry

`Entry` 是最小事实单元，只表达“发生了什么”，不承担恢复策略。

```go
type EntryKind string

const (
    EntryUser       EntryKind = "user"
    EntryAssistant  EntryKind = "assistant"
    EntryToolCall   EntryKind = "tool_call"
    EntryToolResult EntryKind = "tool_result"
    EntrySystem     EntryKind = "system"
    EntryCorrection EntryKind = "correction"
    EntryHandoff    EntryKind = "handoff"
)

type Entry struct {
    Seq         uint64
    Kind        EntryKind
    Content     string
    Metadata    map[string]any
    CorrectsSeq *uint64
    CreatedAt   time.Time
    Actor       string // user | agent | system | tool:<name>
}
```

约束：

- `Entry` 一旦写入即不可变。
- `correction` 必须引用 `CorrectsSeq`，但不能删除原事实。
- `Entry` 不要求携带 `SourceSeqs`；single-session 下顺序链本身就是默认 provenance。

### 3.3 Anchor

`Anchor` 表示当前 session 的线性检查点，用来回答“下一次恢复时从哪里开始”。

```go
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
```

约束：

- `Anchor` 是独立资源，但只形成**线性链**，不支持 DAG / fork-merge。
- `PrevAnchorID` 仅指向上一个 anchor。
- `PhaseTag` 是叙事性标签，不作为恢复查询维度。
- 恢复时只读取**最新 anchor**，而不是“最近相关 anchor 集合”。

### 3.4 Handoff

`Handoff` 用来表示一次阶段收束和继续执行建议，但在 single-session 模型下，它不需要承载多写者事务语义。

推荐输入：

```go
type HandoffInput struct {
    Summary    string
    NextSteps  []string
    SourceSeqs []uint64
    Owner      string
    PhaseTag   string
    StateDelta map[string]any
}
```

推荐最小状态契约：

| 字段 | 含义 | 是否必填 |
| --- | --- | --- |
| `summary` | 当前阶段收束摘要 | 是 |
| `next_steps` | 下一步明确动作列表 | 是 |
| `source_seqs` | 支撑交接结论的关键事实 | 是 |
| `owner` | 当前负责执行者 | 是 |
| `phase_tag` | 阶段标签，仅作叙事/审计 | 否 |
| `open_items` | 未完成事项 | 否 |

示例：

```json
{
  "summary": "Discovery complete.",
  "next_steps": ["Run migration", "Integration tests"],
  "source_seqs": [128, 130, 131],
  "owner": "agent",
  "phase_tag": "implement",
  "open_items": ["confirm prod config"]
}
```

### 3.5 View

`View` 是读取时为当前 session 组装的上下文窗口。

```go
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
```

约束：

- `View` 是瞬时读模型，不持久化，不缓存。
- `View` 总是从当前 session 的最新 anchor 组装；如果没有 anchor，则从序号 1 开始。
- `View` 的 provenance 必须可解释，但不提供多路径恢复选项。

## 4. 设计不变量

1. **History is append-only**：原始事实不可原地修改或删除。
2. **Recovery is linear**：恢复入口只有一个，即当前 session 的最新 anchor。
3. **Derivatives never replace facts**：handoff、摘要、压缩都不能替代原始事实。
4. **Context is constructed**：上下文来自组装，而不是直接继承整段历史。
5. **One session, one tape**：single-session 运行时不支持多 Tape 并行处理。

## 5. 服务职责边界

Tape Service 只负责事实、检查点和视图组装，不负责：

- 解释用户意图；
- 选择是否调用工具；
- 执行模型推理。

与现有模块的边界如下：

- `Session Manager`：负责当前 session 的生命周期与唯一附着关系；Tape Service 直接服务该 session。
- `Agent Loop`：负责单轮编排；在关键阶段通过 Tape Service 追加事实、写 handoff、创建 anchor、组装 view。
- `Tool Engine`：只产出 `tool_call` / `tool_result` 事实；工具执行默认在单轮内闭合，不维护跨 session 的未完成工具队列。

## 6. 内部接口设计

```go
type TapeService interface {
    Append(ctx context.Context, sessionID string, in AppendInput) (*Entry, error)
    AppendCorrection(ctx context.Context, sessionID string, correctsSeq uint64, in AppendInput) (*Entry, error)
    CreateAnchor(ctx context.Context, sessionID string, in CreateAnchorInput) (*Anchor, error)
    Handoff(ctx context.Context, sessionID string, in HandoffInput) (*Anchor, error)

    ListEntries(ctx context.Context, sessionID string, fromSeq, limit uint64) ([]Entry, error)
    GetLatestAnchor(ctx context.Context, sessionID string) (*Anchor, error)
    BuildView(ctx context.Context, req ViewRequest) (*View, error)
}
```

### 6.1 写接口语义

- `Append`：追加普通事实。
- `AppendCorrection`：追加纠错事实，不影响原事实存在性。
- `CreateAnchor`：创建线性检查点。
- `Handoff`：顺序化写入 handoff，并立即尝试创建新的最新 anchor。

### 6.2 读接口语义

- `ListEntries`：按 `Seq` 顺序读取原始事实。
- `GetLatestAnchor`：读取当前 session 最新的恢复点。
- `BuildView`：从最新 anchor（或头部）构造上下文窗口。

## 7. Handoff 写入流程

single-session 模型下，`Handoff` 采用**顺序化写入**，而不是为多并发写者设计事务回滚。

### 7.1 成功路径

1. 校验 `Summary`、`NextSteps`、`SourceSeqs`、`Owner`。
2. 追加一条 `EntryHandoff`，将 `next_steps`、`phase_tag` 等放入 `Metadata`。
3. 基于 `HandoffInput` 生成新的 `Anchor.State`。
4. 写入新的 latest anchor，并通过 `PrevAnchorID` 链接上一个 anchor。
5. 返回新 anchor。

### 7.2 失败路径

- handoff entry 写入失败：调用失败，不产生新 anchor。
- handoff entry 写入成功但 anchor 写入失败：handoff 事实保留，调用返回错误；下一次恢复仍可从上一个 anchor 开始，并读到这条 handoff。
- `SourceSeqs` 缺失或无效：返回校验错误。

这样做的原因是：single-session 下不存在多写者竞争，失败处理可以依赖线性重放，而不必引入复杂事务层。

## 8. View 组装策略

### 8.1 默认算法

1. 获取当前 session 的最新 anchor；若不存在则从 `Seq=1` 开始。
2. 纳入从恢复起点到当前 `HeadSeq` 的事实。
3. 保证最近用户输入和最近 handoff 不会在压缩时被提前丢弃。
4. 如超出 `BudgetTokens`，优先压缩较早事实，保留最近事实原文。
5. 返回 `AnchorID`、`IncludedSeqs`、`OmittedRanges` 和 `Provenance`。

### 8.2 组装原则

- 不支持 `PreferredFrom`、多 anchor path 或策略切换。
- 不缓存 `View`，每次按当前 Tape 即时构造。
- 工具结果默认作为普通事实处理，不额外维护“未闭合工具结果”索引。

## 9. 一致性与错误语义

| 操作 | 成功后保证 | 失败时保证 |
| --- | --- | --- |
| `Append` | 事实可按 `Seq` 重放 | 不产生部分 entry |
| `AppendCorrection` | 原事实保留，纠错可追溯 | 原事实不受影响 |
| `CreateAnchor` | 最新恢复点更新 | 不改变已有事实 |
| `Handoff` | handoff 可审计；若 anchor 成功则最新恢复点推进 | 不回滚已成功写入的 handoff entry |
| `BuildView` | 返回当前 session 的线性上下文 | 不静默制造多路径恢复结果 |

额外约束：

- 不允许静默丢弃 `SourceSeqs`。
- 没有 anchor 时必须显式回退到从头扫描。
- `BuildView` 不引入缓存命中或快照回放逻辑。

## 10. 与现有运行时的对齐方式

### 10.1 Session Manager

- 单实例只维护一个 `current session`。
- 同一时刻只允许一个活动流附着到该 session。
- 当前 session 持有唯一 Tape 引用。

### 10.2 Agent Loop

- 用户输入、模型输出、工具结果继续通过 `Append` 进入 Tape。
- 需要阶段收束时调用 `Handoff`。
- 下一轮上下文始终通过 `BuildView` 从最新 anchor 组装。

### 10.3 Tool Engine

- `tool_call` / `tool_result` 在单轮内尽量闭合。
- 不设计跨 session 的工具未完成状态管理。

## 11. 存储与索引建议（Badger）

```text
session:{session_id}:tape              -> Tape
session:{session_id}:seq               -> uint64(atomic)
session:{session_id}:entry:{seq}       -> Entry
session:{session_id}:anchor:{anchor_id} -> Anchor
session:{session_id}:anchor:latest     -> anchor_id
```

建议：

- `entry:{seq}` 负责顺序扫描与回放。
- `anchor:latest` 负责直接获取当前 session 的恢复点。
- 不引入 `view_cache`、`phase:latest`、多 Tape 聚合索引。

## 12. 非目标

本轮设计不覆盖：

- 对外 `AgentService` 协议调整；
- 多 session 并行执行；
- fork-merge anchor 图；
- View 缓存 / 快照持久化；
- 跨阶段、按 phase 名称的恢复查询。

## 13. 文档落地结论

本设计将 Tape Service 收敛为适合 **single-session runtime** 的内部基础设施，核心变化是：

1. 保留 `Entry / Anchor / Handoff / View` 四个概念，但都改成线性模型。
2. 删除多路径恢复、多 anchor 图和 view cache 等复杂度。
3. 让恢复逻辑收敛为“最新 anchor + 后续事实”。

这样后续实现可以直接围绕单会话、单恢复路径和顺序化 handoff 落地，而不会预埋暂时不需要的多会话能力。
