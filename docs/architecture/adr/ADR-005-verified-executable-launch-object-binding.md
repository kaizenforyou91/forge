# ADR-005: Verified Executable Launch Object Binding

## Status and authorization

**PHASE 12 CLOSED / BLOCKED-DEFERRED — FROZEN INVARIANT RETAINED; NO
VERIFIED-LAUNCH CLAIM.**

- P12-A0: **CLOSED / PASS — SELECTION ACCEPTED**.
- Phase 12: **CLOSED / BLOCKED-DEFERRED — FROZEN INVARIANT RETAINED; NO
  VERIFIED-LAUNCH CLAIM**.
- P12-A1: **CLOSED / PASS — INTEGRATED**.
- P12-B1: private platform-neutral ownership core, **CLOSED / PASS — INTEGRATED,
  DORMANT**.
- P12-B2: **CLOSED / FEASIBILITY STOP — NO IMPLEMENTATION**.
- P12-B3: **CLOSED / FEASIBILITY STOP — NO IMPLEMENTATION**.
- P12-B4: **CLOSED / NOT EXECUTED — NATIVE PREREQUISITES ABSENT**.
- P12-R0: **CLOSED / PASS — REASSESSMENT ACCEPTED**.
- P12-C0: **CLOSED / PASS — DEFERRED CLOSURE INTEGRATED**.
- P12-C0-R1: **CLOSED**; original canonical-main CI attempt 1 failed and remains
  historical evidence.
- P12-C0-R1-CI1: **CLOSED / PASS**.
- P13: **NOT AUTHORIZED / NOT STARTED**.

This ADR preserves the frozen architecture. Current main implements no native
launch-binding mechanism and makes no strengthened platform claim. C0 records the
deferred closure without weakening the invariant.

## Baseline and problem

Canonical A1 baseline is main `1e0c8dee7d9bbe59d7726e9c08a5626b4f46f0be`,
tree `775204c104d9036741c5fc8b325f7984fed101b9`. Phase 11 is CLOSED / PASS —
NATIVE PROCESS SCOPE OWNERSHIP AND DETERMINISTIC CLEANUP. Strict closure CI
`36396310427` is push/main, attempt 1, completed/success on that exact head;
Ubuntu acceptance, Windows acceptance, and Ubuntu race passed.

Forge currently obtains detached verified runnable bytes, writes them into a
controlled private materialization, opens the executable, validates regular-file
status, size, SHA-256, host binary format and architecture, compares open-object
and path identity, closes the validation handle, performs a final pathname check,
and passes `lease.path` to the platform launcher. The launcher resolves that path
again during native process creation.

The remaining race concerns object identity, not merely stale content. An actor
with equivalent filesystem access may attempt to replace, redirect, delete and
recreate, or modify the materialized executable after accepted validation but
before native image admission. Equal hashes do not make two filesystem objects
identical, and a correct identity check followed by an unprotected pathname
launch does not bind the later resolution.

## Security invariant

For every platform on which strengthened verified launch binding is claimed, the
native executable image admitted for process creation **MUST derive from the exact
open filesystem object whose complete bytes, executable format, host architecture,
and object identity Forge accepted**. No mutable pathname may select a different
executable object after that acceptance.

Forge retains binding authority until either:

1. native image admission is proven successful; or
2. start fails closed and all required binding and process-ownership cleanup is
   synchronously complete.

The following are locked consequences:

- content equality is not object identity;
- hash-before-launch alone is insufficient;
- a final `SameFile`, inode/device, or file-ID comparison followed by an
  unprotected pathname launch is insufficient;
- post-start hashing is not admission evidence;
- filesystem cleanup success is not admission evidence;
- no public caller-selected FD, handle, PID, or executable path is introduced;
- provider or model output remains data only.

## Threat model

In scope is ordinary concurrent mutation by another process with the same user
or equivalent filesystem access: rename/replace, delete/recreate, in-place writes,
same-size replacement, same-content/different-object replacement, symlink or
Windows reparse substitution, controlled-directory replacement, supported
namespace-component replacement, and filesystem identity change before admission.

