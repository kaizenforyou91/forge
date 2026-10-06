# P13-A0: Next-Wave Architecture Selection

## Status

**P13-A0 — IMPLEMENTED / DOCUMENTED / UNDER REVIEW.**

**Phase 13 is NOT YET DEFINED. No P13 implementation package is authorized.**

Recommended direction:

> **Phase 13 — Bounded Execution Evidence and Trace Model**

This document is a selection proposal, not an architecture freeze. P13-A1 would
define the detailed contract in a new ADR only after explicit Owner approval.

Phase 12 remains **CLOSED / BLOCKED-DEFERRED — FROZEN INVARIANT RETAINED; NO
VERIFIED-LAUNCH CLAIM**. Nothing in this proposal reopens ADR-005, weakens its
exact-object/native-admission invariant, or claims a verified-launch mechanism.

## Canonical baseline

The selection audit starts from main
`659d2563c964f47b886a5192a88feb5ee622ea78`, tree
`2e47d1a00d84e081abd3b7bd3bf96d273d4433a2`. Exact-main CI
`37425673265` passed attempt 1 on that SHA: Ubuntu acceptance
`112144634094`, Windows acceptance `112144634148`, and Ubuntu race
`112144634065` all passed.

## Current-state audit

### Strongest existing capabilities

- Strict YAML/JSON manifest parsing, validation, dependency ordering, and
  deterministic compiler/package pipelines.
- Package formats v1 and v2, bounded ZIP ingestion, integrity metadata,
  Ed25519 signatures, explicit trust policy, and metadata-only inspection.
- Trusted local runnable-package loading, controlled executable
  materialization, bounded output, cancellation, and platform-native process
  scope cleanup.
- Explicit, bounded AI requests with fixed timeouts, validated results, safe
  error mapping, and opt-in network authority.
- Immutable tool definitions, caller-created tool authority, strict JSON,
  bounded provider/tool round trips, and no model-selected authority expansion.
- Internal single-use `Run` ownership with Ready/Running/terminal states,
  cancellation, stable completion, and no background worker.
- Internal synchronous `Sequence` composition of 1..8 fixed steps with one
  overall deadline, explicit data transfer, aggregate usage, and no scheduler,
  queue, persistence, or autonomous planning.
- `RunHost` composition with application lifetime and deterministic shutdown.
- Structured logger fields, HTTP request IDs, and localized fixed diagnostic
  enums, without a shared execution-evidence contract.

### Most important remaining gaps

1. **Execution evidence is fragmented and transient.** `Run`, `Sequence`, tool
   round trips, CLI commands, and native processes expose different terminal
   facts. There is no common bounded record of admission, step ordering,
   authority use, cancellation, usage, or terminal classification.
2. **Debugging and evaluation cannot rely on a stable internal schema.** Logs
   are text-oriented and optional structured fields accept arbitrary values.
   Provider diagnostic stages are adapter-local. Reconstructing a multi-step
   failure requires test-specific knowledge.
3. **Durable recovery has no safe foundation.** There is no stable execution
   identity, versioned event vocabulary, replay boundary, or persistence-safe
   terminal evidence. Adding persistence now would mix state-machine design,
   sensitive-data policy, storage atomicity, and recovery authority.
4. **Tool/plugin expansion lacks a unified capability descriptor.** AI tool
   authority is strong but internal and invocation-local; the older plugin
   registry is lifecycle-oriented and does not describe filesystem, network,
   subprocess, or secret authority.
5. **Artifact provenance remains bounded.** Package evidence records source
   identity and signatures but not repository commits, reproducible toolchain
   evidence, dependency provenance, or an SBOM.
6. **Distribution is local.** Package registries are in-memory and exact-name /
   version only; remote discovery, acquisition, caching, and trust-domain
   policy do not exist.
7. **Isolation remains a research problem.** Forge has process ownership and
   cleanup, not sandboxing, quotas, syscall isolation, or hostile-code
   containment.

## Evaluation method

Scores are 1..5, and **higher is always more favorable**. For Security
tractability, Dependency risk, and Scope controllability, 5 means easier to
secure, lower dependency risk, and easier to bound. The weighted result is out
of 100:

| Criterion | Weight |
|---|---:|
| Strategic value | 15 |
| Developer usefulness | 12 |
| AI-native leverage | 12 |
| Architecture fit | 12 |
| Security tractability | 10 |
| Cross-platform feasibility | 8 |
| Implementation feasibility | 10 |
| Testability | 8 |
| Dependency risk | 6 |
| Scope controllability | 7 |

