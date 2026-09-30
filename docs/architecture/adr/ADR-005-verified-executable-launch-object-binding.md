# ADR-005: Verified Executable Launch Object Binding

## Status and authorization

**CONTROL ROOM ARCHITECTURE SELECTED; P12-A1 CLOSED / PASS — INTEGRATED; P12-B1 IMPLEMENTED / UNDER REVIEW.**

- P12-A0: **CLOSED / PASS — SELECTION ACCEPTED**.
- Phase 12: **DEFINED — VERIFIED EXECUTABLE LAUNCH OBJECT BINDING**.
- P12-A1: **CLOSED / PASS — INTEGRATED**.
- P12-B1: private platform-neutral ownership core, **IMPLEMENTED / UNDER REVIEW**.
- P12-B2/B3/B4/C0: **NOT AUTHORIZED / NOT STARTED**.

This ADR freezes architecture. It implements no launch mechanism and makes no
strengthened platform claim. Each implementation package remains separately gated.

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
| P12-B1 | Private verified-launch ownership core | Single use, admission state, deterministic finalization, failure disposition; no native mechanism | Implemented / under review |
| P12-B2 | Linux descriptor-bound admission / proof | Actual replacement resistance, direct-object execution, no FD leak, safe Go and Phase 11 integration | NOT AUTHORIZED / NOT STARTED |
| P12-B3 | Windows mutation-exclusion and namespace proof | Identity, namespace, reparse, CreateProcessW, Job, inheritance, filesystem profile | NOT AUTHORIZED / NOT STARTED |
| P12-B4 | Production ProcessRunner integration | End-to-end coordinated replacement tests, lifecycle regressions, fail-closed partial start | NOT AUTHORIZED / NOT STARTED |
| P12-C0 | Final evidence / documentation closure | Truthful platform matrix and exact integrated-main CI | NOT AUTHORIZED / NOT STARTED |

Completion of A1 did not automatically authorize a B package. Control Room
separately authorized B1; B2/B3/B4/C0 remain gated.

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
