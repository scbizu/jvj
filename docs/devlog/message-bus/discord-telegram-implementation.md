# Message Bus Discord Telegram Implementation

## Goal

- Confirm Message Bus plus Discord/Telegram as the primary runtime path.
- Finish the documentation-level description of module boundaries, protocol flow, config shape, and directory layout.
- Leave code implementation for a later execution phase.

## Current Delivery

1. `docs/architecture.md` already reflects the mainline shift:
   - P2P removed from the active path
   - Message Bus ingress added
   - TOML config expanded with `message_bus.discord` and `message_bus.telegram`
2. `docs/devlog/message-bus/bot-design.md` captures the design background, scope, constraints, and acceptance criteria.
3. This file records the documentation-phase implementation handoff for later code work.

## Runtime Backlog

1. **Message Bus core abstraction**
   - `internal/transport/message_bus.go`
   - normalized `BusEvent` and router dispatch interfaces
2. **Discord platform adapter**
   - event ingress
   - signature validation
   - reply delivery
3. **Telegram platform adapter**
   - webhook/polling ingress
   - event parsing
   - reply delivery
4. **Startup wiring and config loading**
   - attach the two platform integrations from `cmd/agent-runtime/main.go`
   - validate required `message_bus` fields in `config/example.toml`

## Acceptance Criteria

- The architecture docs no longer describe P2P as the current implementation path.
- Message Bus, Discord, and Telegram use consistent ingress, protocol, config, and directory language.
- The implementation backlog clearly marks runtime code as a later phase.
