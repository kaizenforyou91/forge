# ADR-006: Bounded Execution Evidence and Trace Model

## Status and authorization

**P13-A1 — IMPLEMENTED / DOCUMENTED / UNDER REVIEW.**

**Phase 13 — Bounded Execution Evidence and Trace Model** is **SELECTED /
ARCHITECTURE FREEZE UNDER REVIEW / IMPLEMENTATION NOT AUTHORIZED**.

- P13-A0: **CLOSED / PASS — SELECTION ACCEPTED**.
- P13-A1: **IMPLEMENTED / DOCUMENTED / UNDER REVIEW**.
- P13-B1/B2/B3/B4/C0: **NOT AUTHORIZED / NOT STARTED**.

This ADR is an architecture and acceptance-test contract only. It authorizes no
collector, Run/Sequence wiring, provider/tool integration, host integration,
public surface, persistence, exporter, or background work.

Phase 12 remains **CLOSED / BLOCKED-DEFERRED — FROZEN INVARIANT RETAINED; NO
VERIFIED-LAUNCH CLAIM**. This architecture neither reopens nor weakens ADR-005.

## Decision

Forge will use one private, synchronous, fixed-schema evidence owner per admitted
execution. The owner accepts only allowlisted lifecycle facts, assigns monotonic
logical sequence numbers, enforces small compile-time bounds, and returns
defensive immutable-by-value snapshots. The execution owner remains authoritative
for work, cancellation, results, errors, and terminal state. Evidence observes
those decisions and cannot select or invoke work.

The conceptual private model is:

```text
Run or Sequence owner
  -> EvidenceOwner (one serialized writer at a time)
     -> bounded []EvidenceEvent
     -> EvidenceSnapshot (defensive copy)
```

These names freeze concepts, not exact Go identifiers. B1 may choose clearer
package-private names without changing the contract.

## Meaning of trace

For Phase 13, **trace** means:

> bounded, local, Forge-owned execution evidence describing approved lifecycle
> facts.

It does not mean distributed tracing, OpenTelemetry, spans with arbitrary
attributes, telemetry exporters, remote monitoring, a metrics backend,
wall-clock analytics, or network transmission. The core records logical order;
it has no wall-clock timestamp contract.

## Current fact-source audit

The architecture derives from current source rather than documentation alone.

| Source | Stable facts | Race or publication boundary | Facts that must remain excluded |
|---|---|---|---|
| `internal/agent.Run` | Ready claim, Running transition, cancellation observation, safe terminal state | `runState.mu`; `complete` or Ready cancellation publishes terminal before closing `Done` | retained operation, request, provider, context, raw result/error |
| `internal/agent.Sequence` | 1..8 declared steps, step index/kind, child admission/order, completed count, known/unknown aggregate usage, terminal state | `sequenceState.mu`; `nextRun` linearizes cancellation with child admission; `complete` or unwind publishes terminal | literal/previous text, child result text, provider, authority, timeout/context |
| `internal/agent.RunHost` | host admission, application cancellation ownership, release/drain | `hostState.mu` plus wait group; Stop closes admission before drain | App pointer/context, cancellation function, active-entry identity |
| `pkg/ai.Executor` | provider boundary, validated success, safe failure category, usage known/unknown | synchronous call plus context checks before and after provider return | request text/model, result text/model, provider object, raw error |
| authorized tool round trip | caller-supplied authority present, provider and tool boundaries, safe terminal result | fixed synchronous two-turn path; admission precedes exactly one handler call | tool arguments/result, call ID, request/response bodies, handler, authority |
| OpenAI diagnostic stages | bounded adapter-local stage enum and safe error classification | stage advances synchronously at fixed checkpoints | HTTP bodies, headers, key, raw provider error/output |
| `ai.Usage` | nonnegative input/output counters or unknown (`nil`) | validation before publication; checked aggregation | invented zero for unknown usage |
| application lifecycle | host running/stopping admission and cancellation | application context and synchronous module Stop | App/module pointers and arbitrary module data |
| `runtime.ProcessResult` | high-level completed/canceled/terminated classification and output-truncation booleans | stable only after `RunningProcess.done` | PID, entrypoint/path, signer identity, stdout/stderr bytes, native handles |

Initial Phase 13 integration is limited to agent, provider, and authorized-tool
lifecycle facts. Native process facts are deferred; the audit above only defines
the safe boundary if a later package explicitly authorizes them.

## Ownership and lifetime

1. Evidence configuration is validated during construction, before execution is
   admitted.
