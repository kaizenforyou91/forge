# ADR-002: Bounded Agent Run Ownership

## Status

Accepted architecture decision for P9-A1, 2026-09-13; implemented through P9-B3.
Phase 9: **CLOSED / PASS — BOUNDED AGENT EXECUTION LIFECYCLE**.
P9-C0: **CLOSED / PASS — INTEGRATED**. No further runtime implementation package
is required. P9-C0 recorded offline validation and closure of the existing
bounded foundation; PR #20 merged and strict push-main CI passed as recorded below.

P9-A0 is **CLOSED / PASS**. P9-A1 approved architecture and changed documentation
only; it did not authorize B1/B2/B3 implementation. Each subsequently received
separate Control Room authorization and is now integrated. The original
decision and B1-specific boundaries below are preserved as historical scope;
the implementation record describes the separately authorized B2/B3 composition.

## Original P9-A1 context and evidence

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

## Original P9-A1 decision and B1 contract

The following decision, lifecycle rules, and alternatives record the P9-A1
architecture as approved. References to future B1/B2/B3 work in this original
record describe that checkpoint, not the current implementation status.

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

## Implementation record and bounded architectural outcome

The three runtime implementation packages and P9-C0 documentation closure are
**CLOSED / PASS — INTEGRATED**:

| Package | Implementation commit | Merge commit |
|---|---|---|
| P9-B1 | `e26e72d4986b721af5ebd204061b992e60c88c1e` | `db732b362ac2e6d6c55df6148b67e2c16dbceaed` |
| P9-B2 | `30e98b320464ce175fa9a60cbb4fc8a8b107d635` | `6c2700ad5a83e03673bdebacd0b5363ce6bc5e38` |
| P9-B3 | `e711907ddfa6f26fbe01174c271f80c18fd9aa55` | `d7d660f65fc5fc794a545f8b64703562e92b8bc7` |
| P9-C0 (documentation) | `d1bd8837a3fa45093954d1c948ebfc2a2e9a678e` | `923fba140bc76ef30128b30011955539f3c1ec86` |

