# Message Bus Bot Access Design

## Background

The architecture direction has shifted away from agent-to-agent P2P and toward a message-bus-first ingress model with Discord and Telegram as the primary platform integrations.

## Goal and Scope

- Remove P2P from the current architecture path.
- Establish Message Bus as the primary ingress for Discord and Telegram traffic.
- Preserve ConnectRPC as the core service boundary without changing the existing `3.1` service definition.
- Keep this round at the documentation-design level only.

## Design

1. **Ingress architecture**
   - Replace `P2P Transport (Optional)` with `Message Bus Ingress (Primary)`.
   - Use the primary flow: Discord Bot / Telegram Bot -> platform adapter -> bus router -> `AgentService`.

2. **Core concepts**
   - Add `ChannelAdapter` for mapping platform-specific events into internal bus events.
   - Add `BusEvent` as the normalized event envelope for `message`, `command`, and `callback`.

3. **Protocol layer**
   - Reframe section `6.2` as the Message Bus protocol for Discord and Telegram.
   - Normalize the flow as platform event -> adapter normalization -> bus router dispatch -> reply adapter.

4. **Config layer**
   - Add a root `[message_bus]` block.
   - Add `[message_bus.discord]` and `[message_bus.telegram]` for platform-specific integration settings.

5. **Directory layer**
   - Replace `internal/transport/p2p.go` with `internal/transport/message_bus.go`.
   - Add platform integration files such as `internal/adapters/discord.go` and `internal/adapters/telegram.go`.

## Error Handling Constraints

- Failed platform signature validation must be rejected and written to the audit log.
- Platform rate-limit errors must use explicit retry and backoff behavior.
- Platform API failures must be reported through the shared error channel instead of being silently swallowed.

## Acceptance Criteria

- `docs/architecture.md` no longer treats P2P as the mainline runtime path.
- Discord and Telegram ingress, routing, config, and directory structure are described consistently.
- The key language is normalized around Message Bus ingress and platform adapters.