Out of scope are administrator/root interference, kernel compromise, filesystem
corruption, malicious behavior of intentionally admitted code, sandboxing,
hostile-code containment, package trust-policy changes, persistent trust,
revocation or rotation, filesystem/network isolation, quotas, provenance, and
SBOM completeness. Same-user mutation of an already-open source ZIP package is a
separate boundary.

## Private ownership abstraction

Phase 12 uses the conceptual private abstraction `verifiedLaunchObject`. This is
not a public Go API and does not freeze an implementation type name. It is:

- private to `runtime`;
- single-use;
- created only after acquisition of `executableLease`;
- owner of the accepted open executable evidence;
- owner of platform identity evidence;
- owner of platform namespace-exclusion capability where required;
- able to state whether native admission is proven; and
- synchronously finalized with an explicit start-failure lease disposition.

Conceptual ownership is:

```text
MaterializedExecutable
  → executableLease
    → verifiedLaunchObject
      → accepted executable object
      → platform identity evidence
      → platform namespace protection where required
      → native admission state
```

## Lifetime contract

1. Forge acquires `executableLease` first.
2. It opens and validates through the object that remains binding authority.
3. Hash, header, architecture, regular-file, size, and object validation consume
   that retained object.
4. Binding authority does not close before native process admission.
5. Native admission consumes that exact object directly, or a path namespace
   proven unable to resolve to another object throughout admission.
6. No executable FD or handle is unintentionally inherited.
7. Successful native admission permits verified-launch resources to close.
8. Phase 11 then owns normal scope and terminal lifecycle.
9. `executableLease` remains held through Phase 11 terminal obligations.
10. Partial admission failure publishes an explicit release disposition.

Finalization is synchronous and exactly once. No retry, detached cleanup worker,
global supervisor, or reusable launch object is introduced.

## Start-failure disposition

The architecture composes with the P11-B4-R2 fail-closed lease model:

| Disposition | Required truth | Lease result |
|---|---|---|
| `PRE_NATIVE_FAILURE` | No target process was created and binding resources are fully closed | May release |
| `POST_CREATE_ADMISSION_PROVEN` | Native process exists, verified admission is established, and ownership transfer succeeded | Continue normal Phase 11 lifecycle |
| `POST_CREATE_CLEANUP_COMPLETE` | Start cannot return success, but binding and Phase 11 cleanup are proven complete | May release |
| `ADMISSION_OR_TERMINAL_PROOF_INCOMPLETE` | Binding or terminal ownership cannot be proven complete | Must retain |

There is no default-to-release path and no error-string inference. Errors remain
observable. Cleanup is synchronous, with no automatic retry.

## Linux profile

Linux targets **direct object-bound native admission**. The preferred candidate is:

```text
execveat(fd, "", argv, envp, AT_EMPTY_PATH)
```

or an equivalently direct descriptor-based native primitive. The final mechanism
must execute the accepted object without mutable pathname re-resolution.

P12-B2 must prove:

- safe integration with the supported Go toolchain;
- no arbitrary or unsupported Go callback after fork;
- preservation of Phase 11 pre-user-code process-group establishment;
- the retained accepted descriptor is the execution object;
- pathname replacement cannot redirect admission;
- controlled argv, environment, cwd, stdin, stdout, and stderr remain intact;
- no executable FD leaks into the final ELF image;
- direct-child PID and result semantics remain compatible; and
- partial native-start failure is synchronously owned and lease-safe.

`/proc/self/fd/N` may be used for research, but it is not the frozen final
guarantee. Phase 12 selects no procfs dependency. A final inode check, hash, or
`SameFile` check followed by pathname execution is not a fallback.

**Feasibility stop:** if direct descriptor-bound execution cannot be integrated
safely without unstable Go runtime internals, unsupported fork-time Go execution,
or weakening Phase 11, P12-B2 must stop. It must not silently lower the invariant.

