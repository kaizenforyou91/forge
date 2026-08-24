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
`forge build`.

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
- Artifact source provenance
- CLI `forge build` for YAML and JSON manifests
- Multi-module dependency-aware builds
- Prebuilt `BuildPlan` compilation and packaging
- Pure manifest-admission preflight and controlled admission
- Snapshot-based dependency/source analysis and commit-time source revalidation
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
- Artifact source provenance.
- CLI build vertical slice for YAML, JSON, and multi-module dependency-aware builds.
- Pure manifest-admission preflight and controlled manifest admission.
- CLI admission and prepared-plan execution.
- No persistent candidate mutation for predictable admission failures.
- Accepted registration remains committed after downstream execution or package
  failure.

Remaining work:

- Secondary-document strict JSON decoding.
- Legacy inspection and migration tooling.
- Compatibility beyond the explicitly supported `(1,1)` and `(2,2)` pairs if
  required.
- Cross-toolchain golden archive validation.
- A durable manifest application-entrypoint contract.
- A user-facing runnable build workflow.
- R2B secure executable materialization: private temporary-directory lifecycle,
  controlled filename, safe write/close and fsync decision, executable
  permissions, post-write validation, cleanup ownership, and the
  materialization-to-process handoff.
- Process runner, lifecycle, cancellation/shutdown, exit propagation,
  supervision, and `forge run`.
- Process memory/CPU limits, sandboxing, and other execution resource controls.
- Binary-header validation and trust snapshot/revocation policy.
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
→ Secure Executable Materialization R2B: Next
```

Completed checkpoints: **Manifest Admission Hardening**, **Package Format
Stabilization**, **Runnable Package Contract R1A**, **Real Executable Output
R1B**, and **Verified Runtime Package Loader R2A**.

The next runtime boundary is **R2B Secure Executable Materialization**. A durable
manifest application-entrypoint contract remains required before a user-facing
runnable build workflow. Executable materialization, application execution, and
a user-facing `forge run` command are not implemented.

## Remaining Platform Work

The following capabilities remain future work:

- Dedicated Validation Engine
- Durable manifest application-entrypoint metadata
- User-facing runnable package build workflow
- Remote package registry and package acquisition
- Package-format production hardening and legacy tooling
- Compiler optimization, build isolation, process resource controls, dependency
  provenance, and reproducibility hardening
- Secure executable materialization with private filesystem lifecycle,
  permissions, post-write validation, cleanup, and process handoff
- Process runner, cancellation/shutdown, supervision, exit propagation, and
  `forge run`
- Binary-header validation and trust snapshot/revocation policy
- Scheduler
- Remote package resolution
- Advanced dependency and version resolution
- AI Runtime

# Long-Term Goal

Forge 1.0 will provide a complete AI-native development platform that enables developers to build modern applications through declarative manifests, modular runtimes, and intelligent automation.
