# Forge Roadmap

> Engineering roadmap for the Forge platform.

**Latest published prerelease:** [v0.4.0-alpha.1](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1).
Publication source: `85d78143db1b8bcf2f96b79d681895b1a0492642`, tree
`e58a864eb6617811cca1dcb8d2d30bbd72e19298`; source-only, zero uploaded assets.
**RR-004-PUB: CLOSED / PASS — PUBLISHED.**
Phase 8 bounded AI/tool foundation is **CLOSED / PASS**, including real-provider
acceptance, and published in 0.4.0-alpha.1. Phase 9 is **CLOSED / PASS — BOUNDED
AGENT EXECUTION LIFECYCLE**, included as internal/pre-stable infrastructure.
P9-A1/B1/B2/B3/C0 are **CLOSED / PASS — INTEGRATED**; no further Phase 9 runtime
implementation is required. RR-005 is the documentation-only post-publication
reconciliation package; it adds no runtime capability. PR #24 integration and
strict push-main CI are the closure gate.
**Phase 10 architecture selected by Control Room: Bounded Workflow Composition
— Synchronous Sequences.** P10-A1 is **CLOSED / PASS — INTEGRATED**.
P10-B1/B2/B3 are **CLOSED / PASS — INTEGRATED**. The selected internal scope is
functionally complete on current main, not included in v0.4.0-alpha.1.
P10-C0 is the final documentation/audit closure package; its integration plus
strict exact push-main CI establishes **CLOSED / PASS — BOUNDED WORKFLOW
COMPOSITION (Synchronous Sequences)**. That gate passed on main
`bad51ed6ea7f476432c656036288f660281dbd50`, strict push-main CI `34924714319`.

**Phase 11 is DEFINED: Native Process Scope Ownership and Deterministic Cleanup.**
P11-A0 is **CLOSED / PASS — SELECTION ACCEPTED**. P11-A1 is
**CLOSED / PASS — INTEGRATED**, as are P11-B1/B2. P11-B3 is separately authorized
for private Windows creation-time Job mechanism/proof only. P11-B4/C0 remain
**NOT AUTHORIZED / NOT STARTED**. The A0 report's earlier undefined status is
historical and superseded by Control Room approval; selection is not reopened.
See [ADR-004](docs/architecture/adr/ADR-004-native-process-scope-ownership.md)
for the accepted architecture and bounded B1/B2/B3 records. B3 integration plus
strict exact push-main CI establishes its CLOSED / PASS — INTEGRATED status.
B3 does not integrate ProcessRunner or authorize release work.

---

# Vision

Forge is an AI-native application platform that enables developers to build, validate, package, and run applications using a manifest-driven architecture.

The roadmap below defines the planned evolution of the platform from foundation to stable release.

---

# Development Principles

Every milestone must satisfy the following principles:

- Manifest First
- Modular Architecture
- AI Native
- Testable
- Extensible
- Developer Friendly
- Open Source

---

# Release Timeline

| Version | Status |
|----------|--------|
| Pre-Alpha | Completed |
| Alpha | In Progress / Active |
| Beta | Planned |
| RC | Planned |
| Stable 1.0 | Planned |

Forge First Alpha is a local, manifest-driven, non-production technical
preview with pre-stable APIs and package formats. It establishes the bounded
validate/package/inspect/trust/run workflow; future roadmap capabilities remain
intentionally incomplete.

---

# Phase 0 — Foundation ✅

Objective:

Build the engineering foundation.

Deliverables:

- GitHub Repository
- Go Workspace
- Git
- VS Code
- Repository Structure
- Bootstrap
- Professional Documentation

Status:

Completed

---

# Phase 1 — Core Foundation

Objective:

Build reusable core libraries used by the entire platform.

Modules:

- Logger
- Configuration
- Errors
- File System
- Version
- Platform

Deliverables:

- pkg/
- internal/

Status:

**Alpha-Bounded Closed.** Configuration, logging, errors, lifecycle,
dependency injection, version hooks, and the required Go-native filesystem and
platform behavior support the First Alpha boundary. Generic filesystem and
platform abstractions remain possible future expansion; release stamping is an
external workflow/release responsibility.

---

# Phase 2 — CLI

Objective:

Provide a professional command-line interface.

Commands:

- forge init
- forge validate
- forge build
- forge inspect
- forge fmt
- forge version

These are long-term Phase 2 command targets. Published `v0.3.0-alpha.1`
provides `forge version`, `forge doctor`, `forge config`, `forge validate`,
`forge build`, `forge build-runnable`, `forge inspect`, and `forge run`.
Published `v0.4.0-alpha.1` additionally provides `forge ai prompt`.
The historical `forge init` and `forge fmt` targets remain deferred.

Status:

**Alpha workflow implemented; long-term CLI expansion remains planned.** This
is not a formal full Phase-2 closure decision.

---

# Phase 3 — Manifest Engine

Objective:

Implement manifest parsing.

Capabilities:

- YAML Loader
- JSON Loader
- Schema Validation
- Manifest Resolution

Status:

Completed for the current strict manifest contract.

---

# Phase 4 — Validation Engine

Objective:

Validate every manifest before execution.

Capabilities:

- Schema Validation
- Dependency Validation
- Runtime Validation
- Semantic Validation

Status:

**Alpha validation workflow implemented; long-term validation expansion
remains planned.** Strict YAML/JSON admission, structural/build/runnable
profiles, deterministic diagnostics, and `forge validate` are implemented.
This is not a formal full Phase-4 closure decision.

---

# Phase 5 — Registry

Objective:

Manage reusable packages.

Capabilities:

- Local Registry
- Remote Registry
- Version Resolution
- Dependency Graph

Status:

**Alpha-Bounded Closed.** The local invocation-scoped contract covers exact
package identities and sources, deterministic dependency resolution/order,
non-mutating admission snapshots, and controlled build-path commit. Remote
registry/acquisition, persistence, version ranges, indexing, and complete
provenance remain planned.

---

# Phase 6 — Compiler

Objective:

Compile manifests into executable runtime bundles.

Capabilities:

- Packaging
- Optimization
- Artifact Generation

Status:

Closed / Pass for the bounded Pre-Alpha compiler/package/runnable pipeline

---

# Phase 7 — Runtime

Objective:

Execute packaged applications.

Capabilities:

- Loader
- Scheduler
- Runtime Engine
- Plugin Loader

Status:

**Alpha-Bounded Closed.** The bounded runtime loads an explicitly trusted,
signed local runnable package, authorizes the host, materializes privately,
executes one direct no-shell child with bounded output, waits/reaps, maps the
result, and cleans up. Scheduling, dynamic plugin loading, richer inputs,
process-tree control, graceful shutdown, sandboxing, quotas, and persistent
trust remain planned.

---

# Phase 8 — AI Runtime

Objective:

Provide AI-native capabilities.

Accepted bounded capabilities (published in v0.4.0-alpha.1):

| Stage | Accepted capability |
|---|---|
| Prompt Execution | Bounded single-prompt core, one OpenAI adapter, and CLI |
| B1 | Function-call response mapping/admission |
| B1-HYG | Test isolation hygiene |
| B2 | Bounded function-tool request serialization |
| C1 | Explicit Tool Execution Authority |
| C2 | `function_call_output` serialization |
| C3 | Correlation / replay-safe invocation coordination |
| C4 | Bounded stateless single function round-trip |
| C5 | Immutable tool authority bundle |
| C6 | One built-in read-only CLI tool: `forge_runtime_info` |
| C7 | Explicit CLI `--allow-tools` opt-in |
| C8 | Real-provider acceptance — PASS; validation gate, not a code package |
| C9 | Terminal reasoning-item compatibility |
| C10 | Safe stage-local diagnostics |

Status: **CLOSED / PASS** for this bounded Phase 8 scope.
Real-provider acceptance: **PASS**.

