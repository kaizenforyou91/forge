# Changelog

# Unreleased

## Added

- Added a dependency-aware compiler execution pipeline.
- Added CLI `forge build` for YAML and JSON manifests.
- Added a package source registry.
- Added deterministic artifact bundle serialization and ZIP packaging.
- Added package integrity metadata and verification.
- Added Ed25519 signing, trust-store verification, and package verification policy.
- Added YAML/JSON and multi-module build coverage.
- Added artifact source provenance through package read-back.
- Added idempotent package-source `Ensure` and the explicit
  `ErrPackageSourceConflict` sentinel.
- Added failure-atomic package and package-source batch `EnsureAll` operations.
- Added the prebuilt `BuildPlan` execution pipeline.
- Added pure manifest-admission preflight and a manifest admission coordinator.
- Added CLI manifest-admission integration.
- Added authoritative `package.json` compatibility metadata with package format
  version 1 and artifact bundle schema version 1.
- Added integrity schema version 2 with an exact package metadata digest.
- Added the independently versioned signature schema version 1 contract.
- Added coordinated current-version ZIP writer and reader support.
- Added package format version 2 and artifact bundle schema version 2 for
  runnable application packages.
- Added `RuntimeDescriptor` with the `application_executable` runtime kind,
  logical entrypoint identity, and host target OS/architecture metadata.
- Added a real host Go application executable builder with main-package
  validation and controlled output.
- Added `RunnablePackageCompiler` with private temporary-output ownership and
  exact executable payload packaging.
- Added `PackageReadResult` for validated package/bundle version evidence,
  caller-owned payload copies, and verified signer KeyID evidence.
- Added configurable `PackageReadLimits` and fixed Alpha runtime limits with
  same-handle archive-size checks, bounded entry reads, overflow-safe total
  accounting, and Store-only runtime ingestion.
- Added `VerifiedRunnablePackageLoader` and a detached verified runnable-package
  result with private metadata and copy-returning executable access.
- Added `SecureExecutableMaterializer` for controlled private materialization of
  `VerifiedRunnablePackage` executable bytes.
- Added `MaterializedExecutable` with explicit ownership of whole-directory
  cleanup, concurrency-safe idempotent `Close`, and retry after cleanup failure.

## Changed

- Manifest module `import_path` is now the package source of truth.
- Artifacts and bundle documents preserve import-path provenance.
- Compiler composition is available through application bootstrap.
- CLI build now executes a prepared `BuildPlan`.
- Predictable manifest-admission failures occur before persistent candidate
  mutation.
- Accepted registry state remains after executor or packaging failure.
- Repeated builds with a shared application are idempotent and byte-deterministic.
- Integrity now binds the exact stored `package.json` bytes.
- The normal reader now rejects legacy unversioned packages without silent
  migration or `ImportPath` inference.
- Unsupported package and bundle versions now fail closed.
- Package compatibility and tamper behavior are now explicit, with reader error
  precedence hardened at security boundaries.
- Package Format Stabilization is now a completed technical checkpoint within
  ongoing Phase 6 package pipeline hardening.
- The package reader now explicitly dispatches the supported `(1,1)` and
  `(2,2)` package-format/bundle-schema pairs.
- Integrity validation is schema-aware internally while integrity schema 2 and
  signature schema 1 remain unchanged.
- Runnable packages preserve exact host executable bytes in their single
  entrypoint artifact.
- `forge build` intentionally remains on package format 1 with
  identity/provenance payloads.
- Runtime loading now always requires integrity, a trusted Ed25519 signature,
  and Store-only archives under the fixed Alpha package-read limits.
- Runtime package reads enforce ceilings of 80 MiB per archive, 16 entries,
  1 MiB per document, 64 MiB per artifact, and 72 MiB total uncompressed data.
- Strict runtime loading classifies validated v1 packages as non-runnable and
  requires runnable v2 packages to match the host GOOS/GOARCH exactly.
- Verified executable bytes are detached from the source archive, and verified
  signer KeyID evidence is retained without extraction or execution.
- Runtime executables now use fresh private `forge-runtime-*` directories and
  internally controlled `application.exe` or `application` filenames.
- Materialized executable creation is exclusive. On platforms where file modes
  are meaningful, the file starts in owner-only non-executable mode and gains
  owner execute permission only after the complete write.
