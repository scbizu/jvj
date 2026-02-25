# Message Bus + Bot Access Design

## 背景

当前方向从 Agent-to-Agent P2P 调整为消息总线优先，先支持 Discord 与 Telegram Bot 访问。

## 目标与范围

- 从架构主线中移除 P2P 方案。
- 以 Message Bus 作为入口，优先支持 Discord/Telegram。
- 保留 ConnectRPC 作为核心服务接口，不改动既有 3.1 服务定义。
- 本轮为文档级设计改造，不包含代码实现。

## 方案

1. **架构层**
   - 将整体架构中的 `P2P Transport (Optional)` 替换为 `Message Bus Ingress (Primary)`。
   - 主链路：Discord Bot/Telegram Bot → Bus Router（事件标准化）→ AgentService。

2. **概念层**
   - 新增 `ChannelAdapter`：平台事件与内部事件转换。
   - 新增 `BusEvent`：统一事件载体（`message`/`command`/`callback`）。

3. **协议层**
   - 将 6.2 改为 `Message Bus 协议（Discord/Telegram）`。
   - 统一流程：平台事件 → 适配器标准化 → Bus Router 分发 → 平台回复适配。

4. **配置层（TOML）**
   - 新增 `[message_bus]` 总配置。
   - 新增 `[message_bus.discord]` 与 `[message_bus.telegram]` 子配置。

5. **目录层**
   - `internal/transport/p2p.go` 替换为 `internal/transport/message_bus.go`。
   - 新增 `internal/adapters/discord.go`、`internal/adapters/telegram.go`。

## 错误处理约束

- 平台签名校验失败必须拒绝并记录审计日志。
- 平台限流错误需要显式重试策略与退避。
- 平台 API 错误必须上报到统一错误通道，不静默吞掉。

## 验收标准

- `docs/architecture.md` 中不再出现 P2P 作为当前主线路径。
- Discord/Telegram 的入口、路由、配置、目录结构描述完整且一致。
- 关键术语统一为 Message Bus + Bot Access。
