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
- CLI builds through `forge build <manifest>`.
- Explicit compatibility metadata for package format v1 and artifact bundle
  schema v1.
- Deterministic versioned ZIP packages with integrity schema v2 and optional
  signature schema v1.
- Exact-byte integrity binding for package metadata, bundles, and artifact
  payloads.
- Strict trust-store-backed signature verification and configurable verification
  policy.
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

Manifest Admission Hardening and Package Format Stabilization are implemented
and validated technical checkpoints. Phase 6 and the Compiler remain in progress.

## Known Limitations

- Only package format v1 and bundle schema v1 are currently supported.
- Legacy packages are rejected by the normal reader; legacy inspection and
  migration tooling are not implemented.
- Multiple-version reader support is not implemented.
- Secondary JSON documents remain permissive toward unknown and duplicate fields.
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