- Materialization now performs `file.Sync`, same-open-handle SHA-256 validation,
  exact-size and regular/non-symlink checks, and open-handle/path identity
  validation before handoff.
- The materialized result exposes no general executable path, and cleanup owns
  removal of the complete private runtime directory.
- Secure executable materialization is complete as a technical checkpoint;
  Process Runner Architecture is the next runtime boundary, while Phase 6
  remains in progress.

## Tests

- Added multi-module dependency-first build coverage.
- Added invalid dependency rejection coverage.
- Added integrity, signature, trust-store, and verification-policy coverage.
- Added provenance serialization and package read-back coverage.
- Added direct `ToolchainExecutor` and Engine-to-Artifact provenance tests.
- Added import-path-only tampering coverage.
- Added concurrent batch registry tests.
- Added repeated-build and deterministic-output coverage.
- Added admission zero-mutation and live conflict revalidation tests.
- Added executor and package failure-state tests.
- Added package metadata duplicate-key and integrity v2 coverage.
- Added version stripping, downgrade, and unsupported-version coverage.
- Added metadata, bundle, payload, and regenerated-integrity tamper coverage.
- Added no-integrity and strict signature policy coverage.
- Added deterministic current-format writer/reader round-trip coverage.
- Added real Go executable build and package-v2 real-binary round-trip coverage.
- Added independent executable SHA-256 verification and signed strict-reader
  verification.
- Added executable payload and runtime metadata tamper rejection coverage.
- Added temporary-output cleanup and source-fixture immutability coverage.
- Added a v1 CLI regression proving `forge build` remains an identity package.
- Added bounded archive, entry-count, document, artifact, total-uncompressed,
  actual-read, overflow, and Store-only package-read coverage.
- Added trusted signed v2 runtime loading and trusted signed v1 non-runnable
  classification coverage.
- Added missing, untrusted, and invalid signature coverage plus executable
  payload and runtime-metadata tamper detection at the runtime-loader boundary.
- Added host-platform mismatch, multi-artifact rejection, executable-byte copy
  isolation, source-package immutability, and no-extraction coverage.
- Added a real executable end-to-end load proof through the public runnable
  compiler and strict verified runtime loader without executing the payload.
- Added the materializer lifecycle matrix covering input and host validation,
  partial and zero-progress writes, file type/size/digest/identity failures,
  cleanup retry, concurrent `Close`, and independent materializations.
- Added real source-to-signed-package-to-strict-load-to-materialization proof
  with exact file-byte and independent SHA-256 equality.
- Added source-package independence, source-fixture immutability, complete
  private-directory cleanup, and no-process-execution coverage.

## Known Limitations

- Legacy package inspection and migration tooling are not implemented.
- Only the explicit package-format/bundle-schema pairs `(1,1)` and `(2,2)` are
  supported; broader compatibility and migration tooling do not exist.
- The v1 bundle codec and integrity/signature document decoders remain
  permissive toward unknown and duplicate fields.
- `forge build` still emits package format 1, and there is no user-facing
  runnable package build workflow.
- Manifest metadata has no durable application-entrypoint contract.
- Process start, a process runner, atomic materialization-to-start leasing,
  arguments/environment/working-directory policy, process lifecycle, and
  `forge run` are not implemented.
- Runnable packages do not serialize dependency provenance or an SBOM.
- Executable builds inherit part of the host build environment and are not
  guaranteed reproducible across toolchains.
- Runtime package ingestion has fixed Alpha byte and entry ceilings. Process
  memory/CPU controls and runtime sandboxing are not implemented.
- Advanced Windows ACL/reparse hardening for materialized executables is not
  complete.
- Binary-header validation and trust snapshot/revocation epoch semantics are not
  implemented.
- Remote package registry is not implemented.
- No-integrity mode provides structural validation without cryptographic tamper
  resistance.
- Unsigned packages do not provide publisher authenticity.
- The production package format remains Pre-Alpha.
- Strict cross-registry atomicity is not guaranteed.
- Process-crash atomicity is not guaranteed.
- Full concurrent-build isolation is not guaranteed.
- Same-output-path concurrency remains unresolved.
- Forge remains Pre-Alpha and is not production-ready.

## FW-030 — Manifest Engine Foundation