The accepted source baseline is
`ac68a1b3e059f173d9b5c71eadaff9fcffba981f`, tree
`e4147daf4e0312e4622266dff5284c95c06ac811`.
[Push-main workflow 34559981995](https://github.com/kaizenforyou91/forge/actions/runs/34559981995)
passed Ubuntu acceptance, Windows acceptance, and Ubuntu race on that exact SHA.
Owner-provided C8 evidence records authentication, direct text, a real function
proposal, authorized local execution, `function_call_output`, a stateless second
provider turn, and final response acceptance. The returned runtime identity
matched local version, commit, and build time (3/3). No live call is part of
this documentation reconciliation.

The production tool boundary remains explicit opt-in, exactly one read-only
built-in tool, at most two provider POSTs and one handler attempt, no retry,
and no recursive tool loop. Both tool requests preserve `store:false`,
`stream:false`, `background:false`, and `parallel_tool_calls:false`; POST #2
uses `tool_choice:"none"`. No stateful response/conversation fields or reasoning
replay are introduced. Diagnostics remain hidden/default-off and use fixed
stage labels rather than raw response logging.

This closure does not claim arbitrary tool support, autonomous Agent Runtime,
AI memory, Workflow Engine, scheduler, multi-provider compatibility, or Beta
readiness. Agent operation ownership begins in Phase 9; memory, workflow, and
scheduling require separate future gates. The existing Phase 6 closure remains
unchanged.

Historical single-prompt evidence (preserved, superseded for current status):

The single-prompt core, one OpenAI Responses adapter, and
`forge ai prompt` are integrated into main through
[PR #1](https://github.com/kaizenforyou91/forge/pull/1), including the accepted
error-sanitization remediation. Code review is **ACCEPTED** and the offline
contract is **PASS**. Main Ubuntu/Windows acceptance and focused race, including
`pkg/ai` and `internal/aiprovider/openai`, are **PASS** at
`bef4874020403e154680f0e682c1b79aed0b937d` in
[workflow 34099866050](https://github.com/kaizenforyou91/forge/actions/runs/34099866050).

At that earlier checkpoint, Phase 8 was **IN PROGRESS**, Tool Calling was
described as **future**, and live acceptance was **NOT AUTHORIZED / NOT RUN**,
pending separate Owner approval. Those statements describe the historical
single-prompt slice; integrated B1/B1-HYG/B2/C1–C7/C9/C10 and C8 evidence supersede
them for current main. The AI additions are published in **v0.4.0-alpha.1**
and absent from historical `v0.3.0-alpha.1`. External pilot remains
**DEFERRED / NON-BLOCKING / NOT RUN** and is not a dependency blocker.
The earlier local Windows focused race remains **NOT RUN** because that session
lacked CGO/compiler support; hosted race PASS does not change that history.

The [single-prompt workflow](docs/AI_PROMPT_WORKFLOW.md) preserves the earlier
text-only contract and offline/live distinction, with old pending-live/no-tool
wording explicitly historical. RR-001 reconciled current-main documentation;
RR-005 reconciles publication status after the separate RR-004-PUB authorization.

---

# Phase 9 — Bounded Agent Execution Lifecycle

Status: **CLOSED / PASS — BOUNDED AGENT EXECUTION LIFECYCLE**.
P9-C0: **CLOSED / PASS — INTEGRATED**. The documentation/offline closure package
merged in PR #20 and passed strict push-main CI. No additional runtime package
is required.
P9-A0 is **CLOSED / PASS**. P9-A1 approved the architecture in
[ADR-002: Bounded Agent Run Ownership](docs/architecture/adr/ADR-002-agent-run-ownership.md).

Delivered objective: give one explicitly requested AI operation one execution
owner, observable lifecycle state, explicit cancellation coordination, and
deterministic terminal completion, including explicit application-host shutdown
composition. `pkg/app.Runtime` remains the application lifecycle owner;
`internal/agent.Run` owns one operation and `RunHost` composes the two.
App startup, module registration, shutdown, and restart semantics are unchanged.
No lower `pkg` package depends on `internal/agent`.

Historical B3 implementation baseline before documentation closure:
`d7d660f65fc5fc794a545f8b64703562e92b8bc7`, tree
`3f78b90c337f7e78ba8797967349a9ac91d14bf6`.

Final Phase 9 closure baseline (P9-C0 merge):
`923fba140bc76ef30128b30011955539f3c1ec86`, tree
`4143a413ad106c60261a3bf180045420951bf6cf`.
[Push-main CI 34761487262](https://github.com/kaizenforyou91/forge/actions/runs/34761487262)
is **completed / success** on that exact SHA: Ubuntu acceptance **PASS**,
Windows acceptance **PASS**, and Ubuntu race **PASS**, without retry or waiver.

| Package | Scope | Authorization/status |
|---|---|---|
| P9-A1 | Architecture decision + roadmap reconciliation | CLOSED / PASS — INTEGRATED |
| P9-B1 | Single-use AI Run ownership over existing text execution | CLOSED / PASS — INTEGRATED |
| P9-B2 | Existing authorized tool round-trip composed with Run lifecycle, preserving Phase 8 bounds | CLOSED / PASS — INTEGRATED |
| P9-B3 | Bounded application-host/shutdown composition | CLOSED / PASS — INTEGRATED |
| P9-C0 | Offline integration / architecture closure | CLOSED / PASS — INTEGRATED |

P9-A1 approved architecture only. B1, B2, and B3 each received separate Control
Room implementation/integration authorization; P9-A1 did not pre-authorize them.

Accepted bounded capabilities (internal/pre-stable, included in v0.4.0-alpha.1):

- Run ownership: Ready / Running / Succeeded / Failed / Canceled; atomic single
  execution claim shared by copies; stable Done; explicit Cancel; fixed/redacted
  state and errors; terminal reference release; no retry, Run-created worker
  goroutine, or persistence. Unknown is the fail-closed invalid state.
- Text execution: caller-supplied `ai.Provider`, unchanged `ai.Executor`, explicit
  timeout/context, and at most one delegated operation per Run.
- Authorized tool composition: caller-supplied `AuthorizedToolRoundTripper` and
  immutable `tool.Authority`; existing C5/C4 execution remains authoritative.
  One Run delegates at most one round trip, with at most two provider POSTs and
  one handler attempt inside it. Two-turn Usage is preserved: aggregate
  `Usage.OutputTokens` may exceed per-turn `MaxOutputTokens` (96 > 64 regression
  PASS). `ai.Result.Validate()` and defensive Usage copying apply; no second
  `ai.Executor` aggregate token check is added.
- Application-host composition: internal `RunHost` implements `app.Module`;
  explicit `App.Add` registration and caller `Host.Execute` only. Register/Start
  execute zero Runs; admission requires a fully Running App and healthy current
  context. App cancellation reaches active Runs; Stop closes admission, cancels,
  and drains hosted work. Restart captures a fresh application context. Shutdown
  is cooperative, with no forced termination or result/error history.

Phase 9 changes lifecycle composition only; it does not enlarge the Phase 8
authority described above. CLI `--allow-tools` remains explicit/default false,
the single built-in remains `forge_runtime_info`, and C10 diagnostics stay safe
and separate from RunHost. No CLI migration or diagnostic composition occurred.

This closure does not deliver autonomous planning/loops, multi-agent execution,
recursive tool conversations, AI memory, durable jobs, persistence, workflow
engine, scheduler, queue, worker pool, background execution, arbitrary tool
catalog, additional built-in tools, filesystem/network/subprocess agent tools,
provider routing, multi-provider compatibility, public `pkg/agent` API, or Beta
readiness. Memory, scheduler, and tool/provider expansion require new selection
gates. Bounded synchronous workflow composition was separately selected by
post-release P10-A0-R1; P10-A1/B1/B2/B3 are CLOSED / PASS — INTEGRATED.
P10-C0 is the final closure package; integration plus strict exact push-main CI
establishes final bounded-scope closure.
Publication required separate
release governance and did not broaden Phase 9 authority.
At the v0.4.0-alpha.1 publication checkpoint, Phase 10 was not yet defined or
authorized. The subsequent Phase 10 architecture and implementation below change
no Phase 9 historical authority or published runtime capability.

Release status: **0.4.0-alpha.1 PUBLISHED**. Separate governance through RR-001
to RR-004-PUB authorized the source-only prerelease (zero assets) at
`85d78143db1b8bcf2f96b79d681895b1a0492642`.
[Publication-source CI 34786373253](https://github.com/kaizenforyou91/forge/actions/runs/34786373253)
passed Ubuntu/Windows acceptance and Ubuntu race. P9-C0 itself did not authorize
publication. RR-005 reconciles documentation afterward, without moving tags.
Historical `v0.3.0-alpha.1` remains at
`5d836931216203aeea0737fc54de9e95091a62ef`.

---

# Phase 10 — Bounded Workflow Composition

## Synchronous Sequences

**Architecture definition selected by Control Room.** P10-A0-R1 is
**CLOSED / PASS — OUTCOME A ACCEPTED**. P10-A1 is
**CLOSED / PASS — INTEGRATED**, as are P10-B1/B2/B3. P10-C0 is the final
closure package, adding documentation and audit only. P10-C0 integration plus
strict exact push-main CI establishes Phase 10
CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION (Synchronous Sequences).

Objective: compose a finite, explicit sequence of already-authorized AI operations
synchronously while preserving caller authority, lifecycle ownership, fail-fast
semantics, cancellation, and bounded resource behavior.

[ADR-003: Bounded Workflow Composition — Synchronous Sequences](docs/architecture/adr/ADR-003-bounded-workflow-composition.md)
is the detailed architecture decision. Historical P10-A1 preparation baseline:
`217be79f938b6f913c08ae86e1c918896e66fd68`, tree
`4a1393af3eceae4dd8a2db6b87a2f5968ebe4680`;
[strict main CI 34799363981](https://github.com/kaizenforyou91/forge/actions/runs/34799363981)
is attempt 1, push/main, completed/success on that SHA with Ubuntu/Windows
acceptance and Ubuntu race PASS. Published v0.4.0-alpha.1 remains immutable;
architecture selection itself added no runtime capability to main or that release.

Historical P10-A1 integrated main / P10-B1 preparation base:
`18e7e51cf366519db5521d7237c218ffd54c6fe6`, tree
`2eb9336df00ca52a479fd9ada72eaf16b2d22ffa`;
[strict main CI 34807777744](https://github.com/kaizenforyou91/forge/actions/runs/34807777744)
is attempt 1, push/main, completed/success with Ubuntu/Windows acceptance and
Ubuntu race PASS. P10-B1 implements standalone internal literal execution only:
`SequenceStep`, `Sequence`, and `SequenceResult{Final, CompletedSteps}` reuse
existing Run paths and State. Construction validates and snapshots without I/O;
one overall context bounds fresh, sequential child Runs. Failure/cancellation
returns zero result. Panic propagates while Sequence becomes consumed/Failed,
releases its references, and closes Done after unwind. Existing Run behavior
is unchanged. There is no previous-text handoff, aggregate Sequence usage, or
application-host admission in B1; the B2 record below extends handoff/accounting,
while the separately authorized B3 record below adds host admission.

Historical P10-B1 integrated main / P10-B2 preparation base:
`9e0c38b70d63446440328aca3fa5c5574ae68df8`, tree
`c0bbdbc747823c87d30c059e574ad93697bafaca`;
[strict main CI 34818568107](https://github.com/kaizenforyou91/forge/actions/runs/34818568107)
is attempt 1, push/main, completed/success for both acceptance jobs and Ubuntu
race. B2 adds explicit previous-step text constructors while preserving literal
constructors. Step 1 must remain literal; only the immediately preceding text
is copied unchanged into a fresh request and validated against the 16 KiB input
bound before constructing a child Run. Invalid handoff prevents all later work.
SequenceResult now adds AggregateUsage separately from Final.Usage: fresh checked
int64 sums when all usage is known, nil permanently after any unknown usage.
Known overflow fails immediately with ErrMalformedResponse and zero result;
unknown usage disables arithmetic, including possible later overflow. B1
lifecycle, deadlines, panic cleanup, and caller-fixed authority remain unchanged.
The B2 slice adds no host/app integration, public API, CLI, persistence, or background work.

### P10-B3 implementation record — whole-sequence host ownership

Historical P10-B2 integrated main / B3 preparation base:
`6d3cac19982129f785ce0a3153ace4feade055e4`, tree
`dc8f618ced2fd9c826e41c8ca30d8362280abb67`;
[strict main CI 34825138783](https://github.com/kaizenforyou91/forge/actions/runs/34825138783)
is push/main on that exact SHA, attempt 1, completed/success for Ubuntu/Windows
acceptance and Ubuntu race. P10-A1/B1/B2 are CLOSED / PASS — INTEGRATED.

B3 adds internal synchronous RunHost.ExecuteSequence(ctx, *Sequence). One unique
cancel-only entry owns the entire call, including children, inter-step gaps,
handoff/accounting, and panic unwind. Child Runs are not admitted separately.
Shared private admission preserves existing host/App running-generation checks.
Caller context remains parent; App cancellation only cancels the linked context.
Sequence alone owns timeouts, results, errors, handoff, and accounting unchanged.

Stop closes admission, snapshots owners, cancels Run or Sequence, and drains
every admitted call before publishing stopped. Non-cooperative work may block
shutdown; no worker, detachment, early Done, or fake completion exists. Unique
per-call entries ensure a losing duplicate cannot untrack a winner. Release
unlinks context cancellation, deletes the exact entry, clears its owner reference,
and completes wait accounting, including panic propagation. Restart captures
a fresh App context; consumed Sequences remain consumed.

Existing RunHost.Execute and module identity remain compatible. Sequence and
pkg/app source remain unchanged. No public API, CLI, persistence, background
work, or authority expansion. B3 is CLOSED / PASS — INTEGRATED.

### P10-C0 final offline closure record

P10-B3 integrated main / P10-C0 audit baseline:
`92ea08c5af041e6a204891059724811ab582111a`, tree
`00e178657f0fa746c6e1261df55251c46bae277e`; parents
`6d3cac19982129f785ce0a3153ace4feade055e4` and
`bc64d2f19e09fce0f40a6e08b52ec4ddea7d5a88`.
[Strict main CI 34920655351](https://github.com/kaizenforyou91/forge/actions/runs/34920655351)
is push/main on that exact SHA, attempt 1, completed/success: Ubuntu acceptance,
Windows acceptance, and Ubuntu race PASS. No C0 retry is pre-authorized.

P10-A1/B1/B2/B3 are CLOSED / PASS — INTEGRATED. The selected implementation is
functionally complete. C0 audits existing source/tests and reconciles current
documentation, including narrow ADR-002 status corrections; it adds no runtime
capability. P10-C0 integration plus strict exact push-main CI establishes Phase 10
CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION (Synchronous Sequences).
This closure gate passed at `bad51ed6ea7f476432c656036288f660281dbd50`,
with strict push-main CI `34924714319`, attempt 1, completed/success.
The immutable published release excludes Phase 10. The subsequent P11-A0
selection was accepted; Phase 11 is defined below, A1/B1/B2 are integrated, and
B3 is the separately authorized Windows mechanism/proof package. B4/C0 remain gated.

### Selected Phase 10 architecture

- INTERNAL, single-use Sequence in `internal/agent`; Ready / Running /
  Succeeded / Failed / Canceled, atomic execution claim, explicit Cancel,
  stable Done, deterministic normal terminal publication, reference release.
- Synchronous caller-goroutine execution; ZERO sequence worker goroutines.
  Exactly 1..8 steps. Eight is an intentional initial safety bound, not a measured
  product requirement. Static linear declaration order only, with no concurrency
  inside one sequence.
- Only text Run and authorized-tool Run semantics. Immutable specifications fix
  operation kind, provider/round-tripper, model, MaxOutputTokens, timeout,
  applicable immutable tool Authority, and input source before Execute.
- Inputs are exactly LITERAL_TEXT or PREVIOUS_STEP_TEXT; step 1 is literal.
  Previous text is the immediately preceding result, unchanged. Apply normal
  ai.Request.Validate before next provider work: blank, invalid UTF-8, >16 KiB,
  or otherwise invalid handoff fails. No truncation, summarization, template,
  concatenation, transformation callback, or automatic extra call.
- Construct each fresh Run only after input is known, through NewRun or
  NewAuthorizedToolRun. Do not mutate preconstructed Runs or export a generic
  executable operation constructor.
- Step timeouts satisfy ai.ValidateTimeout; overall timeout is positive and
  at most 120 seconds. The earliest caller, overall, and step deadline wins;
  subsequent steps receive only remaining time.
- Fail fast: zero later work after failure/cancellation. No retry, replay,
  resume, compensation, rollback, fallback, or alternate provider.
- Success returns final step Result and bounded aggregate metadata; final Run
  Usage is distinct from sequence usage. Failure returns zero success data and
  a safe error, never prior successful text as final output. Intermediate text
  exists only as needed; reference release is not secure erasure.
- Checked aggregate usage only when all completed steps report Usage; any nil
  means UNKNOWN/nil, never zero. Overflow fails safely. No per-Run token cap
  applies to the aggregate; retain the 64-cap / 96-aggregate tool regression.
- For T text and U tool steps, T+U<=8: accepted OpenAI paths allow at most T+2U
  POSTs and U handler attempts. Eight tool steps permit at most 16 POSTs and
  eight handler attempts; per-Run authority and bounds remain unchanged.
- Provider/custom trusted implementation panics may propagate. Defer/unwind
  must release applicable references and host admission bookkeeping; no
  universal panic-to-safe-error conversion or detached execution is introduced.

### Application ownership and authority

Whole-sequence host ownership includes inter-step gaps. Independent calls to
RunHost.Execute for each child Run are insufficient. Select private generalized
active-execution bookkeeping inside internal/agent/host.go, with narrow Run and
Sequence entry points. Admit a sequence once and drain the entire synchronous
call; no public/generic operation API is introduced. Existing RunHost.Execute
behavior remains compatible.

Stop closes admission, cancels the Sequence and active Run, prevents the next
step, and waits for the whole admitted call. Register/Start execute zero AI work.
Restart captures a fresh application context; old Sequence objects remain
consumed. Cancellation remains cooperative and cannot force a broken provider
to return. pkg/app remains lifecycle owner, with no production API change.

Authority comes only from the trusted caller and is fixed before execution.
Handoff transfers text data, never provider/model/tool/timeout/node authority.
No environment credential lookup, prompt/result/authority persistence, raw data
logging, credential serialization, history, or resume state is added. The CLI
remains direct Phase 8 composition, explicit network/tool opt-in, one read-only
forge_runtime_info built-in, and no RunHost/Sequence migration.

| Public surface decision | Answer |
|---|---|
| Public API / CLI change | NO / NO |
| Manifest / package format change | NO / NO |
| Persistence / background execution | NO / NO |
| Network / tool authority expansion | NO / NO |

Architecture Freeze v1.0 remains unchanged. Composition stays above pkg/app,
pkg/ai, pkg/ai/tool, and existing internal/agent primitives. Lower layers must
not import higher composition; no Freeze exception is required.

### Non-goals and separate debt

No autonomous agents, planning loops, multi-agent systems, durable AI memory or
jobs, scheduler, queue, worker pool, background work, generic workflow engine,
DAG, parallel sequence execution, branching/join, dynamic/model-created steps,
expression language, generic tool marketplace, new built-ins, filesystem or
shell/subprocess tools, provider routing/fallback, remote registry/distribution,
public agent API/CLI, native/runtime/package steps, or production sandboxing.

Validated-object-to-path execution binding, same-user package mutation, Windows
ACL/reparse/share-mode hardening, process-tree/graceful native shutdown,
persistent trust lifecycle, key rotation/revocation, and provenance/SBOM
completeness remain IMPORTANT but are NOT P10 blockers. Control Room selected
native process scope ownership as Phase 11; P11-A1/B1/B2 are integrated and
P11-B3 Windows mechanisms are separately authorized. Production runner
integration remains unimplemented. The other hardening families remain separate accepted debt. Closed
Phase 10 authority does not expand through the later selection.

### Package status and acceptance

| Package | Defined scope | Status / integration dependency |
|---|---|---|
| P10-A1 | Architecture definition, ADR, roadmap | CLOSED / PASS — INTEGRATED; architecture only |
| P10-B1 | Internal sequence lifecycle and static literal-step execution | CLOSED / PASS — INTEGRATED |
| P10-B2 | Previous-text handoff and bounded aggregate accounting | CLOSED / PASS — INTEGRATED |
| P10-B3 | Whole-sequence application-host integration | CLOSED / PASS — INTEGRATED |
| P10-C0 | Offline integration / architecture closure | Final documentation/audit closure package; integration plus strict exact push-main CI establishes bounded Phase 10 closure |

P10-A1 integration did not authorize implementation. B1/B2/B3 each integrated
under separate approval. B3 preserved Sequence and pkg/app source. C0 is separately
authorized for documentation and read-only audit only, with no new test or source.

Existing deterministic offline tests cover single claim, sequential order,
zero/>8 rejection, valid step types, literal/handoff validation, zero next calls
on failure, all deadline/cancellation boundaries, application Stop during and
between steps, drain/restart, known/unknown/overflow usage, 64/96 preservation,
immutable authority, redaction, terminal reference release, concurrent Execute,
race behavior, and panic unwind bookkeeping. Use synchronization rather than
sleeps where possible. Preserve Ubuntu/Windows acceptance for dependency
metadata/cleanliness, listing, vet, full tests and build; Ubuntu race retains
pkg/compiler, runtime, internal/cli, pkg/ai, internal/aiprovider/openai,
pkg/ai/tool, and internal/agent. No new dependency is currently justified.

**NO NEW LIVE PROVIDER VALIDATION REQUIRED** for composition of unchanged
accepted paths. Wire serialization, HTTP, decoding, or provider/tool round-trip
changes in a future package must reopen freshness review. C0 makes no live
call and accesses no API key.

No version or release is selected/authorized. Architecture definition has no
automatic publication effect; the integrated Phase 10 scope may be considered
for an alpha feature release only through a separate gate. No Beta readiness
or new production guarantee is implied.

---

# Phase 11 — Native Process Scope Ownership and Deterministic Cleanup

**DEFINED. P11-A0: CLOSED / PASS — SELECTION ACCEPTED.**
P11-A1: **CLOSED / PASS — INTEGRATED**.
P11-B1: **CLOSED / PASS — INTEGRATED**.
P11-B2: **CLOSED / PASS — INTEGRATED**.
P11-B3: **SEPARATELY AUTHORIZED — WINDOWS MECHANISM/PROOF ONLY**.
P11-B4/C0: **NOT AUTHORIZED / NOT STARTED**.

Objective: own one native launch scope from creation through termination and
cleanup while preserving direct-child results, caller cancellation, and existing
launch authority. B1 supplies coordination; B2/B3 supply private Linux/Windows
primitives; production scope ownership is not integrated and Phase 11 is not
functionally complete.

[ADR-004: Native Process Scope Ownership and Deterministic Cleanup](docs/architecture/adr/ADR-004-native-process-scope-ownership.md)
records the accepted detailed contract. Historical A1 preparation baseline:
`bad51ed6ea7f476432c656036288f660281dbd50`, tree
`2aaca2b07ebdeaf3694d170b16c49f1ca1711dfa`; strict push-main CI
[34924714319](https://github.com/kaizenforyou91/forge/actions/runs/34924714319),
attempt 1, PASS for Ubuntu/Windows acceptance and Ubuntu race.

The proposed boundary is one private process-scope owner, membership established
before child code runs, immediate force termination, exact direct-child reaping,
and explicit cleanup/error outcomes. Linux process-group control and Windows
creation-time Job Object membership have different guarantees. macOS retains
historical direct-child behavior only; no strengthened descendant claim is made. A successful
termination request is not proof that every descendant has exited. Safe identifier
lifetime, platform launch mechanics, and native acceptance remain proof obligations
for separately authorized implementation packages.

Preserve public result shape, CLI grammar, package/manifest formats, direct launch
without shell injection, output bounds, and caller-fixed authority. No persistence,
scheduler, worker pool, graceful shutdown protocol, hostile-process containment,
AI integration, or provider/tool authority expansion is selected. pkg/app remains
application lifecycle owner; Architecture Freeze remains unchanged.

| Package | Proposed purpose | Authorization |
|---|---|---|
| P11-A1 | Architecture/ADR and focused roadmap definition | CLOSED / PASS — INTEGRATED |
| P11-B1 | Private platform-neutral scope ownership/terminal coordination | CLOSED / PASS — INTEGRATED |
| P11-B2 | Private Linux group and identifier-lifetime mechanism/proof | CLOSED / PASS — INTEGRATED |
| P11-B3 | Windows creation-time job membership and resource ownership | Separately authorized; integration + strict exact push-main CI establishes closure |
| P11-B4 | Runtime integration and compatibility evidence | NOT AUTHORIZED / NOT STARTED |
| P11-C0 | Final architecture/integration closure | NOT AUTHORIZED / NOT STARTED |

Existing Phases 6–10 stay closed. Executable object binding, same-user package
mutation, Windows filesystem parity, durable trust, and provenance remain separate
accepted debt. The A0 sentence that Phase 11 was not defined describes the earlier
selection output only; Control Room approval supersedes it. No A0 rerun is needed.

Published v0.4.0-alpha.1 contains neither Phase 10 nor Phase 11 and stays immutable.
No release or version is selected. A1 integration does not authorize later
packages. B3 is separately authorized and adds no production runner integration,
Phase 11 closure, Beta readiness, or production readiness.

### P11-B1 implementation record

`runtime/process_scope.go` defines one private owner and a two-operation private
platform contract. PREPARED resources can finalize without termination; ACTIVE
owners serialize manual, cancellation and natural-exit cleanup requests. Only a
successful request records a winner, and success is not descendant quiescence.
Ordinary control failure permits a later caller request and preserves the first
failure. Finalization is synchronous, exactly once and caches its outcome,
releasing the platform reference. Trusted panics propagate; interrupted control
is not reusable, and interrupted finalization cannot release resources twice.
Fake-platform tests prove call counts and concurrent ordering without native
processes or sleeps. B1 creates no goroutines and changes no existing execution
path. P11-B1 is CLOSED / PASS — INTEGRATED at main
`cf1bb4069210f63ba7bf415b93c992ab753350c5`, strict push-main CI `34941814984`,
attempt 1 PASS. These are B1 slice boundaries, not a claim that B2 is absent.

### P11-B2 implementation and native proof gate

Linux-only private preparation sets Setpgid=true/Pgid=0 before Start and consumes
a preparation receipt after successful launch; identity comes from the direct
child PID. Conflicting group/session/tracing/namespace settings fail closed.
Blocking waitid(P_PID, WEXITED | WNOWAIT) observes exit without reaping and
retries EINTR only. Owned-group SIGKILL maps ESRCH to os.ErrProcessDone, which
cannot become a B1 success winner. Control success still does not prove quiescence.

The required sole-owner ordering is observation, terminal control, control-identity
retirement, then normal reap. Retirement clears the PGID and OS seams before reap;
no stale group signal follows it. This is a structural ordering proof, not a claim
of safety if another waiter reaps early. B4 must compose it with the existing
runner, output and lease lifecycle; no production path calls these primitives yet.

Deterministic native Go helpers cover membership, non-reap then normal reap,
leader-first exit with inherited-pipe closure, an independent sentinel, and a
controlled setsid escape followed by explicit cleanup. Hosted Ubuntu acceptance
and race supply native evidence. Windows acceptance is regression evidence only;
Linux cross-compilation is compile-only. x/sys remains v0.13.0, promoted from
indirect to direct use with go.sum unchanged. B2 is CLOSED / PASS — INTEGRATED
at main `f7337587d72315a77dc21bc29ad18c71b49069ce`, strict push-main CI
[34949872571](https://github.com/kaizenforyou91/forge/actions/runs/34949872571),
attempt 1 PASS. B3 is separately authorized below. B4/C0 remain
NOT AUTHORIZED / NOT STARTED. macOS remains direct-child only.

### P11-B3 Windows mechanism and native proof gate

The private Windows launcher creates one unnamed, non-inheritable Job, sets only
KILL_ON_JOB_CLOSE, and supplies exactly that Job in the creation-time JOB_LIST
attribute. Explicit duplicated stdio handles are the only HANDLE_LIST entries.
No post-start assignment, suspended fallback, host-Job mutation, breakaway or
administrator operation is used. Unsupported attributes or nesting fail start.
Job control and direct-child handles have separate owners. TerminateJobObject
with exit code 1 acknowledges a request only; process signaling and pipe EOF are
separate proof. Finalization retires/closes the Job once and preserves failure.
Kill-on-close is failure safety, not the normal B1 control winner.

Fake tests cover exact handle cleanup and partial failures. Native Go helpers
cover membership, ordinary descendants, outside sentinels, test-owned outer/inner
Jobs and kill-on-close. Hosted Windows acceptance is canonical native B3 proof;
Ubuntu acceptance/race remain regression gates. B3 integration plus strict exact
push-main CI establishes CLOSED / PASS — INTEGRATED. B1/B2 and production runner
paths remain unchanged; no public API, CLI, format, persistence or dependency
change. B4/C0 remain NOT AUTHORIZED / NOT STARTED; Phase 11 is not closed.

---

# Engineering Milestones

## Milestone 1

Engineering Documentation

Status:

Completed

---

## Milestone 2

Foundation Library

Status:

Alpha-Bounded Closed; long-term foundation expansion remains planned

---

## Milestone 3

CLI

Status:

Alpha workflow implemented; long-term CLI expansion remains planned

---

## Milestone 4

Manifest Engine

Status:

Completed for the current strict manifest contract

---

## Milestone 5

Validation Engine

Status:

Alpha validation workflow implemented; long-term expansion remains planned

---

## Milestone 6

Registry

Status:

Alpha-Bounded Closed for the local exact-identity registry boundary; remote,
persistent, and advanced-resolution capabilities remain planned

---

## Milestone 7

Compiler

Status:

Closed / Pass for the bounded Pre-Alpha compiler/package/runnable scope;
broader compiler capabilities remain planned

---

## Milestone 8

Runtime

Status:

Alpha-Bounded Closed for the trusted local direct-child boundary; scheduler,
dynamic loading, richer inputs, and isolation remain planned

---

## Milestone 9

AI Runtime

Status:

The historical Milestone 9 label covers the broader AI Runtime family; it is
not the new Phase 9 number. The bounded Phase 8 AI/tool foundation is
**CLOSED / PASS**, including real-provider acceptance, and is published in v0.4.0-alpha.1.
Bounded agent Run and application-host ownership are integrated through P9-B3;
P9-C0 is CLOSED / PASS — INTEGRATED and Phase 9 is CLOSED / PASS. Autonomous agents, memory,
and Workflow Engine remain future, separately gated work.

---

# Definition of Done

A milestone is considered complete when:

- Design completed
- Documentation updated
- Implementation finished
- Tests passing
- Code reviewed
- Changes merged into main

---

# Current Engineering Status

The original roadmap describes the long-term evolution of Forge.
Implementation progress is tracked separately through engineering milestones.

## Current Phase Status

| Phase | Status |
|---|---|
| Phase 0 — Foundation | ✅ Completed |
| Phase 1 — Core Foundation | ✅ Alpha-Bounded Closed; long-term expansion planned |
| Phase 2 — CLI | Alpha workflow implemented; long-term expansion planned |
| Phase 3 — Manifest Engine | ✅ Complete for current strict contract |
| Phase 4 — Validation Engine | Alpha validation workflow implemented; long-term expansion planned |
| Phase 5 — Registry | ✅ Alpha-Bounded Closed; local exact-identity boundary |
| Phase 6 — Compiler | ✅ CLOSED / PASS — bounded Pre-Alpha compiler/package/runnable pipeline |
| Phase 7 — Runtime | ✅ Alpha-Bounded Closed; trusted local direct-child boundary |
| Phase 8 — AI Runtime | CLOSED / PASS — bounded AI/tool foundation, real-provider PASS; published in v0.4.0-alpha.1 |
| Phase 9 — Bounded Agent Execution Lifecycle | CLOSED / PASS — BOUNDED AGENT EXECUTION LIFECYCLE |
| Phase 10 — Bounded Workflow Composition (Synchronous Sequences) | CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION; A1/B1/B2/B3/C0 integrated, exact main CI passed |
| Phase 11 — Native Process Scope Ownership and Deterministic Cleanup | DEFINED; A1/B1/B2 integrated; B3 separately authorized Windows mechanism/proof; B4/C0 NOT AUTHORIZED / NOT STARTED |

## Engineering Milestones

| Milestone | Status | Scope |
|---|---|---|
| FW-024 | ✅ Completed | Middleware Foundation |
| FW-025 | ✅ Completed | Observability Foundation |
| FW-026 | ✅ Completed | Roadmap Verification |
| FW-027 | ✅ Completed | Runtime Engine Foundation |
| FW-028 | ✅ Completed | CLI Foundation Completion |
| FW-029 | ✅ Completed | Plugin System Foundation |
| FW-030 | ✅ Completed | Manifest Engine Foundation |

## Post-FW-030 Implemented Checkpoints

The formal milestone list above currently ends at FW-030. The following
checkpoints describe tested and committed implementation evidence without
assigning new milestone or task identifiers:

- Exact dependency resolution, dependency graph construction, and deterministic
  dependency-first build order.
- CLI `forge build` vertical slice for YAML and JSON manifests.
- Multi-module dependency-aware compilation and packaging.
- Package identity and package source registries.
- Compiler execution abstraction, operating-system command runner, and
  import-path-aware toolchain execution.
- Artifact generation, deterministic artifact bundles, and deterministic ZIP
  packaging.
- Package integrity metadata and verification.
- Ed25519 signing and verification, trust store, and package verification policy.
- ZIP package read-back and artifact source provenance preservation.
- Direct artifact provenance coverage from toolchain execution through artifacts.
- Import-path-only integrity tampering coverage.
- Idempotent package-source `Ensure` with explicit source-conflict semantics.
- Failure-atomic `Registry.EnsureAll` and `PackageSourceRegistry.EnsureAll` with
  deterministic successful insertion order.
- Prebuilt `BuildPlan` execution through `CompileAndPackagePlan`.
- Pure `ManifestAdmissionPlan` preflight with snapshot-based dependency and
  source analysis.
- Commit-time source revalidation and controlled manifest admission.
- CLI integration through `AdmitManifest` and `CompileAndPackagePlan`.
- Predictable admission failures leave no candidate registration.
- Downstream executor or package failures retain accepted admission state.
- Repeated shared-application builds remain byte-deterministic.
- Explicit `package.json` compatibility metadata for package format v1 and
  artifact bundle schema v1.
- Independently versioned integrity schema v2 and signature schema v1.
- Duplicate-key rejection for authoritative package metadata.
- Exact-byte package metadata integrity binding and coordinated writer/reader
  version dispatch.
- Fail-closed unsupported-version behavior and explicit legacy/unversioned
  package rejection.
- Compatibility, downgrade, and tamper hardening with deterministic
  current-format writer/reader round trips.
- Package format v2 and bundle schema v2 with `RuntimeDescriptor`, the
  `application_executable` runtime kind, logical entrypoint identity, and host
  target metadata.
- Explicit reader dispatch for supported `(1,1)` and `(2,2)` package/bundle
  version pairs, schema-aware integrity validation, and internal v2 ZIP
  assembly while preserving v1 `forge build` output.
- A separate context-aware Go application executable builder with explicit
  working directory and environment, main-package validation, host GOOS/GOARCH,
  and `go build -trimpath -buildvcs=false -o`.
- `RunnablePackageCompiler` ownership of entrypoint/source resolution, private
  temporary output, exact executable-byte capture, one-artifact package v2
  assembly, cleanup, and optional signing.
- Real host executable integration proof covering exact read-back, independent
  SHA-256 verification, tamper rejection, cleanup, source non-mutation, and no
  application execution.
- R2A-1 bounded versioned reads with detailed validated package/bundle version
  evidence, verified signer evidence, same-handle archive-size checks, bounded
  entry reads, overflow-safe accounting, fixed Alpha limits, and Store-only
  runtime policy.
- R2A-2 strict verified runtime loading with mandatory integrity and trusted
  signatures, v1 non-runnable classification, v2 Alpha one-artifact validation,
  exact host GOOS/GOARCH authorization, detached executable bytes, source
  immutability, and no extraction or execution.
- The intended R2A-3 security-integration scope was already covered directly by
  R2A-2, so no redundant implementation checkpoint was required.
- R2B-1 secure executable materialization accepts only
  `VerifiedRunnablePackage`, rechecks the host target, and owns a private
  `forge-runtime-*` directory with an internally controlled executable name.
  It uses exclusive creation, a `0600` initial mode, complete-write-before-`0700`
  ordering, `Sync`, same-handle SHA-256 verification, and regular-file,
  non-symlink, size, and file/path identity validation.
- R2B-1 also adds lifecycle-managed cleanup through concurrency-safe,
  idempotent, retryable `MaterializedExecutable.Close`, while intentionally
  exposing no public executable path.
- R2B-2 proves the production path from a real Go executable through a trusted
  signed package v2, strict runtime loading, and exact materialization. Coverage
  includes source-ZIP independence, source-fixture immutability, independent
  materializations, cleanup, and no application execution.
- PR-1 adds a package-private atomic single-use execution lease with irreversible
  claim semantics, Close/acquire linearization, pending cleanup coordination,
  retryable cleanup failure, and no public executable path.
- PR-2 adds direct no-shell child start from `MaterializedExecutable`, start-time
  host/file/digest/identity and PE/ELF/Mach-O validation, a controlled working
  directory and reduced environment, zero arguments, null stdin, bounded output,
  with a 1 MiB ceiling per stdout/stderr stream, and one background Wait/reap
  owner with context-cancellation support.
- PR-3 adds concurrency-safe immediate direct-child termination, serialized
  natural/cancellation/manual-termination outcomes, stable result preservation
  across cleanup failures, and full trusted package-to-real-execution proof.
- Manifest Entrypoint Slice 1 adds an optional top-level application entrypoint
  with exact module/version structural validation, library-manifest
  compatibility, no duplicated `ImportPath`, and explicit `BuildPlan`
  non-authority while preserving v1 `forge build` behavior.
- Manifest Entrypoint Slice 2 carries the entrypoint and normalized
  `PackageSource` candidates as immutable admission evidence with copy-isolated
  accessors. Successful preparation is immutable build authority, while
  `AdmitManifest` additionally publishes shared-registry candidates and applies
  existing live-conflict behavior.
- Manifest Entrypoint Slice 3 adds the thin `RunnableManifestCompiler`, which
  mechanically converts admitted identity to `RuntimeEntrypoint` and delegates
  real executable construction and signed package-v2 assembly to the existing
  runnable compiler.
- Admission-bound source hardening makes the normalized admission snapshot the
  manifest-driven build authority. A private one-source resolver prevents
  external resolver injection, rejects missing, duplicate, or noncanonical
  selected-source evidence, and preserves the invariant that admitted,
  builder, and artifact import paths are identical.
- User-Facing Runnable Workflow Architecture freezes an explicit signed
  `forge build-runnable` operation while preserving `forge build` as the v1
  identity/provenance workflow.
- Runnable signing and safe-publication primitives add strict PKCS#8 PEM
  Ed25519 key loading, explicit KeyID validation, private same-filesystem
  staging, bounded package verification, and atomic no-replace publication.
- The explicit signed `build-runnable` CLI uses prepared immutable admission,
  the process working directory, host GOOS/GOARCH, admission-bound source
  authority, and mandatory signing to produce package format v2.
- Runnable Workflow Formal Closure proves real command-to-signed-package
  integration, registry non-mutation, default/custom output behavior, target
  preservation, strict read-back, and no package execution.
- `forge run` Architecture Review freezes explicit local package input, one
  command-local trusted Ed25519 public key and KeyID, strict runtime authority,
  direct-child execution, bounded output, cleanup, and exit semantics.
- Trusted Run Support Primitives add strict local package/public-key input,
  signal-aware command context, and pure child/cancellation exit mapping.
- Explicit Trusted `forge run` composes strict package-v2 loading, exact host
  authorization, private materialization, direct-child Start and Wait/reap,
  bounded output presentation, cleanup, and exact child/cancellation status.
- `forge run` Formal Closure confirms the narrow Pre-Alpha workflow without
  sandbox, process-tree, resource-isolation, remote-acquisition, or generalized
  application-execution claims.
- Package / Materialization TOCTOU Hardening Architecture Review defines the
  package-selection identity boundary and separates it from later
  materialization-to-exec hardening.
- Atomic Package-Open Identity Binding moves symlink/nonregular rejection and
  Lstat/Open/Stat/SameFile authority into `ZIPPackageReader`, which consumes
  archive size and ZIP bytes through the accepted handle without reopening the
  path.
- CLI Package Preflight Simplification leaves `forge run` with lexical/local
  path policy only and delegates filesystem existence and identity to the
  compiler reader.
- TOCTOU Hardening Formal Closure completes the package-selection boundary
  while retaining same-user in-place mutation and materialized
  validation-to-exec races as separate debt.
- Linux runtime reaping tests now use portable `cmd.Wait`/`ProcessState`
  evidence for signal-terminated children; no production ProcessRunner
  lifecycle change was required, and Ubuntu GitHub Actions is green.
- Exact KeyID Alignment establishes one valid-UTF-8, nonempty,
  non-trimming, non-normalizing identifier contract across `build-runnable`,
  the signer, `PackageSignature`, `TrustStore`, verifier, and `forge run`.
  Surrounding Unicode whitespace and ASCII controls U+0000 through U+001F and
  U+007F are rejected; all other Unicode identity remains exact.
- Secondary-Document Strict JSON Alignment applies one shared structural
  contract to `package.json`, bundle schemas v1 and v2, `integrity.json`, and
  `signature.json`: valid raw UTF-8, one object, trailing whitespace only,
  recursive duplicate rejection, unknown-field rejection, and subsequent
  domain/schema validation. This subsumes the signature-specific invalid-UTF-8
  guard by rejecting malformed raw bytes before JSON repair. Canonical writer
  bytes, schema versions, hashes, and signature payloads remain unchanged.
- Linux / Windows Continuous Acceptance adds independent Ubuntu and Windows
  matrix checks with dependency cleanliness, package enumeration, vet, full
  uncached tests, and full builds. A separate focused Ubuntu race check covers
  `pkg/compiler`, `runtime`, and `internal/cli`; the first hosted Windows run
  passed on Windows Server 2025 with Go 1.26.7 on windows/amd64.
- Phase 6 Compiler / Package Pipeline Hardening Formal Closure accepts the
  bounded Pre-Alpha / First Alpha implementation as technically complete with
  no implementation blocker. Residual hardening, testing debt, and future
  capabilities remain explicit and do not reopen Phase 6.
- Strict Manifest Admission Alignment establishes one strict YAML/JSON
  document boundary with UTF-8, single-document/object, exact field-type,
  recursive duplicate-field, unknown-field, and unsafe-YAML rejection.
- `forge validate` Vertical Slice adds non-mutating structural, build, and
  runnable admission profiles with deterministic human-readable results.
- Verified Package Inspection Architecture and implementation establish a
  bounded, version-aware, metadata-only read path with integrity verification,
  self-signature verification, and optional explicit trust verification.
- `forge inspect` Vertical Slice exposes v1/v2 package identity, runtime,
  artifact, integrity, and exact signature-state evidence without execution.
- Bounded Phase 1 / 5 / 7 Closure Review accepts those three First Alpha
  scopes as Alpha-Bounded Closed while preserving their long-term work.
- Single-Prompt Execution integrates the bounded core, OpenAI Responses adapter,
  and CLI into main, with offline contract acceptance and explicit network/
  credential boundaries. Model output remains data, never executed instructions.
- Error-sanitization remediation preserves unknown joined failures as safe
  `ErrProvider` categories, so mixed cancellation remains exit 1 and pure
  cancellation remains exit 130 without exposing raw messages or causes.
- Historical single-prompt checkpoint: main acceptance for the then-unreleased
  slice passed on Ubuntu and Windows;
  the existing Ubuntu race gate also covers the AI core and OpenAI adapter.
  At that historical checkpoint live acceptance was NOT AUTHORIZED / NOT RUN;
  the accepted C8 evidence in the Phase 8 section supersedes that status.
- B1/B1-HYG/B2 and C1–C7/C9/C10 integrate the bounded tool foundation;
  C8 closes the real-provider validation gate. See the Phase 8 stage inventory
  above; these additions are published in v0.4.0-alpha.1.

## Current Implemented Foundation

Forge currently provides:

- Configuration management
- Structured logging
- Framework error handling
- Dependency injection
- Application lifecycle and runtime context
- HTTP and middleware infrastructure
- CLI foundation
- Plugin contract and registry
- Plugin configuration enablement
- Manifest contract
- Strict YAML manifest document loading
- Strict JSON manifest document loading
- Manifest validation with recursive duplicate-field and unknown-field
  rejection
- CLI `forge validate` with non-mutating structural, build, and runnable
  profiles
- Optional exact application-entrypoint declaration and structural validation
- Exact module resolution
- Dependency-aware deterministic build planning
- Concurrency-safe in-memory package identity registry
- Exact package name/version resolution
- Package source registry
- Strict source registration, idempotent source `Ensure`, and explicit source
  conflict behavior
- Failure-atomic package and source batch `EnsureAll` operations
- Deterministic dependency graph/order integration
- Compiler execution abstraction
- Operating-system command runner
- Import-path-aware toolchain executor
- Dependency-first build plan execution
- Artifact generation and artifact bundles
- Deterministic bundle serialization and ZIP packaging
- Explicit package compatibility metadata and package format version dispatch
- Package format v1/bundle schema v1 identity packages and package format
  v2/bundle schema v2 runnable application packages
- Integrity schema v2 with exact package metadata, bundle, and payload binding
- Signature schema v1 with Ed25519 signing and verification
- Trust store and verification policy
- One exact KeyID contract across producers, serialized signatures, trust
  routing, verification, and execution, without trimming, case folding, or
  Unicode normalization
- One strict package-document JSON contract across `package.json`, bundle v1,
  bundle v2, `integrity.json`, and `signature.json`
- Versioned ZIP package read-back with explicit `(1,1)` and `(2,2)` dispatch
- Legacy package rejection and package tamper/downgrade hardening
- Runnable application metadata with `RuntimeDescriptor`, logical entrypoint,
  `application_executable` kind, and host OS/architecture target
- Host Go application executable builder and `RunnablePackageCompiler`
- Exact real executable-byte packaging as a one-artifact package v2 with
  integrity and optional signing
- Detailed validated package-read evidence and bounded Alpha runtime ingestion
  with finite archive, entry, document, artifact, and total-uncompressed limits
- Strict trusted runtime package loading with v1 non-runnable classification,
  v2 one-artifact validation, exact host authorization, verified signer
  evidence, and detached executable bytes without extraction or execution
- Secure executable materialization from verified runnable packages into a
  fresh private directory with controlled filename, exclusive creation,
  complete-write permission transition, `Sync`, exact-byte validation, and
  explicit lifecycle cleanup without public path exposure or execution
- A completed direct-child Process Runner checkpoint with atomic single-use
  acquisition, start-time executable/header revalidation, controlled execution
  inputs, bounded output, background Wait/reap, cancellation and manual
  termination, stable process results, and Close-coordinated cleanup
- Artifact source provenance
- CLI `forge build` for YAML and JSON manifests
- Multi-module dependency-aware builds
- Prebuilt `BuildPlan` compilation and packaging
- Pure manifest-admission preflight and controlled admission
- Snapshot-based dependency/source analysis and commit-time source revalidation
- Immutable admitted application identity and normalized source authority
- Admission-bound runnable manifest composition into signed package v2 without
  entrypoint or source inference
- CLI `forge build-runnable` for mandatory-signed, host-target package-v2
  creation with strict staged verification and no-replace publication
- CLI `forge run` for an existing local signed runnable package v2 with one
  explicit command-local Ed25519 public key and exact KeyID
- CLI `forge inspect` for bounded, non-executing v1/v2 inspection with verified
  integrity, unsigned/signed-unverified/signed-trusted states, and optional
  explicit command-local trust
- Declaration-order admission with dependency-first execution
- Predictable admission failures without persistent candidate mutation
- Accepted admission state retained after downstream failures
- Shared-application repeated deterministic builds
- Continuous Ubuntu and Windows acceptance with dependency-cleanliness, list,
  vet, full-test, and full-build gates, plus a focused Ubuntu race gate for the
  `pkg/compiler`, `runtime`, `internal/cli`, `pkg/ai`,
  `internal/aiprovider/openai`, `pkg/ai/tool`, and `internal/agent` boundaries
- Bounded single-prompt core, OpenAI Responses adapter, and `forge ai prompt`,
  integrated into main, offline-accepted, and published in v0.4.0-alpha.1
- Bounded function-tool admission, explicit immutable execution authority,
  invocation-local replay coordination, stateless continuation, one read-only
  CLI tool with explicit opt-in, terminal reasoning compatibility, and safe
  stage diagnostics; Phase 8 CLOSED / PASS with real-provider acceptance PASS
- Single-use text and authorized-tool Run lifecycles with explicit application-host
  cancellation/drain/restart composition; Phase 9 CLOSED / PASS, internal/pre-stable

## Phase 5 — Registry Alpha-Bounded Scope

Implemented Alpha contract:

- Concurrency-safe in-memory package identity registry.
- Exact package name/version resolution.
- Package source registry.
- Strict source registration.
- Idempotent source `Ensure` with explicit source-conflict behavior.
- Failure-atomic package batch `EnsureAll`.
- Failure-atomic source batch `EnsureAll`.
- Deterministic successful insertion order.
- Deterministic dependency graph/order integration.

Long-term remaining work:

- Remote registry and remote resolution.
- Package acquisition.
- Advanced version constraints.
- Registry persistence.
- Remote and distributed transaction semantics.

Status: **ALPHA-BOUNDED CLOSED.** These deferred distribution and persistence
capabilities are not marked complete.

## Phase 6 — Compiler Foundation

Implemented foundation:

- Compiler execution abstraction and operating-system command runner.
- Import-path-aware toolchain executor.
- Deterministic `BuildPlan` with dependency-first execution.
- Prebuilt `BuildPlan` compilation and packaging pipeline.
- Artifact generation and artifact bundles.
- Artifact provenance.
- Deterministic bundle serialization and ZIP packaging.
- Explicit package compatibility metadata and package format version dispatch.
- Package format v1/bundle schema v1 identity packages and package format
  v2/bundle schema v2 runnable application packages.
- Integrity schema v2 with exact package metadata, bundle, and payload binding.
- Signature schema v1 with Ed25519 signing and verification, trust store, and
  verification policy.
- One exact KeyID validator shared by `build-runnable`, the signer,
  `PackageSignature`, `TrustStore`, verifier, and `forge run`. TrustStore
  routing preserves exact Go-string identity and performs no trimming, case
  folding, or Unicode normalization.
- Shared strict JSON decoding for `package.json`, bundle schema v1, bundle
  schema v2, `integrity.json`, and `signature.json`, including raw UTF-8,
  object-root, single-value, recursive duplicate-key, unknown-field, and
  trailing-whitespace enforcement before domain/schema validation.
- Versioned ZIP read-back with explicit `(1,1)` and `(2,2)` dispatch.
- Legacy/unversioned package rejection and current-format tamper/downgrade
  hardening.
- `RuntimeDescriptor` with the `application_executable` kind, logical
  entrypoint, and host OS/architecture target.
- Context-aware host Go executable building with explicit working directory and
  environment, main-package validation, and controlled `go build -o` output.
- `RunnablePackageCompiler` with private temporary-output ownership, exact
  executable-byte capture, one-artifact package v2 assembly, cleanup, and
  optional signing.
- Real executable package v2 integration with exact read-back, digest, tamper,
  cleanup, and source non-mutation proof.
- Bounded version-aware package reads with validated version and signer evidence,
  same-handle archive-size validation, bounded actual reads, overflow-safe total
  accounting, fixed Alpha limits, and Store-only runtime policy.
- Atomic package acquisition with reader-owned Lstat/Open/Stat/SameFile
  identity binding, symlink/nonregular rejection, same-handle ZIP consumption,
  and unchanged public path-based reader APIs.
- Strict verified runtime package loading with mandatory integrity and trusted
  signatures, v1 non-runnable classification, v2 Alpha one-artifact validation,
  exact host GOOS/GOARCH authorization, detached bytes, source-package
  non-mutation, and no extraction or execution.
- Secure executable materialization with verified-input and host revalidation,
  private directory ownership, controlled filename, exclusive file creation,
  complete-write-before-executable-permission ordering, `Sync`, same-handle
  SHA-256 and file-identity validation, and explicit concurrency-safe cleanup.
- Real executable materialization integration proving source-to-package-to-load-
  to-file byte equality, source independence, isolated materializations,
  cleanup, and no application execution.
- Artifact source provenance.
- CLI build vertical slice for YAML, JSON, and multi-module dependency-aware builds.
- Pure manifest-admission preflight and controlled manifest admission.
- CLI admission and prepared-plan execution.
- Optional exact manifest application-entrypoint contract with `BuildPlan`
  non-authority and library-manifest compatibility.
- Immutable admitted entrypoint and normalized source evidence with
  copy-isolated accessors.
- Admission-bound `RunnableManifestCompiler` composition through the existing
  executable builder and runnable package compiler.
- Real signed package-v2 integration with exact runtime/artifact identity and
  the invariant that admitted, builder, and artifact import paths match.
- An explicit `forge build-runnable` CLI that requires an admitted entrypoint,
  an unencrypted PKCS#8 PEM Ed25519 signing key, and an explicit KeyID.
- Immutable prepared-admission authority, process-cwd build policy, host-only
  targeting, strict bounded staged read-back, and atomic no-replace output
  publication without shared-registry mutation or package execution.
- Default runnable output uses
  `build/<name>-<version>-runnable-<goos>-<goarch>.zip`; custom output requires
  an exact `.zip` path, and no force/overwrite mode exists.
- No persistent candidate mutation for predictable admission failures.
- Accepted registration remains committed after downstream execution or package
  failure.
- Continuous `acceptance (ubuntu-latest)` and `acceptance (windows-latest)`
  gates with `fail-fast: false`, dependency-file cleanliness, `go list`, vet,
  full uncached tests, and full builds.
- Focused `race (ubuntu-latest)` coverage for `pkg/compiler`, `runtime`, and
  `internal/cli`.
- Formal technical closure of Phase 6 for the bounded Pre-Alpha / First Alpha
  compiler-package-runnable pipeline.

Status: **CLOSED / PASS for the bounded Pre-Alpha / First Alpha scope.** This
does not mean Beta readiness, production readiness, completion of all security
hardening, or completion of future compiler and runtime capabilities.

Deferred hardening and test debt (does not reopen Phase 6):

- Same-user coherent in-place mutation of an already-open package remains
  outside immutable-snapshot guarantees; package parsing, integrity, and
  signatures remain authoritative.
- The materialized executable validation-to-exec architecture review is
  complete; stronger object-bound execution implementation remains deferred.
- Windows ACL, reparse-point, and share-mode hardening, plus
  capability-dependent Windows package-symlink coverage.
- Direct package-handle Close failure injection and command-level
  Start/Wait/Close/output-write failure-injection seams.
- Cross-toolchain golden validation, reproducibility hardening, and build
  environment isolation.
- Strict cross-registry transaction visibility and process-crash recovery
  between registry commits.
- Concurrent full-build isolation and broader generic/full-build output-path
  coordination. `build-runnable` already uses same-filesystem staging and
  atomic hard-link no-replace publication: one concurrent publisher may win,
  while existing targets are not overwritten.
- Manifest decoder strictness remains separate from the completed package-
  document strictness contract.

Future capabilities (do not keep Phase 6 open):

- Legacy inspection and migration tooling.
- Package-format compatibility or future schemas beyond the explicitly
  supported `(1,1)` and `(2,2)` pairs.
- Persistent trust, multiple configured trusted keys, rotation, revocation,
  trust snapshots, and start-time reauthorization.
- Source-content and repository-commit provenance, dependency provenance, and
  SBOM support.
- Runtime arguments, caller environment, stdin, caller working directory, and
  live streaming.
- Process-tree/descendant lifecycle, graceful shutdown protocols, and Windows
  Job Object or Unix process-group policy.
- Sandboxing and CPU, memory, process-count, filesystem, network, syscall, and
  privilege controls.
- Compiler optimization, remote registry negotiation and package acquisition,
  scheduler work, and AI runtime capabilities beyond the closed bounded
  Phase 8/9 foundations (autonomous agents, memory, and workflow engine).

## Evidence-Based Current Roadmap Position

```text
Published release: v0.4.0-alpha.1 — source-only, zero uploaded assets
P10-C0 audit base: main 92ea08c5af041e6a204891059724811ab582111a, integrated P10-A1/B1/B2/B3
P10-C0: final documentation/audit package; integration + strict exact push-main CI establishes bounded Phase 10 closure
→ Phase 1 — Core Foundation: Alpha-Bounded Closed
→ Phase 2 — Alpha workflow implemented; long-term expansion planned
→ Phase 3 — Manifest Engine: Complete for current contract
→ Phase 4 — Alpha validation workflow implemented; long-term expansion planned
→ Phase 5 — Registry: Alpha-Bounded Closed
→ Phase 6 — Compiler / Package Pipeline: CLOSED / PASS
→ Phase 7 — Runtime: Alpha-Bounded Closed
→ Phase 8 — AI Runtime: CLOSED / PASS; published in 0.4.0-alpha.1
→ Single-Prompt Core / OpenAI Adapter / CLI: Integrated / offline contract PASS
→ Main Ubuntu / Windows Acceptance and Focused AI Race: PASS
→ Bounded Tool Calling / Real Provider Acceptance: PASS
→ Phase 9 — Bounded Agent Execution Lifecycle: CLOSED / PASS; included, internal/pre-stable
→ Phase 9 Packages: A1/B1/B2/B3/C0 CLOSED / PASS — INTEGRATED; no further runtime package required
→ Phase 10 — Bounded Workflow Composition (Synchronous Sequences): A1/B1/B2/B3 integrated; functionally complete internal scope; C0 integration + strict exact push-main CI establishes closure
→ Autonomous Agents / Memory / Workflow Engine / Scheduler: Future
→ Release: 0.4.0-alpha.1 PUBLISHED; RR-004-PUB CLOSED / PASS
→ RR-005 — post-publication documentation reconciliation; documentation-only, no runtime changes; PR #24 merge + strict push-main CI are the closure gate
→ Package Pipeline Hardening checkpoints
→ Package Format Stabilization: Completed
→ Runnable Package Contract R1A: Completed
→ Real Executable Output R1B: Completed
→ Verified Runtime Package Loader R2A: Completed
→ Secure Executable Materialization R2B: Completed
→ Process Runner: Completed
→ Manifest Application Entrypoint: Completed
→ User-Facing Runnable Workflow: Completed
→ forge run Architecture Review: Completed
→ Trusted Run Support Primitives: Completed
→ Explicit Trusted forge run Command: Completed
→ forge run Formal Closure: Completed
→ Package / Materialization TOCTOU Hardening Architecture Review: Completed
→ Atomic Package-Open Identity Binding: Completed
→ CLI Package Preflight Simplification: Completed
→ TOCTOU Hardening Formal Closure: Completed
→ Runtime Reaping CI Portability Defect: Resolved
→ Exact KeyID Alignment: Completed
→ Secondary-Document Strict JSON Alignment: Completed
→ Linux / Windows Acceptance Matrix: Completed
→ Focused Ubuntu Race Gate: Completed
→ Strict Manifest Admission Alignment: Completed
→ forge validate Vertical Slice: Completed
→ Verified Package Inspection Architecture / Implementation: Completed
→ forge inspect Vertical Slice: Completed
→ Bounded Phase 1 / 5 / 7 Closure Review: Completed
→ Materialized Executable Validation-to-Exec Architecture Review: Completed
→ Materialized Executable Validation-to-Exec Implementation: Deferred
→ Phase 6 Compiler / Package Pipeline Hardening: CLOSED / PASS
```

Completed checkpoints: **Manifest Admission Hardening**, **Package Format
Stabilization**, **Runnable Package Contract R1A**, **Real Executable Output
R1B**, **Verified Runtime Package Loader R2A**, **Secure Executable
Materialization R2B**, **Process Runner**, and **Manifest Application
Entrypoint**, **User-Facing Runnable Workflow**, **forge run Architecture
Review**, **Trusted Run Support Primitives**, **Explicit Trusted forge run
Command**, **forge run Formal Closure**, **Package / Materialization TOCTOU
Hardening Architecture Review**, **Atomic Package-Open Identity Binding**,
**CLI Package Preflight Simplification**, and **TOCTOU Hardening Formal
Closure**, **Exact KeyID Alignment**, **Secondary-Document Strict JSON
Alignment**, **Linux / Windows Continuous Acceptance**, and the **Phase 6
Compiler / Package Pipeline Hardening Formal Closure Review**.

`forge run` is completed for its narrow Pre-Alpha boundary: existing local
signed runnable package v2, one explicit command-local trusted Ed25519 public
key and KeyID, strict runtime verification, host-only authorization, secure
materialization, one direct child, Wait/reap, bounded output, cleanup, and exact
child/cancellation exit mapping. Phase 6 is **CLOSED / PASS** for this bounded
Pre-Alpha / First Alpha compiler-package-runnable pipeline. This closure does
not mean Beta or production readiness and does not erase the deferred
hardening and future-capability work listed above.

The frozen grammar is `forge run <package.zip> --trusted-key
<public-key.pem> --key-id <key-id>`. It accepts one local regular, non-symlink
lowercase `.zip` package, performs no manifest/source/build or remote acquisition,
and creates one invocation-local TrustStore. CLI input handling is lexical;
the compiler reader owns filesystem existence, symlink/nonregular rejection,
atomic open identity binding, same-handle ZIP reads, and Close. The key is an
explicit X.509 PKIX
`PUBLIC KEY` PEM Ed25519 key and KeyID. The strict runtime loader remains the
signature, integrity, bounded-read, runnable-v2, and host authorization
authority; verified bytes alone reach private materialization and the direct
no-shell child.

The First Alpha command accepts no user args, environment injection, or stdin.
It buffers at most 1 MiB per stream until completion, preserves natural child
exit codes, maps clean cancellation to 130, and maps infrastructure failures to
1. Trusted native code can execute arbitrary code: Forge provides no sandbox,
filesystem/network or resource isolation, privilege drop, process-tree
containment, graceful shutdown guarantee, or production-safety claim.

Package-selection TOCTOU hardening is complete for the current Alpha boundary.
It does not claim an immutable package snapshot: same-user in-place mutation of
the accepted file remains bounded by ZIP, integrity, and signature validation.
The **Materialized Executable Validation-to-Exec Race Hardening Architecture
Review** is complete, and implementation remains deferred technical debt. It
is not an active Phase-6 implementation and does not authorize expansion to
arguments, environment, stdin, sandboxing, process-tree containment, or
resource controls.

## Remaining Platform Work

The following capabilities remain future work:

- Long-term validation expansion and an explicit manifest schema-version policy
- `forge init`, `forge fmt`, and machine-readable validation/inspection output
- Remote package registry and package acquisition
- Persistent local registry, package indexing, advanced version constraints,
  and complete provenance/SBOM
- Package-format production hardening and legacy tooling
- Compiler optimization, build isolation, process resource controls, dependency
  provenance, and reproducibility hardening
- Materialized executable validation-to-exec implementation (architecture
  review complete; implementation deferred)
- Process-tree/descendant lifecycle, graceful shutdown, and optional Windows Job
  Object or Unix process-group policy
- Richer arguments/environment/working-directory contracts, process resource
  controls, and sandboxing
- Trust snapshot/revocation and start-time authorization policy
- Persistent trust, multiple configured keys, rotation, and revocation
- Scheduler and multi-application orchestration
- Dynamic plugin discovery/loading
- Remote package resolution
- Advanced dependency and version resolution
- AI runtime expansion beyond the accepted bounded foundation: autonomous
  agents, memory, and workflow engine, each requiring separate scope decisions
- Further publication decisions require separate Owner authorization; RR-005
  reconciles the completed 0.4.0-alpha.1 publication. Phase 8 real-provider
  acceptance is already PASS and is not an outstanding implementation task

# Long-Term Goal

Forge 1.0 will provide a complete AI-native development platform that enables developers to build modern applications through declarative manifests, modular runtimes, and intelligent automation.
