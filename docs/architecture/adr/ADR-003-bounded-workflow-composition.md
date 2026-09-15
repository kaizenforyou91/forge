# ADR-003: Bounded Workflow Composition — Synchronous Sequences

## Status and authorization

**ACCEPTED BY CONTROL ROOM — ARCHITECTURE ONLY.**

P10-A0-R1 is CLOSED / PASS — OUTCOME A ACCEPTED BY CONTROL ROOM.
P10-A1 is CLOSED / PASS — INTEGRATED after strict push-main CI. That architecture
decision did not authorize implementation. P10-B1 is CLOSED / PASS — INTEGRATED
under separate authorization, as are P10-B2 and P10-B3. The selected internal
implementation is functionally complete on current main. P10-C0 is the separately
authorized final documentation/audit closure package, with no new runtime capability.
P10-C0 integration plus strict exact push-main CI establishes Phase 10
CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION (Synchronous Sequences).
No closure is claimed before that gate. The architecture-only status above
classifies the original decision; implementation authority was granted separately.

Phase 10 — Bounded Workflow Composition (Synchronous Sequences) has the objective:
compose a finite, explicit sequence of already-authorized AI operations
synchronously while preserving caller authority, lifecycle ownership, fail-fast
semantics, cancellation, and bounded resource behavior.

## Evidence and immutable publication baseline