## Windows profile

Windows has **conditional strengthened support pending native P12-B3 proof**.
The accepted architecture recognizes that `CreateProcessW` is path based and
selects no documented CreateProcess-from-file-handle primitive.

The candidate strengthened design combines:

- a private, non-inheritable verified executable handle;
- no `FILE_SHARE_WRITE`;
- no `FILE_SHARE_DELETE`;
- only the read sharing required for native image admission;
- `FILE_ID_INFO` volume serial and 128-bit file ID;
- explicit reparse-point rejection;
- held directory and namespace authority sufficient for the supported local
  filesystem profile;
- controlled `CreateProcessW`;
- post-create binding corroboration where required; and
- unchanged Phase 11 creation-time Job admission.

Binding handles must be excluded from the child `HANDLE_LIST`. Forge must not
modify a host Job, request breakaway, or weaken the B3 creation-time `JOB_LIST`
guarantee.

A file handle alone is not sufficient. File identity alone is not sufficient.
The complete path resolution relevant to `CreateProcessW` must be proven unable
to select another executable object during admission.

### Windows namespace proof gate

P12-B3 must determine whether documented Win32 operations can establish this
combination for an explicitly supported local filesystem profile:

```text
accepted file object
  + stable identity
  + mutation exclusion
  + namespace non-redirection
  + CreateProcessW compatibility
```

Native proof must cover file write, rename, delete, and replacement denial;
final-file reparse substitution; containing-directory replacement and rename;
relevant ancestor substitution; stable `FILE_ID_INFO`; `CreateProcessW` under
the selected share modes; absence of inherited binding handles; and preservation
of creation-time Job admission.

P12-A1 does not promise all Windows filesystems. B3 may select a documented,
tested local profile such as NTFS or ReFS only when evidence supports it. Remote
filesystems, unstable or unavailable 128-bit identity, or incompatible reparse
and namespace semantics must fail closed under the strengthened internal path.

If namespace non-redirection cannot be proven, strengthened Windows Verified
Launch Binding is **UNSUPPORTED / FAIL-CLOSED**. Forge must not downgrade the
claim to best effort or infer full binding from file locking. Undocumented
`NtCreateUserProcess` or other undocumented native interfaces are rejected.

## macOS and other GOOS

macOS retains historical path-based direct-child launch only. Other GOOS retain
historical behavior unless separately proven. No strengthened Verified Launch
Binding claim is made. A future internal caller requiring the strengthened
guarantee on an unsupported platform fails closed. A1 adds no public feature flag
or CLI switch.

## Composition with Phase 11

Phase 12 owns **native admission identity**. Phase 11 owns **process scope and
terminal cleanup**. These authorities remain distinct.

Linux target ordering:

```text
retain accepted executable object
→ prepare Phase 11 process group
→ descriptor-bound native admission
→ establish admission proof
→ close launch-binding resources
→ Phase 11 observe/control/retire/reap/output
→ executable lease release
```

Windows target ordering:

```text
retain accepted executable and namespace authority
→ creation-time Phase 11 Job admission
→ CreateProcessW under protected namespace
→ establish verified binding proof
→ close launch-binding resources
→ Phase 11 result/control/quiescence/finalization/output
→ executable lease release
```

Phase 12 preserves Linux PGID ordering, Windows creation-time Job admission,
direct-child result authority, output bounds, cancellation semantics, and the R2
failed-start lease rule.

## Rejected architectures

The following do not establish the invariant and are rejected as the final
binding guarantee:

- hashing immediately before launch;
- hashing after process creation;
- `SameFile` followed by pathname launch;
- inode/device comparison followed by pathname launch;
- file-ID comparison followed by unprotected pathname launch;
- permissions or ACLs alone;
- a Windows executable handle without namespace proof;
- `/proc/self/fd` as the final cross-environment Linux contract;
- a memfd copy as the default architecture;
- undocumented Windows process-creation APIs;
- suspended-process post-create hash verification; and
- cleanup or deletion success as admission evidence.

