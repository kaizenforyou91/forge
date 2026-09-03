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
| Compiler | CLOSED / PASS — Phase 6 bounded Pre-Alpha pipeline |
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
- CLI builds identity/provenance packages through `forge build <manifest>` and
  signed runnable packages through the explicit `forge build-runnable`
  workflow.
- Explicit reader dispatch for package format/bundle schema pairs `(1,1)` and
  `(2,2)`.
- Package format 1 / bundle schema 1 remains the supported readable
  identity/provenance format and is non-runnable. Package format 2 / bundle
  schema 2 is the supported runnable application format. Crossed,
  unsupported, and future pairs fail closed; integrity remains schema 2 and
  signatures remain schema 1.
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
- One strict structural JSON contract covers `package.json`, bundle schema v1,
  bundle schema v2, `integrity.json`, and `signature.json`: raw bytes must be
  valid UTF-8, the root must be one object followed only by JSON whitespace,
  duplicate member names are rejected recursively, unknown fields are
  rejected, and domain/schema validation follows structural decoding.
- Strict package-document decoding does not canonicalize or reserialize
  verification inputs. The exact stored `package.json`, `bundle.json`, artifact,
  and `integrity.json` bytes remain the hashing and signature authority.
- Verified read-back of real executable package payloads; optional trusted
  signatures authenticate their integrity transitively.
- Strict trust-store-backed signature verification and configurable verification
  policy.
- Bounded, version-aware package reads expose validated package and bundle
  versions, copied payloads, and a signer KeyID only after successful trusted
  verification.
- Package selection is bound inside `ZIPPackageReader`: it rejects symlink and
  non-regular paths, resolves the pre-open identity, opens once, requires the
  handle `Stat` to identify the same file, and uses that handle for archive
  sizing and every ZIP read.
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
- Linux tests for signal-terminated children use portable `cmd.Wait` and
  matching `ProcessState` PID evidence rather than treating
  `ProcessState.Exited()` as reap proof; this was a test portability correction,
  not a production ProcessRunner lifecycle change.
- Continuous CI runs independent `acceptance (ubuntu-latest)` and
  `acceptance (windows-latest)` checks plus `race (ubuntu-latest)`. Acceptance
  enforces dependency-file cleanliness, package enumeration, vet, uncached
  full tests, and a full build; the focused race check covers `pkg/compiler`,
  `runtime`, and `internal/cli`.
- The real production path is proven end to end from a trusted signed package
  through strict loading, secure materialization, direct child execution,
  deterministic result capture, and explicit cleanup.
- `forge run` executes an existing local signed runnable package v2 only after
  explicit command-local Ed25519 trust, strict runtime verification, exact host
  authorization, private materialization, direct-child execution, Wait/reap,
  bounded output presentation, and cleanup.
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
`forge config`, `forge build`, `forge build-runnable`, and `forge run`.

Forge remains **Pre-Alpha**. The compiler and package pipeline are a tested
foundation, not a production-ready package ecosystem or stable production
format.

Manifest Admission Hardening, Package Format Stabilization, Runnable Package
Contract R1A, Real Executable Output R1B, Verified Runtime Package Loader R2A,
Secure Executable Materialization R2B, the direct-child Process Runner, and the
Manifest Application Entrypoint and User-Facing Runnable Workflow are
implemented and validated technical checkpoints. **Phase 6 — Compiler /
Package Pipeline Hardening is CLOSED / PASS for the bounded Pre-Alpha / First
Alpha compiler-package-runnable pipeline.** The `forge run` Architecture
Review, trusted-run primitives, explicit command, and formal closure are
completed checkpoints.
The package-selection TOCTOU review, atomic package-open identity binding, CLI
preflight simplification, and formal hardening closure are also complete.

This bounded closure does not make Forge Beta or production-ready, and it does
not complete all future compiler, runtime, trust, provenance, isolation, or
security-hardening work.

## Package Identity and Parsing Contracts

Forge supports exactly these package-format/bundle-schema pairs:

| Package format | Bundle schema | Current contract |
|---|---|---|
| 1 | 1 | Readable identity/provenance package; non-runnable |
| 2 | 2 | Runnable `application_executable` package |

All other pairs fail closed. Integrity schema 2 and signature schema 1 remain
the only supported security-document versions.

The package documents `package.json`, bundle v1, bundle v2, `integrity.json`,
and `signature.json` share one strict decoding contract: valid raw UTF-8, one
top-level JSON object, only trailing JSON whitespace, recursive duplicate-key
rejection, unknown-field rejection, and domain/schema validation afterward.
Valid canonical Forge v1 and v2 writer output remains accepted. Verification
uses the exact ZIP-stored bytes; decoding never reserializes documents for
hashing or signature verification.

One exact KeyID contract applies from `forge build-runnable` through the signer,
`PackageSignature`, `TrustStore`, verifier, and `forge run`. A KeyID must be a
nonempty valid UTF-8 Go string, must have no surrounding Unicode whitespace,
and must contain neither ASCII controls U+0000 through U+001F nor U+007F.
Other Unicode is allowed. Forge does not trim, case-fold, or apply NFC/NFD or
other normalization; trust routing uses exact Go-string identity.

