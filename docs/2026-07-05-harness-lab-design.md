# Harness Lab — Design Spec

Date: 2026-07-05
Status: approved
Vault project record: `dev/projects/harness-lab/index.md` (planning, backlog, session log)

## Overview

A hand-rolled minimal agent harness in Go, built stage by stage against the 8-part
control roadmap on the vault's `agent-harness` wiki page. The goal is to feel each
"commodity" harness problem firsthand (provider abstraction, skill loading, context
compaction, action validation, lifecycle hooks) rather than read about it built
elsewhere. This spec covers the architecture for all 6 backlog stages; implementation
plans are written and executed incrementally (see **Implementation Planning** below).

This started as a pure learning exercise but may evolve into, or directly influence,
a production tool — see the Open Questions section.

## Repo & Project Structure

Code lives in a new repo: `~/Source/harness-lab` (module path to be finalized when
`go.mod` is created — follow the `github.com/kbartnik/harness-lab` convention used by
sibling projects). The vault's `dev/projects/harness-lab/` stays the planning home:
`index.md`, `backlog/`, and the session log. `index.md`'s `repo:` field gets filled in
once this repo is pushed.

```
harness-lab/
  go.mod
  cmd/harness-lab/       # main.go — REPL entrypoint
  internal/core/         # Message, ToolCall, ToolResult, Turn, Role
  internal/provider/     # Provider interface + anthropic.go, openai.go
  internal/tool/         # Tool interface + read.go, write.go
  internal/loop/         # the five-step loop itself
  docs/                  # this spec and future design docs
```

## Core Architecture