## Deterministic proof contract

Shared future tests coordinate with channels, native events, or equivalent
deterministic barriers rather than sleeps. They cover replacement after accepted
validation; rename/replace; delete/recreate; same-size/different-byte and same-
byte/different-object replacement; in-place mutation; symlink/reparse substitution;
directory replacement; cancellation during admission; partial native start;
exactly-once binding-resource finalization; retained lease while proof is
incomplete; Phase 11 lifecycle regressions; and outside-scope sentinel survival.

Linux proof additionally accepts object A, replaces its path with B, proves A
executes, proves descriptor identity, proves no executable FD leak, preserves the
Phase 11 process group, requires no procfs, and covers partial exec cleanup. If the
selected mechanism supports execution after unlink, that behavior is proved too.

Windows proof additionally blocks write/rename/delete/supersede while binding is
active, rejects reparse substitution, blocks or detects directory and namespace
replacement before admission, proves stable `FILE_ID_INFO`, proves
`CreateProcessW` compatibility, proves the fixture corresponds to the accepted
object, excludes binding handles from inheritance, preserves Job membership, and
fails closed on unsupported filesystem semantics.

Tests prove binding itself. Hash equality, timing, output, or cleanup success is
not a substitute.

## Package sequence and gates

| Package | Purpose | Required gate | Status |
|---|---|---|---|
| P12-A1 | Architecture freeze / this ADR | Accurate invariant, threat model, profiles, lifetime and tests | CLOSED / PASS — INTEGRATED |
| P12-B1 | Private verified-launch ownership core | Single use, admission state, deterministic finalization, failure disposition; no native mechanism | CLOSED / PASS — INTEGRATED, DORMANT |
| P12-B2 | Linux descriptor-bound admission / proof | Actual replacement resistance, direct-object execution, no FD leak, safe Go and Phase 11 integration | CLOSED / FEASIBILITY STOP — NO IMPLEMENTATION |
| P12-B3 | Windows mutation-exclusion and namespace proof | Identity, namespace, reparse, CreateProcessW, Job, inheritance, filesystem profile | CLOSED / FEASIBILITY STOP — NO IMPLEMENTATION |
| P12-B4 | Production ProcessRunner integration | End-to-end coordinated replacement tests, lifecycle regressions, fail-closed partial start | CLOSED / NOT EXECUTED — NATIVE PREREQUISITES ABSENT |
| P12-C0 | Deferred closure evidence and documentation | Truthful feasibility record and exact integrated-main CI | CLOSED / PASS — DEFERRED CLOSURE INTEGRATED |

Completion of A1 did not automatically authorize a B package. Control Room
separately authorized B1, B2, and B3. B2 and B3 ended at their required
feasibility stops without implementation; B4 was not authorized or executed.

## P12-B1 implementation record

`runtime/verified_launch.go` implements one private pointer-owned, single-use
coordination owner and a one-operation private platform contract whose only
authority is synchronous binding-resource finalization. The owner starts no
goroutine and performs no filesystem, syscall, process-creation, PID, path, FD,
or handle operation. No production execution path uses it in B1.

The explicit lifecycle is PREPARED to FINALIZED, with independent monotonic
evidence for native process creation and verified admission. Native creation does
not imply admission; admission cannot be recorded before creation; finalization
does not fabricate either fact. Finalization makes one serialized platform call,
caches its returned result, retires the platform reference, and never retries.
A propagating trusted-platform panic leaves explicit interrupted evidence and
poisons the retired authority against a second release attempt.

