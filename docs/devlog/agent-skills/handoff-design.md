# Agent Native Skills and Built-in Handoff Design

## Background

`skills` 本身就是 agent 原生支持的概念，因此这里不需要再额外发明一套很重的 `Skill Engine / SkillRegistry / SkillRuntime` 抽象。
我们真正需要的是两件事：

1. 让 runtime 在启动时把内建 skills 预加载进 agent；
2. 提供一个 built-in `handoff` skill，标准化 handoff 内容并接入 Tape 的 `Handoff(...)` 语义。

本设计参考 `https://agentskills.io/specification`，但重点是**直接复用 skill 这个原生概念**，而不是围绕 skill 再造一层平台。

## Goals

1. 使用标准 skill bundle 目录组织 built-in skills。
2. Agent 在初始化阶段预加载全部 built-in skills。
3. 第一版只提供 built-in `handoff` skill。
4. skill 的元数据与说明以 `SKILL.md` 为事实来源。
5. runtime 只实现最小的 bootstrap、配置和 handoff bridge。

## 3. 总体思路

### 3.1 不过度抽象

第一版明确不做：

- 自定义 `Skill Engine`
- 自定义 `SkillRuntime`
- 一套新的 `Match/Execute` 通用接口层
- 为了 skill 再造一套复杂注册表 DSL

第一版只做：

- built-in skill bundles
- 启动时预加载
- 针对 `handoff` 的最小 runtime bridge

### 3.2 built-in skill bundle

内建 skill 直接按 skill spec 的目录结构组织：

```text
skills/
└── builtins/
    └── handoff/
        ├── SKILL.md
        ├── references/
        └── assets/        # optional
```

说明：

- `SKILL.md` 包含 frontmatter 和技能说明；
- `references/` 存放补充说明；
- `assets/` 可用于模板或静态资源；
- 第一版不强制 `scripts/`。

这样未来如果要扩展更多 built-in skills，只需要继续增加标准 bundle。

## 4. 启动时预加载

Agent 在初始化时预加载全部 built-in skill bundles。

### 4.1 启动流程

```text
runtime init
  -> discover built-in skill bundle roots
  -> validate SKILL.md presence
  -> read skill metadata
  -> apply config:skill:{name}
  -> preload into agent
```

### 4.2 运行时最小对象

```go
type BuiltinSkillBundle struct {
    Name   string
    Root   string
    Config SkillConfig
}

type SkillConfig struct {
    Enabled  bool
    Defaults map[string]string
}

func LoadBuiltinSkillBundles() ([]BuiltinSkillBundle, error)
```

这里的对象只负责：

- 指出 skill bundle 在哪里；
- 读取配置；
- 参与启动时预加载。

它不定义新的 skill 运行模型。

## 5. 与 Agent Skills Spec 的关系

第一版直接使用 spec 的核心事实来源：

- `SKILL.md`
- 标准目录结构
- YAML frontmatter

也就是说：

- **metadata 的权威来源是 `SKILL.md`**
- 不是 Go 里的 `SkillSpec` 结构
- runtime 最多做轻量解析和校验，不重新定义完整 spec

这样更贴近 skill 的原生形态，也避免“spec-compatible but not really spec-driven”的重复抽象。

## 6. Built-in Handoff Skill

### 6.1 定位

`handoff` 是第一个 built-in skill，用于完成标准化阶段交接。

它的职责是：

1. 读取当前 session 的必要上下文；
2. 生成标准化 handoff payload；
3. 调用 Tape Service 的 `Handoff(...)`；
4. 返回结构化交接结果。

### 6.2 payload

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

### 6.3 为什么它适合作为 built-in skill

- `handoff` 是高层语义动作，不是单纯 tool；
- 它天然适合作为 agent 的内建能力；
- 它和 single-session tape / anchor 模型高度耦合，适合作为第一条跑通的 built-in skill。

## 7. Runtime Bridge

虽然不希望过度抽象 skill 概念，但 `handoff` 仍需要一个轻量 bridge 才能接到 Tape Service。

这个 bridge 只做一件事：把 skill 产出的 handoff payload 转成 runtime 可执行的 handoff 调用。

```go
type HandoffBridge interface {
    Handoff(ctx context.Context, sessionID string, input HandoffInput) (*HandoffResult, error)
}
```

约束：

- bridge 只服务 `handoff` skill；
- 不演化成通用 skill runtime；
- 不负责 skill 的匹配、发现或解释。

## 8. 与现有模块的集成

### 8.1 Agent 初始化

- runtime 启动时调用 `LoadBuiltinSkillBundles()`
- 把 `skills/builtins/*` 预加载进 agent
- 应用 `config:skill:{name}`

### 8.2 Agent Loop

- `Agent Loop` 不需要拥有一个新的 skill engine；
- 它只需要假设：agent 初始化后，built-in skills 已经可用；
- 在需要阶段交接时，agent 可以直接使用 built-in `handoff` skill。

### 8.3 Tape Service

- `handoff` skill 最终通过 `HandoffBridge` 调用 `TapeService.Handoff(...)`
- handoff entry 和 latest anchor 的语义保持不变

### 8.4 Config

继续沿用：

```text
config:skill:{name} -> Skill 配置
```

第一版主要用于：

- 开关 skill
- 默认 owner
- 默认 `phase_tag`
- handoff 默认模板参数

## 9. 审计与错误处理

### 9.1 审计

Tape 建议记录：

- `skill_name`
- `handoff_payload_summary`
- `handoff_result_summary`

默认不强制记录完整 `SKILL.md` 正文。

### 9.2 错误处理

- `SKILL.md` 缺失：启动时加载失败
- built-in skill 配置非法：启动时失败或禁用对应 skill
- handoff payload 生成失败：不写 Tape
- handoff entry 成功但 anchor 失败：沿用当前 tape 设计，保留 handoff 事实并显式报错

## 10. 第一版边界

第一版明确不做：

- 外部 skill marketplace
- 运行时按需发现技能
- 一套新的通用 skill 执行框架
- skill 目录热更新
- 非 handoff 的复杂 built-in skills 编排

第一版聚焦于：

1. built-in skill bundle 预加载
2. built-in `handoff` skill
3. handoff -> tape bridge

## 11. 结论

这版设计把 `skills` 当成 **agent 原生概念** 来处理：

- skill 本体直接按 spec bundle 组织；
- runtime 只做启动预加载和最小配置；
- `handoff` 作为第一个 built-in skill 接到 Tape/Handoff。

这样既满足当前需求，也避免为一个原生概念再套一层不必要的平台抽象。