The historical B3 implementation baseline before documentation closure is
`d7d660f65fc5fc794a545f8b64703562e92b8bc7`, tree
`3f78b90c337f7e78ba8797967349a9ac91d14bf6`.
[Push-main CI 34760583942](https://github.com/kaizenforyou91/forge/actions/runs/34760583942)
is **completed / success** for this exact SHA, with Ubuntu acceptance **PASS**,
Windows acceptance **PASS**, and Ubuntu race **PASS**, without retry or waiver.
Acceptance includes dependency metadata/cleanliness, package listing, vet,
unit tests, and build; hosted race includes `internal/agent` and the OpenAI adapter.

The final Phase 9 closure baseline is the P9-C0 merge in PR #20:
`923fba140bc76ef30128b30011955539f3c1ec86`, tree
`4143a413ad106c60261a3bf180045420951bf6cf`.
[Push-main CI 34761487262](https://github.com/kaizenforyou91/forge/actions/runs/34761487262)
is **completed / success** for this exact SHA: Ubuntu acceptance **PASS**,
Windows acceptance **PASS**, and Ubuntu race **PASS**, without retry or waiver.
Phase 9 closure became effective with that integration and strict CI success.

- **P9-B1:** [`Run`](../../../internal/agent/run.go) is internal/pre-stable with
  an atomic single-use claim shared by copies, Ready / Running / Succeeded /
  Failed / Canceled states, stable Done, explicit Cancel, fixed/redacted
  state/errors, and terminal reference release. Unknown is fail-closed.
  `NewRun` still uses caller-supplied `ai.Provider` and unchanged `ai.Executor`
  with explicit context/timeout. At most one operation is delegated; no retry,
  Run-created worker goroutine, or persistence exists.
- **P9-B2:** [`NewAuthorizedToolRun`](../../../internal/agent/tool_run.go) accepts
  a caller-supplied `AuthorizedToolRoundTripper` and immutable `tool.Authority`.
  It delegates directly to the existing C5/C4 path, never through `ai.Executor`.
  The private operation closure shares the B1 lifecycle and classifier; there
  is no exported generic operation constructor. Each Run delegates at most one
  round trip, with the existing maximum two provider POSTs and one handler.
  `ai.Result.Validate()` and defensive Usage copying remain; aggregate two-turn
  `Usage.OutputTokens` may exceed per-turn `MaxOutputTokens`. The 96 > 64
  regression passes, while the text executor's single-turn limit remains
  enforced. No second aggregate token-limit check, new tool authority, retry,
  or recursive loop is introduced.
- **P9-B3:** [`RunHost`](../../../internal/agent/host.go) implements `app.Module`
  with explicit `App.Add` registration and one-App binding. Register/Start
  execute zero Runs. Caller `Host.Execute` requires a fully Running App and a
  healthy current lifecycle context, then calls Run synchronously. Caller
  context remains the parent; `context.AfterFunc` bridges application
  cancellation only, with no execution worker. Unique per-call active entries
  prevent a losing duplicate Run claim from untracking the winner. Stop closes
  admission, cancels active Runs, and drains admitted calls; entry removal
  precedes WaitGroup completion. Restart snapshots a fresh app context;
  consumed Runs remain consumed. Old contexts and active entries are released,
  with no result/error history. Shutdown is cooperative; work is never forcibly
  terminated or abandoned merely to report completion.

`pkg/app.Runtime` remains the application lifecycle owner, Run owns one
operation, and RunHost composes them. Phase 9 did not replace or fork App
startup, module registration, shutdown, or restart. No lower `pkg` package
depends on `internal/agent`; B3 changed no `pkg/app` or OpenAI production source.
There is no persisted Run/Host state, agent global registry, or CLI migration.
Only `NewRun` and `NewAuthorizedToolRun` expose executable Run construction;
the shared operation factory remains private.

Phase 8 authority remains unchanged: exactly one read-only built-in
`forge_runtime_info`; explicit/default-false `--allow-tools`; at most two POSTs
and one handler; no retry or recursive tool loop; `store:false`, `stream:false`,
`background:false`, `parallel_tool_calls:false`; POST #2 `tool_choice:"none"`;
no provider conversation state or reasoning replay. C9 terminal compatibility
and safe, fixed-vocabulary C10 diagnostics remain intact. Phase 9 adds no C10
diagnostic composition and does not enlarge tool authority.

### B3 race retry history

[PR #19 CI 34760085814](https://github.com/kaizenforyou91/forge/actions/runs/34760085814)
attempt 1 passed both acceptance jobs but failed Ubuntu race in unchanged
`internal/aiprovider/openai`, test
`TestFunctionRoundTripDiagnosticContext/true/second`, with
`context stage/category/bounds changed`. The test blob at base and head was
`1ca17deb4118cb8d8e76a616224c44eb938f2366`; `internal/agent` passed.
One separately authorized targeted retry of race job `103731275343` passed as
attempt-2 job `103732225210`, and the workflow aggregate became success.
Acceptance jobs were not rerun. No code remediation occurred; this was not
classified as a B3 source defect. B3 push-main run 34760583942 passed all
three canonical jobs without retry or waiver.

## Package sequence and current authorization/status

| Package | Intended scope | Status |
|---|---|---|
| P9-A1 | Architecture decision + roadmap reconciliation | CLOSED / PASS — INTEGRATED |
| P9-B1 | Single-use AI Run ownership over existing text execution | CLOSED / PASS — INTEGRATED |
| P9-B2 | Existing authorized tool round-trip with Run lifecycle, preserving Phase 8 bounds and aggregate-usage semantics | CLOSED / PASS — INTEGRATED |
| P9-B3 | Bounded application-host/shutdown composition | CLOSED / PASS — INTEGRATED |
| P9-C0 | Offline integration / architecture closure | CLOSED / PASS — INTEGRATED |

The separate B1/B2/B3 authorizations do not retroactively broaden P9-A1.
Memory, workflow, scheduler, and tool/provider expansion remain future work
requiring a new architecture/roadmap selection gate. Phase 9 did not authorize
publication; release governance proceeded separately through RR-001 to RR-004-PUB.
P9-C0 authorized no runtime capability. **Phase 10: NOT DEFINED / NOT AUTHORIZED.**

## Original P9-B1 acceptance intent (now satisfied)

The original P9-A1 acceptance intent required P9-B1 to prove, using offline tests:

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

P9-A1 implemented no tests, Go source, or CI changes. P9-C0 changed only
documentation; its offline validation passed agent, App, and OpenAI tests plus
full package listing/tests/vet/build and dependency cleanliness. Local race was
not required; the normal hosted PR workflow and strict push-main CI passed,
including canonical agent/OpenAI race coverage.

## Non-capabilities and remaining scope boundaries

- Autonomous planning or autonomous loops.
- Multi-agent execution or recursive provider conversations.
- Memory/history, durable jobs, or persistence.
- Workflow engine, scheduler, queues, worker pools, or automatic/background execution.
- New provider, provider routing, or multi-provider compatibility.
- Arbitrary tools, new built-in tools, or filesystem/network/subprocess agent handlers.
- CLI agent command, CLI migration, or application lifecycle behavior changes
  beyond the separately authorized, integrated RunHost composition.
- Public `pkg/agent` API or Beta readiness.
- Native-process expansion.
- Release or tag creation.

## Release status

**v0.4.0-alpha.1 is PUBLISHED**, following separate RR-001 through RR-004-PUB
governance and explicit Owner publication authorization. The source-only
[GitHub prerelease](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1)
has zero uploaded assets and selects
`85d78143db1b8bcf2f96b79d681895b1a0492642`.
RR-005 reconciles current publication wording only. P9-C0 did not authorize
publication; the original architecture decision and separate B1/B2/B3
authorizations are unchanged. Publication does not broaden Phase 9 authority.
**Phase 10: NOT DEFINED / NOT AUTHORIZED.**

Published `v0.3.0-alpha.1` remains at
`5d836931216203aeea0737fc54de9e95091a62ef`; RR-005 creates or moves no tag.
