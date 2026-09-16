# ADR-004: Native Process Scope Ownership and Deterministic Cleanup

## Status and authorization

**CONTROL ROOM ARCHITECTURE ACCEPTED; P11-A1 CLOSED / PASS — INTEGRATED.**

- P11-A0: **CLOSED / PASS — SELECTION ACCEPTED**.
- Phase 11: **DEFINED**.
- Canonical title: **Native Process Scope Ownership and Deterministic Cleanup**.
- P11-A1: **CLOSED / PASS — INTEGRATED** (architecture only).
- P11-B1: **CLOSED / PASS — INTEGRATED** (private coordination only).
- P11-B2: **CLOSED / PASS — INTEGRATED** (private Linux mechanism/proof).
- P11-B3: **CLOSED / PASS — INTEGRATED** (private Windows mechanism/proof).
- P11-B4: **SEPARATELY AUTHORIZED — PRODUCTION RUNNER INTEGRATION**.
- P11-C0: **NOT AUTHORIZED / NOT STARTED**.

Control Room approval supersedes the A0 selection report's historical statement
that Phase 11 was not defined. Selection is not being reopened. This record
defines the accepted architecture. B1 supplies integrated private coordination;
B2/B3 supply integrated Linux/Windows mechanisms and native proof. B4 separately
composes them into the production runner, with its closure conditional on integration
and strict exact push-main CI. C0 and publication remain unauthorized.

## Baseline and existing behavior