Historical P10-A1 preparation base: `217be79f938b6f913c08ae86e1c918896e66fd68`, tree
`4a1393af3eceae4dd8a2db6b87a2f5968ebe4680`; parents
`4d9c79b0ce7a3fc151bc3e4e7b8131754528de6c` and
`eb51041491412e982205c2e63e4cee3998779b71`.
[Strict main CI 34799363981](https://github.com/kaizenforyou91/forge/actions/runs/34799363981)
is push/main on that exact SHA, attempt 1, completed/success: Ubuntu acceptance,
Windows acceptance, and Ubuntu race PASS.

Historical P10-A1 integrated main / P10-B1 preparation base:
`18e7e51cf366519db5521d7237c218ffd54c6fe6`, tree
`2eb9336df00ca52a479fd9ada72eaf16b2d22ffa`; parents
`217be79f938b6f913c08ae86e1c918896e66fd68` and
`b8a661b1d8d0b592b4c0d3a7af5f20d6e48feee2`.
[Strict main CI 34807777744](https://github.com/kaizenforyou91/forge/actions/runs/34807777744)
is push/main on that exact SHA, attempt 1, completed/success for Ubuntu acceptance,
Windows acceptance, and Ubuntu race. The earlier P10-A1 PR had a separately
authorized targeted race retry; this main run passed attempt 1. No retry is
pre-authorized for B1.

Historical P10-B1 integrated main / P10-B2 preparation base:
`9e0c38b70d63446440328aca3fa5c5574ae68df8`, tree
`c0bbdbc747823c87d30c059e574ad93697bafaca`; parents
`18e7e51cf366519db5521d7237c218ffd54c6fe6` and
`2f1a053b879808219f14ecef768aed1f113b26d2`.
[Strict main CI 34818568107](https://github.com/kaizenforyou91/forge/actions/runs/34818568107)
is push/main on that exact SHA, attempt 1, completed/success for Ubuntu acceptance,
Windows acceptance, and Ubuntu race. No B2 CI retry is pre-authorized.

Historical P10-B2 integrated main / P10-B3 preparation base:
`6d3cac19982129f785ce0a3153ace4feade055e4`, tree
`dc8f618ced2fd9c826e41c8ca30d8362280abb67`; parents
`9e0c38b70d63446440328aca3fa5c5574ae68df8` and
`edc75b975a25d1c588da00ca9083cd9d5669a80d`.
[Strict main CI 34825138783](https://github.com/kaizenforyou91/forge/actions/runs/34825138783)
is push/main on that exact SHA, attempt 1, completed/success for Ubuntu acceptance,
Windows acceptance, and Ubuntu race. No B3 retry was pre-authorized.

P10-B3 integrated main / P10-C0 audit baseline:
`92ea08c5af041e6a204891059724811ab582111a`, tree
`00e178657f0fa746c6e1261df55251c46bae277e`; parents
`6d3cac19982129f785ce0a3153ace4feade055e4` and
`bc64d2f19e09fce0f40a6e08b52ec4ddea7d5a88`.
[Strict main CI 34920655351](https://github.com/kaizenforyou91/forge/actions/runs/34920655351)
is push/main on that exact SHA, attempt 1, completed/success: Ubuntu acceptance,
Windows acceptance, and Ubuntu race PASS. No C0 retry is pre-authorized.

Published [v0.4.0-alpha.1](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1)
remains source-only, non-production, and pre-stable. Annotated tag object
`c02b81a94c3c7bead6a7fcf4030ce03cc98af52a` targets
`85d78143db1b8bcf2f96b79d681895b1a0492642`; Release ID `388084778` remains
draft=false, prerelease=true, assets=0. Phase 10 is not included in that release.
The release/publication chain is CLOSED / PASS; C0 does not mutate it.

The existing [Run](../../../internal/agent/run.go) snapshots its request,
owns a single execution claim, and releases terminal references. The
[authorized-tool constructor](../../../internal/agent/tool_run.go) delegates
directly without applying text Executor limits to aggregate usage.
[RunHost](../../../internal/agent/host.go) admits explicit synchronous calls,
links application cancellation, and drains active work. These accepted Phase 9
primitives originally supplied operation ownership; B1/B2 add Sequence and
handoff, and the B3 record below adds whole-sequence host ownership.

## Placement and Architecture Freeze

The Sequence is INTERNAL and pre-stable in `internal/agent`, implemented in
`sequence.go` / `sequence_test.go`, with whole-sequence host composition in
`host.go` / `host_test.go`. Historical P10-A1 created none of those source files;
B1/B2/B3 integrated them under separate authorization. C0 changes documentation only.

Composition builds above `pkg/app`, `pkg/ai`, `pkg/ai/tool`, and existing
`internal/agent` ownership. Lower layers must not import higher agent composition.
`pkg/app` remains application lifecycle owner, with no production API change.
The [Architecture Freeze v1.0](../ARCHITECTURE_FREEZE_V1.md) and
[ADR-001](ADR-001-architecture-freeze.md) remain unchanged; no Freeze exception
is required. [ADR-002](ADR-002-agent-run-ownership.md) retains its Phase 9 decision
and historical separate implementation authorizations.

## Sequence shape and immutable specifications

The following sections define the selected full Phase 10 architecture.
The B1, B2, and separately authorized B3 records distinguish their delivered
slices. P10-C0 closure remains separately gated.

Define one single-use Sequence with Ready, Running, Succeeded, Failed, and
Canceled states. Execute runs on the caller goroutine; Forge starts ZERO
sequence worker goroutines. Exactly 1..8 steps are admitted. Eight is an
intentional initial safety bound, not a measured product requirement.

The topology is STATIC LINEAR SEQUENCE: declaration order is execution order.
There are no DAGs, branches, joins, cycles, parallel nodes, dynamic/model-created
nodes, conditions, expressions, templates, or transformation callbacks.
There is no concurrency inside one sequence and no global concurrency quota
claim for independent caller work.

Only two operation classes are allowed: text Run and authorized-tool Run.
There is no native/shell/runtime/package step, arbitrary operation constructor,
or generic executable API. The existing private operation function remains
private; this ADR approves only `NewRun` and `NewAuthorizedToolRun` as executable
Run construction paths.

Before Execute, immutable internal step specifications fix:

- operation kind and the caller-supplied Provider or authorized round-tripper;
- model, MaxOutputTokens, and per-step timeout;
- immutable tool Authority for an authorized-tool step;
- input mode and literal text, when applicable.

Construction snapshots the specification slice and owned values, rejects invalid
step counts/kinds, nil or typed-nil providers, invalid authority, model/token
limits, timeouts, and input combinations without performing I/O. Existing
ownership conventions apply: provider/handler implementations remain trusted
caller-supplied objects, not deep-cloned or made thread-safe by Sequence.
There are no specification setters. Model output can never choose or mutate
provider, model, tools, authority, timeout, next kind, or node count.

## Inputs and fresh Run construction

Exactly two input modes exist: LITERAL_TEXT and PREVIOUS_STEP_TEXT.
Step 1 MUST use literal text; later steps may use either mode. Previous text
means the immediately preceding successful step's Result.Text, passed unchanged.
There is no concatenation, implicit rewriting, summarization, or transformation.

Once a step's input is known, build its request and apply normal
`ai.Request.Validate()` before constructing a fresh Run through its narrow
constructor. Literal inputs can be validated at Sequence construction; dynamic
handoff must be validated at the step boundary. Blank, invalid UTF-8, oversized,
or otherwise invalid handoff fails before any next provider/handler work.
`MaxInputBytes` remains 16 KiB even though a result may reach the existing
64 KiB result bound. No truncation, fallback, or extra provider call is allowed.

Do not preconstruct mutable Runs and change their captured requests later.
Sequence owns specifications and constructs each fresh Run only after its input
is resolved. A canceled sequence must not proceed to the next step.

## Time bounds and fail-fast execution

Each step timeout satisfies `ai.ValidateTimeout`: positive, at most 120 seconds.
Sequence overall timeout is also explicit, positive, and at most 120 seconds.
Effective execution uses the earliest caller deadline, overall deadline, or
step deadline. A later step receives only remaining overall time; no reset
extends sequence lifetime or a shorter caller deadline.

First failure or cancellation ends execution. Later steps perform ZERO work.
There is no retry, resume, rollback, compensation, fallback, alternate provider,
or replay. A new Sequence requires a new caller action. Provider processing,
billing, or handler side effects already performed cannot be rolled back.

## Lifecycle, result, and panic contracts

Copies share one private ownership core, one atomic execution claim, and one
stable Done channel. Nil context/invalid Sequence fails closed without execution.
Pre-start Cancel consumes the sequence with zero provider/handler work. A valid
Execute with an already-canceled context consumes it without I/O. Concurrent or
repeated Execute cannot duplicate work. Running cancellation cancels the active
Run and prevents later steps, but does not publish completion before owned work
returns. Late Cancel cannot rewrite a terminal state or returned result.

Normal terminal publication follows the existing Run classification: pure
context cancellation is Canceled; non-context or mixed failures are Failed;
success is suppressed if cancellation wins the terminal publication boundary.
Handoff validation and accounting overflow are failures. State and Done are
published consistently after owned work returns.

Success returns the final step Result and bounded aggregate metadata/accounting.
Final Result.Usage retains that final Run's semantics; sequence aggregate usage
is a separate field. Bounded metadata may include step count/completed count and
terminal step index, never an unbounded event stream. Failure returns a zero
success result and sanitized error; terminal state is observable without
returning earlier successful text as final data. No partial text/history is
retained for later inspection. Intermediate text exists only while needed for
execution/handoff; terminal references to specifications, active Runs, authority,
and text are released. Caller-owned returned values are not retained by Sequence.
Reference release is NOT secure memory erasure.

There is no universal panic containment. Provider/custom trusted implementation
panics may propagate. Defers must release execution references and host admission
bookkeeping on unwind where applicable, without converting arbitrary panic into
a safe provider error. Normal terminal-result guarantees describe returned
execution, not a successfully recovered provider panic. A panic cannot make the
single-use claim reusable or justify detached work. Existing tool-handler panic
handling remains unchanged.

B1 makes Sequence unwind bookkeeping explicit: a propagated panic leaves the
Sequence consumed and observably Failed, clears specifications/active child/
cancel references, and closes Done. The original panic propagates unchanged;
there is no ordinary returned error or result. Existing Run panic behavior is
not modified. This is reference release, not panic containment or secure erasure.

## Checked usage and aggregate execution bounds

Every Run retains existing usage behavior. Text execution continues through
`ai.Executor`; authorized-tool execution continues through its direct bounded
round-trip path. The 64 per-turn cap / 96 aggregate output regression remains
valid and is not a production provider guarantee.

If ALL completed steps report Usage, sum InputTokens and OutputTokens using
checked integer arithmetic. If ANY completed step reports nil/unknown Usage,
sequence aggregate usage is UNKNOWN/nil, never zero-filled. Known arithmetic
must not wrap; overflow fails safely and prevents later execution. No Run's
MaxOutputTokens is applied as a cap to the Sequence aggregate. Missing usage
and failed provider work do not establish a billing total.

For declared T text steps and U authorized-tool steps, T+U=N<=8:

- text steps permit at most T provider executions;
- accepted OpenAI paths permit at most T+2U provider POSTs;
- tool steps permit at most U handler attempts.

Worst case: eight tool steps, 16 POSTs and 8 handler attempts. These are explicit
aggregate possibilities, not increased per-Run authority or a guarantee that
every permitted call occurs. A custom trusted provider still must honor its
existing contract; Sequence cannot enforce its internals.

## Authority, data, and secrets

Trusted caller authority is fixed before execution. Immutable tool Authority
is passed through unchanged; handoff transfers TEXT only. Provider output never
grants authority. Steps cannot dynamically register tools, select outside their
authority, add network/filesystem/shell/subprocess authority, or mutate it.

Production CLI remains direct Phase 8 composition with explicit --allow-network,
--allow-tools default false, and exactly one read-only built-in,
`forge_runtime_info`. No CLI migration occurs. Trusted caller-supplied library
handlers are not made read-only merely by composition.

Sequence performs no environment credential lookup and persists no prompts,
results, authorities, credentials, history, or resume state. It logs no raw
prompts, model output, provider error causes, or secrets. Safe diagnostics must
redact data and authority. Existing stateless provider settings, no retry, and
no recursive tool loop remain unchanged. `store:false` is not a zero-retention
guarantee; local cancellation is not a provider processing/billing guarantee.

## Whole-sequence application ownership

Calling RunHost.Execute independently per Run is insufficient: inter-step gaps
must remain part of ONE application-owned admitted execution.

Select private generalized active-execution bookkeeping inside
`internal/agent/host.go`, with narrow internal Run and Sequence entry points.
A private cancellation/bookkeeping abstraction may support these two owners;
it must not expose generic executable callbacks or a public operation API.
Existing RunHost.Execute behavior remains compatible. A narrow Sequence host
entry point admits the whole sequence once, executes it synchronously, and
releases admission only after the whole call returns/unwinds. Child Runs execute
under this owned lifetime without independent per-step host admissions.

Admission verifies the same running application generation under the host lock.
Stop closes admission before waiting, cancels each admitted owner, thereby
cancels the active Sequence and its active Run, prevents the next step, and
drains the whole call including inter-step gaps. Cancellation is checked and
coordinated at each step transition; an application restart cannot admit another
step from an old generation. Defers release host entries and wait accounting
on errors and panic unwind.

Register and Start perform zero AI executions. `pkg/app` owns application
lifecycle; no app production API change, scheduler, worker pool, or global
registry is introduced. Restart captures a fresh application context; old
Sequence objects remain consumed. Stop is cooperative: a broken provider may
block shutdown. Do not fake early Done or detach work to meet a deadline.

## Public surface and non-goals

| Decision | Answer |
|---|---|
| Public API change | NO |
| CLI change | NO |
| Manifest change | NO |
| Package format change | NO |
| Persistence | NO |
| Network authority expansion | NO |
| Tool authority expansion | NO |
| Background execution | NO |

No public `pkg/agent`, `forge agent`, or `forge workflow` command is introduced.
Non-goals: autonomous agents, planning loops, multi-agent systems, durable AI
memory, durable jobs, scheduler, queue, worker pool, background execution,
generic workflow engine, DAG, parallel sequence execution, branching/join,
generic tool marketplace, new built-ins, filesystem tools, shell/subprocess
tools, provider routing/fallback, remote registry/distribution, public agent
API/CLI, runtime/native execution steps, and production sandboxing.

## Accepted debt outside Phase 10

Important but NOT P10 blockers: validated-object-to-path execution binding,
same-user package mutation, Windows ACL/reparse/share-mode hardening,
process-tree/graceful native shutdown, persistent trust lifecycle, key
rotation/revocation, and provenance/SBOM completeness. Closed Phase 6–9 scopes
are not reopened. Runtime/Trust Hardening is the preferred next architecture
wave after bounded composition unless future evidence changes priority; it is
not folded into P10 or automatically authorized. No Beta readiness is claimed.

## P10-B1 implementation record — literal steps only

Historical B1 slice, now CLOSED / PASS — INTEGRATED. Its literal constructors
and lifecycle remain compatible; the separate B2 extension below adds input modes
and aggregate accounting to the current Sequence.

The separately authorized B1 slice is internal/agent/sequence.go and
sequence_test.go, with ErrInvalidSequence/ErrSequenceConsumed in errors.go.
The errors have fixed, redacted text. No existing Run, tool Run, host, State,
provider, CLI, or app production file changes.

- SequenceStep has immutable unexported fields and only two narrow constructors:
  NewTextSequenceStep(provider, request, timeout) and
  NewAuthorizedToolSequenceStep(roundTripper, request, authority, timeout).
  Validation rejects invalid requests/timeouts, nil/typed-nil implementations,
  and unusable tool authority, with zero provider/handler work or retained Run.
- NewSequence(steps, overallTimeout) validates 1..8 steps and snapshots the slice
  and request values. Implementations remain trusted caller-owned references;
  tool Authority is the existing immutable object, passed unchanged.
- Execute, Cancel, State, Done, String, and GoString share a private single-use
  core across copies. Execute is synchronous with no worker goroutine; every
  fresh child uses NewRun or NewAuthorizedToolRun at execution time.
- All requests are declared literals. Overall timeout is explicit and at most
  120 seconds; each child retains its own timeout and any shorter caller
  deadline. Cancellation is checked before constructing each child, propagates
  to active work, and prevents later steps. Done waits for return/unwind.
- SequenceResult contains only Final ai.Result and CompletedSteps int. Success
  returns the last child result/count; failure or cancellation returns zero.
  Final.Usage is only that child's usage, including valid tool 64/96 semantics.
  No aggregate usage is present or inferred; earlier results are not retained.
- Failure is fail-fast with existing safe classification; pure context means
  Canceled and ordinary/mixed failures mean Failed. Cancellation can suppress
  success before terminal publication; late Cancel cannot rewrite it.
- Terminal cleanup releases specifications, active Run, and cancel references
  before closing Done. String/GoString redact Sequence and step contents.

No PREVIOUS_STEP_TEXT, aggregate accounting, host/app admission, public API,
CLI, persistence, background work, or authority expansion is implemented by B1.
B2 subsequently integrated; B3 received separate host-integration authorization.

## P10-B2 implementation record — handoff and aggregate accounting

B2 modifies only sequence.go / sequence_test.go and its three governance
documents. No errors.go, Run, tool Run, host, app, provider, or public product
surface changes. B1's single claim/shared core, synchronous lifecycle, deadlines,
cancellation, fail-fast, panic propagation/cleanup, and redaction are preserved.

- Unexported sequenceLiteralText / sequencePreviousStepText modes are explicit.
  Existing constructors still select literal input with unchanged signatures.
  NewTextSequenceStepFromPrevious(provider, model, maxOutputTokens, timeout) and
  NewAuthorizedToolSequenceStepFromPrevious(roundTripper, model, maxOutputTokens,
  authority, timeout) validate fixed configuration without I/O. A fixed probe is
  used only in a local Request copy for canonical model/token validation; it is
  never retained as prompt text or passed to a provider.
- NewSequence revalidates modes/specifications and requires step 1 literal.
  A previous-text specification retains no literal Text. The actual request
  copies only the immediate preceding successful Result.Text unchanged; model,
  token limit, timeout, operation kind, provider, and immutable authority remain
  caller-fixed. Literal steps ignore preceding output.
- Request.Validate runs before child Run construction. Exactly 16 KiB valid text
  can pass; 16 KiB + 1 fails before next provider/handler work. Blank/invalid UTF-8
  or malformed child results still fail existing child Result validation first.
  There is no truncation, rewriting, repair, summarization, or extra call.
- Only the preceding transition value is retained, not history. After the fresh
  Run captures needed text, the preceding Result reference is dropped. Terminal
  cleanup retains no intermediate text/specification; this is not secure erasure.
- SequenceResult contains Final ai.Result, CompletedSteps int, and
  AggregateUsage *ai.Usage. Final.Usage belongs only to the final child.
  AggregateUsage is a fresh, non-aliasing checked sum of both int64 counters
  when all completed children report usage. Zero known usage remains known.
- Any nil child Usage permanently makes the aggregate UNKNOWN/nil and disables
  further arithmetic. While still known, overflow of either counter fails
  immediately with sanitized ai.ErrMalformedResponse, Failed state, zero result,
  and zero later work. A hypothetical later nil cannot undo an earlier overflow.
- Tool 64-cap / 96-output usage remains valid and contributes to the sum. No
  child request cap is imposed on aggregate usage. Any failure or cancellation
  returns zero SequenceResult, never partial text or accounting. A panic still
  propagates unchanged with consumed/Failed state, released references, and Done
  closed after unwind; no usage is fabricated for the panicking child.

B2 adds no automatic provider calls or authority. For T+U<=8, accepted paths
retain T+2U POSTs / U handler attempts at most (16 / 8 worst case). No host/app
admission, public API, CLI, persistence, or background execution is delivered by
B2. B3 subsequently integrated the separately authorized host composition below.
C0 is the separately gated final documentation/audit closure package.

## P10-B3 implementation record — whole-sequence host integration

B3 changes only host.go / host_test.go and its three governance documents.
Sequence, Run, tool Run, errors, pkg/app, AI/provider, CLI, CI, and dependency
source remain unchanged. No new Sequence semantics or public API is introduced.

- ExecuteSequence(ctx context.Context, sequence *Sequence) (SequenceResult, error)
  is the sole new narrow internal execution entry point. It explicitly calls
  Sequence.Execute once, synchronously, with no worker goroutine. Existing
  Execute(ctx, *Run), module name, registration, lifecycle, and redaction remain
  compatible. Register/Start perform zero Run, Sequence, provider, or handler work.
- Each call creates one unique private hostExecution{cancel func()} entry,
  containing only Run.Cancel or Sequence.Cancel authority. No generic execution
  callback, result, history, prompt, or authority data is stored. A losing
  duplicate claim releases only its own entry; mixed Run/Sequence calls remain
  distinct. Children are never individually admitted to RunHost.
- Shared private admit checks nil context, valid/bound host, hostRunning,
  App.Started, non-nil/live appCtx, and exact current App.Context generation
  under the host lock. Admission and WaitGroup.Add occur before Stop can close
  the gate. Admission errors remain existing host errors; admitted operation
  errors and results belong to Run or Sequence unchanged.
- Caller context stays parent. context.AfterFunc links application cancellation
  to a derived caller context and performs cancellation only. There is no new
  timeout policy. The whole call remains registered across active children,
  handoff/accounting, inter-step gaps, terminal return, and panic unwind.
- Stop closes admission under lock, snapshots owner cancellation functions,
  cancels them outside the lock, and waits for all admitted calls. Sequence.Cancel
  cancels its overall context and active child and prevents later construction.
  Stop never equates canceled context or closed Done with a returned host call.
  Non-cooperative providers/handlers may block Stop; no work is detached.
- Deferred release stops the application-context callback, cancels the linked
  context, deletes the exact entry and clears its cancel reference before
  WaitGroup.Done. Context-link panic also releases admission. Trusted panic
  propagates unchanged; Sequence retains its existing Failed/consumed cleanup,
  and the running host remains usable after external recovery.
- Restart captures a fresh application context. Fresh Sequences execute; old
  consumed Sequences remain consumed. Result, Final.Usage, AggregateUsage,
  literal/previous-text semantics, bounds, and error classifications pass through
  unchanged. Cancellation callbacks never execute AI work.

Deterministic tests cover the real narrow entry point, eight-child single-entry
identity, literal/previous text and 64/96 usage, caller/App cancellation, text
and tool child Stop/drain, concurrent Stops, duplicate claims/copies, mixed
Run/Sequence ownership, admission/Stop races, deadlines, panic recovery, and
restart. A package-local fixture uses real private admission and Sequence
transition methods to hold a no-active-child gap after a successful child;
Stop cancels it and cannot return before exact call release. No production
hook or Sequence change is needed. Existing Run host tests remain intact.

B3 is CLOSED / PASS — INTEGRATED on the exact baseline recorded above.
C0 integration plus strict exact push-main CI establishes final Phase 10 closure.
No public API, CLI, manifest/package change, persistence, background execution,
network/tool authority expansion, live call, or credential access is introduced.

## P10-C0 final offline source audit

C0 audits the exact B3 integration baseline above. A1/B1/B2/B3 are
CLOSED / PASS — INTEGRATED. The following existing source and tests establish
the implemented bounded scope; C0 changes no Go, tests, CI, dependencies,
manifest/schema, or runtime capability.

| Evidence | Audited boundary |
|---|---|
| [Sequence](../../../internal/agent/sequence.go), [tests](../../../internal/agent/sequence_test.go) | Shared single claim; synchronous 1..8 static steps; literal/previous-text modes; fresh Request validation before fresh Run; deadlines, cancellation, fail-fast, panic cleanup, redaction and reference release |
| [RunHost](../../../internal/agent/host.go), [tests](../../../internal/agent/host_test.go) | One unique cancel-only admission for the entire Sequence call; active-child and inter-step Stop cancellation/drain; duplicate/mixed owners; panic unwind; fresh App generation on restart |
| [Run](../../../internal/agent/run.go), [tests](../../../internal/agent/run_test.go), [tool Run](../../../internal/agent/tool_run.go), [tests](../../../internal/agent/tool_run_test.go) | Existing narrow execution paths and lifecycle remain authoritative; safe errors, explicit authority and tool 64-cap/96-output usage preserved |
| [AI contracts](../../../pkg/ai/types.go), [Executor](../../../pkg/ai/executor.go), [tool Authority](../../../pkg/ai/tool/authority.go), [tool execution](../../../pkg/ai/tool/execution.go) | Canonical request/result validation; 16 KiB input; immutable caller authority; bounded handler execution |
| [OpenAI round trip](../../../internal/aiprovider/openai/function_call_roundtrip.go), [authority path](../../../internal/aiprovider/openai/function_call_authority.go) | Unchanged production wire path; at most two POSTs and one handler attempt per authorized-tool Run; no retry or recursive tool loop |
| [CLI AI path](../../../internal/cli/ai.go) | Direct Phase 8 composition, not Run/Sequence/RunHost; no new CLI grammar or built-in tool |
| [App lifecycle](../../../pkg/app/lifecycle.go), [start](../../../pkg/app/start.go), [stop](../../../pkg/app/stop.go), [Architecture Freeze](../ARCHITECTURE_FREEZE_V1.md), [ADR-002](ADR-002-agent-run-ownership.md) | App remains lifecycle owner; no app production API change, lower-layer dependency inversion, new dependency, or Freeze exception; ADR-002 receives only current-status reconciliation |

Handoff uses only the immediate successful predecessor's text, unchanged.
Request.Validate rejects invalid or over-16-KiB input before the next child is
constructed. No truncation, summarization, repair call, or history is added.
Final.Usage belongs to the final child; AggregateUsage is a fresh checked sum
only while every successful child reports usage. Nil stays unknown and stops
arithmetic; known int64 overflow fails safely with zero result and no later work.
The existing 64/96 tool regression remains valid without an aggregate token cap.

Model output remains text data only: provider, model, token limit, timeout,
operation kind/count and immutable tool authority stay caller-fixed. For T text
and U tool steps with T+U<=8, accepted OpenAI paths retain at most T+2U POSTs and
U handler attempts (16/8 worst case), without increasing per-Run authority.
Trusted provider/custom panic propagates unchanged while Sequence and host
bookkeeping unwind; existing Run and tool-handler panic contracts are unchanged.
Stop owns the exact call until return/unwind and cannot detach non-cooperative work.

No public API, CLI, manifest/package change, persistence, background execution,
new network/tool authority, autonomous planning, scheduler, generic workflow
engine, DAG/parallel nodes, provider routing, or production isolation is added.
Source comparison confirms CLI, pkg/app, AI/tool, OpenAI, manifest/package/runtime,
dependency metadata, and Architecture Freeze remain unchanged from publication.
The production AI path also matches the accepted live baseline
`ac68a1b3e059f173d9b5c71eadaff9fcffba981f`; no new live validation is required.

C0 preserves published v0.4.0-alpha.1 and its historical pre-Phase-10 truth.
It selects no version or release. The accepted debt above, cooperative shutdown,
provider processing/billing limits, and pre-stable APIs/formats remain; this is
not Beta/production readiness. Runtime/Trust Hardening is a later selection gate.

P10-C0 integration plus strict exact push-main CI establishes
**CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION (Synchronous Sequences)**.
This conditional rule does not assert that C0 has already integrated. Normal
PR acceptance on Ubuntu/Windows and canonical Ubuntu race must pass before
that separate guarded integration; no retry or waiver is pre-authorized.

## Package plan and separate authorization

| Package | Purpose / future family | Proof obligation | Authorization / dependency |
|---|---|---|---|
| P10-A1 | Architecture / ADR / roadmap only | Accurate bounded decision, historical truth, links, offline checks | CLOSED / PASS — INTEGRATED; architecture only |
| P10-B1 | Internal sequence lifecycle and static literal steps; sequence.go / sequence_test.go / errors.go | Single claim, order, bounds, fail-fast, cancellation, redaction, reference release | CLOSED / PASS — INTEGRATED |
| P10-B2 | Previous-text handoff and aggregate accounting; sequence.go / sequence_test.go | No next call on bad input, immutable authority, known/unknown/overflow usage | CLOSED / PASS — INTEGRATED |
| P10-B3 | Whole-sequence host composition; host.go / host_test.go | Admission, inter-step cancellation/drain, restart, panic bookkeeping | CLOSED / PASS — INTEGRATED |
| P10-C0 | Offline integration / architecture closure | Integrated scope and invariant audit, strict main CI | Final documentation/audit closure package; integration plus strict exact push-main CI establishes bounded Phase 10 closure |

A1/B1/B2/B3 are integrated under their separate authorizations. C0 is authorized
for final documentation and offline audit only. Its guarded integration and strict
exact push-main CI remain the final Phase 10 closure gate.

## Future tests and acceptance evidence

The selected proof obligations below are now covered by integrated tests.
C0 uses that evidence and adds no runtime test; future changes must preserve it.

- Admission: zero-node and >8 rejection, step-kind/input validation, nil and
  typed-nil rejection, construction with zero I/O, immutable snapshots.
- Ownership: single claim, repeated/concurrent Execute and copies, stable Done,
  late Cancel, terminal reference release, redaction, panic unwind bookkeeping.
- Execution: declaration order, literal input, previous-result handoff, blank,
  invalid UTF-8 and oversized handoff rejected before any next provider call.
- Failure: fail-fast, zero later calls, pure versus mixed cancellation, safe
  zero result, no retry/fallback/replay, no raw error or output leakage.
- Deadlines: per-step, overall, shorter caller precedence, remaining-time
  behavior, pre-start/mid-step/between-step cancellation.
- Host: application Stop during a step and between steps, admission race,
  drain completion, fresh-context restart, consumed old sequences, no detached
  work, unchanged RunHost.Execute and zero Register/Start execution.
- Accounting: known sums, unknown/nil propagation, overflow, final Run usage
  distinct from sequence usage, and the 64-cap/96-aggregate regression.
- Authority/resources: fixed provider/model/authority, no model-created steps,
  no next call after denial, 1..8 bounds, maximum call counts, bounded metadata.

Use deterministic fake providers/handlers and synchronization rather than
sleeps where possible. Preserve Ubuntu and Windows acceptance: dependency
metadata/cleanliness, go list, go vet, full go test, and go build. Preserve Ubuntu
race for `pkg/compiler`, `runtime`, `internal/cli`, `pkg/ai`,
`internal/aiprovider/openai`, `pkg/ai/tool`, and `internal/agent` at minimum.
Check existing CLI, manifest, and package compatibility. No new dependency is
currently justified. Normal PR CI and strict exact push-main CI are required
for later integration/closure; this ADR does not weaken them.

## Live validation and release implications

**NO NEW LIVE PROVIDER VALIDATION REQUIRED** for the selected composition
design: it reuses accepted execution paths without changing provider wire or
adapter behavior. A future change to wire serialization, HTTP, response decoding,
or provider/tool round-trip behavior must reopen freshness review. C0 makes
zero live calls and accesses no API key.

No version is selected, release authorized, tag created, or Release edited.
Architecture definition and C0 closure have no automatic publication effect.
The integrated Phase 10 scope may be considered for an alpha feature release
only through a separate release gate. This decision does not promise production or Beta readiness.
