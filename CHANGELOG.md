# Changelog

All notable changes to Forge are documented in this file. The format follows
Keep a Changelog, and release identities follow Semantic Versioning.

## Unreleased

No changes recorded for a subsequent release.

## 0.3.0-alpha.1 - 2026-09-05

This section summarizes the complete First Alpha baseline accumulated across
the preceding engineering checkpoints; it does not imply that every feature
was introduced by one recent commit. Forge entered First Alpha as a
non-production technical preview with pre-stable APIs and package formats.

### Added

- Added strict YAML and JSON manifest document admission with valid UTF-8,
  single-document/object, exact field-type, recursive duplicate-field, and
  unknown-field enforcement plus rejection of ambiguous YAML features.
- Added `forge validate <manifest> [--profile structural|build|runnable]` for
  non-mutating manifest and compiler-admission checks.
- Added bounded verified `forge inspect <package.zip>` for v1/v2 package,
  runtime, artifact, integrity, and signature-state evidence without execution.
- Added optional explicit `--trusted-key` and `--key-id` inspection trust for
  invocation-local Ed25519 signer verification.
- Added the canonical external Alpha example and documented validation,
  v1-build, v2-build, inspection, explicit-trust, and execution workflow.
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
- Added a package-private atomic single-use executable lease with Close/start
  lifecycle coordination and no public executable path.
- Added `ProcessRunner`, `RunningProcess`, and `ProcessResult` for direct-child
  start, background Wait/reap, and stable exit evidence.
- Added immediate manual direct-child termination and distinct cancellation and
  termination result evidence.
- Added host PE, ELF, and Mach-O executable-family and architecture validation.
- Added the optional `ApplicationEntrypoint` manifest contract with exact
  declared module/version identity and no duplicated source path.
- Added immutable application-entrypoint and normalized `PackageSource`
  evidence to `ManifestAdmissionPlan` with copy-returning accessors.
- Added `RunnableManifestCompiler` and `RunnableManifestRequest` for
  programmatic admitted-manifest composition into runnable package v2.
- Added a private immutable one-source resolver that binds manifest-driven
  compilation to the selected source in the admission snapshot.
- Added the explicit `forge build-runnable <manifest>` command with required
  `--signing-key` and `--key-id` flags plus optional `--output`.
- Added strict unencrypted PKCS#8 PEM Ed25519 private-key loading with a 16 KiB
  ceiling, regular/non-symlink identity checks, and owner-only Unix permissions.
- Added private same-filesystem runnable-package staging and atomic no-replace
  hard-link publication.
- Added strict bounded staged-package verification using the Alpha package-read
  limits before final publication.
- Added `forge run <package.zip> --trusted-key <public-key.pem> --key-id
  <key-id>` for explicit execution of existing local signed runnable packages
  v2; both trust flags are required.
- Added X.509 PKIX `PUBLIC KEY` PEM Ed25519 trust input with a 16 KiB ceiling
  and exact explicit KeyID.
- Added signal-aware CLI execution, typed child exit propagation, and trusted
  runtime composition through strict loading, host authorization, secure
  materialization, direct-child execution, Wait/reap, bounded output, and
  cleanup.
- Added independent `acceptance (ubuntu-latest)` and
  `acceptance (windows-latest)` CI checks with matrix `fail-fast: false`.
- Added a focused `race (ubuntu-latest)` CI check for `pkg/compiler`, `runtime`,
  and `internal/cli`.

### Changed

- Entered **First Alpha — 0.3.0-alpha.1** with Phase 1, Phase 5, and Phase 7
  Alpha-Bounded Closed and Phase 6 Closed / Pass.

- Accepted Phase 1, Phase 5, and Phase 7 as Alpha-bounded closed scopes while
  retaining their long-term core, distribution, trust, runtime, scheduling,
  isolation, and platform work.
- Shared neutral Alpha package read limits now support both runtime loading and
  safe non-executing inspection without weakening the stricter runtime policy.
- Aligned README, roadmap, release-stamping guidance, and external-user
  workflow documentation with the implemented command and security boundaries.
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
- Package Format Stabilization is a completed technical checkpoint within the
  now-closed bounded Phase 6 compiler/package/runnable pipeline.
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
- Process execution now consumes `MaterializedExecutable` through one
  irreversible execution claim rather than accepting an arbitrary path; Close
  cannot delete an actively leased executable.
- Before direct no-shell start, the runner revalidates target, size, SHA-256,
  regular/non-symlink file identity, and host executable headers.
- The initial child policy uses zero arguments, a reduced explicit environment,
  a controlled private working directory, non-interactive stdin, and separate
  stdout/stderr capture bounded to 1 MiB per stream.
