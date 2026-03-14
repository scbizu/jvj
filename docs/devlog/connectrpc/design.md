# ConnectRPC Design

## Background

The `3.1 Transport Layer` section in `docs/architecture.md` mixed ConnectRPC language with WebSocket/gRPC semantics, and the examples did not read as a clean transport-layer blueprint.

## Goal and Scope

- Refine only `3.1 Transport Layer` and `3.1.1` through `3.1.4`.
- Normalize the protocol description around ConnectRPC: one proto surface with Connect, gRPC, and gRPC-Web compatibility.
- Correct the server and client examples so they express a realistic service boundary.
- Leave `3.2+` unchanged.

## Design Direction

1. **Transport layer summary**  
   Reframe the section around ConnectRPC’s multi-protocol compatibility and remove any wording that equates Connect bidirectional streaming with WebSocket.

2. **Protocol notes**  
   Keep the service contract stable and only refine comments so transport capability is described in ConnectRPC terms.

3. **Server-side service boundary**  
   Preserve `NewAgentServiceHandler`, health/reflection, and h2c, while cleaning up wording so the example clearly represents the service boundary exposed by the transport layer.

4. **Client-side protocol examples**  
   Separate the default Connect client path from the gRPC compatibility path using `connect.WithGRPC()`, while keeping the bidirectional streaming example intact.

5. **Benefits language**  
   Normalize the stated advantages to transport unification, HTTP-based streaming support, and client interoperability.

## Acceptance Criteria

- The 3.1 section uses consistent ConnectRPC terminology and no longer treats Connect as WebSocket.
- The examples in 3.1 are internally consistent and aligned with the intended protocol/service boundary.
- `docs/architecture.md` sections `3.2+` are untouched.
