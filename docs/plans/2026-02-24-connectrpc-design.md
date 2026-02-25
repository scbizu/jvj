# ConnectRPC 设计改造（architecture.md）

## 背景

`docs/architecture.md` 的 3.1 章节存在 ConnectRPC 与 WebSocket/gRPC 语义混用问题，且示例代码有可实现性不一致点。

## 目标与范围

- 仅改造 `3.1 Transport Layer`（含 3.1.1~3.1.4）。
- 统一术语为 ConnectRPC 语义：单一 proto，同时支持 Connect/gRPC/gRPC-Web。
- 修正服务端与客户端示例，使其结构可作为实现蓝本。
- 不改动 3.2+ 章节。

## 方案

1. **总述改写**  
   明确 ConnectRPC 的多协议兼容关系，删除“Connect bidi streaming = WebSocket”表达，强调统一基于 HTTP 语义与流式能力。

2. **Proto 说明收敛**  
   保持接口不扩展，仅修正注释，避免把传输层能力绑定到 WebSocket。

3. **服务端示例修正**  
   保留 `NewAgentServiceHandler`、health/reflection、h2c；修复示例中的变量遮蔽与说明不一致，确保示例结构清晰可实现。

4. **客户端示例修正**  
   拆分为默认 Connect 客户端与 `connect.WithGRPC()` 兼容客户端，保留双向流式调用示例。

5. **优势表述统一**  
   优势改为“传输统一、流式支持（基于 HTTP）、多客户端互操作”。

## 验收标准

- 3.1 内术语一致，不再出现 Connect 直接等同 WebSocket 的说法。
- 3.1 示例代码自洽，语义与 ConnectRPC 一致。
- 3.2+ 内容不受影响。