The immutable-by-value private status snapshot contains only phase, creation,
admission, and finalization outcome evidence. It exposes no platform resource.
Later B4 integration can combine these facts with Phase 11 terminal evidence to
derive the frozen start-failure dispositions without a default-to-release guess.
Deterministic fake-platform tests cover invalid and typed-nil construction,
ordering, single transitions, preservation across finalization, returned failure,
panic interruption, concurrent evidence reports, concurrent finalization, exact
call counts, and control/finalization serialization. B1 adds zero native binding
mechanism and makes no strengthened Linux or Windows claim.

## Feasibility outcome and deferred closure

ADR-005 remains the authoritative desired security invariant. Its architecture
is security-valid, but feasibility stopped on both primary target platforms
within Forge's supported toolchain and documented-native-API boundaries. This is
an architecture and platform boundary, not a test failure, flaky result, reduced
threat model, or partial implementation.

P12-B2 established that Linux `execveat(fd, "", argv, envp, AT_EMPTY_PATH)` can
select a retained descriptor object at the kernel boundary. The supported Go
child-start APIs cannot safely and sustainably combine that operation with the
runtime-managed fork protocol, pre-user-code `setpgid`, controlled cwd, argv,
environment and stdio, descriptor remapping and closure, synchronous exec-failure
evidence, and Phase 11 ownership. Private runtime hooks, `go:linkname`, copied
standard-library fork internals, unsafe post-fork Go, unsupported clone/vfork,
procfs binding, and an unbound helper-process workaround remain rejected. An
independent blocker also remains: an ordinary retained file descriptor preserves
object identity but does not guarantee accepted-byte immutability against every
pre-existing mutation authority. B2 produced no tracked implementation.

P12-B3 evaluated documented Windows behavior on build `10.0.26100` and local
NTFS. A restrictive non-inheritable file handle blocked a new writer, deletion,
and rename; held directory authority blocked the tested containing-directory
rename; a pre-existing writable mapping caused restrictive acquisition to fail
closed; `FILE_ID_INFO` was available; and `CreateProcessW` remained compatible
with the candidate protection. The complete invariant was still unproven:
`CreateProcessW` accepts a name or path rather than the verified handle, returns
no documented admitted-image `FILE_ID_INFO`, and documented operations did not
prove complete ancestor and reparse namespace non-redirection. A post-create
path, hash, or file-ID comparison remains corroboration rather than admission
proof. B3 produced no tracked implementation.

The shared missing proof chain is:

```text
accepted bytes and object identity
  → immutable binding authority
  → supported native admission
  → authoritative admission evidence
```

B1 remains integrated and dormant. Its private single-use ownership and evidence
model changes no production path, exposes no public API, and creates no process,
network, or filesystem authority. It remains reusable for a future original-
object or immutable-snapshot architecture, so deferral does not require removal.
P12-B4 was never implemented: production `ProcessRunner` has no verified-launch
integration, and current main makes no strengthened verified-launch claim.

That closure gate passed when PR #39 head
`ca04feca740e695731fb07203b1169a866d24ff0` merged as
`917def87aebb4d4351448f4bd84bd7f6c8357179`, tree
`e8290f89c69135ea965c26061582e758091f658c`, with parents
`8a3b13af37a5d8742e5a105a5d38b7a0bdfb6f85` and
`ca04feca740e695731fb07203b1169a866d24ff0`. Strict push-main CI
`36985492452` passed attempt 1 on the exact merge head: Ubuntu acceptance
`110769456688`, Windows acceptance `110769457708`, and Ubuntu race
`110769457050` all passed. Phase 12 is therefore **CLOSED / BLOCKED-DEFERRED —
FROZEN INVARIANT RETAINED; NO VERIFIED-LAUNCH CLAIM**. This means the property
remains desirable but current
supported platform and toolchain primitives cannot establish it within Forge's
accepted boundaries. It does not mean the specification failed, implementation
partially shipped, path-based launch became verified, or the threat model was
reduced.

### Final closure-evidence reconciliation

