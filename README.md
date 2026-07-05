# Harness Lab

A hand-rolled agent harness in Go, built stage by stage to develop first-hand,
demonstrable competency in agent harness engineering — the discipline
Anthropic's own [Claude Certified Architect: Foundations (CCA-F)](https://medium.com/towards-artificial-intelligence/claude-certified-architect-the-complete-guide-to-passing-the-cca-foundations-exam-9665ce7342a8)
exam certifies.

## Why this exists

CCA-F tests practitioners on a specific set of patterns: tool-call loops,
provider/protocol abstraction, context management under degradation,
action-boundary validation, lifecycle hooks, and permission models. Industry
commentary on the exam has gone as far as calling it, in substance, "a
certification in harness engineering" — the patterns map almost one-to-one.

This project is a deliberate alternative (or complement) to studying for that
exam by recall: instead of preparing to recognize the right answer on a
multiple-choice question, I build the thing the question is about, from
scratch, and prove it works with tests. Two concrete examples already hit in
this build:

- **"Hard guarantees belong in code, not prompts."** This shows up in the CCA-F
  material as a hooks-domain principle. Stage 6 of this project builds that
  guarantee directly — a lifecycle hook and a permission gate that enforce a
  boundary structurally, not by asking the model nicely.
- **Self-evaluation bias.** The CCA-F material names "Independent Review
  Instance vs. Self-Review" as an exam topic. Stage 5's action-boundary
  validator is a working instance of exactly that pattern: a check that runs
  *before* a tool executes, structurally separate from the model that
  requested the action.

The output of this project isn't a test score — it's a working system, a set
of design decisions I can defend in detail, and a body of code and tests that
demonstrate the competency directly rather than asserting it.

## What it is

A minimal agent harness: a tool-calling loop against a real LLM provider, with
each "commodity" harness problem (provider abstraction, context management,
skill loading, action validation, lifecycle hooks, permissions) added
deliberately, one stage at a time, in the order a harness actually needs to
solve them. Two providers (Anthropic, then OpenAI) are used specifically to
force real design decisions rather than working against a single API's
assumptions.

This is not aimed at replacing Claude Code or any other production agent
tool — the goal is understanding the mechanisms well enough to design, audit,
and harden agent tooling built on them, not to compete with mature tools.

## Build stages

| Stage | What it adds | Harness concept |
|---|---|---|
| 1 | Naked tool-call loop, one provider, two tools (`read`, `write`) | The five-step loop, sandboxed tool execution |
| 2 | Second provider (OpenAI) | Provider/protocol abstraction — normalizing genuinely different tool-call and stop-reason shapes behind one interface |
| 3 | Naive context resend, observed until output degrades | Context rot, made to happen on purpose before being fixed |
| 4 | Skill loading | Progressive disclosure — catalog-tier metadata with lazy full-body load |
| 5 | Action-boundary validator | Structural/semantic/policy checks before tool execution — independent review vs. self-review |
| 6 | Lifecycle hooks + permission model | Hard guarantees enforced in code, not in the prompt |

Each stage is implemented test-first, with a working, runnable system at the
end of every stage — not a big-bang build at the end.

## Status

Currently in progress. See commit history for the stage currently underway.

## Tech stack

Go, standard library only for provider integration (`net/http` +
`encoding/json`, no vendor SDKs) — so the actual wire-level behavior of each
provider's tool-calling API is visible in the code, not abstracted away by a
client library.

## Non-goals

- Not a production agent tool, though the design leaves that door open rather
  than closing it off for convenience (see Open Questions in the design spec).
- Not aiming for provider or feature completeness — each stage builds exactly
  enough to make its underlying decision real, then stops.