The numerical result guides comparison; it does not replace architectural
judgment.

## Ranked candidates

| Rank | Candidate | Strategic | Developer | AI-native | Fit | Security | Cross-platform | Implementation | Testability | Dependency | Scope | Weighted |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | Bounded Execution Evidence and Trace Model | 5 | 4 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 97.6 |
| 2 | Crash-Safe Run and Workflow State Ledger | 5 | 5 | 5 | 4 | 3 | 5 | 3 | 4 | 5 | 3 | 85.2 |
| 3 | Artifact Provenance and SBOM Evidence | 4 | 4 | 3 | 5 | 4 | 5 | 4 | 5 | 5 | 4 | 84.4 |
| 4 | Capability-Scoped Tool and Plugin Admission | 4 | 4 | 5 | 4 | 3 | 5 | 4 | 4 | 5 | 3 | 81.8 |
| 5 | Remote Package Acquisition Foundation | 5 | 5 | 3 | 4 | 2 | 5 | 3 | 4 | 3 | 2 | 74.6 |
| 6 | Cross-Platform Isolation Profile Research | 4 | 3 | 3 | 3 | 2 | 1 | 2 | 3 | 3 | 2 | 54.4 |

## Candidate 1 — Bounded Execution Evidence and Trace Model

### Problem and value now

Forge can own executions but cannot describe them through one stable, safe,
bounded evidence vocabulary. Developers need to answer which step was admitted,
which fixed authority class was used, where cancellation won, what terminal
state was published, and whether usage/output bounds were reached without
inspecting raw provider payloads or component-specific logs.

This wave would enable deterministic debugging, test assertions, evaluation,
and future persistence design while adding no execution authority.

### Existing foundation

`agent.Run`, `agent.Sequence`, `RunHost`, AI/tool diagnostic stages, usage
accounting, `ProcessResult`, and structured logger fields already expose the
facts from which bounded evidence can be derived.

### Proposed boundary and invariants

- One private, versioned execution-evidence vocabulary.
- Fixed event kinds and bounded field types; no arbitrary maps.
- Redacted by construction: no prompts, model output, tool arguments/results,
  API keys, environment values, executable paths, native handles, or raw errors.
- Observation only: evidence cannot select providers, tools, workflow steps,
  processes, or cancellation outcomes.
- Bounded event count and bounded string/enumeration sizes.
- Monotonic sequence numbers, exactly one terminal classification, and an
  immutable snapshot after completion.
- Evidence collection failure cannot silently change execution authority or
  fabricate success. A1 must choose explicit fail-open/fail-closed behavior for
  optional diagnostics versus required audit evidence.
- No persistence, network export, telemetry upload, global registry, or
  background worker in the initial phase.

### Likely package sequence

P13-A1 freezes the schema, redaction policy, ownership, and failure semantics.
P13-B1 implements a dormant private bounded collector. P13-B2 integrates
`Run`/`Sequence` lifecycle evidence. P13-B3 integrates authorized tool/provider
metadata and safe error categories. P13-B4 hardens host composition and optional
diagnostic projection without adding a public CLI by default. P13-C0 audits
end-to-end evidence, platform truth, and documentation.

### Security, platform, dependencies, and tests

The main risk is accidental sensitive-data capture. The design must use
allowlisted fields and fixed enums rather than sanitizing arbitrary data after
collection. A pure-Go private model is portable across Linux, Windows, macOS,
and other supported Go platforms. No new dependency is expected.

Tests should prove exact event order, terminal uniqueness, cancellation races,
step bounds, redaction, snapshot immutability, overflow behavior, panic/failure
finalization, concurrent inspection, and zero effect on provider/tool/process
authority.

### Blockers, non-goals, and Phase 12 interaction

No known platform blocker exists. The main architecture work is defining a
minimal schema that does not become a generic tracing framework.

Non-goals: durable history, distributed tracing, OpenTelemetry export,
analytics, model-quality scoring, prompt capture, scheduler, retries, replay,
public workflow API, or production monitoring service.

This candidate is independent of Phase 12, does not weaken ADR-005, and does
not require reopening verified-launch work.

**Decision: select.** It provides high leverage with the smallest new authority
surface and creates the evidence foundation needed by several later waves.

## Candidate 2 — Crash-Safe Run and Workflow State Ledger

### Problem and value now

All agent/workflow state is in memory. A process crash loses admission,
progress, terminal classification, and usage evidence. Developers cannot inspect
or reconcile interrupted work.

