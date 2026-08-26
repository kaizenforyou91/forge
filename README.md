# Forge

> **Build AI-Native Applications with Structured Manifests**

![Status](https://img.shields.io/badge/status-pre--alpha-orange)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-Apache--2.0-blue)

Forge is an AI-native application platform written in Go. It is designed to build, validate, package, and run applications using a manifest-driven architecture.

The long-term vision of Forge is to become a modern application platform where applications are described declaratively, validated automatically, and executed through a modular runtime.

> **Current Status:** Pre-Alpha (Under Active Development)

---

# Vision

Forge aims to simplify application development by treating application manifests as the primary source of truth.

Instead of writing large amounts of configuration and boilerplate code, developers define applications declaratively and allow Forge to:

- Parse
- Validate
- Build
- Execute
- Extend

through a unified runtime.

---

# Core Principles

Forge is built around several engineering principles.

- Manifest First
- Modular Architecture
- AI Native
- Strong Validation
- Extensible Runtime
- Developer Friendly
- Open Source

---

# Platform Status

Forge is being developed incrementally. A status of "foundation complete" means
that a tested implementation exists; it does not mean the component is complete
or production-stable.

| Component | Status |
|-----------|--------|
| Workspace | ✅ Bootstrap |
| Documentation | 🚧 In Progress |
| CLI | 🚧 Foundation Complete / In Progress |
| Manifest Engine | ✅ Complete |
| Validation Engine | 🚧 Partial Manifest-Layer Foundation |
| Registry | 🚧 Foundation Complete / In Progress |
| Compiler | 🚧 Foundation Complete / Package Pipeline Hardening |
| Runtime | 🚧 Foundation Complete / In Progress |
| Plugin System | 🚧 Foundation Complete / In Progress |
| AI Runtime | ⏳ Planned |

---

# Current Capabilities

The current tested foundation includes:

- Manifest-driven application architecture.
- YAML and JSON manifest loading.
- Exact module and version resolution.
- Dependency-aware deterministic build planning.
- Multi-module, dependency-first builds.
- Import-path-aware compilation.
- An optional manifest application entrypoint selects exactly one declared
  module and version without making `BuildPlan` ordering authoritative.
- CLI builds through `forge build <manifest>`.
- Explicit reader dispatch for package format/bundle schema pairs `(1,1)` and
  `(2,2)`.
- `forge build` remains on package format v1 and bundle schema v1, with
  identity/provenance artifact payloads.
- Package format v2 and bundle schema v2 define runnable application packages
  with a `RuntimeDescriptor`, the `application_executable` runtime kind, a
  logical entrypoint, and host target OS/architecture metadata.
- A separate runnable compiler path builds a real host Go application
  executable and packages its exact bytes as the single v2 entrypoint artifact.
- Deterministic versioned ZIP packages with integrity schema v2 and optional
  signature schema v1.
- Exact-byte integrity binding for package metadata, bundles, and artifact
  payloads.
- Verified read-back of real executable package payloads; optional trusted
  signatures authenticate their integrity transitively.
- Strict trust-store-backed signature verification and configurable verification
  policy.
- Bounded, version-aware package reads expose validated package and bundle
  versions, copied payloads, and a signer KeyID only after successful trusted
  verification.
- The strict runtime loader always requires integrity, a trusted signature,
  Store-only ZIP entries, and the fixed Alpha read limits: 80 MiB per archive,
  16 entries, 1 MiB per document, 64 MiB per artifact, and 72 MiB total
  uncompressed data.
- General inspection can read v1 packages, but the runtime loader classifies
  them as non-runnable. Runnable v2 packages must satisfy the Alpha
  one-artifact contract and exactly match the host OS and architecture.
- Successful runtime loading returns detached verified executable bytes and
  verified signer identity without extracting files or executing the payload.
- `SecureExecutableMaterializer` accepts only a `VerifiedRunnablePackage` and
  writes its detached bytes into a fresh private `forge-runtime-*` directory
  with the internally controlled name `application.exe` on Windows or
  `application` elsewhere.
- Materialized files are created exclusively with an initial owner-only,
  non-executable write mode where meaningful. Owner execute permission is
  applied only after the complete write, followed by `file.Sync`, exact-size
  and same-handle SHA-256 validation, and file/path identity checks.
- `MaterializedExecutable.Close` owns removal of the complete private directory;
  cleanup is concurrency-safe, idempotent after success, and retryable after a
  failure. The production API intentionally exposes no general executable path.
- `ProcessRunner` consumes a package-private, atomic, single-use execution lease
  from `MaterializedExecutable`; the controlled executable path remains private
  and `Close` cannot remove it while the lease is active.
- Before direct no-shell start, the runner revalidates the host target, exact
  size and SHA-256, regular/non-symlink file identity, and the host PE, ELF, or
  Mach-O executable family and architecture.
- The initial child contract uses zero arguments, a controlled private working
  directory, a reduced explicit environment, non-interactive stdin, and separate
  stdout/stderr capture bounded to 1 MiB per stream while excess output is
  drained.
- One background waiter owns process reap and lease release. Normal non-zero
  exits remain process results; context cancellation and immediate manual
  direct-child termination retain distinct result evidence, and pending cleanup
  runs after reap when `Close` was requested.
- The real production path is proven end to end from a trusted signed package
  through strict loading, secure materialization, direct child execution,
  deterministic result capture, and explicit cleanup.
- Normal-reader rejection of legacy/unversioned packages and unsupported package
  or bundle versions.
- Artifact source provenance preserved through package read-back.
- Strict package-source registration with idempotent `Ensure` and explicit
  source-binding conflict rejection.
- Failure-atomic package and package-source batch registration through
  `EnsureAll`.
- Shared-application repeated builds with byte-deterministic package output.
- Prepared `BuildPlan` execution through `CompileAndPackagePlan`.
- Pure manifest-admission preflight with snapshot-based package and source
  conflict analysis.
- Commit-time live source conflict revalidation and declaration-order admission.
- Dependency-first execution after successful admission.
- No persistent candidate registration after predictable admission failures:
  invalid manifests, missing import paths, source conflicts, missing
  dependencies, or dependency cycles.
- Accepted admission state remains committed after executor or package-output
  failure.
- Configuration, structured logging, dependency injection, application lifecycle,
  HTTP, middleware, and plugin foundations.

The implemented top-level CLI commands are `forge version`, `forge doctor`,
`forge config`, and `forge build`.

Forge remains **Pre-Alpha**. The compiler and package pipeline are a tested
foundation, not a production-ready package ecosystem or stable production
format.

Manifest Admission Hardening, Package Format Stabilization, Runnable Package
Contract R1A, Real Executable Output R1B, Verified Runtime Package Loader R2A,
Secure Executable Materialization R2B, the direct-child Process Runner, and the
Manifest Application Entrypoint are implemented and validated technical
checkpoints. Phase 6 and the Compiler remain in progress; User-Facing Runnable
Workflow Architecture is next.

## Manifest Runnable Contract

Manifests may optionally declare one application entrypoint:

```yaml
entrypoint:
  module: app
  version: v1
```

Generic and library manifests do not require this field. When present, it must
identify one exact declared module and version. The selected module and its
admitted package-source evidence continue to own the canonical `ImportPath`;
the entrypoint itself contains no source path, and `BuildPlan` order never
infers it.

The programmatic runnable-manifest path is:

```text
manifest entrypoint
→ immutable admission identity and normalized source snapshot
→ admission-bound RuntimeEntrypoint and source resolver
→ signed runnable package v2
```

`PrepareManifestAdmission` success provides immutable build evidence.
`AdmitManifest` additionally publishes package/source candidates into shared
registries and performs live conflict checks. `RunnableManifestCompiler`
derives its selected source only from the admission snapshot; callers cannot
inject a resolver or raw source path. Consequently, the admitted canonical
`ImportPath` is the one passed to the executable builder and recorded in the
artifact.

The manifest declaration is build intent, not runtime trust authorization.
Runtime authority still begins with a signed package and proceeds through
strict trusted verification, host authorization, secure materialization,
executable-header validation, and `ProcessRunner` controls.

This is currently a programmatic compiler capability. `forge build` remains the
package-format-v1 identity/provenance workflow even when an entrypoint is
present. No user-facing runnable-build command or `forge run` exists.

## Known Limitations

- `forge build` still emits package format v1 identity/provenance packages; no
  user-facing runnable-package build command exists.
- Manifest loading does not provide a strict unknown-field contract, JSON
  duplicate keys are not rejected, and there is no separate manifest
  `schema_version` field.
- Process execution is limited to one direct child with zero arguments, a fixed
  reduced environment, and a controlled working directory. Descendants are not
  managed; graceful shutdown, arbitrary arguments/environment/working-directory
  policy, process-tree supervision, and `forge run` are not implemented.
- Runnable package v2 currently contains one entrypoint artifact and does not
  serialize dependency provenance or an SBOM.
- Runtime package ingestion has fixed Alpha byte and entry ceilings. Process
  memory/CPU controls and runtime sandboxing are not implemented.
- Advanced Windows ACL/reparse hardening for materialized executables is not
  complete.
- Host PE/ELF/Mach-O family and architecture validation is implemented, but it
  is not malware analysis. Trust snapshot/revocation epoch semantics and
  start-time trust reauthorization are not implemented.
- The same-user path-to-Start replacement window is not eliminated, and there
  are no CPU, memory, process-count, filesystem, network, syscall, or privilege-
  dropping sandbox controls.
- Executable builds partly inherit the host build environment and are not
  guaranteed to be reproducible across toolchains.
- Admission freezes the canonical source `ImportPath`, not source repository
  contents, commit/digest provenance, or an SBOM.
- Legacy packages are rejected by the normal reader; legacy inspection and
  migration tooling are not implemented.
- The reader supports only the explicit `(1,1)` and `(2,2)` package/bundle
  version pairs; broader multi-version compatibility is not implemented.
- Bundle schema v1 and the integrity/signature JSON decoders retain permissive
  unknown- or duplicate-field behavior in some paths.
- Remote package registry, resolution, and package acquisition are not implemented.
- Strict atomic visibility across the package and package-source registries is
  not guaranteed.
- Process-crash atomicity between source and package commits is not guaranteed.
- Full concurrent-build isolation is not implemented.
- Concurrent builds targeting the same output path are not coordinated.
- No-integrity mode provides structural validation only and no cryptographic
  tamper resistance.
- Unsigned integrity proves consistency, not publisher authenticity.
- The package format remains Pre-Alpha and is not production-stable.

---

# Repository Structure

```text
forge/

├── cmd/
├── internal/
├── pkg/
├── runtime/
├── schemas/
├── docs/
├── examples/
├── scripts/
├── tools/
├── assets/
└── test/
```

---

# Development Roadmap

## Phase 1

- Workspace Bootstrap
- Documentation
- Engineering Standards

## Phase 2

- CLI
- Foundation Library
- Manifest Engine

## Phase 3

- Validation Engine
- Registry
- Runtime

## Phase 4

- Compiler
- Plugin System
- AI Runtime

---

# Getting Started

Clone the repository.

```bash
git clone https://github.com/kaizenforyou91/forge.git
```

Enter the project.

```bash
cd forge
```

Initialize dependencies.

```bash
go mod tidy
```

Run tests.

```bash
go test ./...
```

---

# Documentation

Project documentation is located inside:

```text
docs/
```

This directory contains:

- Architecture
- Specifications
- ADR
- API Documentation
- Roadmap

---

# Engineering Workflow

Every change follows the engineering workflow below.

```text
Specification
        │
        ▼
Implementation
        │
        ▼
Review
        │
        ▼
Test
        │
        ▼
Commit
        │
        ▼
Release
```

---

# Contributing

Forge is currently under active development.

Contribution guidelines will be published after the first alpha release.

---

# License

Apache License 2.0

---

# Current Development Stage

```text
Core Foundation
████████████████████ 100%

CLI Foundation
████████████████████ 100%

Runtime Foundation
████████████████████ 100%

Plugin Foundation
████████████████████ 100%

Manifest Engine
████████████████████ 100%

Validation Engine
Partial foundation in the manifest layer

Package Registry
Foundation complete / in progress

Compiler
Foundation complete / package pipeline hardening in progress

AI Runtime
░░░░░░░░░░░░░░░░░░░░ 0%
```
> Progress percentages represent the completed foundation scope for each
> engineering area. They do not imply that the entire long-term platform
> capability has been completed.

---

## Project Status

Forge is currently in the **Pre-Alpha** stage.

The core engineering foundation is now substantially established. Package Format
Stabilization is a completed technical checkpoint within ongoing package pipeline
hardening, while registry, validation, and runtime capabilities continue to
evolve.

AI-First Engineering Operating System
