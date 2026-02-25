# Message Bus 细化设计（架构级）

## 背景

在 Message Bus + Discord/Telegram 主线已确定后，继续细化两块核心能力：
1) Command 执行安全模型（禁止直接 eval）  
2) 依赖安装策略（黑名单优先）

## 目标与范围

- 将 Message Bus 从“概念入口”提升为模块级设计（3.2）。
- 明确事件类型、路由状态、错误分层与审计要求。
- 保持架构级描述，不进入实现级伪代码细节。

## 设计决策

### 1) 模块重排

- 新增 `3.2 Message Bus Ingress`。
- 原 `3.2~3.7` 顺延为 `3.3~3.8`。
- 目的：把平台接入层职责（适配、校验、路由）与核心引擎职责清晰分离。

### 2) Command Policy（禁止 eval）

- 不接受原始 shell 字符串直执行，不以 `sh -c` 作为主路径。
- 执行链路固定为：`BusEvent(command)` → `Router.Parse` → `ToolRegistry.SchemaValidate` → `Sandbox Executor(argv)`。
- 约束：固定 cwd、非 root、超时、资源配额、输出上限、审计必填。

### 3) Dependency Install Policy（黑名单优先）

- 以 `system` 事件类型承载 `deps.install`，与普通用户命令分流。
- 允许常规生态安装流程（go/npm/pip），禁止高风险命令/参数：
  - `curl|bash`
  - `sudo`
  - `npm -g`
  - 任意远程脚本 URL 直执行
- 安装失败必须显式回传，不允许静默降级。

### 4) 事件与错误模型

- 事件类型：`message | command | callback | system`。
- 状态流转：`received -> validated -> routed -> executing -> replied/failed`。
- 错误分层：
  1. Platform Error（签名/API/限流）
  2. Policy Reject（命令或依赖策略拒绝）
  3. Execution Error（工具或运行时失败）
- 所有错误必须携带 `correlation_id` 并写入 Tape。

## 文档落地范围

- `docs/architecture.md`
  - 新增/重排第 3 章模块结构
  - 强化 6.2 Message Bus 协议
  - 在 TOML 配置增加 command/deps policy 节

## 验收标准

- 文档存在独立 `3.2 Message Bus Ingress` 模块。
- 文档明确“禁止直接 eval”与“黑名单优先依赖安装策略”。
- 6.2 包含事件类型、状态机与分层错误处理。