A bounded ledger could provide durable terminal history and explicit recovery
classification for a fixed execution model.

### Foundation and boundary

The single-use `Run`/`Sequence` state machines, configuration transaction code,
atomic file replacement patterns, and application lifecycle are useful
foundations. The boundary should be an append-only or transactional local ledger
of metadata-only execution facts; it must never persist prompts, secrets, tool
payloads, or native authority.

Major invariants include stable execution IDs, schema versioning, atomic
terminal publication, crash-prefix validity, idempotent reconciliation, and no
automatic re-execution after restart.

### Sequence, security, platform, and testing

A plausible wave would require architecture freeze, a private record model,
an atomic local store, crash recovery classification, integration, and closure.
It is pure-Go and cross-platform in principle, but Windows file-sharing and
rename semantics require native proof.

Testing must use fault injection around create/write/sync/rename, truncated
records, concurrent readers, stale locks, cancellation, disk-full errors, and
restart classification.

### Blockers and non-goals

The blocker is not API availability; it is defining durable semantics before a
stable evidence schema exists. Persistence also expands sensitive-data,
retention, permissions, migration, and cleanup obligations.

Non-goals would include automatic resume, retries, queues, distributed workers,
remote coordination, and exactly-once external effects.

This candidate is independent of Phase 12 but should consume a stable Phase 13
evidence model rather than invent a second one.

**Decision: runner-up.** Reconsider immediately after bounded execution
evidence is integrated and proven useful.

## Candidate 3 — Artifact Provenance and SBOM Evidence

### Problem and value now

Forge signs and inspects bounded packages but does not bind source repository
commit/digest, reproducible toolchain facts, dependency provenance, or an SBOM.
That limits supply-chain transparency and independent build evidence.

### Foundation and boundary

Package metadata versioning, integrity/signature schemas, deterministic ZIP
writing, inspection results, and strict readers provide a strong foundation.
The boundary should be a versioned, deterministic evidence document whose bytes
are included in integrity/signature coverage.

Invariants: deterministic canonical encoding; explicit absent/unknown values;
no network lookup during verification; bounded entry/count sizes; package trust
remains distinct from provenance completeness.

### Packages, risk, and tests

Likely packages: evidence schema, compiler emission, reader/inspection support,
CLI presentation, compatibility/downgrade hardening, closure. Standard library
support may suffice for a Forge-specific evidence envelope; adopting SPDX or
CycloneDX could add dependencies and substantial schema scope.

Tests should cover determinism, tampering, downgrade, unknown fields, size
limits, unsupported versions, and signed round trips across platforms.

Non-goals: remote attestation, transparency logs, reproducible-build guarantee,
publisher identity beyond existing trust, or automatic dependency discovery.

This candidate is independent of Phase 12 and does not weaken ADR-005.

**Decision: defer.** Valuable and feasible, but it advances package evidence
more than AI execution capability and should follow the execution-evidence wave.

## Candidate 4 — Capability-Scoped Tool and Plugin Admission

### Problem and value now

AI tool authority is explicit and safe but limited to caller-built in-process
bindings. The older plugin registry describes lifecycle identity, not authority.
Forge lacks one declarative capability model for future tool/plugin expansion.

### Foundation and boundary

`tool.Authority`, immutable definitions, strict argument validation, app module
lifecycle, and deterministic plugin registration are the foundations. A bounded
wave could define private capability descriptors and admission policy without
dynamic loading.

Invariants: provider output remains data; only host code constructs authority;
capabilities are allowlisted and immutable; no ambient filesystem/network/
subprocess authority; admission never loads code or discovers plugins.

### Packages, risk, and tests

Likely packages: capability vocabulary, policy/admission core, tool binding,
plugin lifecycle reconciliation, integration hardening, closure. No dependency
is expected. Tests must prove confused-deputy resistance, duplicate/conflicting
capabilities, immutable snapshots, typed-nil handling, cancellation, and absence
of undeclared authority.

Non-goals: dynamic plugin loader, marketplace, remote tools, arbitrary shell,
MCP client, secret broker, or public manifest syntax.

This candidate is independent of Phase 12. Its main risk is creating a broad
abstraction before Forge has evidence showing which capability facts developers
need.

**Decision: defer.** Strong AI-native value, but execution evidence should first
make authority use observable and testable.

## Candidate 5 — Remote Package Acquisition Foundation

### Problem and value now

Developers must supply local package files and keys manually. There is no remote
registry, resolution, acquisition, cache, or trust-domain configuration.

### Foundation and boundary