- A background waiter keeps the lease until Wait/reap. Cancellation and manual
  termination remain distinct, and process results survive joined cleanup
  failures.
- The optional manifest entrypoint is explicit build intent; `BuildPlan` order
  is not entrypoint authority, and generic/library manifests remain valid
  without an entrypoint.
- Manifest-driven runnable source authority now derives exclusively from the
  normalized admission snapshot. The coordinator no longer permits a
  caller-selected source resolver or raw source path.
- Missing, duplicate, or noncanonical selected-source evidence is rejected
  before executable construction or packaging. The admitted canonical
  `ImportPath` is passed to the builder and recorded as artifact provenance.
- A successful `PrepareManifestAdmission` result is immutable build authority;
  `AdmitManifest` additionally publishes candidates to shared registries and
  performs live source-conflict checks.
- Manifest entrypoint metadata remains build intent rather than runtime trust
  authorization; runtime authority still begins at strict signed-package
  verification.
- Manifest Application Entrypoint and User-Facing Runnable Workflow are
  complete technical checkpoints. The `forge run` Architecture Review,
  support primitives, explicit trusted command, and formal closure are also
  completed checkpoints.
- `forge build-runnable` uses immutable `PrepareManifestAdmission` evidence and
  requires its admitted application entrypoint without publishing package or
  source candidates into shared registries.
- Runnable CLI source authority remains bound to the normalized admission
  snapshot; no entrypoint, `ImportPath`, or resolver override is accepted.
- Runnable builds are host-target-only and use the process current working
  directory as the build working directory.
- Runnable output defaults to
  `build/<name>-<version>-runnable-<goos>-<goarch>.zip`; custom output requires
  an exact `.zip` path, and no force/overwrite mode exists.
- `forge build` remains package format 1 / bundle schema 1 even when the
  manifest contains an application entrypoint.
- Main dispatch now preserves exact natural child exit codes from 1 through
  255; clean context cancellation maps to 130, while infrastructure failures
  remain exit 1.
- `forge run` buffers stdout and stderr to 1 MiB per stream, presents retained
  output after child completion, warns on truncation, and emits no success
  banner.
- The ZIP package reader now rejects symlink and nonregular package paths,
  resolves pre-open identity, binds it to the accepted open handle with
  `SameFile`, and uses that handle for archive size and all ZIP reads while
  preserving the public path-based reader APIs.
- `forge run` package input now owns lexical/local path policy only; filesystem
  existence and object identity are delegated to the compiler reader. Trust
  validation can therefore precede package filesystem loading.
- One exact KeyID validator now governs `build-runnable`, the signer,
  `PackageSignature`, `TrustStore`, verifier, and `forge run`. KeyIDs must be
  valid nonempty UTF-8 without surrounding Unicode whitespace, ASCII controls
  U+0000 through U+001F, or U+007F; other Unicode remains exact. TrustStore no
  longer trims identifiers, and no case folding or Unicode normalization is
  applied.
- All supported package JSON documents now use one shared strict structural
  decoder: `package.json`, bundle schemas v1 and v2, `integrity.json`, and
  `signature.json` require valid raw UTF-8, one object followed only by JSON
  whitespace, recursive duplicate-key rejection, unknown-field rejection, and
  domain/schema validation afterward.
- Package JSON hardening is a pre-stable compatibility tightening. Canonical v1
  and v2 writer output remains accepted; schema versions, serialized layouts,
  hashing, Ed25519, and exact-byte signature payloads are unchanged, with no
  verification-time canonicalization or reserialization.
- Phase 6 — Compiler / Package Pipeline Hardening is CLOSED / PASS for the
  bounded Pre-Alpha / First Alpha compiler-package-runnable pipeline. This is
  not a Beta, production-readiness, or all-hardening claim.

### Security

- KeyID handling is exact across producers, signatures, trust routing,
  verification, and execution: identifiers are not trimmed, case-folded, or
  Unicode-normalized, and invalid UTF-8, surrounding Unicode whitespace, and
  ASCII control characters are rejected.
- Manifest admission now fails closed on unknown fields, recursive duplicate
  fields, malformed UTF-8, multiple/trailing documents, wrong field types, and
  unsupported YAML alias/anchor/merge/tag behavior.
- Package inspection distinguishes unsigned, self-consistent but explicitly
  untrusted signatures, and signatures verified against a supplied trusted
  key. Signed-unverified packages are not described as authentic or trusted.
- Runnable CLI signing is mandatory and uses an explicit KeyID; there is no
  unsigned mode or private-key input through command arguments or environment.