P12-C0-R1 merged as `e8a7100ec14e4da2124a439a10a51c77e67b5dd0`.
Its original canonical-main workflow `37279190953` failed attempt 1 in Ubuntu
race at `TestFunctionRoundTripDiagnosticContext/true/second`. The failure was
diagnosed as nondeterminism in the scripted test transport: after deterministically
observing context cancellation, the fixture could still return a synthetic
successful response. It was not a production semantic defect.

P12-C0-R1-CI1 changed only that test fixture and its assertion diagnostics; it
made no production behavior change and did not alter this ADR's invariant or
feasibility conclusions. Remediation PR #41 at head
`d8a667deb39d6ddfa8772b8cb1e07ab50fcbe1b8` merged as canonical main
`0a13ab776e66c35e3a9137b465ee1fce2abae8ff`. Fresh push-main CI
`37402403722` passed attempt 1 on that exact head: Ubuntu acceptance
`112072276131`, Windows acceptance `112072276373`, and Ubuntu race
`112072276420` all passed. P12-C0-R1-CI1 is therefore **CLOSED / PASS**. The
original failed run remains part of the permanent evidence chain.

This reconciliation completes evidence for the deferred closure. Phase 12
remains **CLOSED / BLOCKED-DEFERRED — FROZEN INVARIANT RETAINED; NO
VERIFIED-LAUNCH CLAIM**. It does not imply that native verified launch binding
was implemented, that the B2/B3 feasibility stops were reversed, that B4 ran, or
that the invariant was weakened.

### Reassessment triggers

Linux reassessment requires a supported Go API or toolchain facility that
provides an executable descriptor target or `execveat` equivalent while retaining
the runtime-supported fork protocol, pre-exec `setpgid`, controlled argv,
environment, cwd and stdio, deterministic descriptor remapping and closure,
synchronous exec-failure reporting, no leaked executable descriptor, and no user
Go callback after fork.

Windows reassessment requires a documented supported API that either creates a
process from an already-open executable handle or returns authoritative admitted-
image identity tied to a specified open file object. It must compose with
`STARTUPINFOEX`, `JOB_LIST`, `HANDLE_LIST`, non-inherited binding handles,
documented namespace and reparse behavior, and non-administrator operation.

If those native-admission prerequisites become viable, a separately selected
future direction may begin as **P13-A0 — Verified Immutable Execution Snapshot
Selection**, with a new ADR. A sealed Linux memfd could address snapshot byte
immutability, but it does not by itself solve supported Go descriptor execution.
ADR-005 is not rewritten into snapshot semantics. Windows executable mutation
exclusion is likewise a potentially useful, separately named weaker hardening
family; it is not Verified Executable Launch Object Binding.

## Public surface and dependencies

Expected architecture impact is: no public Go API, CLI grammar, manifest/schema,
package-format, trust-semantics, persistence, scheduler, network authority, or
agent/AI authority change. No dependency version change is selected. Existing
`golang.org/x/sys` remains the preferred native boundary where sufficient; any
future dependency or version change requires separate review.

Architecture Freeze v1 remains in force with no exception. `runtime` retains
executable-lease, verified-launch, native-start, process-scope, and terminal
lifecycle ownership. No public native handle, caller-provided PID/path, global
supervisor, worker pool, daemon, service, or plugin loader is introduced.

## Non-goals and deferred work

This phase does not include persistent trust, rotation/revocation, provenance or
SBOM, general Windows ACL hardening, sandboxing, privilege drop, filesystem or
network isolation, quotas, graceful shutdown, scheduling, autonomous agents,
remote registries, or same-user source-package snapshotting. A strict prerequisite
discovered later must be reported as a dependency rather than silently absorbed.

## Release boundary

Published `v0.4.0-alpha.1` remains immutable: annotated tag object
`c02b81a94c3c7bead6a7fcf4030ce03cc98af52a`, peeled source
`85d78143db1b8bcf2f96b79d681895b1a0492642`, Release `388084778`,
`draft=false`, `prerelease=true`, zero assets. Phases 10, 11, and 12 are not in
that release. A1 authorizes no version selection, tag, Release, or asset action.