Exact package identity, strict readers, signatures, local registries, package
sources, and verified inspection provide foundations. A bounded slice could
define metadata discovery and download-to-quarantine with explicit caller
network authority and existing verification before admission.

Invariants: network opt-in; HTTPS policy; byte/count/time bounds; no execution
before local verification; immutable cache keys; no implicit trust from origin;
atomic cache publication; offline repeatability after verified acquisition.

### Packages, risk, and tests

Likely packages span protocol architecture, client bounds, cache, trust policy,
CLI integration, and closure. HTTP can use the standard library, but registry
protocol and credential handling create durable compatibility and security
obligations. Tests need hostile servers, truncation, redirects, cache poisoning,
concurrent acquisition, cancellation, and offline verification.

Non-goals: package publishing, federation, dependency solver, account system,
key transparency, or automatic updates.

This candidate is independent of Phase 12, but it adds network and filesystem
authority before Forge has a shared execution evidence model.

**Decision: defer.** High product value, but its security and scope surface is
too large for the immediate next wave.

## Candidate 6 — Cross-Platform Isolation Profile Research

### Problem and value now

Trusted runnable packages execute with caller privileges. Process scope controls
cleanup, not malicious-code containment, resource quotas, network isolation, or
filesystem isolation.

### Boundary and feasibility

A truthful wave would begin with profile selection and feasibility gates for
Linux namespaces/seccomp/cgroups, Windows AppContainer/Job/resource policy, and
macOS sandbox facilities. It must not claim parity where native proof is absent.

Invariants: fail closed when a requested profile is unsupported; no elevation;
no host-policy mutation; explicit resource ownership; Phase 11 lifecycle remains
intact.

Testing requires native hostile fixtures for filesystem, network, process,
syscall, and quota boundaries. Significant OS-specific code and CI capacity are
unavoidable.

Non-goals: root/admin resistance, kernel-compromise defense, perfect hostile-code
containment, or one universal cross-platform profile.

This candidate is independent of the Phase 12 invariant but shares its native
feasibility risk. It does not satisfy ADR-005 reassessment triggers.

**Decision: reject for Phase 13.** Security value is high, but cross-platform
truth and implementation feasibility are currently too weak for the immediate
wave.

## Recommended Phase 13 direction

### Proposed title

**Phase 13 — Bounded Execution Evidence and Trace Model**

### Objective

Define and integrate a private, bounded, redacted, immutable-by-snapshot record
of Forge-owned AI Run, Sequence, and authorized-tool lifecycle facts without
adding execution, persistence, network, or public caller authority.

### Primary problem

Forge has strong component-level ownership but no shared evidence contract for
explaining multi-step execution and terminal outcomes safely and
deterministically.

### Architecture thesis

Observation should be a separately owned, bounded data product of execution—not
an effectful callback surface. Execution owners publish fixed, allowlisted facts
to a private collector; callers receive an immutable terminal snapshot. The
collector never chooses work, blocks on external I/O, or retains sensitive
payloads.

### Value proposition

- Makes agent/workflow behavior diagnosable without exposing prompts or secrets.
- Creates deterministic evidence for tests and future evaluation tooling.
- Unifies currently fragmented Run, Sequence, tool, usage, and cancellation
  facts.
- Provides the schema prerequisite for a later crash-safe ledger without
  prematurely authorizing persistence or resume.
- Gives future capability admission and remote execution work a stable audit
  vocabulary.

### Security boundary and initial invariants

- Private runtime/internal API only unless a later package explicitly approves a
  public projection.
- Fixed enums and bounded scalar fields only; arbitrary metadata is rejected.
- Raw input/output/tool arguments/results/errors/credentials/paths/handles are
  forbidden.
- No global collector, goroutine, network exporter, file writer, or callback to
  untrusted code.
- Exactly one ordered terminal classification; no event after terminal.
- Snapshot data cannot recover provider, authority, handler, context, process,
  or cancellation capabilities.
- Evidence cannot change execution outcome by default. A1 must explicitly
  classify construction/invariant failures rather than silently dropping them.
- Copying/snapshotting is defensive and bounded.

### Platform boundary

The core model is pure Go and platform-neutral. Initial integration concerns
AI/agent lifecycle facts that already have common semantics. Native-process
evidence, wall-clock export, and OS telemetry are excluded unless separately
proven. No Linux, Windows, or macOS privilege is added.

### Success criteria

- One frozen, versioned private schema and redaction contract.
- Deterministic lifecycle evidence for successful, failed, canceled, and panic
  paths.
