# ADR-002: Bounded Agent Run Ownership

## Status

Accepted architecture decision for P9-A1, 2026-09-13.
Phase 9: **ARCHITECTURE SELECTED / IMPLEMENTATION NOT STARTED**.
P9-A0: **CLOSED / PASS**. This ADR authorizes no P9-B1 implementation.
P9-A1 changes documentation only.

## Context and evidence

The [roadmap](../../../ROADMAP.md) names Agent Runtime as unfinished work.
Phase 8 is **CLOSED / PASS** for the bounded AI/tool foundation at
`ac68a1b3e059f173d9b5c71eadaff9fcffba981f`, tree
`e4147daf4e0312e4622266dff5284c95c06ac811`.
Prompt execution, B1/B1-HYG/B2, C1–C7, C9, and C10 are integrated.
C8 is the accepted real-provider validation gate, not a code package.
Owner-provided evidence covers authentication, direct text, a real function
proposal, authorized local handler execution, `function_call_output`, the
stateless second provider turn, final response, and runtime identity 3/3 MATCH.
[Push-main CI 34559981995](https://github.com/kaizenforyou91/forge/actions/runs/34559981995)
passed Ubuntu acceptance, Windows acceptance, and Ubuntu race on that SHA.
These are existing acceptance records; this decision performs no live call.

Memory, Workflow Engine, and scheduler contracts remain undefined. No next
provider requirement has been identified. Tool authority expansion would
enlarge side effects prematurely. The next bounded increment is ownership of
one explicitly requested AI operation, not another execution primitive.

The [Architecture Freeze v1.0](../ARCHITECTURE_FREEZE_V1.md) requires explicit
concurrency, context propagation, owned work, deterministic lifecycle, small
APIs, and downward dependencies. Its [accepted ADR-001](ADR-001-architecture-freeze.md)
remains unchanged. This decision complies with that freeze; it does not alter
its application lifecycle or require a replacement architecture version.

This ADR belongs to `docs/architecture/adr/`. The separate historical
repository-layout ADR series in `docs/adr/` is not moved or renumbered.

## Decision

Select **Phase 9 — Bounded Agent Execution Lifecycle**. One explicitly requested
AI operation receives one execution owner, observable lifecycle state, explicit
cancellation coordination, and deterministic terminal completion.

P9-B1 will use an internal, pre-stable `Run` object in `internal/agent`.
It will not create `pkg/agent`. The existing
[`ai.Provider`](../../../pkg/ai/types.go) contract MUST NOT change, and
[`ai.Executor`](../../../pkg/ai/executor.go) remains the execution primitive.
Run owns the operation lifecycle around that primitive, not HTTP transport,
provider response parsing, timeout replacement, or tool execution.

The intended API shape below is architecture intent only, not implemented code:

```go
NewRun(
    provider ai.Provider,
    request ai.Request,
    timeout time.Duration,
) (*Run, error)

(*Run).Execute(context.Context) (ai.Result, error)
(*Run).Cancel()
(*Run).State() State
(*Run).Done() <-chan struct{}
```

Construction validates existing AI request/provider/timeout contracts through
the existing primitive and validators. It performs zero provider I/O, no
credential lookup, and no hidden execution. State is a fixed classification;
observing it grants zero authority. Nil/incomplete Run misuse must fail closed
without delegated work rather than treating a zero value as a constructed Run.

## Operation lifecycle and execution claim

```text
Ready -> Running -> Succeeded
                 -> Failed
                 -> Canceled

Ready -- Cancel() --> Canceled
```

There is no restart, reset, second execution, retry, hidden execution, or
background scheduling. The Run lifetime admits at most ONE delegated AI
execution attempt; the text primitive may reject a request/context before
calling its provider, so an admitted Run can still make zero provider calls.

- A nil Execute context fails before claiming the Run.
- Other pre-claim invalid input fails without provider work or consumption.
- The first valid Execute atomically claims `Ready -> Running`.
- Concurrent Execute calls cannot both claim the Run.
- Any Execute after the Run has been claimed returns a fixed, payload-free
  consumed-run error, whether the first execution is running or terminal.
- Pre-start Cancel also consumes the Run. Subsequent Execute returns the same
  consumed-run classification without provider work.
- A valid but already-canceled/deadline-expired caller context cannot cause
  provider work: its claimed execution terminates Canceled through the context
  contract. It cannot make the Run executable again.

## Synchronous ownership and cancellation

P9-B1 MUST NOT create a worker goroutine. Execute delegates synchronously on
the caller's goroutine. The caller decides whether to call Execute from its own
goroutine. Run owns state transitions, cancellation coordination, terminal
publication, and Done closure. It is not a queue, scheduler, worker pool, or
background agent runtime.

| Cancel timing | Required behavior |
|---|---|
| Ready | Atomically publish Canceled, perform zero provider calls, close Done exactly once |
| Running | Record the cancellation request and invoke the run-owned cancellation function; do not publish terminal completion until delegated execution returns |
| Terminal | No state change, re-execution, or terminal rewrite |

The caller's context/deadline and the existing ai.Executor timeout remain
authoritative. Run may derive its cancellation context from the supplied
caller context; it must not detach to a background context or replace the
executor timeout with a custom timeout mechanism. The execution claim and
cancel-function publication must be coordinated so a racing Cancel cannot be
lost. Cancellation and state observation must remain safe during execution.

The provider must honor context and finish owned work before returning, as
required by the existing Provider contract. Run cannot forcibly terminate a
non-cooperative provider and must not manufacture early completion by leaving
it running in a detached goroutine.

## Terminal precedence

Terminal publication and Cancel must have a synchronized ordering. The table
applies once delegated execution has returned; a cancellation request alone
does not make a running operation terminal.

| Delegated outcome / publication ordering | Terminal state | Returned outcome |
|---|---|---|
| Success, with no cancellation or expired context winning before publication | Succeeded | Validated result, nil error |
| Success, but run cancellation, caller cancellation, or deadline wins before success publication | Canceled | Zero result; preserve `context.Canceled` or `context.DeadlineExceeded` identity |
| Context-only failure, including executor timeout | Canceled | Zero result; preserve existing context/deadline classification |
| Non-context failure, including a failure mixed with cancellation | Failed | Zero result; preserve existing safe AI failure classifications |

Use the existing [`ai.SafeError`](../../../pkg/ai/errors.go) behavior. Do not
reinterpret an independent provider failure as success or pure cancellation
because cancellation raced. Mixed failures remain failures; recognizing one
context category is not sufficient to discard other safe error categories.
An explicit Run Cancel uses context cancellation identity; an already-reported
deadline failure must not be relabeled as ordinary cancellation.

After terminal publication, late cancellation cannot rewrite Succeeded,
Failed, or Canceled. No partial result is returned on failure or cancellation.

## Done contract

- Done is non-nil from successful construction and always returns the same
  run-owned receive-only channel.
- It closes exactly once and only after terminal state is published.
- It may close immediately for pre-start Cancel.
- It never closes merely because execution was requested or cancellation was
  requested while delegated work is still running.
- It cannot be reopened. No result is sent through Done.

Observers may use Done to know that this operation's lifecycle is terminal;
they receive neither execution authority nor result payload through it.

## Observability and data lifetime

State exposes only a fixed lifecycle classification. State, Done, errors, and
diagnostics must never expose prompts, model responses, API credentials,
provider internals, tool arguments, call IDs, reasoning, or raw provider
errors. They must not surface a model identifier from the stored request;
separately supplied caller metadata does not authorize payload disclosure by
Run. P9-B1 adds no model-metadata observability API.

Run may retain its bounded ai.Request only as required for its one execution.
After terminal publication it should release internal stored-request and
executor/provider references where practical, avoiding unnecessary lifetime
extension. This is reference release, NOT secure memory erasure.
The result is returned to the executing caller, not retained as run history.
No request, result, error, or history is persisted. No unbounded event list or
global collection is introduced; Run holds bounded per-operation state only.

## Authority and side-effect boundaries

The CALLER supplies ai.Provider, constructs Run, and explicitly invokes Execute.
Construction alone does not execute. State, provider output, and Cancel cannot
authorize or create execution. `ADMITTED != EXECUTED` remains a Forge-wide
principle. P9-B1 contains NO tool authority or handler dispatch; tool composition
belongs to later P9-B2.

P9-B1 introduces no filesystem, network, or subprocess handler. Its only
delegated work is the existing caller-supplied text execution primitive; that
provider retains its existing side-effect and context contract. Existing CLI
network permission remains explicit and is not bypassed or changed by this
internal architecture.

Phase 8 stays unchanged: exactly one exposed built-in, read-only
`forge_runtime_info`; explicit default-false `--allow-tools`; at most two
provider POSTs and one handler attempt; no retry or recursive tool loop.
`store:false`, `stream:false`, `background:false`, `parallel_tool_calls:false`,
POST #2 `tool_choice:"none"`, no previous-response/conversation/response-ID
state, no reasoning replay or encrypted-reasoning include remain intact.
Provider proposals remain fail closed, authority is explicit and immutable,
replay protection remains invocation-local, and diagnostics remain fixed and
safe. Provider output never creates authority; credentials are never persisted.

## Persistence boundary

P9-B1 has NO file writes, database, config store, serialized Run, resume,
durable job ID, global run registry, execution history, or AI memory.
Run identity means in-process object ownership only. Cancellation does not
imply rollback of provider processing, and this design promises no durable
exactly-once or crash-recovery semantics.

## Application lifecycle separation

[`pkg/app.Runtime`](../../../pkg/app/runtime.go) owns the application lifecycle.
`internal/agent.Run` will own one AI operation lifecycle. Run does not replace
or reimplement application startup, module registration, restart, or shutdown.
The frozen application lifecycle model remains authoritative.

P9-B1 MUST NOT import the application lifecycle package, create another
application Runtime, add module start/stop hooks, auto-run at application
startup, or wire application shutdown. Application-host composition belongs
only to provisional P9-B3 after a separate gate. No lower-layer `pkg` package
may depend on `internal/agent`.

## Alternatives and consequences

- Reuse ai.Executor alone: retains correct request execution and deadlines,
  but does not provide a single-use Run claim, observable operation state, or
  a shared completion signal. Reuse it underneath Run rather than duplicate it.
- Extend pkg/app.Runtime: would conflate application lifetime with one AI
  operation. Keep operation ownership separate and defer host composition.
- Start with memory, workflow, or scheduling: introduces undefined ownership,
  retention, timing, and failure policies before a bounded operation owner.
- Start with provider/tool expansion: no concrete next provider requirement
  exists; broader tool authority would enlarge side effects unnecessarily.

This decision adds a small internal lifecycle contract that can later support
explicit host composition without changing the AI/provider primitives.
Its costs are synchronization and cancellation/publication race testing.
It intentionally provides neither a complete autonomous agent runtime nor a
hard-stop guarantee for a broken custom provider. Keep the API internal until
its consumers and later composition contracts are demonstrated.

## Package sequence and authorization

| Package | Intended scope | Status |
|---|---|---|
| P9-A1 | Architecture decision + roadmap reconciliation | Documentation/governance only |
| P9-B1 | Single-use AI Run ownership over existing text execution | Architecture intent; separate implementation authorization required |
| P9-B2 | Existing authorized tool round-trip with Run lifecycle, preserving Phase 8 bounds and aggregate-usage semantics | PROVISIONAL; no implementation authorization |
| P9-B3 | Bounded application-host/shutdown composition | PROVISIONAL; separate gate, no implementation authorization |
| P9-C0 | Offline integration / architecture closure | PROVISIONAL; no implementation authorization |

Memory, workflow, and scheduler remain separate future gates. This ADR does
not authorize any implementation package or broaden the Phase 8 boundary.

## P9-B1 acceptance intent

Future P9-B1 must prove, using deterministic offline tests:

- Existing AI construction contracts are validated with zero provider I/O.
- At most one delegated provider execution exists per Run lifetime.
- Concurrent/repeated Execute cannot double-run; pre-claim invalid input and
  consumed-run errors are fixed, safe, and perform no provider work.
- Pre-start Cancel makes zero provider calls; in-flight cancellation propagates.
- Done is stable/non-nil, closes exactly once after terminal publication, and
  running completion is published only after delegated work ends.
- Cancellation/success ordering is deterministic; late cancellation cannot
  rewrite terminal state; deadlines preserve context identity.
- Independent non-context provider failure remains Failed, including mixed
  failures under existing safe error behavior.
- Prompt/result/credential canaries cannot leak through state or diagnostics.
- There is no retry, goroutine created by Run, persistence, or tool execution.
- Tests use fake providers, controlled channels, and virtual time as needed;
  no live provider or actual environment credential access is required.
- Hosted race coverage includes the future package, alongside normal offline
  package listing, full tests, vet, build, and dependency cleanliness checks.

No tests, Go source, or CI changes are implemented in P9-A1. Local race is not
required for this documentation-only package; hosted PR CI remains canonical.

## Out of scope

- Autonomous planning or autonomous loops.
- Multi-agent execution or recursive provider conversations.
- Memory/history, durable jobs, or persistence.
- Workflow engine, scheduler, queues, or worker pools.
- New provider or provider routing.
- Arbitrary tools, new built-in tools, or filesystem/network/subprocess agent handlers.
- CLI agent command or application lifecycle wiring.
- Native-process expansion.
- Release or tag creation.

## Release status

**RELEASE DEFERRED.** Phase 8 is accepted, but current release documentation
requires broader synchronization before a publication decision. README and
CHANGELOG reconciliation remain a separate release-readiness package.
P9-A1 does not change them, release identity, or the Architecture Freeze.
Published `v0.3.0-alpha.1` remains at
`5d836931216203aeea0737fc54de9e95091a62ef`; no tag is created or moved.