## Continuous Acceptance

The required continuous checks are `acceptance (ubuntu-latest)`,
`acceptance (windows-latest)`, and `race (ubuntu-latest)`. Both acceptance
jobs run `go mod tidy` followed by a `go.mod`/`go.sum` cleanliness diff,
`go list ./...`, `go vet ./...`, `go test ./... -count=1`, and
`go build ./...`. The focused Ubuntu race job runs the compiler, runtime, and
CLI package boundaries under the race detector.

The first hosted Windows acceptance passed on Windows Server 2025 with Go
1.26.7 on windows/amd64. Routine CI output is intentionally non-verbose, so
this does not claim that every capability-dependent Windows symlink fixture
executed rather than skipped.

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

`forge build` remains the package-format-v1 identity/provenance workflow even
when an entrypoint is present. Runnable package creation is an explicit,
separate CLI operation; manifest metadata never switches `forge build` to v2.

## Signed Runnable Build

Create a signed host-target runnable package with:

```text
forge build-runnable <manifest> \
  --signing-key <private-key.pem> \
  --key-id <key-id> \
  [--output <package.zip>]
```

The manifest must declare an application entrypoint. Signing is mandatory: the
key file must be an unencrypted PKCS#8 PEM Ed25519 private key with PEM type
`PRIVATE KEY`, no larger than 16 KiB, and the explicit KeyID must satisfy the
exact package KeyID contract above. The key path must name a regular,
non-symlink file; owner-only permissions are enforced on Unix. Windows ACL
validation is not currently claimed.

The command uses the process current working directory as its Go build working
directory and targets the host `GOOS/GOARCH`; it has no working-directory or
cross-compilation flags. `PrepareManifestAdmission` supplies immutable
entrypoint and normalized source evidence without committing shared registries.

The default output is
`build/<name>-<version>-runnable-<goos>-<goarch>.zip`. A custom `--output`
must end in exactly `.zip`; relative paths are resolved from the process current
working directory. Existing targets are never overwritten and there is no
`--force` mode. Forge stages the package in the final parent directory,
strictly verifies its signature and v2 runtime/artifact metadata under the
fixed Alpha read limits, then publishes it atomically with no-replace hard-link
semantics. An existing target is never overwritten; if concurrent publishers
select the same path, one may win and the others fail safely. Filesystems
without required hard-link support fail safely. Broader generic/full-build
coordination and a global multi-process ownership policy remain deferred.

`build-runnable` creates a package only; it never executes its output. Signing
authenticates the produced package bytes and metadata, but does not prove
reproducible source contents, a repository commit/digest, an SBOM, or that the
signed code is safe.

## Trusted Package Execution

Run an existing local signed runnable package v2 with exactly one explicitly
trusted public key:

```text
forge run <package.zip> \
  --trusted-key <public-key.pem> \
  --key-id <key-id>
```

The CLI accepts one nonblank local path with no surrounding whitespace and the
exact lowercase `.zip` extension. Relative paths resolve from the process
current working directory and become absolute, clean paths internally. Remote
URLs, manifests, source resolution, and implicit builds are not accepted. The
CLI does not inspect package filesystem identity. `forge build` remains the
package-v1 identity/provenance command; `forge build-runnable` remains the
explicit signed package-v2 producer.

Both trust flags are required. The key must be an X.509 PKIX
SubjectPublicKeyInfo encoded as one `PUBLIC KEY` PEM block containing an
Ed25519 public key. The regular, non-symlink file is limited to 16 KiB; because
it is public material, owner-only permissions are not required. Certificates,
private keys, alternate PEM types, and trailing data are rejected. KeyID is
explicit rather than derived from the key or filename and follows the same
exact, non-normalizing contract used by the producer, signature model,
TrustStore, and verifier. Each invocation creates a command-local TrustStore
holding exactly this one KeyID/key pair; no trust is persisted globally or in
config.

```text
local signed package v2
â†’ explicit trusted Ed25519 key and KeyID
â†’ strict VerifiedRunnablePackageLoader
â†’ exact host GOOS/GOARCH authorization
â†’ detached verified executable bytes
â†’ private forge-runtime-* materialization
â†’ direct no-shell child
â†’ Wait/reap
â†’ bounded output presentation
â†’ cleanup
```

The compiler ZIP reader owns filesystem existence/open failures, `Lstat`,
symlink and non-regular rejection, stable identity resolution, open-handle
`Stat`, `SameFile` binding, archive size, same-handle ZIP consumption, and
handle closure. For one read, the regular non-symlink object selected
immediately before open is the object represented by the handle used for
`zip.NewReader`, bounded entry reads, signature and integrity verification,
and detached package construction. The package pathname is not re-resolved
after that handle is accepted.

The strict runtime loader remains the sole authority for signature and
integrity verification, Alpha bounded package reads, runnable-v2 structure,
and host authorization. The CLI creates no second verifier, ZIP reader,
verification policy, or build/source path. After strict verification produces
the detached result, the source ZIP path no longer participates in execution
authority: materialization consumes copied executable bytes only, with no
source path, inode, file index, or complete-ZIP digest retained as runtime
authority.