- Added manifest contract.
- Added YAML manifest loader.
- Added JSON manifest loader.
- Added manifest validation.
- Added exact module resolution.
- Added end-to-end YAML and JSON manifest pipeline coverage.
- Added race, vet, build, and regression validation.

## FW-029 — Plugin System Foundation

- Added plugin contract.
- Added plugin registry.
- Integrated plugin registry with application lifecycle.
- Added plugin configuration enablement.
- Verified plugin integration with the application container.

## FW-028 — CLI Foundation Completion

- Completed CLI command construction.
- Added configuration path handling.
- Added configuration validation, show, init, watch, and doctor flows.
- Added CLI regression tests.
- Added configuration watcher cancellation hardening.

## FW-027 — Runtime Engine Foundation

- Added explicit application runtime states.
- Added deterministic lifecycle transitions.
- Added restart semantics.
- Added startup rollback.
- Added shutdown error aggregation.
- Added deterministic lifecycle concurrency tests.

### Architecture

- Added Forge Architecture Freeze v1.0.
- Added ADR-001 documenting the Core Runtime architecture baseline.
- Established dependency direction and package boundary rules.
- Established architecture change and ADR policies.

All notable changes to this project will be documented in this file.

The format is based on **Keep a Changelog**.

This project follows **Semantic Versioning**.

---

## [Unreleased]

### Remaining Platform Work

- Dedicated Validation Engine
- Remote package registry, acquisition, and resolution
- Advanced dependency and version constraints
- Compiler package-format production hardening, legacy tooling, and optimization
- Durable manifest application-entrypoint metadata and a user-facing runnable
  build workflow
- Build isolation, process resource controls, dependency provenance, and
  reproducibility hardening
- Atomic materialized-executable acquisition/lease and materialization-to-start
  TOCTOU minimization
- Process runner, start-failure handling, arguments/environment/working-directory
  policy, cancellation/shutdown, stdout/stderr policy, exit propagation,
  supervision, and `forge run`
- Binary-header validation and trust snapshot/revocation policy
- Runtime scheduler
- AI Runtime

---

## [0.1.0] - Pre-Alpha

### Added

#### FW-001 — Workspace Bootstrap

- Initialized Git repository.
- Configured Go module.
- Created project directory structure.
- Added development workspace.
- Configured GitHub repository.

---

#### FW-002.1 — Professional README

Added:

- Professional project overview.
- Vision statement.
- Repository structure.
- Development roadmap summary.
- Engineering workflow.
- Getting Started guide.

---

#### FW-002.2 — Professional ROADMAP

Added:

- Engineering roadmap.
- Development phases.
- Core milestones.
- Long-term vision.
- Release planning.

---

#### FW-002.3 — CONTRIBUTING Guide

Added:

- Contribution guidelines.
- Branch strategy.
- Commit convention.
- Pull request checklist.
- Code style principles.
- Engineering workflow.

---

## Upcoming Milestones

### FW-002.4

- Professional CHANGELOG

### FW-002.5

- SECURITY Policy

### FW-002.6

- ADR-001 Repository Layout

### FW-003

- Engineering Automation

### FW-004

- Foundation Library

### FW-005

- CLI Bootstrap

### FW-006

- Manifest Engine

### FW-007

- Validation Engine

### FW-008

- Registry

### FW-009

- Compiler

### FW-010

- Runtime

### FW-011

- AI Runtime

---

## Versioning Strategy

Forge follows Semantic Versioning.

Version format:

MAJOR.MINOR.PATCH

Example:

- 0.1.0 — Pre-Alpha
- 0.2.0 — Alpha
- 0.5.0 — Beta
- 0.9.0 — Release Candidate
- 1.0.0 — Stable Release

---

## Release Philosophy

Every release must satisfy the following requirements:

- Documentation updated
- Tests passing
- Code reviewed
- Changelog updated
- Roadmap synchronized
- Repository tagged

---

## Notes

Forge is currently in the **Pre-Alpha** stage.

The engineering foundation has been established, and active development continues toward the first Alpha release.

# Changelog

## v0.2.0-core-runtime

### Added

- Dependency Injection Container
- Constructor Injection
- Generic Resolve
- Auto Wiring
- Recursive Dependency Resolution
- Module System
- Application Host
- Runtime Context
- Startup Pipeline
- Shutdown Pipeline
- Fluent Builder API

### Internal

- Benchmark
- Freeze Container
- Dependency Graph
- Cycle Detection
