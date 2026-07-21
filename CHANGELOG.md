# Changelog

## Stage 1 — Naked tool-call loop

The foundational five-step loop: receive user input, send to the model, execute
any tool calls, append results, repeat until the model stops. Establishes the
core architectural decisions the rest of the project builds on.

**What was built**
- `core` — shared wire types: `Message`, `ToolCall`, `ToolResult`, `Turn`
- `provider` — Anthropic Messages API client with request/response conversion
- `tool` — `Tool` interface with sandboxed `Read` and `Write` implementations
- `loop` — agent loop with circuit breaker (`maxToolIterations`) and defensive slice copy
- `cmd/harness-lab` — REPL entrypoint wiring all packages together

**Key decisions**
- Handled tool errors (sandbox violations, file not found) are returned as `ToolResult` content with `IsError: true` — the model sees them and can reason about them, rather than the loop aborting
- Unknown tool names are fed back to the model as error content rather than hard-failing
- The loop never mutates the caller's message slice (defensive copy on entry)
- Provider integration uses `net/http` + `encoding/json` directly — no SDK — keeping the wire-level behavior visible in the code