Child output is buffered until completion, with a 1 MiB limit independently
for stdout and stderr. Retained stdout bytes go only to Forge stdout; retained
child stderr and Forge diagnostics go to Forge stderr. There is no success
banner or added child-output newline. Truncation produces:

```text
forge: child stdout truncated after 1048576 bytes
forge: child stderr truncated after 1048576 bytes
```

Natural child exit 0 maps to Forge exit 0, and natural child exit 1 through 255
is preserved exactly. Clean context cancellation maps to 130. Input, trust,
runtime, output, wait, or cleanup infrastructure failures map to 1 and override
child/cancellation status. The CLI wires `os.Interrupt` through context
cancellation to direct-child termination and reap, but does not claim complete
terminal signal behavior across all platforms.

> **Security warning:** `forge run` executes trusted native code directly.
> Trust authenticates the signer and package integrity; it does not make code
> safe. Forge provides no sandbox, filesystem or network isolation, privilege
> drop, process-tree containment, CPU/memory limits, or production-safety
> guarantee.

The First Alpha runner has zero user arguments, no environment injection, nil
stdin, a private runtime working directory, a reduced environment, one direct
child, no live streaming, and no remote package acquisition.

## Known Limitations

- `forge build` still emits package format v1 identity/provenance packages;
  `forge build-runnable` is the separate signed package-v2 workflow, and
  `forge run` accepts only an existing local trusted package v2.
- Manifest loading does not provide a strict unknown-field contract, JSON
  duplicate keys are not rejected, and there is no separate manifest
  `schema_version` field.
- Process execution is limited to one host-target direct child with zero user
  arguments, no environment injection, nil stdin, a reduced environment, and a
  private working directory. Descendants are not managed; graceful shutdown,
  process-tree supervision, resource controls, and generalized runtime input
  policy are not implemented.
- Runnable package v2 currently contains one entrypoint artifact and does not
  serialize dependency provenance or an SBOM.
- Runtime package ingestion has fixed Alpha byte and entry ceilings. Process
  memory/CPU controls and runtime sandboxing are not implemented.
- Advanced Windows ACL/reparse hardening for materialized executables is not
  complete. Windows share-mode behavior and capability-dependent symlink
  security fixtures also remain bounded platform-hardening and coverage debt.
- Host PE/ELF/Mach-O family and architecture validation is implemented, but it
  is not malware analysis. Trust snapshot/revocation epoch semantics and
  start-time trust reauthorization are not implemented.
- Open-once package acquisition prevents directory-entry replacement from
  redirecting verification after handle acceptance, but it is not an immutable
  snapshot. A same-user process may still attempt in-place modification of the
  already-open file; ZIP parsing, integrity, and signature checks remain the
  content authority.
- `ProcessRunner` revalidates the materialized executable's type, size, digest,
  identity, target, and binary header before Start, but closes its validation
  handle before OS pathname execution. Its architecture review is complete;
  stronger validation-to-exec object binding remains explicitly deferred
  technical debt.
- There are no CPU, memory, process-count, filesystem, network, syscall, or
  privilege-dropping sandbox controls.
- Runtime trust is invocation-local with one key; persistent trust
  configuration, multiple configured keys, rotation, and revocation policy are
  not implemented.
- Exact cross-platform terminal signal acceptance, package-handle Close failure
  injection, and command-level fault injection for Start, Wait, Close, and
  output-write failures remain test debt.
- Executable builds partly inherit the host build environment and are not
  guaranteed to be reproducible across toolchains.
- Admission freezes the canonical source `ImportPath`, not source repository
  contents, commit/digest provenance, or an SBOM.
- Legacy packages are rejected by the normal reader; legacy inspection and
  migration tooling are not implemented.
- The reader supports only the explicit `(1,1)` and `(2,2)` package/bundle
  version pairs; broader multi-version compatibility is not implemented.
- Remote package registry, resolution, and package acquisition are not implemented.
- Strict atomic visibility across the package and package-source registries is
  not guaranteed.
- Process-crash atomicity between source and package commits is not guaranteed.
- Full concurrent-build isolation is not implemented.
- `forge build-runnable` rejects existing or racing output targets rather than
  overwriting them; broader same-output-path coordination for other workflows
  is not implemented.
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
Phase 6 compiler/package/runnable pipeline CLOSED / PASS for the bounded
Pre-Alpha / First Alpha scope

AI Runtime
░░░░░░░░░░░░░░░░░░░░ 0%
```
> Progress percentages represent the completed foundation scope for each
> engineering area. They do not imply that the entire long-term platform
> capability has been completed.

---

## Project Status

Forge is currently in the **Pre-Alpha** stage.

The core engineering foundation is now substantially established. Phase 6 —
Compiler / Package Pipeline Hardening is closed for its bounded Pre-Alpha /
First Alpha compiler-package-runnable contract, while registry, validation,
runtime expansion, and additional security hardening continue to evolve.

AI-First Engineering Operating System