Historical A1 architecture preparation main: `bad51ed6ea7f476432c656036288f660281dbd50`.
Tree: `2aaca2b07ebdeaf3694d170b16c49f1ca1711dfa`.
Parents: `92ea08c5af041e6a204891059724811ab582111a` and
`9a1fc045e070056753c9e910ea757a5460a1a83c`.
[Strict main CI 34924714319](https://github.com/kaizenforyou91/forge/actions/runs/34924714319)
is push/main on this exact commit, attempt 1, completed/success: Ubuntu
acceptance, Windows acceptance, and Ubuntu race PASS. This is baseline evidence,
not evidence that future Phase 11 code has passed.

Phases 6–10 remain closed within their accepted scope. In particular, Phase 10
is CLOSED / PASS — BOUNDED WORKFLOW COMPOSITION (Synchronous Sequences).

At the historical A1 baseline, [ProcessRunner](../../../runtime/process_runner.go) starts a validated
materialized executable directly. [RunningProcess](../../../runtime/running_process.go)
owns direct-child cancellation/termination, one wait/reap path, bounded output,
and coordinated [execution-lease](../../../runtime/executable_lease.go) release.
Its existing Terminate requests immediate termination, not graceful shutdown.
Production-runner descendant scope ownership is not currently integrated. The AI lifecycle in
ADR-002/ADR-003 is separate and is not a native-process supervisor.

## Objective and invariant

Own one native launch scope from creation through termination and cleanup while
preserving direct-child results, caller cancellation, and existing launch authority.

The required invariant is: establish the supported scope before child code runs;
retain one lifecycle owner through direct-child reap and terminal scope-control
obligations; never direct a control operation at an unrelated process or reused
identifier. A successful signal request is not proof that every descendant exited.

The strengthened Linux/Windows target is controlled termination and explicit
cleanup outcomes for the supported scope, not a uniform process-tree containment
guarantee. macOS retains its historical direct-child guarantee only. OS profile
limitations must remain visible in architecture and acceptance evidence.

| Platform | Architecture boundary |
|---|---|
| Linux | Strengthened scope target; identifier-reuse safety and Go/OS integration proof required |
| Windows | Strengthened scope target subject to creation-time Job admission proof |
| macOS | Historical direct-child baseline only; no strengthened descendant claim |
| Other GOOS | Unsupported/unclaimed unless separately proven and authorized |

## Threat and authority boundary

In scope: trusted native programs with ordinary or non-cooperative descendants
that remain in the established process group/job; cancellation races; premature
leader exit; inherited output handles; partial startup and cleanup failure.

Out of scope: administrator/root interference, deliberate Unix group/session
escape, process injection, privilege changes, malicious-code safety, fork bombs,
resource quotas, arbitrary filesystem/network side effects, and guaranteed cleanup
after Forge itself is forcibly terminated. Windows job-handle closure may provide
stronger behavior, but that is not a portable parent-crash guarantee.

The caller's existing native launch decision supplies authority. Private scope
identity derives only from that launch. No caller-selected PID/PGID/job name,
process-name enumeration, global kill operation, AI-generated control request,
or model-selected executable is accepted. Scope handles are not inherited by the
child. Provider output remains data only; no Phase 8–10 authority expands.

## Ownership and terminal policy

1. Validate caller context and acquire the existing single-use executable lease.
2. Prepare output/working-directory resources and the platform scope. Preserve
   all executable verification and host-target checks.
3. Establish scope membership as part of launch, before native user code runs.
   Scope-establishment failure is a start failure, not permission to silently
   fall back to unowned direct-child execution.
4. Transfer all successfully created resources to one private lifecycle owner.
   Register/constructor work must not introduce an execution service or worker pool.
5. Caller cancellation or Terminate requests immediate force termination of the
   owned scope. Serialize control, exit observation, and identifier retirement.
   Preserve idempotence and the existing natural-exit/cancellation precedence.
6. Natural direct-child exit also triggers terminal scope shutdown: surviving
   in-scope descendants are not permitted to become intentional background jobs.
   This cleanup action alone does not mark the direct child Canceled/Terminated.
7. Complete required platform scope finalization and direct-child reaping exactly
   once, then finalize bounded output and release the executable lease. Wait must
   not complete before required scope cleanup. Preserve the existing deferred
   materialization cleanup ownership; lease release follows scope finalization.
   Publish a stable result only after owned finalization. Cleanup failures must
   be observable; they do not turn into a successful all-descendants-gone claim.

These strengthened scope obligations apply to the Linux/Windows targets. macOS
retains the existing direct-child cancellation, reap, and cleanup contract.

There is no grace interval or cooperative shutdown protocol in this selected
slice. Terminate remains an immediate request. No automatic retry, restart,
resumption, detached cleanup worker, or background process supervisor is added.

A failed force request does not authorize abandoning the direct child. Preserve
wait ownership even when an OS failure prevents prompt termination. Cancellation
is not a hard upper bound on kernel process exit. Preserve bounded output waiting;
the existing two-second WaitDelay is not a tree-drain or total-shutdown guarantee.

## Linux profile: process-group identity and terminal ordering

Create a separate process group at launch, with the direct child as leader.
Do not first start ordinary child execution and assign a group afterward.
Descendants inherit membership unless they explicitly leave it. Signals target
only the owned group while its identity is protected.

The architecture requires a non-reaping leader-exit observation followed by
serialized terminal group control before final leader reap. On Linux,
waitid with WNOWAIT is a candidate primitive. Merely retaining a numeric PGID
after cmd.Wait has reaped the leader does not satisfy the identity invariant.
No concurrent generic cmd.Wait may consume the leader before this ordering is
complete. No group signals may be issued after identifier retirement.

At the historical A1 checkpoint, the Linux package still had to prove supported
Go/OS integration for this ordering; A1 did not claim mechanism design complete.
The B2 record below supplies the private primitive/proof boundary; production
launcher/wait integration remains a B4 obligation. If this proof cannot be met, stop for Control Room
review rather than retain a stale numeric PGID, weaken identity protection, or
silently remove existing platform support. No inference about macOS follows.

Group signaling does not enumerate/reap all descendants, prevent setsid/setpgid
escape, or prove full quiescence. Do not poll kill(group, 0) and interpret it as
a portable all-descendants-reaped proof. A successful terminal force request
fulfills the group-control obligation; it does not prove immediate cessation of
all work. Escaped descendants and arbitrary kernel delays remain outside scope.
File/lease cleanup must preserve errors if a remaining OS resource prevents it.

## macOS boundary: historical direct-child behavior

macOS retains the historical direct-child guarantee only. This phase does not
infer Linux process-group/wait semantics on Darwin and does not authorize adding
a strengthened macOS descendant profile. Such a claim requires a separately
reviewed mechanism and native macOS acceptance evidence first. Preserve existing
macOS behavior and compatibility; cross-compilation is not native acceptance.

## Windows profile: creation-time Job Object ownership

Use one private Job Object per native launch with termination on last handle
close and no deliberate breakaway permission. Require creation-time assignment
before the first user instruction, using the documented job-list process
attribute on a supported Windows baseline. The implementation package must
record the exact supported Windows/Go matrix before integration.

Starting a running process and subsequently calling AssignProcessToJobObject is
not acceptable. A suspended-create/assign/resume fallback introduces an orphan
window if the parent dies before assignment and is not silently authorized.
Do not treat setting CREATE_SUSPENDED on exec.Cmd alone as a complete launcher;
the necessary startup attributes and process/thread handle lifetimes need proof.

Outer/nested Job policy: never modify, close, or request breakaway from a host or
CI runner's outer Job. Attempt only compatible nested membership in Forge's own
private Job. The same admission contract applies under hosted Windows CI and an
ordinary user account; no administrator/elevation requirement is introduced.
Acceptance must cover launching under an outer Job and prove that closing Forge's
Job does not terminate unrelated host/CI processes. Hosted CI is not an exemption
from pre-user-code admission or evidence of every possible host Job policy.

Nested-job restrictions or unavailable creation-time assignment must fail the
start explicitly, before child work. No automatic degraded direct-child fallback
is selected for the strengthened Windows profile; a degradation policy would need
separate Control Room approval and explicit truthful guarantees. Preserve the
existing restricted environment, working
directory, stdio, direct-child handle, and single wait ownership. A private
platform launch implementation may be necessary; no unsafe general launcher API
or undocumented image-creation API is selected.

Cancellation/termination operates on the owned job; finalization retains ownership
until the job's terminal outcome is observed or an explicit failure is retained.
Do not advertise universal host-crash recovery or hostile-process containment.
Job membership/control, process handles, thread handles, and notification resources
must have exact owners and release paths, including partial initialization.

## Compatibility and result semantics

Preserve public Start, Terminate, Wait, and ProcessResult signatures and fields.
ExitCode continues to describe the direct child. Natural nonzero exit is an
application result, not by itself an infrastructure failure. Cancellation and
manual termination retain their existing classification and late-exit precedence.
No aggregate descendant exit code or new public process-tree result is introduced.

Wait returns defensive cached output with existing independent stdout/stderr
bounds. Output can include inherited pipe writers; it is not authenticated per
PID. Scope-control failures use existing process start/termination/wait error
families as appropriate; preserve simultaneous primary and cleanup errors.
Do not suppress infrastructure failure because the direct child exited zero.
No new public error contract is authorized implicitly.

This is a behavioral expansion of termination to the owned scope, even though
API shape and CLI grammar remain unchanged. Compatibility tests must make that
visible. No automatic fallback to narrower direct-child semantics is allowed
within a strengthened profile. The explicit historical macOS profile is not a
failed Linux/Windows launch fallback.

## Architecture Freeze and surface decisions

Architecture Freeze v1.0 is sufficient; no exception is proposed. Dependency
direction remains CLI to runtime to compiler evidence, with private platform
mechanisms below runtime. pkg/app remains application lifecycle owner. Lower
layers must not import CLI or agent composition. No RunHost/Sequence migration
or new pkg/app production API is selected.

| Surface | Decision |
|---|---|
| New public API / result fields | NO |
| New CLI grammar | NO |
| Manifest or package format change | NO |
| Persistence / durable policy | NO |
| Provider/network/tool authority expansion | NO |
| Background jobs / daemon / worker pool | NO |
| Native termination scope | LIMITED expansion to the caller-authorized launch scope |

Existing owned native wait/output goroutines are not background-job capability.
Any new internal notification goroutine must have bounded ownership and be joined
or released by the launch owner. No dependency change is made by A1; a later
platform dependency must be justified and separately included in authorized scope.

## Non-goals and accepted debt

No graceful shutdown protocol, sandbox, quotas, process-tree escape prevention,
remote registry, persistent trust, key rotation/revocation, SBOM, provider routing,
new tools, AI autonomy, public agent API, public process manager, new CLI grammar,
workflow CLI, scheduler/background service, or durable state. Verified executable
launch binding (including execveat/memfd), general Windows filesystem hardening,
SBOM/provenance, and package-format changes are explicitly excluded.

Executable validation-to-path launch binding, same-user package mutation,
Windows ACL/reparse/share-mode parity, build provenance, and reproducibility
remain separate accepted debt. The A0 trusted-key identity and compiler output
findings require separate triage, not opportunistic remediation in this phase.
None of this reopens Phases 6–10 or implies Beta/production readiness.

## Package sequence and proof gates

These are bounded package boundaries. A1/B1/B2/B3 are integrated; B4 production
integration is separately authorized. C0 remains gated.

| Package | Proposed family / purpose | Required proof | Status |
|---|---|---|---|
| P11-A1 | This ADR, README, CHANGELOG Unreleased, and focused roadmap status | Accurate authority, platform limitations, ownership and acceptance contract | CLOSED / PASS — INTEGRATED |
| P11-B1 | Private platform-neutral scope ownership and terminal coordination | Single owner/control winner; partial-start and resource-release invariants | CLOSED / PASS — INTEGRATED |
| P11-B2 | Private Linux runtime platform mechanisms and tests | Pre-exec grouping; non-reaping observation; safe PGID lifetime; native Linux evidence | CLOSED / PASS — INTEGRATED |
| P11-B3 | Private Windows runtime platform mechanisms and tests | Creation-time job membership; nested-job failures; handle cleanup | CLOSED / PASS — INTEGRATED |
| P11-B4 | ProcessRunner/RunningProcess integration and compatibility tests | Direct-child results, scope termination, output, cancellation and lease ordering | Separately authorized; integration + strict exact push-main CI establishes closure |
| P11-C0 | Documentation and final architecture/acceptance audit | Reviewed integration and strict exact push-main acceptance | NOT AUTHORIZED / NOT STARTED |

B1 depends on separately reviewed A1. B2/B3 depend on the accepted B1 ownership
contract. B4 depends on accepted platform proofs; C0 depends on integrated B4.
No package may silently expand into the deferred trust, filesystem, build, or
AI work. Exact future file scope belongs in each separate authorization.

## B1 implementation record — private coordination only

B1 baseline main: `f8f780c71cfc6eccb7b1ea21be8f89ba06b76627`, tree
`7d6d536127c83a73c7650da84b48e541bbb119e6`; strict push-main CI
[34937937688](https://github.com/kaizenforyou91/forge/actions/runs/34937937688),
attempt 1, PASS for Ubuntu/Windows acceptance and Ubuntu race.

[Private core](../../../runtime/process_scope.go) and
[deterministic tests](../../../runtime/process_scope_test.go) implement only:

- One pointer-owned, non-copyable `processScopeOwner`; all production identifiers
  are private. The two-method `processScopePlatform` supplies only owned-scope
  termination and synchronous resource finalization. Nil/typed-nil platforms and
  zero/incomplete owners are rejected; missing platform methods fail at compile time.
- PREPARED -> ACTIVE -> FINALIZED, with PREPARED -> FINALIZED for partial starts.
  Activation is single-use. Partial-start finalization issues zero termination
  calls. No reset, reuse, PID, output, direct-child result or lease ownership.
- One mutex serializes activation, each entire control call, finalization and
  status observation. Platform methods must not re-enter their owner. Blocking
  private platform work blocks its callers; no cleanup is detached.
- Manual termination, cancellation and natural-exit cleanup are distinct causes.
  Only the first successful platform control records a winner; later requests
  are no-ops. A winner acknowledges successful control, not descendant exit,
  reap or quiescence. Natural cleanup does not classify the direct child as
  manually terminated or canceled.
- Ordinary control failure is returned without a winner. A later caller may
  request control; no automatic retry occurs. The first returned failure is
  retained as bounded internal diagnostic evidence even after later success.
- Finalization closes control admission, invokes resource release once, caches
  its returned error and drops the platform reference even on failure. Repeated
  callers get the cached outcome. FINALIZED means the release attempt ended,
  not that release succeeded or all descendants disappeared. Future integration
  owns the ordering of active control, finalization and public result mapping.
- No universal panic recovery: the original trusted platform panic propagates.
  Interrupted control remains ineligible for another control attempt because its
  effects are unknown, but separate finalization is still available. Interrupted
  finalization retains a private fixed failure, clears the platform reference and
  cannot perform a second release. No panic payload is retained as diagnostics.

Tests use fake private platforms only and assert terminate/finalize call counts,
concurrent winners, ordering in both directions, activation misuse, ordinary
failure preservation, partial starts, panic propagation and reference release.
B1 starts zero goroutines and performs zero native OS operations. It changes no
ProcessRunner, RunningProcess, ProcessResult or public error contract.

At the B1 checkpoint, Linux pre-exec membership, leader-exits-first/non-reaping
wait ordering, PGID reuse/retirement and native proof remained B2 obligations.
Windows creation-time Job membership, nested-host compatibility and native handle
lifetimes are B3 obligations, addressed below. Neither platform proof is supplied by B1;
ProcessRunner integration remains B4. macOS retains direct-child behavior only.
B1 is CLOSED / PASS — INTEGRATED at main
`cf1bb4069210f63ba7bf415b93c992ab753350c5`, tree
`fa8467438a3b5712b425d79dd12c747a2d8e2411`, strict push-main CI
[34941814984](https://github.com/kaizenforyou91/forge/actions/runs/34941814984),
attempt 1 PASS. B2/B3 are now integrated; B4 is separately authorized below.
C0 remains NOT AUTHORIZED / NOT STARTED. Phase 11 is not closed.

## B2 Linux mechanism and identifier-lifetime proof

B2 baseline is the B1 integration commit/tree and strict main CI recorded above.
The mandatory pre-mutation audit inspected installed Go **1.26.5** source:
`src/syscall/exec_linux.go` (SysProcAttr lines 75-91, child setpgid lines 392-398,
execve/error-pipe lines 667-677), `src/syscall/exec_unix.go` (parent error-pipe
handling lines 217-245), and `src/os/exec/exec.go` (StartProcess/error return
lines 733-740). Setpgid executes before user exec; a child setup error returns
through Start instead of silently running outside the requested group.

Installed `golang.org/x/sys@v0.13.0/unix/zsyscall_linux.go` provides Waitid,
Getpgid and Kill; `zerrors_linux.go` provides P_PID, WEXITED, WNOWAIT and SIGKILL.
x/sys v0.13.0 was already present transitively. B2 promotes that exact version to
direct use for supported Linux waitid/WNOWAIT primitives rather than handwritten
raw syscalls. No module/version upgrade or go.sum change is included.

[Linux primitives](../../../runtime/process_scope_linux.go),
[unit tests](../../../runtime/process_scope_linux_test.go) and
[native fixtures](../../../runtime/process_scope_linux_integration_test.go)
are each guarded by `//go:build linux`; all production identifiers are private.

- Preparation accepts only an unstarted, exclusively Forge-owned Cmd. It copies
  SysProcAttr and sets Setpgid=true/Pgid=0 without starting a process. Nonzero Pgid,
  Setsid, Setctty, Foreground, Ptrace or namespace/clone flags are rejected rather
  than overwritten. Unrelated attributes are preserved. A single-use receipt
  binds only that prepared, successfully started, unreaped child. No external
  PID/PGID is accepted. PGID derives from child PID; Getpgid is test evidence,
  not the source of the pre-user-code guarantee.
- Non-reaping observation uses blocking Waitid(P_PID, leaderPID, WEXITED|WNOWAIT).
  EINTR retries the interrupted observation; other errors return unchanged. No
  sleep polling, WNOHANG spin, group-existence probe or final reap is performed.
  ECHILD also retires the now-unowned control identity without publishing success.
- Control uses only Kill(-ownedPGID, SIGKILL). ESRCH maps to os.ErrProcessDone,
  so B1 cannot publish a successful winner for an absent group. Other failures
  remain observable. No signal retry, graceful interval or descendant traversal.
- A control mutex serializes signaling and retirement. A separate observation
  mutex prevents retirement during a blocking wait while allowing control to
  unblock that wait. Finalization retires the numeric identity and clears OS
  seams; it performs no extra kill, reap or nonexistent group-handle close.
  Linux finalization has no other owned resource whose release can fail.
- Required sole-owner order: non-reaping observation, terminal group control,
  **control-identity retirement before normal reap**, then Cmd.Wait. Retiring
  before reap is stronger than merely clearing a remembered PGID afterward and
  closes that race window. The unreaped leader protects ordinary PID reuse during
  control. No method may signal after retirement. This does not protect against
  a future caller that violates sole-reaper ownership or mutates the Cmd.
  Future B4 still owns the full call through reap, output completion and lease
  release; no B2 production function starts a child or calls Cmd.Wait.

Native tests use this test binary and pipes/ExtraFiles, not shells, root or
external programs. Readiness messages and explicit exit commands establish
ordering. WNOWAIT followed by another observation and one successful normal wait
proves non-consumption. The leader-first fixture leaves a same-group descendant
holding a test-owned pipe; group termination produces EOF. A separately grouped
sentinel still answers ping. A controlled descendant calls setsid, survives group
control and answers ping, then exits through its independent control pipe; EOF
confirms fixture cleanup. Failure cleanup preserves pre-reap signaling order and
uses control-pipe closure for the escaped fixture. Deadlines are failsafes only.
No brute-force PID reuse test or all-descendants-reaped claim is made.

Windows local tests and Linux cross-compilation are not native Linux proof.
B2 is CLOSED / PASS — INTEGRATED at main
`f7337587d72315a77dc21bc29ad18c71b49069ce`, tree
`4ed8a9422711d1fa2b117de53ccdb79045935de0`, parents
`cf1bb4069210f63ba7bf415b93c992ab753350c5` and
`a235f0babf9a8f19a999c1eefcda687d7ce02252`.
[Strict push-main CI 34949872571](https://github.com/kaizenforyou91/forge/actions/runs/34949872571)
passed attempt 1: native Ubuntu acceptance/race and Windows acceptance. B1 files, ProcessRunner, RunningProcess, ProcessResult,
errors, output bounds and lease code are unchanged; no production runner path
used B2 at that historical checkpoint. B3 is now integrated and B4 separately
authorized below. C0 remains NOT AUTHORIZED / NOT STARTED; macOS remains direct-child only.
Success still means accepted control, not immediate cessation, universal reap,
closed inherited handles or containment of deliberately escaped processes.

## B3 Windows creation-time Job mechanism and proof gate

B3 starts from the integrated B2 main/tree/parents and strict CI recorded above.
All three new files are Windows-tagged and contain no public runtime API:
[private launcher](../../../runtime/process_scope_windows.go),
[unit tests](../../../runtime/process_scope_windows_test.go), and
[native fixtures](../../../runtime/process_scope_windows_integration_test.go).
No production ProcessRunner call path uses this launcher. B1, B2, runner/result,
lease/output and public error code remain unchanged. go.mod/go.sum are unchanged;
the only non-stdlib dependency remains the already-direct x/sys **v0.13.0**.

### Feasibility and supported evidence profile

The pre-mutation audit inspected installed Go **1.26.5 windows/amd64**:
`src/syscall/exec_windows.go` exposes creation flags, inherited handles and parent
process, but no Job-list attribute in SysProcAttr. Ordinary exec.Cmd.Start cannot
express the selected creation-time contract. x/sys v0.13.0 supplies the Job,
CreateProcess, wait/exit-code, duplication, StartupInfoEx and extended-limit
representations. Required kernel32 exports were checked locally. A no-process
probe successfully installed JOB_LIST with a freshly created private Job.

Microsoft documents JOB_LIST for Windows 10 / Server 2016 and newer, not every
historical Windows version. Its value is the SDK input attribute 13 (`0x2000d`).
The narrow attribute wrapper calls documented Initialize/Update/Delete functions
and owns an aligned, pinned Go buffer, including second-initialization failure.
Attribute values are pinned until list deletion; unpinning and reference release
occur synchronously afterward. No opaque attribute pointer outlives this storage.
This avoids the allocation leak on that failure path in the installed x/sys
container implementation; no dependency upgrade is needed. Missing exports or
unsupported attributes return start failure, never another launch profile.

Local native evidence: Windows **10.0 build 26100**, Go **1.26.5 windows/amd64**,
x/sys **v0.13.0**. The hosted profile inspected before B3 was Windows Server 2025
**10.0.26100**, Go **1.26.8 windows/amd64**, image
**windows-2025-vs2026 / 20260907.229.1**, from baseline
Windows acceptance job **104318019443**. That earlier run is environment evidence,
not B3 proof. The exact-head B3 Windows acceptance run must establish native B3
PASS and its job log supplies the actual runner image/toolchain identity; a moving
windows-latest label is not a permanent platform guarantee. Tests request no
elevation, admin operation, registry/service installation or host-policy change.
They run as the ordinary invoking/hosted runner account; this is not an assertion
that the runner account itself has no administrative group membership.

### Launch authority and handle lifetime

- One private, unnamed Job is created with nil security attributes (non-inheritable).
  Only KILL_ON_JOB_CLOSE is configured, before process creation. No breakaway,
  quotas, UI restrictions, external Job handle, Job name or reusable registry.
- One narrow declaration supplies an absolute executable, explicit command line,
  absolute directory, explicit environment and three borrowed stdio handles.
  The caller keeps these handles alive through start. No inherited environment,
  shell, token/parent override, PID selector or arbitrary creation flags.
- File/pipe/console stdio is duplicated into owned inheritable handles without
  changing the caller's handle flags. Exactly those three duplicates enter
  HANDLE_LIST; the private Job never does. JOB_LIST contains exactly that Job.
  Attribute values remain live until list deletion. Explicit Unicode environment
  and EXTENDED_STARTUPINFO_PRESENT accompany CreateProcessW.
- Creation-time JOB_LIST admission is the guarantee. Post-create IsProcessInJob
  is evidence only. Neither Start-then-Assign nor suspended-create/Assign/Resume
  is implemented. Unsupported capability or rejected nesting fails start.
- The Job object owns scope control only. The primary thread handle is closed
  after creation; the private launch result transfers a separate direct process
  handle to its caller for wait/result/close. B3 does not own ProcessResult,
  output accounting, executable leases or the production runner's waiter.
- Partial-start cleanup deletes attributes, closes duplicated stdio and thread
  handles, and finalizes the Job. A failure after process creation also requests
  owned-Job termination and closes the process handle. Cleanup errors are joined,
  not discarded or retried. No cleanup goroutine is detached.

### Scope control, nesting and native proof

TerminateJobObject(privateJob, **1**) acknowledges a control request only. It does
not prove process objects are signaled, pending I/O completed, handles closed or
malicious escape contained. Native tests separately wait for process signaling
and pipe EOF. The private platform implements unchanged B1 terminate/finalize
semantics; mutex serialization prevents control using a retired handle.
Finalization retires the Job handle before its single CloseHandle attempt, caches
close failure and clears control references. Empty/PREPARED Job closure is resource
cleanup. KILL_ON_JOB_CLOSE is failure safety; B4 must order normal active terminal
control before finalization and must not treat CloseHandle as a B1 control winner.

Windows decides whether an inherited outer Job and Forge's new nested Job form
a compatible hierarchy. Forge does not open, mutate or close the host Job,
request breakaway, or bypass host policy. Rejected nesting is an explicit failure.
The deterministic native nesting fixture creates its own outer Job, launches an
intermediate helper, then creates a private inner Job and ordinary outer sibling.
Inner termination/closure must leave both the intermediate and sibling responsive.

Fake tests assert exact create/terminate/close/list-delete counts and handle sets,
invalid declarations, attribute/creation failures, post-create cleanup, preserved
errors and concurrent exactly-once finalization. Self-contained Go test helpers
prove immediate membership plus first handshake, ordinary descendant inheritance,
process signaling/inherited-pipe closure, independent sentinel survival, nested
Job isolation and isolated kill-on-close behavior. Pipes/messages establish order;
timeouts are failsafes, not sleep-based proof. No descendant enumeration or external
helper executable is required. The existing full Windows acceptance command runs
these tests without skips; Linux acceptance/race remain regression gates.

B3 integration plus strict exact push-main CI establishes **CLOSED / PASS —
INTEGRATED**. PR/native proof alone is not integration or Phase 11 closure.
At the historical B3 checkpoint B4/C0 were **NOT AUTHORIZED / NOT STARTED**.
B3 is now integrated; B4 is separately authorized below. macOS retains historical
direct-child behavior; other GOOS are unclaimed. There is no public API, CLI,
package-format, persistence, background-service or AI-authority expansion.

References: [creation attributes](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute),
[CreateProcessW](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw),
and [nested Jobs](https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs).

## B4 production runner integration record

Canonical B3 integration: main `0439ac342219482520aaeb236c02ecc4cb3807d9`,
tree `b116de08a74f949627d927f9b1e3b6f55acda7b4`, parents
`f7337587d72315a77dc21bc29ad18c71b49069ce` and
`302a039489a4eb880019a343b7293605dce24e13`. Strict push-main
[35063351382](https://github.com/kaizenforyou91/forge/actions/runs/35063351382)
passed attempt 1: Windows acceptance, Ubuntu acceptance and Ubuntu race.

B4 changes only runner integration, private execution adapters, bounded fixtures
and focused documentation. B1/B2/B3 source and public result/error surfaces remain
unchanged. Build tags select Linux, Windows, or historical direct-child execution.
The private execution adapter separates exit observation from final wait/output
ownership. Each successful launch activates one B1 owner. A separate mutex gate
serializes manual/cancel admission against observed completion; successful control
is still not quiescence. Natural cleanup never sets public cancellation/termination.
Manual failure stays cached; failed cancellation remains observable with context
and control error evidence even if the child subsequently exits naturally.

Linux observes WNOWAIT, closes classification, requests natural group cleanup only
without an earlier winner, retires the identity, then calls Cmd.Wait once. Binding
uses exclusively owned command preparation; no foreign reaper or attribute mutation
is permitted. Windows uses unchanged B3 JOB_LIST admission, observes the process
handle, closes classification, controls/finalizes the Job, collects exit status,
joins output drains and closes the direct handle. B4 composes a synchronous wait
into B3's existing per-call close seam for post-create launch failures, before B3
releases the process handle. It does not change Job membership or termination.

The joined per-process cancellation watcher performs only gated scope control.
Windows stdout/stderr each drain continuously into the existing 1 MiB writer;
a two-second post-exit failsafe closes only owned readers and joins both helpers,
retaining an infrastructure error. Stdin is the null device, arguments remain
empty and runtimeProcessEnvironment remains the sole logical environment policy.
The executable lease releases only after scope finalization, native completion,
output completion and watcher join. Nonzero child exit remains application data;
cleanup failures preserve that result and existing error classifications.

Self-executable fixtures use marker handshakes and inherited pipes to exercise
natural leader-first cleanup, manual/cancel descendant control and output EOF
through the actual production runner. Existing compatibility tests remain, with
native completion assertions adapted to both exec.Cmd and process-handle ownership.
No native mechanism, new dependency, public API, CLI, package format, persistence,
macOS strengthening, executable binding or hostile-process containment is added.
B4 integration plus strict exact push-main CI establishes **CLOSED / PASS — INTEGRATED**.
P11-C0 remains **NOT AUTHORIZED / NOT STARTED**; this record does not close Phase 11.

## Acceptance model

Use deterministic native fixtures and synchronization, with bounded test deadlines
and cleanup for failed assertions. Cover child/grandchild launch, parent-first
exit, descendants retaining pipes, mixed output saturation, cancellation before
start/during execution/after exit, concurrent Terminate/Wait, partial start, and
scope-control/handle-close/lease-cleanup failures.

Require proof that no child work precedes scope establishment, no reused/unrelated
identifier is signaled, no background survivor is intentionally authorized, and
no failure produces an unsupported containment claim. Test Unix escape as an
explicit limitation and ensure test-owned escape fixtures are cleaned up safely.
Test Windows restrictions/nesting, ordinary-user hosted CI compatibility, and all
handle-lifetime failure paths. Preserve the macOS direct-child baseline; stronger
macOS descendant acceptance is outside the selected implementation target.

Preserve existing executable validation, trusted-package, output, result, CLI
exit mapping, single-use lease, and cancellation regressions. A1 adds no runtime
test or capability. Later code requires format, dependency cleanliness, package
listing, vet, full tests/build, Ubuntu/Windows acceptance, and canonical hosted
race coverage. Any later strengthened macOS claim requires separately reviewed
mechanism and native macOS evidence first. Cross-compilation alone
is not filesystem/process semantic evidence. No retry or waiver is pre-authorized.

Native proof of the non-reaping Linux integration and Windows creation-time job
launcher is required before those platform packages can pass; this document does
not substitute architecture assertions for the native evidence recorded for each package.

## Publication and closure boundary

Published v0.4.0-alpha.1 remains immutable and contains neither Phase 10 nor
Phase 11. Its annotated tag object is
`c02b81a94c3c7bead6a7fcf4030ce03cc98af52a`, targeting
`85d78143db1b8bcf2f96b79d681895b1a0492642`; Release `388084778` is
draft=false, prerelease=true, assets=0. No version, tag, release, or asset action
is selected. No live OpenAI call or API-key access is required.

Phase 11 is defined and not closed. A1/B1/B2/B3 are integrated. B4 supplies
separately authorized production integration; its integration and C0 closure
transactions require their own review and authorization. A0's superseded status remains historical;
it is not a reason to repeat architecture selection.

## OS references and design consequences

- [Linux process groups](https://man7.org/linux/man-pages/man2/setpgid.2.html):
  group membership is inherited but can change; this is not containment.
- [Linux wait semantics](https://man7.org/linux/man-pages/man2/waitpid.2.html):
  WNOWAIT permits observation without consuming the child's waitable state;
  integration must preserve that distinction until group control finishes.
- [Windows Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects):
  job policy and membership are OS-specific and require explicit handle ownership.
- [Windows creation-time job assignment](https://devblogs.microsoft.com/oldnewthing/20230209-00/?p=107812):
  assignment before user code avoids the post-start admission race.
- [Windows console control events](https://learn.microsoft.com/en-us/windows/console/generateconsolectrlevent):
  console-specific behavior is not a general graceful shutdown protocol.

These references inform the proposed design; they are not evidence that Forge's
future implementation is complete or that macOS behavior has been validated.