2. A successful construction creates one private owner for one Run or Sequence.
3. The executing caller writes running-path events. Before execution is claimed,
   `Cancel` may instead publish the Ready-to-canceled transition. These paths
   are serialized by the Run or Sequence lifecycle mutex: there is one active
   writer at a time, not necessarily one goroutine for the owner's lifetime.
   The evidence owner starts no goroutine.
4. Evidence append and terminal publication are synchronous with the lifecycle
   transition they describe. The lifecycle lock orders competing `Execute` and
   `Cancel` calls; an evidence mutex may additionally protect concurrent
   snapshots. Snapshot never calls back into the lifecycle owner or reverses
   that lock order.
5. Readers may request snapshots concurrently; they never receive backing
   storage or authority references.
6. A completed or interrupted owner is immutable. No reset, retry, resume, or
   reuse operation exists.
7. The owner performs no filesystem, network, provider, handler, process, or
   other external I/O and invokes no caller callback.

## Schema version and fixed event representation

Every snapshot carries schema version `1`. Version is a fixed unsigned integer,
not a caller-selected value. Events contain only fields from this fixed logical
shape:

| Field | Type | Meaning |
|---|---|---|
| `Sequence` | positive bounded unsigned integer | owner-assigned logical order, starting at 1 |
| `Kind` | fixed enum | semantic lifecycle fact |
| `Boundary` | fixed enum | none, begin, or end where the event is a boundary |
| `StepIndex` | optional bounded unsigned integer | zero-based Sequence index, 0..7 |
| `StepKind` | optional fixed enum | text or authorized-tool |
| `UsageKnown` | boolean | distinguishes present counters from unknown usage |
| `InputTokens` | nonnegative `int64` | meaningful only when usage is known |
| `OutputTokens` | nonnegative `int64` | meaningful only when usage is known |
| `ErrorCategories` | fixed bit set | zero or more existing redacted safe categories |
| `Terminal` | fixed enum | none, succeeded, failed, or canceled |

An event kind determines exactly which fields are allowed. Unused fields must be
zero. Invalid combinations are contract violations. There is no
`map[string]any`, attribute bag, extension payload, interface value, pointer, or
free-form string.

## Minimum event vocabulary

The version-1 vocabulary is deliberately small:

| Event | Allowed payload | Rule |
|---|---|---|
| `ExecutionAdmitted` | none | first event only when `Execute` wins the Ready claim |
| `ExecutionStarted` | none | follows admission before delegated work; absent if Ready cancellation wins |
| `StepBoundary` | boundary, step index, step kind | Sequence only; begin/end for an admitted child |
| `ProviderBoundary` | boundary, optional step index | begin immediately before provider delegation; end after synchronous return |
| `AuthorizedToolBoundary` | boundary, optional step index | begin only after tool admission and before handler execution; end after return |
| `CancellationObserved` | optional step index | records the execution owner's authoritative observation, once |
| `UsageObserved` | known marker and optional counters, optional step index | unknown is explicit and never encoded as zero |
| `Terminal` | terminal enum and safe error-category bit set | exactly one for normally completed execution |

Provider and tool boundary events state only that a boundary was crossed. They
do not identify the provider, model, tool, handler, request, response, or
authority. Adapter-local diagnostic stages remain adapter-local in the initial
schema; B3 may map only reviewed stable stages to a smaller fixed enum if needed.

## Identity decisions

| Candidate identity | Version-1 decision | Reason |
|---|---|---|
| execution ID | excluded | no persistence or cross-process correlation exists; a durable/global ID would imply a future authority and privacy contract |
| Sequence step | zero-based index 0..7 plus fixed step-kind enum | already stable, bounded, and needed to explain ordering |
| tool identity | excluded | names can disclose application semantics and are not needed to prove the authority boundary |
| provider identity | excluded | provider objects and configuration are caller-owned; lifecycle explanation needs only the boundary |
| model identity | excluded | model strings are request data and can reveal configuration |
| digests | excluded | no version-1 object requires correlation; digest semantics and linkability would need separate review |

Snapshot identity is therefore only object ownership: the snapshot came from the
specific private owner on which `Snapshot` was called. It is not globally named.

## Redaction by construction

The evidence API must be unable to accept prohibited values. Evidence must never
contain:

- prompts or other request text;
- model output or result text;
- tool arguments or result payloads;
- provider request/response bodies, headers, or raw diagnostic bodies;
- raw errors, error strings, wrapped causes, panic values, or stack traces;
- credentials, API keys, tokens, or environment values;
- filesystem or executable paths;
- PIDs, file descriptors, native/process handles, filesystem identities, or
  verified-launch evidence;