- Staged runnable packages are signature-checked and semantically verified as
  package format 2 / bundle schema 2 under fixed Alpha read limits before
  publication.
- Existing or concurrently appearing output targets are preserved. Publication
  is atomic and no-replace, with no unsafe fallback when hard links are
  unavailable.
- `build-runnable` still does not execute its output. `forge run` keeps the
  strict runtime loader as verification authority, authorizes the exact host,
  materializes privately without exposing a raw executable path, runs one
  direct child, waits/reaps, and owns cleanup.
- `forge run` executes trusted native code directly. Trust does not imply safe
  code; there is no sandbox, filesystem/network isolation, privilege drop,
  process-tree containment, resource control, or production-safety guarantee.
- Package-selection TOCTOU hardening is closed for the current Alpha boundary:
  directory-entry replacement cannot redirect verification after handle
  acceptance. This is not an immutable snapshot; same-user in-place mutation
  and the materialized validation-handle-close-to-OS-exec race remain separate
  hardening debt.
- Raw invalid UTF-8 in package documents, including `signature.json`, is
  rejected before `encoding/json` can repair malformed bytes. Exact stored
  `integrity.json` bytes remain the Ed25519 verification payload.

### Tests

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
- Added Close/acquisition race, single-use lease, pending cleanup, and cleanup
  retry coverage.
- Added successful direct-child start, zero/non-zero exit, repeated/concurrent
  Wait, controlled cwd/environment/stdin, bounded output, and executable-header
  matrix coverage.
- Added context cancellation, concurrent/manual termination, winner-race,
  cleanup-error/result-preservation, and no-auto-cleanup coverage.
- Added full real trusted package-to-execution proof with signer/entrypoint
  preservation, source-package independence, and no shell or public path API.
- Added YAML/JSON application-entrypoint decoding, optional/null behavior,
  exact-membership, whitespace, validation-precedence, and parity coverage.
- Added `BuildPlan` entrypoint-independence, immutable admission identity,
  normalized source, admission idempotence/conflict, and accessor-copy coverage.
- Added divergent-resolver elimination plus missing, duplicate, and
  noncanonical admitted-source rejection coverage.
- Added prepared- and committed-plan runnable composition coverage with exact
  builder/artifact `ImportPath`, non-main rejection, and source mutation
  isolation.
- Added real signed package-v2 read-back proving exact runtime entrypoint, host
  target, sole artifact provenance, non-empty payload, and signer evidence.
- Added real root/Cobra-to-signed-package-v2 command integration with default,
  relative custom, and absolute custom output coverage.
- Added runnable CLI coverage for signing-key and KeyID failures, missing
  entrypoint precedence, non-main rejection, pre-cancellation, shared-registry
  non-mutation, staging cleanup, and existing-target preservation.
- Added an entrypoint-present regression proving `forge build` remains package
  format 1 / bundle schema 1.
- Added real `forge run` exit-0 and exit-23 execution, OS-observed exit-23
  propagation, deterministic cancellation-to-130, and dual-stream truncation
  coverage.
- Added command-level malformed/wrong key, wrong KeyID, unsigned v2, signed v1,
  tampered package, and non-package input rejection coverage.
- Added deterministic package-open identity-race, post-open replacement,
  symlink/nonregular, and truncation coverage plus command-level missing and
  directory package delegation coverage.
- Corrected Linux runtime reaping tests to use portable `cmd.Wait` completion,
  non-nil `ProcessState`, and direct-child PID evidence instead of treating
  `ProcessState.Exited()` as reap proof for signal-terminated children. No
  production ProcessRunner behavior changed; Ubuntu GitHub Actions is green.
- Added package-document matrices covering object roots, unknown fields,
  recursive duplicates, trailing content, malformed raw UTF-8, canonical
  writer output, v1/v2 round trips, and security error precedence.
- CI acceptance now runs `go mod tidy` plus dependency-file diff, `go list`,
  vet, uncached full tests, and full builds on Ubuntu and Windows. The first
  hosted Windows acceptance passed on Windows Server 2025 with Go 1.26.7 on
  windows/amd64; focused Ubuntu race acceptance also passed.

### Limitations

- Legacy package inspection and migration tooling are not implemented.
- Only the explicit package-format/bundle-schema pairs `(1,1)` and `(2,2)` are
  supported; broader compatibility and migration tooling do not exist.
- `forge build` still emits package format 1; signed runnable package-v2
  creation is the separate explicit `forge build-runnable` workflow.
- Manifest decoding has no separate `schema_version` field; the strict current
  manifest contract remains pre-stable and is not a stable compatibility
  guarantee.
