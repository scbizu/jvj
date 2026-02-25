# Message Bus Discord/Telegram 实施计划（文档阶段）

> 当前按你的要求：**只更新架构与实施计划文档，不创建代码**。

## 目标

- 明确系统主线路径为 Message Bus + Discord/Telegram Bot。
- 完成文档层面的模块边界、协议流程、配置结构与目录约定。
- 将代码实现留到后续阶段。

## 当前交付（本阶段）

1. `docs/architecture.md` 已完成主线调整：
   - 移除 P2P 主线；
   - 增加 Message Bus Ingress；
   - 配置改为 TOML 并新增 `message_bus.discord/telegram`。
2. `docs/plans/2026-02-24-message-bus-bot-design.md` 已记录设计背景、范围、约束与验收标准。
3. 本文件作为“文档阶段实施计划”，定义后续代码化入口。

## 后续代码化 Backlog（下一阶段）

1. Message Bus 核心抽象
   - `internal/transport/message_bus.go`
   - 统一 `BusEvent` 与路由分发接口
2. Discord Adapter
   - 事件接入、签名校验、响应发送
3. Telegram Adapter
   - Webhook/Polling 接入、事件解析、响应发送
4. 启动装配与配置加载
   - 在 `cmd/agent-runtime/main.go` 挂载两个适配器
   - 校验 `config/example.toml` 的 message_bus 必需项

## 验收标准（文档阶段）

- 架构文档中不再把 P2P 作为当前实现主线。
- Message Bus、Discord、Telegram 三者在架构图、协议、配置、目录描述一致。
- 实施计划明确标注“代码实现延期”。