- contexts, cancellation functions, function pointers, providers, handlers,
  authorities, executors, admissions, Apps, Runs, Sequences, or processes.

Post-capture sanitization is not an accepted defense. B1 must expose typed
methods or typed internal facts whose parameters cannot carry these values.

## Safe error categories

Evidence may carry only a fixed bit set derived from existing safe categories:

- canceled;
- deadline exceeded;
- invalid request;
- authentication;
- authorization;
- rate limited;
- quota exceeded;
- transport;
- malformed response;
- response too large;
- incomplete response;
- refused;
- provider failure;
- tool execution denied;
- tool handler failure;
- tool output invalid;
- tool output too large;
- agent/host/sequence contract failure.

The mapping consumes only already-sanitized errors at the owning boundary. It
never retains or formats the source error. Multiple safe categories may coexist;
their bit positions and ordering are versioned. Unknown errors map to a single
generic internal/provider failure category without retaining their content.

## Logical ordering

- Sequence numbers start at 1 and increase by exactly one.
- If `Execute` wins the Ready claim, `ExecutionAdmitted` is first and
  `ExecutionStarted` precedes any provider, tool, step, usage, cancellation, or
  terminal event. A pre-canceled context still follows this claimed-execution
  path, although delegated provider work may be prevented.
- If `Cancel` wins while the Run or Sequence is Ready, no execution is admitted
  or started. Under the lifecycle lock it publishes `CancellationObserved`
  followed by `Terminal(canceled)` and closes the existing `Done` channel only
  after terminal evidence is visible. A later `Execute` remains consumed and
  cannot append events. No provider, tool, or step boundary is fabricated.
- Provider/tool begin and end boundaries are properly nested. An authorized tool
  boundary occurs within its owning provider round trip and never grants or
  selects authority.
- A Sequence child begin precedes all child events; child end follows them.
  Children are ordered by step index and never overlap under the current
  synchronous Sequence contract.
- Cancellation is recorded at the existing authoritative observation point.
  A cancellation request alone is not evidence that cancellation won.
- The execution owner's existing terminal decision resolves cancellation versus
  natural completion. Evidence publishes the same decision, never a second one.
- Exactly one terminal event exists for normally completed evidence. It is last.
  No append is valid after terminal.
- No deterministic timestamp, duration, host clock, goroutine schedule, or
  cross-execution order is promised.

## Bounds

Version 1 freezes these limits:

| Scope | Bound | Derivation |
|---|---:|---|
| events in one standalone Run | 16 | admission/start, bounded provider/tool boundaries, cancellation, usage, terminal, with headroom for the fixed two-turn tool path |
| provider plus authorized-tool boundary events in one round trip | 8 | two provider turns and one tool execution each have at most begin/end; two positions remain reserved inside the Run limit without authorizing recursion |
| Sequence steps | 8 | existing `maxSequenceSteps` contract |
| step boundary events | 16 | begin/end for each of eight steps |
| total events in a maximum Sequence | 148 | four sequence-level events plus `8 * (16 child events + 2 step boundaries)` |
| identifier/string bytes | 0 | version 1 permits no string fields or identifiers |
| usage counters | nonnegative `int64` | identical to existing `ai.Usage`; checked addition remains authoritative |

The Sequence owner preallocates no more than 148 fixed-size events. A standalone
Run preallocates no more than 16. Snapshot memory is therefore bounded by the
same fixed event count and contains no variable-size payload. B1 must verify the
concrete event representation has no hidden pointer, slice, map, interface, or
string fields except the snapshot's private event slice, which is defensively
copied.

The bounds are ceilings, not required event counts. B1/B2/B3 must prove every
authorized path fits them. Raising a bound or adding an event kind requires an
architecture review and schema-version decision.

## Usage evidence

- `UsageKnown=false` means the provider or any aggregated child reported unknown
  usage. Both counters must then be zero and must not be interpreted as measured
  zero.
- `UsageKnown=true` requires nonnegative counters copied from validated
  `ai.Usage`.
- Sequence aggregate usage remains unknown if any successful child is unknown.
- Existing checked `int64` addition remains authoritative; evidence performs no
  independent accounting and grants no token authority.
- The accepted 64-token request-cap / 96-token authorized-round-trip aggregate
  behavior remains valid. Evidence records the already-validated aggregate and
  does not apply the single-turn cap again.

## Failure semantics

Observation is not execution authority. Version 1 uses the following coherent
model:

### Construction-time rejection

Invalid schema version, invalid bound configuration, nil/incomplete private
owner inputs, or an impossible static path budget reject construction before
Run or Sequence execution is admitted. Construction performs no execution or
I/O. No partially usable owner is returned.

