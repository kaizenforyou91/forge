# Forge Roadmap

> Engineering roadmap for the Forge platform.

**Project Status:** Pre-Alpha

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
| Pre-Alpha | 🚧 In Progress |
| Alpha | Planned |
| Beta | Planned |
| RC | Planned |
| Stable 1.0 | Planned |

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

Planned

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
top-level commands are `forge version`, `forge doctor`, `forge config`, and
`forge build`, plus the explicit signed `forge build-runnable` producer and
explicit trusted `forge run` execution workflow.

Status:

Planned

---

| Phase 3 — Manifest Engine | ✅ Foundation Complete |

Objective:

Implement manifest parsing.

Capabilities:

- YAML Loader
- JSON Loader
- Schema Validation
- Manifest Resolution

Status:

Planned

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

Planned

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

Planned

---

# Phase 6 — Compiler

Objective:

Compile manifests into executable runtime bundles.

Capabilities:

- Packaging
- Optimization
- Artifact Generation

Status:

Planned

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

Planned

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

Planned

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

Planned

---

## Milestone 3

CLI

Status:

Planned

---

## Milestone 4

Manifest Engine

Status:

Planned

---

## Milestone 5

Validation Engine

Status:

Planned

---

## Milestone 6

Registry

Status:

Planned

---

## Milestone 7

Compiler

Status:

Planned

---

## Milestone 8

Runtime

Status:

Planned

---

## Milestone 9

AI Runtime

Status:

Planned

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
| Phase 1 — Core Foundation | 🔄 Substantially Complete |
| Phase 2 — CLI | 🔄 Foundation Complete / In Progress |
| Phase 3 — Manifest Engine | ✅ Complete |
| Phase 4 — Validation Engine | 🔄 Partial Foundation in Manifest Layer / Dedicated Validation Engine Not Started |
| Phase 5 — Registry | 🔄 Foundation Complete / In Progress |
| Phase 6 — Compiler | 🔄 Foundation Complete / Package Pipeline Hardening In Progress |
| Phase 7 — Runtime | 🔄 Foundation Complete / In Progress |
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
  SHA-256 verification, tamper rejection, cleanup, source immutability, and no
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
- YAML manifest loading
- JSON manifest loading
- Manifest validation
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
- Declaration-order admission with dependency-first execution
- Predictable admission failures without persistent candidate mutation
- Accepted admission state retained after downstream failures
- Shared-application repeated deterministic builds

## Phase 5 — Registry Foundation

Implemented foundation:

- Concurrency-safe in-memory package identity registry.
- Exact package name/version resolution.
- Package source registry.
- Strict source registration.
- Idempotent source `Ensure` with explicit source-conflict behavior.
- Failure-atomic package batch `EnsureAll`.
- Failure-atomic source batch `EnsureAll`.
- Deterministic successful insertion order.
- Deterministic dependency graph/order integration.

Remaining work:

- Remote registry and remote resolution.
- Package acquisition.
- Advanced version constraints.
- Registry persistence.
- Remote and distributed transaction semantics.

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
  cleanup, and source-immutability proof.
- Bounded version-aware package reads with validated version and signer evidence,
  same-handle archive-size validation, bounded actual reads, overflow-safe total
  accounting, fixed Alpha limits, and Store-only runtime policy.
- Strict verified runtime package loading with mandatory integrity and trusted
  signatures, v1 non-runnable classification, v2 Alpha one-artifact validation,
  exact host GOOS/GOARCH authorization, detached bytes, source immutability, and
  no extraction or execution.
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

Remaining work:

- Secondary-document strict JSON decoding.
- Legacy inspection and migration tooling.
- Compatibility beyond the explicitly supported `(1,1)` and `(2,2)` pairs if
  required.
- Cross-toolchain golden archive validation.
- Stronger same-user path-to-Start TOCTOU mitigation.
- Process-tree/descendant lifecycle, graceful termination, and platform policy
  for Windows Job Objects or Unix process groups if selected.
- Richer argument, environment, and working-directory contracts if selected,
  plus process memory/CPU limits, sandboxing, and other execution controls.
- Trust snapshot/revocation and start-time authorization policy.
- Dependency provenance and SBOM support for runnable packages.
- Build isolation and cross-toolchain reproducibility hardening.
- Compiler optimization.
- Remote registry negotiation and package acquisition.
- Strict cross-registry transaction visibility.
- Process-crash recovery between registry commits.
- Concurrent full-build isolation.
- Same-output-path concurrency coordination.

## Evidence-Based Current Roadmap Position

```text
Pre-Alpha
→ Phase 6 — Compiler
→ Package Pipeline Hardening
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
→ Same-User Package/Materialization Path-to-Start TOCTOU Hardening Review: Next
```

Completed checkpoints: **Manifest Admission Hardening**, **Package Format
Stabilization**, **Runnable Package Contract R1A**, **Real Executable Output
R1B**, **Verified Runtime Package Loader R2A**, **Secure Executable
Materialization R2B**, **Process Runner**, and **Manifest Application
Entrypoint**, **User-Facing Runnable Workflow**, **forge run Architecture
Review**, **Trusted Run Support Primitives**, **Explicit Trusted forge run
Command**, and **forge run Formal Closure**.

`forge run` is completed for its narrow Pre-Alpha boundary: existing local
signed runnable package v2, one explicit command-local trusted Ed25519 public
key and KeyID, strict runtime verification, host-only authorization, secure
materialization, one direct child, Wait/reap, bounded output, cleanup, and exact
child/cancellation exit mapping. Phase 6 remains in progress.

The frozen grammar is `forge run <package.zip> --trusted-key
<public-key.pem> --key-id <key-id>`. It accepts one local regular, non-symlink
lowercase `.zip` package, performs no manifest/source/build or remote acquisition,
and creates one invocation-local TrustStore. The key is an explicit X.509 PKIX
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

The next architecture focus is **stronger same-user package/materialization
path-to-Start TOCTOU hardening review**. This is a review target, not an
approved implementation or an expansion to arguments, environment, or stdin.

## Remaining Platform Work

The following capabilities remain future work:

- Dedicated Validation Engine
- Manifest decoder strictness, JSON duplicate-key handling, and explicit
  manifest schema versioning
- Remote package registry and package acquisition
- Package-format production hardening and legacy tooling
- Compiler optimization, build isolation, process resource controls, dependency
  provenance, and reproducibility hardening
- Stronger materialization-to-start TOCTOU mitigation
- Process-tree/descendant lifecycle, graceful shutdown, and optional Windows Job
  Object or Unix process-group policy
- Richer arguments/environment/working-directory contracts, process resource
  controls, and sandboxing
- Trust snapshot/revocation and start-time authorization policy
- Scheduler
- Remote package resolution
- Advanced dependency and version resolution
- AI Runtime

# Long-Term Goal

Forge 1.0 will provide a complete AI-native development platform that enables developers to build modern applications through declarative manifests, modular runtimes, and intelligent automation.