- Exact step and authorized-tool boundary evidence for bounded Sequences.
- Strict event/count/size limits and immutable terminal snapshots.
- Race-safe concurrent inspection and cancellation tests.
- Existing execution results, authority, timings, and errors remain compatible.
- Zero persistence, network export, scheduler, retry, or new public behavior.

### Explicit exclusions

Durable run history, resume/replay, queues, workers, scheduling, autonomous
planning, multi-agent orchestration, prompt logging, arbitrary attributes,
distributed tracing, OpenTelemetry, metrics backend, remote export, model
evaluation service, public CLI trace command, native sandboxing, package
provenance, and Phase 12 verified-launch work.

## Proposed package map

### P13-A0 — Selection

- Objective: select the next bounded wave.
- In scope: repository audit, candidate comparison, recommendation.
- Out of scope: architecture freeze or implementation.
- Dependency: Phase 12 closure.
- Gate: Owner accepts or rejects this proposal.

### P13-A1 — Evidence Architecture Freeze

- Objective: define ADR-006 with schema, ownership, redaction, bounds, failure
  semantics, and compatibility rules.
- In scope: private evidence model and deterministic lifecycle vocabulary.
- Out of scope: source implementation, persistence, exporters, public API.
- Dependency: explicit approval of P13-A0.
- Gate: architecture/security review and frozen acceptance tests.

### P13-B1 — Private Bounded Evidence Core

- Objective: implement a dormant single-owner collector and immutable snapshot.
- In scope: event validation, bounds, ordering, terminal state, concurrency.
- Out of scope: production Run/Sequence wiring.
- Dependency: P13-A1.
- Gate: deterministic unit/race proof; no production behavior change.

### P13-B2 — Run and Sequence Lifecycle Evidence

- Objective: integrate lifecycle and step evidence into internal owners.
- In scope: admission, start, step, cancellation, usage-known/unknown, terminal
  categories.
- Out of scope: tool payloads, persistence, CLI, process/runtime tracing.
- Dependency: P13-B1.
- Gate: exact evidence order across success/failure/cancel/panic and unchanged
  Run/Sequence semantics.

### P13-B3 — Authorized Tool and Provider Boundary Evidence

- Objective: add fixed metadata-only facts for provider requests and authorized
  tool execution.
- In scope: stage enums, admitted tool identity digest or fixed safe identity,
  call counts, safe error categories, usage evidence.
- Out of scope: arguments, results, prompts, provider bodies, keys, new tools.
- Dependency: P13-B2 and existing authority invariants.
- Gate: redaction/adversarial tests, exact bounds, no authority expansion.

### P13-B4 — Host Integration and Diagnostic Projection

- Objective: compose evidence with `RunHost` and define an optional private
  diagnostic projection.
- In scope: application shutdown ordering, immutable retrieval, compatibility
  hardening.
- Out of scope: public CLI/API by default, storage, telemetry export.
- Dependency: P13-B2/B3.
- Gate: lifecycle/race/native-regression CI and explicit confirmation that
  observation cannot control execution.

### P13-C0 — Final Evidence and Documentation Closure

- Objective: audit the integrated wave and publish the truthful capability
  matrix.
- In scope: exact-main evidence, limitations, deferred ledger/export work.
- Out of scope: new runtime capability.
- Dependency: all authorized B packages.
- Gate: strict exact-main attempt-1 CI on supported platforms.

## Runner-up

**Crash-Safe Run and Workflow State Ledger** is the strongest follow-on wave.
It would make interrupted work inspectable and establish a basis for carefully
bounded recovery. It should not be Phase 13 because Forge lacks the stable,
redacted execution-evidence schema needed to decide what is safe and meaningful
to persist. Starting persistence first would couple state-machine semantics,
storage atomicity, retention, permissions, migrations, and recovery authority in
one wave.

Reconsider it after Phase 13 proves a versioned evidence vocabulary, terminal
classification, redaction contract, and immutable snapshot behavior. A later
selection must still exclude automatic resume and exactly-once external effects
unless separately proven.

## Controlled conclusion

P13-A0 proposes **Phase 13 — Bounded Execution Evidence and Trace Model**.
This is the highest-value bounded next step because it strengthens debugging,
evaluation readiness, and future reliability work while adding no new execution
authority and requiring no unsupported native mechanism.

No P13-A1 or implementation package is authorized by this document. Phase 13
remains **NOT YET DEFINED / IMPLEMENTATION NOT AUTHORIZED** until the Owner
accepts the selection and separately authorizes an architecture-freeze package.
