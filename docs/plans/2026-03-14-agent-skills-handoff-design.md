# Agent Skills 模块与 Built-in Handoff Skill 设计

## 1. 背景

当前运行时已经有 Tool、Tape、Agent Loop 和单会话 handoff 设计，但缺少一个介于“模型推理”和“底层能力调用”之间的**高层语义能力层**。  
我们希望增加一个 `agent skills` 模块，并引入第一个内建 skill：`handoff`。

本设计参考 `https://agentskills.io/specification`，但第一版不做外部目录加载，而是：

- 先建立**内建 skills 框架**；
- skills 在 **agent 初始化阶段** 统一加载；
- `handoff` skill 作为内建 skill 标准化地生成交接内容，并调用 Tape/Handoff 机制完成阶段交接。

## 2. 设计目标

1. 增加一个独立的 `Skill Engine / Skill Registry` 模块。
2. 第一版只支持内建 skills，不做运行时目录扫描和外部技能安装。
3. Skill 元数据结构尽量对齐 `agentskills` spec，便于未来扩展。
4. Agent 在初始化时预加载全部内建 skills，而不是按需懒加载。
5. 提供内建 `handoff` skill，统一 handoff payload 与 Tape/Handoff 调用路径。

## 3. 总体架构

### 3.1 模块位置

`Skill Engine` 位于 `Agent Loop` 和底层 `Tool Engine / Tape Service` 之间：

```text
Agent Loop
  -> Skill Registry
  -> Built-in Skill
     -> Tool Engine / Tape Service
```

职责拆分：

- `SkillRegistry`：注册、初始化加载、发现和调用内建 skills。
- `SkillRuntime`：在一次调用中组织 skill 的上下文输入、结果输出和错误回传。
- `Built-in Handoff Skill`：作为首个内建 skill，负责标准化阶段交接。

### 3.2 与 Tool 的关系

- **Tool**：偏底层执行能力，如命令执行、查询、文件处理。
- **Skill**：偏高层语义能力，如 handoff、规划、复盘等。
- Skill 可以调用 Tool，也可以直接调用 Tape Service 或其他运行时模块。

因此，Skill 不是 Tool 的别名，而是位于 Tool 之上的编排层。

## 4. 与 Agent Skills Spec 的对齐方式

第一版不直接从 `SKILL.md` 目录加载，但会对齐其核心描述字段：

```go
type SkillSpec struct {
    Name          string
    Description   string
    License       string
    Compatibility string
    Metadata      map[string]string
    AllowedTools  []string
}
```

说明：

- `Name` / `Description` 与 `agentskills` spec 保持一致。
- `License` / `Compatibility` / `Metadata` / `AllowedTools` 也保留对应字段。
- 第一版这些字段主要用于**注册、审计、配置与未来兼容**，而不是驱动目录加载。

也就是说，第一版是**spec-compatible metadata**，不是 **spec-driven loader**。

## 5. 初始化加载模型

Agent 在启动时完成全部内建 skills 的加载。

### 5.1 初始化步骤

1. runtime 创建 `SkillRegistry`
2. 注册全部 built-in skills
3. 读取每个 skill 的 `SkillSpec`
4. 建立 `name -> skill` 索引
5. 将 registry 注入 `Agent Loop`

### 5.2 为什么必须初始化时加载

- Agent 在进入会话前就应知道自己有哪些高层能力。
- `handoff` 属于 built-in 能力，不应依赖运行时发现。
- 初始化加载有助于稳定 skill 匹配顺序、配置解析和审计。

因此，第一版明确**不采用懒加载 / 按需发现**模型。

## 6. 运行时接口

```go
type SkillInvocation struct {
    SessionID string
    Name      string
    Goal      string
    Input     map[string]any
}

type SkillResult struct {
    Name      string
    Success   bool
    Summary   string
    Payload   map[string]any
}

type Skill interface {
    Spec() SkillSpec
    Match(ctx context.Context, input SkillInvocation) bool
    Execute(ctx context.Context, input SkillInvocation) (*SkillResult, error)
}
```

约束：

- `Match` 用于判断该 skill 是否适合当前调用。
- `Execute` 负责真正的技能逻辑。
- `SkillResult` 必须可写入 Tape 摘要，不依赖隐藏状态。

## 7. Built-in Handoff Skill

### 7.1 职责

`handoff` skill 的职责不是只输出一段文本，而是完成一个标准化交接动作：

1. 读取当前 session 的上下文；
2. 生成标准化 handoff payload；
3. 调用 Tape Service 的 `Handoff(...)`；
4. 返回结构化交接结果。

### 7.2 输出 payload

推荐最小 payload：

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

### 7.3 执行流程

```text
Agent Loop
  -> invoke handoff skill
  -> handoff skill builds payload
  -> handoff skill calls TapeService.Handoff(...)
  -> Tape appends handoff entry
  -> Tape updates latest anchor (if successful)
  -> skill returns structured result
```

### 7.4 返回结果

建议结果至少包含：

- `success`
- `handoff_written`
- `anchor_written`
- `summary`
- `anchor_id`（如果存在）
- `error_stage`（如果失败）

## 8. 与现有模块的集成

### 8.1 Agent Loop

- `Agent Loop` 在需要阶段收束、任务转交或上下文压缩前显式调用 `handoff` skill。
- `handoff` 不再由上层各处自行拼装，而是走统一 skill 入口。

### 8.2 Tape Service

- `handoff` skill 是 Tape `Handoff(...)` 的第一条标准化高层调用链。
- 它将 handoff 内容与 Tape 的线性交接语义绑定起来。

### 8.3 Tool Engine

- Skill 可以调用 Tool，但 `handoff` 第一版主要调用 Tape Service。
- 它与命令执行类 `planner -> script builder -> executor` 流程互补，不冲突。

### 8.4 Config

沿用现有配置存储思路：

```text
config:skill:{name} -> Skill 配置
```

第一版可用于：

- skill 启停
- 默认 owner
- 默认 `phase_tag`
- `handoff` payload 模板参数

## 9. 错误处理与审计

### 9.1 错误处理

- skill 未命中：返回未命中，由 Agent Loop 继续其他路径。
- payload 生成失败：不写 Tape，直接返回错误。
- handoff entry 成功但 anchor 失败：保留 handoff 事实，并返回结构化错误。

### 9.2 审计

Tape 建议记录：

- `skill_invocation_summary`
- `skill_name`
- `handoff_payload_summary`
- `skill_result_summary`

默认不强制记录完整 skill body 或完整 prompt。

## 10. 第一版边界

第一版明确不做：

- 外部 `SKILL.md` 目录加载器
- 动态安装 / marketplace
- 多 skill 复杂调度
- 让 `handoff` 脱离 Tape/Handoff 独立运行

第一版只做两件事：

1. 建立内建 skills 框架
2. 跑通 built-in `handoff` skill

## 11. 结论

本设计把 `Skill` 引入为一个新的高层语义层：

- 它按 `agentskills` spec 对齐元数据；
- 在 agent 初始化阶段统一加载；
- 第一版只支持 built-in skills；
- `handoff` 作为第一个 built-in skill，统一生成交接内容并调用 Tape/Handoff。

这样后续再扩展更多 skills 或引入目录加载时，可以在不推翻第一版模型的前提下逐步演进。
