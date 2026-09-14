# ADR-003: Bounded Workflow Composition — Synchronous Sequences

## Status and authorization

**ACCEPTED BY CONTROL ROOM — ARCHITECTURE ONLY.**

P10-A0-R1 is CLOSED / PASS — OUTCOME A ACCEPTED BY CONTROL ROOM.
Canonical effectiveness requires integration of P10-A1 into main with strict
push-main CI. Implementation packages require separate authorization. These
are effectiveness criteria, not a claim that integration is pending or complete.
P10-A1 defines architecture only; it starts no P10-B1/B2/B3/C0 implementation.

Phase 10 — Bounded Workflow Composition (Synchronous Sequences) has the objective:
compose a finite, explicit sequence of already-authorized AI operations
synchronously while preserving caller authority, lifecycle ownership, fail-fast
semantics, cancellation, and bounded resource behavior.

## Evidence and immutable publication baseline

Preparation base: `217be79f938b6f913c08ae86e1c918896e66fd68`, tree
`4a1393af3eceae4dd8a2db6b87a2f5968ebe4680`; parents
`4d9c79b0ce7a3fc151bc3e4e7b8131754528de6c` and
`eb51041491412e982205c2e63e4cee3998779b71`.
[Strict main CI 34799363981](https://github.com/kaizenforyou91/forge/actions/runs/34799363981)
is push/main on that exact SHA, attempt 1, completed/success: Ubuntu acceptance,
Windows acceptance, and Ubuntu race PASS.

Published [v0.4.0-alpha.1](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1)
remains source-only, non-production, and pre-stable. Annotated tag object
`c02b81a94c3c7bead6a7fcf4030ce03cc98af52a` targets
`85d78143db1b8bcf2f96b79d681895b1a0492642`; Release ID `388084778` remains
draft=false, prerelease=true, assets=0. Phase 10 is not included in that release.
The release/publication chain is CLOSED / PASS; P10-A1 does not mutate it.

The existing [Run](../../../internal/agent/run.go) snapshots its request,
owns a single execution claim, and releases terminal references. The
[authorized-tool constructor](../../../internal/agent/tool_run.go) delegates
directly without applying text Executor limits to aggregate usage.
[RunHost](../../../internal/agent/host.go) admits explicit synchronous calls,
links application cancellation, and drains active work. These accepted Phase 9
primitives supply operation ownership, not sequence ownership or input handoff.

## Placement and Architecture Freeze

The Sequence is INTERNAL and pre-stable in `internal/agent`; likely future files
are `internal/agent/sequence*.go` and `sequence*_test.go`, with narrow changes
to `internal/agent/host.go` and `host_test.go` for host composition.
P10-A1 creates none of those files.

Composition builds above `pkg/app`, `pkg/ai`, `pkg/ai/tool`, and existing
`internal/agent` ownership. Lower layers must not import higher agent composition.
`pkg/app` remains application lifecycle owner, with no production API change.
The [Architecture Freeze v1.0](../ARCHITECTURE_FREEZE_V1.md) and
[ADR-001](ADR-001-architecture-freeze.md) remain unchanged; no Freeze exception
is required. [ADR-002](ADR-002-agent-run-ownership.md) retains its Phase 9 decision
and historical separate implementation authorizations.

## Sequence shape and immutable specifications

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

## Future package plan — no implementation authorization

| Package | Purpose / future family | Proof obligation | Authorization / dependency |
|---|---|---|---|
| P10-A1 | Architecture / ADR / roadmap only | Accurate bounded decision, historical truth, links, offline checks | ARCHITECTURE ONLY; integration plus strict push-main CI establishes canonical effectiveness |
| P10-B1 | Internal sequence lifecycle and static literal steps; sequence*.go and tests | Single claim, order, bounds, fail-fast, cancellation, redaction, reference release | NOT AUTHORIZED; requires integrated A1 and separate approval |
| P10-B2 | Previous-text handoff and aggregate accounting; sequence*.go and tests | No next call on bad input, immutable authority, known/unknown/overflow usage | NOT AUTHORIZED; requires B1 integration and separate approval |
| P10-B3 | Whole-sequence host composition; host.go / host_test.go | Admission, inter-step cancellation/drain, restart, panic bookkeeping | NOT AUTHORIZED; requires B2 integration and separate approval |
| P10-C0 | Offline integration / architecture closure | Integrated scope and invariant audit, strict main CI | NOT AUTHORIZED; requires B1/B2/B3 integration and separate approval |

After P10-A1 integration and strict push-main PASS, its classification is
CLOSED / PASS — INTEGRATED. This is the classification rule, not a pre-merge
claim. B1/B2/B3/C0 remain NOT AUTHORIZED until separately approved.

## Future tests and acceptance evidence

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
or provider/tool round-trip behavior must reopen freshness review. P10-A1 makes
zero live calls and accesses no API key.

No version is selected, release authorized, tag created, or Release edited.
Architecture definition has no automatic publication effect. Later implemented
Phase 10 may be considered for an alpha feature release only through a separate
release gate. This decision does not promise production or Beta readiness.