- Process management is host-only and direct-child-only. Descendants, process
  trees, graceful shutdown, arbitrary arguments/environment/working-directory
  policy, live streaming, and resource controls are not implemented.
- Runnable packages do not serialize dependency provenance or an SBOM.
- Admission freezes a canonical source `ImportPath`, not repository contents,
  commit/digest provenance, or reproducible toolchain output.
- Executable builds inherit part of the host build environment and are not
  guaranteed reproducible across toolchains.
- Runtime package ingestion has fixed Alpha byte and entry ceilings. Process
  memory/CPU controls and runtime sandboxing are not implemented.
- Advanced Windows ACL/reparse/share-mode hardening for materialized
  executables is not complete, and some Windows symlink security fixtures
  remain capability-dependent.
- Runnable signing-key files are permission-checked on Unix, but Windows ACL
  validation is not implemented; filesystems without hard-link support cannot
  publish runnable output and fail safely.
- Host executable family/architecture validation is implemented but is not
  malware analysis. Trust snapshot/revocation epochs and start-time trust
  reauthorization are not implemented.
- Open-once package selection does not provide immutable-snapshot protection
  against same-user in-place modification of the accepted file.
- ProcessRunner closes its validation handle before OS pathname execution. The
  architecture review is complete; stronger materialized validation-to-exec
  binding implementation remains deferred technical debt.
- No process resource isolation or filesystem/network/syscall sandbox exists.
- Runtime trust is command-local and limited to one explicit key per
  invocation; persistent trust configuration, multiple configured trusted
  keys, rotation/revocation, and remote package acquisition are absent.
- Cross-platform signal acceptance, package-handle Close failure injection,
  and command-level Start/Wait/Close/output fault-injection seams remain test
  debt.
- Remote package registry is not implemented.
- No-integrity mode provides structural validation without cryptographic tamper
  resistance.
- Unsigned packages do not provide publisher authenticity.
- The package formats remain pre-stable and are not production-stable.
- Strict cross-registry atomicity is not guaranteed.
- Process-crash atomicity is not guaranteed.
- Full concurrent-build isolation is not guaranteed.
- `build-runnable` uses same-filesystem staging and atomic hard-link no-replace
  publication: one concurrent publisher may win, and existing targets are not
  overwritten. Broader generic/full-build coordination and global
  multi-process output ownership remain unresolved.
- Forge First Alpha remains a non-production technical preview.

### Historical checkpoint: FW-030 — Manifest Engine Foundation

- Added manifest contract.
- Added YAML manifest loader.
- Added JSON manifest loader.
- Added manifest validation.
- Added exact module resolution.
- Added end-to-end YAML and JSON manifest pipeline coverage.
- Added race, vet, build, and regression validation.

### Historical checkpoint: FW-029 — Plugin System Foundation

- Added plugin contract.
- Added plugin registry.
- Integrated plugin registry with application lifecycle.
- Added plugin configuration enablement.
- Verified plugin integration with the application container.

### Historical checkpoint: FW-028 — CLI Foundation Completion

- Completed CLI command construction.
- Added configuration path handling.
- Added configuration validation, show, init, watch, and doctor flows.
- Added CLI regression tests.
- Added configuration watcher cancellation hardening.

### Historical checkpoint: FW-027 — Runtime Engine Foundation

- Added explicit application runtime states.
- Added deterministic lifecycle transitions.
- Added restart semantics.
- Added startup rollback.
- Added shutdown error aggregation.
- Added deterministic lifecycle concurrency tests.

#### Architecture

- Added Forge Architecture Freeze v1.0.
- Added ADR-001 documenting the Core Runtime architecture baseline.
- Established dependency direction and package boundary rules.
- Established architecture change and ADR policies.

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

## 0.1.0 - Pre-Alpha

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

### Historical planning snapshot recorded at 0.1.0

#### FW-002.4

- Professional CHANGELOG

#### FW-002.5

- SECURITY Policy

#### FW-002.6

- ADR-001 Repository Layout

#### FW-003

- Engineering Automation

#### FW-004

- Foundation Library

#### FW-005

- CLI Bootstrap

#### FW-006

- Manifest Engine

#### FW-007

- Validation Engine

#### FW-008

- Registry

#### FW-009

- Compiler

#### FW-010

- Runtime

#### FW-011

- AI Runtime

---

### Historical versioning strategy recorded at 0.1.0

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

### Release philosophy

Every release must satisfy the following requirements:

- Documentation updated
- Tests passing
- Code reviewed
- Changelog updated
- Roadmap synchronized
- Repository tagged

---

### Historical note

At the time of 0.1.0, Forge was in the **Pre-Alpha** stage and active
development was progressing toward the first Alpha release.
