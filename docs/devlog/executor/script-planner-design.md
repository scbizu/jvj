# Scripted Executor Design

## Background

当前设计里，命令执行路径仍接近“解析后直接交给执行器跑命令”。这会带来两个问题：

- 执行层过早绑定到单条命令形状，复杂问题不够灵活；
- 每次失败都容易回到重新组织命令的模式，增加模型 token 消耗。

本设计将命令执行类 tool call 收敛为三段式：**先分析，再生成一次性临时脚本，最后执行脚本**。

## Goals

1. 命令执行类 tool call 不直接携带最终 shell 命令。
2. 执行前先产出结构化 `ExecutionPlan`。
3. 执行层统一把 plan 编译成一次性临时脚本。
4. 失败重试优先基于已有 plan 做局部修订，减少重新推理成本。
5. 保持 single-session、append-only、可审计的运行时约束。

## 3. 总体架构

### 3.1 三段式执行链

```text
ToolCall(command-like)
  -> ToolPlanner
  -> ExecutionPlan
  -> ScriptBuilder
  -> ScriptArtifact(temp shell script)
  -> Sandbox Script Executor
  -> ExecutionResult
```

职责拆分：

- `ToolPlanner`：理解问题和目标，输出结构化执行计划。
- `ScriptBuilder`：把计划编译成一次性脚本，并统一注入执行约束。
- `Sandbox Script Executor`：只负责受控执行、采集结果和清理。

## 4. 核心对象

### 4.1 ExecutionPlan

```go
type ExecutionPlan struct {
    Goal          string
    Preconditions []CheckSpec
    Steps         []PlanStep
    Expected      []ArtifactSpec
    Cleanup       []CleanupStep
}
```

语义：

- `Goal`：本次执行要解决的问题。
- `Preconditions`：执行前必须检查的条件，如文件存在、工具可用、目录正确。
- `Steps`：脚本要执行的分步动作。
- `Expected`：预期产生的文件、输出或状态变化。
- `Cleanup`：执行后的清理动作。

### 4.2 ScriptArtifact

```go
type ScriptArtifact struct {
    Path    string
    Hash    string
    Content string
}
```

语义：

- 脚本是一次性临时文件。
- 默认执行后删除。
- 运行时默认只在 Tape 中记录 `hash / path / 生命周期`，而不是完整正文。

### 4.3 ExecutionResult

```go
type ExecutionResult struct {
    ExitCode  int
    Stdout    string
    Stderr    string
    Retryable bool
    Artifacts []ArtifactSummary
}
```

语义：

- `Retryable` 由执行器结合退出码和失败类型判断。
- `Artifacts` 用于摘要本次执行实际产物。

## 5. 执行数据流

1. `Router` 或 `Model Runner` 识别出命令执行类 tool call。
2. `ToolPlanner` 基于当前任务、上下文和约束生成 `ExecutionPlan`。
3. `ScriptBuilder` 将 `ExecutionPlan` 编译为一次性临时脚本。
4. 执行器统一加上：
   - `set -euo pipefail`
   - 固定 cwd
   - 环境白名单
   - 超时包装
   - 日志采集
5. `Sandbox Script Executor` 执行脚本，得到 `ExecutionResult`。
6. `Tape` 记录 plan 摘要、script 摘要和执行结果摘要。

## 6. 失败与重试策略

失败后按三层处理：

### 6.1 局部修订

若失败原因只是参数、顺序、目录或环境检查问题：

- 保留原 `ExecutionPlan`
- 调整局部 step 或 precondition
- 重新生成临时脚本并执行

### 6.2 计划修订

若失败表明当前方案本身不完整：

- 回到 `ToolPlanner`
- 基于已有 plan + `ExecutionResult` 生成新 plan
- 再编译新脚本

### 6.3 明确终止

若失败属于黑名单、权限限制、资源上限或策略禁止：

- 不做脚本重编译
- 直接返回显式错误

## 7. 为什么这样更省 token

相比“每次都直接生成 bash 命令再试”，三段式模型可以把 token 主要花在：

- 第一次生成 `ExecutionPlan`
- 必要时修订少量 step

而不是每次都重新展开完整命令推理。  
大多数失败可以只根据执行结果做局部修正，不需要回到问题起点重新分析。

## 8. 与现有模块的对齐

### 8.1 Command Policy

- 继续禁止 raw shell eval。
- 新增约束：命令执行类 tool call 必须先经过 planner 和 script builder。

### 8.2 Tool Engine

- 从“工具注册和执行”升级为“工具注册、执行规划和脚本化执行”。
- 对纯读取类 tool call 不强制生成脚本。
- 对命令执行类 tool call 统一走 planner -> script builder -> executor。

### 8.3 Tape Service

Tape 建议记录：

- `execution_plan_summary`
- `script_hash`
- `script_path`（可选，短期）
- `execution_result_summary`

默认不记录完整脚本正文，除非调试或审计显式要求。

## 9. 非目标

本轮设计不覆盖：

- 所有 tool call 都强制脚本化；
- 持久化可复用脚本库；
- 多 session 下的共享脚本缓存；
- 直接让模型输出完整最终 shell 作为执行主路径。

## 10. 结论

命令执行类 tool call 的推荐路径是：

**先分析目标 -> 生成结构化计划 -> 编译一次性临时脚本 -> 受控执行 -> 基于结果局部修订**

这样能同时满足：

- 更灵活的复杂问题处理；
- 更低的重复 token 消耗；
- 更清晰的执行边界与审计能力。