**Core types (fixed at stage 1 — this vocabulary doesn't change per stage):**

```go
type Role string // "user" | "assistant" | "tool"

type Message struct {
    Role       Role
    Text       string
    ToolCalls  []ToolCall  // present on assistant messages that call tools
    ToolCallID string      // present on tool-result messages
}

type ToolCall struct {
    ID   string
    Name string
    Args map[string]any
}

type ToolResult struct {
    ToolCallID string
    Output     string
    IsError    bool
}
```

**Interface extraction principle: fixed core vocabulary, interfaces extracted at
second use** (standard Go idiom — accept concrete types, don't speculate interfaces —
applied deliberately here because it also matches the project's pedagogy: an interface
boundary should get discovered under real pressure, not designed in advance).

- **`Tool` interface exists from stage 1** — `read` and `write` are two implementations
  on day one, so the "second use" threshold is already met at the start:
  ```go
  type Tool interface {
      Name() string
      Schema() map[string]any
      Execute(args map[string]any) (ToolResult, error)
  }
  ```
- **`Provider` interface does not exist at stage 1.** Stage 1 calls
  `anthropic.SendMessage(...)` as a concrete function. Stage 2 extracts the interface,
  forced into existence by OpenAI's genuinely different tool-call/stop-reason shape —
  that normalization decision is the actual point of stage 2, and designing the
  interface before hitting the friction would pre-answer the question the stage exists
  to test.
- **`Validator` interface doesn't exist until stage 5**, **`Hook` interface doesn't
  exist until stage 6** — same reasoning, extracted only when each stage's second
  concrete instance (structural/semantic/policy; session-start/session-stop) appears.

## Stage-by-Stage Build Plan

| Stage | Backlog item | What gets added | Package(s) touched |
|---|---|---|---|
| 1 | Naked loop, one provider, two tools | Core types, `Tool` interface, concrete `anthropic.SendMessage`, the loop itself, CLI REPL | `core`, `tool`, `loop`, `cmd` (new) |
| 2 | Add second provider | Extract `Provider` interface from the stage-1 concrete call; add `openai.go`; document the one concrete normalization point (tool schema + stop-reason mapping) | `provider` (new) |
| 3 | Naive context resend, observe degradation | No new abstraction — deliberately let `loop` keep resending full history; add a session log capturing the observed degradation | `loop` (no interface change), session notes in vault |
| 4 | Add skill loading | New `skill` package: directory scan, frontmatter parse (catalog tier), lazy full-body load via the `read` tool | `skill` (new) |
| 5 | Action boundary validator | Extract `Validator` interface (structural/semantic/policy are the second/third use that justifies it); wire into `loop` before tool execution | `validator` (new), `loop` (modified) |
| 6 | Lifecycle hooks + permission model | Extract `Hook` interface (session-start/session-stop = second use); naive always-ask permission gate alongside it | `hook` (new), `permission` (new) |

Each stage after 1 is additive to `loop`'s call sites, not a rewrite: stage 2 swaps a
concrete call for an interface call, stage 5 inserts a check before `tool.Execute`,
stage 6 wraps the loop's entry/exit points. None require touching `core`'s type
definitions.

## Cross-Cutting Concerns

**CLI / REPL:**
```
$ harness-lab chat --workdir ./sandbox
> read the file foo.txt
[tool call] read(path="foo.txt")
[tool result] (contents...)
The file contains...
>
```
`--workdir` (default `./sandbox`, created if missing) is the root both `read` and
`write` are confined to — enforced inside `tool/read.go` and `tool/write.go` by
resolving the requested path and rejecting anything that escapes the root, not by
trusting the model's input. This gives stage 5's validator a concrete boundary to test
against later (a path-traversal attempt becomes a deniable case, alongside the source
material's "March 32nd" booking-date example).

**Providers:** Anthropic first (stage 1), OpenAI second (stage 2) — genuinely
different tool-call schema and stop-reason shape, which is what makes stage 2's
normalization decision real rather than academic.

**Calling style:** blocking (non-streaming) request/response for stages 1-2. No
partial-chunk assembly, no goroutines/channels for stream handling — the loop shape
and provider normalization are the point, not streaming UX. Streaming is deferred (see
Advanced Techniques).

**Persistence:** in-memory only, for the life of one REPL process, for now. Stage 3's
context-rot exercise just needs one long-running session; stage 6's lifecycle hooks can
still fire on process start/stop without durable storage. Persistence is deferred (see
Advanced Techniques).

**Error handling philosophy, by stage:**
- Stages 1-4: naive on purpose — a tool error becomes a `ToolResult{IsError: true}`
  message sent back to the model as-is; no retries, no recovery logic (matches backlog
  item 1's acceptance criteria: "no error handling beyond don't crash").
- Stage 5 changes this at exactly one point: before `tool.Execute` runs, not after —
  the validator either allows the call through unchanged or returns a deny `ToolResult`
  with a model-readable reason, so the loop's error-handling shape doesn't otherwise
  change.
- Recovery (retry/rollback/replay) is explicitly out of scope for the 6-stage backlog
  (see Advanced Techniques).

**Testing approach:**
- Unit tests for all deterministic logic: core type (de)serialization, provider
  request/response normalization, validator layers, skill frontmatter parsing,
  permission gate logic.
- For unfamiliar API shapes (first real Anthropic/OpenAI tool-call round trip), write
  exploratory tests first — hit the real API, assert on the actual response shape —
  before committing to the `core` type mapping. This is "test to learn the shape,"
  distinct from regression testing.
- No automated tests for stage 3's degradation observation or stage 6's permission-gate
  friction — both are explicitly manual/observational per their acceptance criteria (a
  written note, not an assertion).

## Advanced Techniques (Future Work)

Not part of the 6-stage implementation plan — called out explicitly so each can be
picked up as its own follow-on exercise once the core loop is solid.

**Deferred from this spec's scoping decisions:**
- **Streaming responses** — replace blocking `SendMessage` with SSE chunk assembly. A
  natural use of goroutines/channels once the loop shape itself isn't in question.
- **Durable session storage** — persist conversation history to disk (JSON/JSONL) so
  sessions survive process restarts; likely motivated once stage 3's context-rot work
  makes you want to inspect old sessions after the fact.
- **Tool sandbox loosening** — multiple allowed roots or an unrestricted mode, if the
  single-`--workdir` constraint ever gets in the way.

**Roadmap parts with no current backlog item** (from the 8-part control roadmap on
`agent-harness`, not covered by the 6 items above):
- **Recovery** — rollback/retry/replay for runaway loops (roadmap part 5).
- **Orchestration and subagents** — one agent doing too much (roadmap part 6);
  background at the vault's `subagent-orchestration` page.
- **Observability and drift detection** — beyond stage 6's basic lifecycle hooks
  (roadmap part 8); closes the "tool failure reported as success" failure mode.

**Stage stretch goals already noted inline in the backlog** (not required for that
stage's acceptance criteria):
- Stage 4: point the skill loader at this vault's own `.agents/skills/` directory as a
  real-world progressive-disclosure test.
- Stage 4: SkillOpt — a rollout/reflect/edit loop that improves skill text
  automatically, once triggering works reliably.

**Cross-reference, not duplicated here:** the Rust port (applying the ownership model
to context/state, once the Go version works end to end) is tracked in
`dev/projects/harness-lab/index.md`'s Future Work section.

## Open Questions

- **Toy vs. production tool.** Originally scoped as a pure learning exercise, but there
  is real possibility this evolves into, or directly informs, a production tool — not
  resolved yet, but worth keeping in mind when making stage 2+ decisions (e.g. don't
  choose the throwaway version of something if the sturdier version is nearly as cheap).
  Revisit explicitly once stages 1-2 are working.
- **Stage 3's "long enough" threshold.** The backlog says "a session long enough to
  visibly degrade output" — better discovered empirically (turn count / token budget /
  wall clock) when actually running it than guessed at now.

## Implementation Planning

This spec covers the full 6-stage roadmap, but implementation plans are written and
executed **incrementally, one to two stages at a time** — the next step is a plan for
stages 1-2 only (naked loop + second provider). Once those are working, the
writing-plans skill is re-invoked for the next stage(s), reading this same spec for
architecture context rather than re-deriving it.