### Post-admission operations

For a correctly constructed owner, every allowed append and snapshot operation
is allocation-bounded and has no ordinary error path. Evidence never asks a
provider, tool, process, or caller whether execution may continue.

### Programmer and invariant failures

An invalid transition, invalid field combination, duplicate terminal, append
after terminal/interruption, non-monotonic sequence, impossible-state detection,
integer overflow, or capacity overflow is a trusted Forge programmer bug. It
panics synchronously. The owner is poisoned before the panic is allowed to
propagate, cannot accept another event, and cannot fabricate a successful
terminal record. There is no retry or recovery callback.

Capacity overflow is not treated as normal runtime pressure: the limits are
derived from existing fixed execution bounds and must be proven by tests.
Silently dropping, truncating, overwriting, or wrapping events is forbidden.

### Snapshot misuse

A nil, zero, or incomplete owner returns an explicitly invalid zero snapshot.
An in-progress owner returns an in-progress snapshot. These states are not
success and contain no terminal classification. Calling Snapshot never changes
owner state and never panics solely because execution is still in progress.

### Diagnostic projection

Any optional private projection introduced by B4 operates on a snapshot after
the fact. Projection failure returns only to the diagnostic caller and cannot
change execution, terminal result, authority, or the owner snapshot.

## Panic and interruption semantics

Evidence does not recover or swallow trusted panics and never stores the panic
value. A synchronous defer may retire/poison evidence state and then allow the
original panic to propagate unchanged; it may not call user code or start
cleanup work.

- If the existing execution owner publishes `StateFailed` during unwind, as
  Sequence currently does, evidence may publish terminal failed before the
  original panic continues.
- If the existing owner does not publish a terminal state during unwind,
  evidence must not invent one. It freezes as **interrupted**, with no execution
  terminal classification and no further events accepted.
- An interrupted snapshot is immutable and explicitly incomplete. It is never
  interpreted as succeeded, failed, or canceled.

This preserves existing panic behavior. B2 may not add recovery merely to obtain
a nicer trace.

## Snapshot and concurrency contract

Snapshots have three fixed phases: invalid, in-progress, and final. Final has
either a normally completed terminal classification or interrupted evidence.

- `Snapshot` is available after successful construction and may be called before,
  during, or after execution. Before an `Execute` or `Cancel` claim, it is a
  valid version-1 in-progress snapshot with zero events and no terminal.
- It locks the evidence owner only, copies scalar state and the event slice,
  unlocks, and returns. It neither acquires a Run/Sequence lifecycle lock nor
  performs a callback or I/O.
- Concurrent Snapshot calls are supported. Event writing is serialized across
  the executing caller and a possible Ready `Cancel` caller; lifecycle
  synchronization serializes writes with readers.
- Every returned snapshot owns its event backing storage. Mutation of one copy
  cannot affect the owner or another snapshot.
- In-progress snapshots are consistent prefixes and may differ over time. A
  Ready-canceled owner instead has a valid final snapshot containing only
  `CancellationObserved` and `Terminal(canceled)`; it has no admission or start
  event. This final snapshot differs from an invalid zero-owner snapshot.
- A final snapshot is byte-for-byte stable across repeated/concurrent calls.
- Schema version is present in every valid snapshot, including in-progress and
  interrupted snapshots.
- Invalid zero snapshots have version zero, no events, no terminal, and cannot
  be confused with success.
- Terminal state and all preceding events become visible atomically under the
  same mutex before readers observe final status.

No serialization, persistence format, durable identity, or public projection is
implied by this in-memory snapshot.

## Compatibility and authority boundary

P13 must preserve:

- existing Run/Sequence single-use claims, result/error values, cancellation
  winners, `Done` publication, timeouts, and panic behavior;
- fixed caller-created tool authority, one admitted handler call, no recursion,
  no retry, and no model-created authority;
- provider request/result validation and safe error reduction;
- RunHost admission, shutdown cancellation, drain, and restart semantics;
- Phase 11 process ownership and Phase 12's absence of a verified-launch claim.

The evidence core cannot retain or invoke a provider, handler, authority,
context, cancel function, process, or host. Snapshot inspection cannot mutate
execution. Evidence types remain private/internal.

Public Go API: **NO**. CLI: **NO**. Manifest/schema: **NO**. Package format:
**NO**. Persistence: **NO**. Network/exporter: **NO**. Background worker:
**NO**. New execution authority: **NO**. Dependency change: **NO**.

## Deterministic acceptance-test contract

