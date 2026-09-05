# Forge Roadmap

> Engineering roadmap for the Forge platform.

**Project Status:** First Alpha — 0.3.0-alpha.1

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

These are long-term Phase 2 command targets. The currently implemented
top-level commands are `forge version`, `forge doctor`, `forge config`,
`forge validate`, `forge build`, `forge build-runnable`, `forge inspect`, and
`forge run`. The historical `forge init` and `forge fmt` targets remain
deferred.

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

Capabilities:

- Prompt Execution
- Tool Calling
- Agent Runtime
- Memory
- Workflow Engine

Status:

Not Started

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

Not Started

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
| Phase 8 — AI Runtime | ⏳ Not Started |

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
  compiler/runtime/CLI boundary

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
  scheduler work, and AI runtime capabilities.

## Evidence-Based Current Roadmap Position

```text
First Alpha — 0.3.0-alpha.1
→ Phase 1 — Core Foundation: Alpha-Bounded Closed
→ Phase 2 — Alpha workflow implemented; long-term expansion planned
→ Phase 3 — Manifest Engine: Complete for current contract
→ Phase 4 — Alpha validation workflow implemented; long-term expansion planned
→ Phase 5 — Registry: Alpha-Bounded Closed
→ Phase 6 — Compiler / Package Pipeline: CLOSED / PASS
→ Phase 7 — Runtime: Alpha-Bounded Closed
→ Phase 8 — AI Runtime: Not Started
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
- AI Runtime

# Long-Term Goal

Forge 1.0 will provide a complete AI-native development platform that enables developers to build modern applications through declarative manifests, modular runtimes, and intelligent automation.