Future packages must use fakes, channels, barriers, atomics, and existing
lifecycle hooks rather than sleeps or live providers.

| Case | Required evidence proof |
|---|---|
| successful text Run | admitted, started, provider begin/end, usage, succeeded terminal in exact order |
| failed Run | safe category only, zero raw error data, failed terminal |
| canceled Run | cancellation recorded only when observed; canceled terminal matches Run winner |
| Ready cancellation of Run or Sequence | `Cancel` wins the Ready claim under the lifecycle lock; final snapshot has cancellation then canceled terminal, no admission/start/delegation event, and terminal evidence precedes `Done` closure |
| `Execute` versus Ready `Cancel` race | exactly one claim wins; the winning path alone writes its valid ordered events and later calls cannot append |
| panic unwind | original panic identity propagates; failed terminal only where existing owner publishes it, otherwise immutable interrupted snapshot |
| successful Sequence | indices 0..N-1 ordered, child boundaries nested, one sequence terminal |
| mid-sequence failure | completed prefix only; no later step admission; failed terminal |
| cancellation between steps | no next child begin; cancellation and terminal order is deterministic |
| cancellation during step | active child prefix closes consistently; no detached work |
| authorized tool success | provider/tool nesting, one tool begin/end, no identity or payload |
| tool-handler failure | fixed safe category; no handler error/panic/output retained |
| provider failure | fixed safe category and boundary end; no raw provider data |
| known usage | exact validated counters with known marker |
| unknown usage | unknown marker preserved; never converted to known zero |
| 64/96 tool usage | accepted aggregate recorded without reapplying single-turn cap |
| maximum Sequence | eight steps fit within exactly the 148-event ceiling |
| event overflow | deterministic panic, owner poisoned, no dropped/wrapped events or fabricated success |
| integer overflow | deterministic invariant failure; no wrapped sequence or usage count |
| terminal uniqueness | second terminal and any post-terminal append panic without mutating final snapshot |
| snapshot immutability | defensive copies cannot mutate owner or sibling snapshots |
| concurrent snapshots | race-free consistent prefixes and stable final copies |
| zero/incomplete owner | explicit invalid snapshot; no success/terminal evidence |
| forbidden data adversary | canary prompts, output, arguments, errors, keys, paths, IDs, handles, and pointers cannot enter or format through evidence |
| race detector | concurrent cancellation, terminal publication, and snapshot inspection are clean |
| zero authority effect | with evidence enabled/inspected versus absent test harness, provider/tool call counts, ordering, results, errors, cancellation, and panic identity are identical |

Tests must also verify fixed struct field types by reflection, reject arbitrary
metadata, prove no collector goroutine/I/O, and prove final snapshots expose no
capability-bearing value.

## Package sequence

### P13-A1 — Architecture freeze / ADR-006

Documentation, schema, ownership, redaction, bounds, failure semantics,
compatibility, and acceptance-test contract. No implementation.

### P13-B1 — Dormant private bounded evidence core

Implement the private owner, fixed events, validation, bounds, terminal/
interruption state, snapshots, concurrency, and unit/race tests. No production
Run/Sequence wiring.

### P13-B2 — Run and Sequence lifecycle integration

Compose the owner with existing Run and Sequence transitions while preserving
results, cancellation, usage, panic, and host-independent behavior.

### P13-B3 — Authorized tool and provider boundary evidence

Add fixed boundary and safe-category evidence without identity or payload data.
Prove authority and provider behavior are unchanged.

### P13-B4 — RunHost integration and private diagnostic projection

Compose evidence lifetime with host admission/shutdown and add only an
explicitly approved private projection. No public API/CLI, persistence, or
exporter.

### P13-C0 — Final audit / documentation / evidence closure

Audit integrated schema, redaction, bounds, authority neutrality, limitations,
and exact-main CI. Add no runtime capability.

Completion of A1 authorizes none of the B packages.

## Deferred and rejected scope

Deferred: durable history, execution IDs, serialization, crash recovery,
resume/replay, public inspection, CLI output, OpenTelemetry, distributed traces,
metrics, remote export, model evaluation, arbitrary tags, process/native events,
provenance, and capability-policy expansion.

Rejected for this architecture: arbitrary maps, caller-defined fields,
free-form event names, raw error strings, payload capture followed by redaction,
unbounded event lists, ring-buffer overwrite, best-effort silent drops,
background collectors, callbacks, and using evidence to decide execution.

## Release boundary

The published `v0.4.0-alpha.1` release remains unchanged and contains none of
Phases 10, 11, 12, or 13. P13-A1 selects no version, tag, release, or asset.
