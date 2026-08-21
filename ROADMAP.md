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
- Deterministic dependency graph/order integration
- Compiler execution abstraction
- Operating-system command runner
- Import-path-aware toolchain executor
- Dependency-first build plan execution
- Artifact generation and artifact bundles
- Deterministic bundle serialization and ZIP packaging
- Package integrity metadata and verification
- Ed25519 signing and verification
- Trust store and verification policy
- ZIP package read-back
- Artifact source provenance
- CLI `forge build` for YAML and JSON manifests
- Multi-module dependency-aware builds

## Phase 5 — Registry Foundation

Implemented foundation:

- Concurrency-safe in-memory package identity registry.
- Exact package name/version resolution.
- Package source registry.
- Deterministic dependency graph/order integration.

Remaining work:

- Remote registry and remote resolution.
- Package acquisition.
- Advanced version constraints.
- Registry persistence and distribution policy.

## Phase 6 — Compiler Foundation

Implemented foundation:

- Compiler execution abstraction and operating-system command runner.
- Import-path-aware toolchain executor.
- Dependency-first build plan execution.
- Artifact generation and artifact bundles.
- Deterministic bundle serialization and ZIP packaging.
- Package integrity metadata and verification.
- Ed25519 signing and verification, trust store, and verification policy.
- ZIP read-back and artifact source provenance.
- CLI build vertical slice for YAML, JSON, and multi-module dependency-aware builds.

Remaining work:

- Package schema/version contract.
- Backward-compatibility policy.
- Repeated-build package-source semantics.
- Direct provenance unit coverage.
- Import-path-only tamper acceptance test.
- Compiler optimization.
- Stable production package format.
- Production-grade execution semantics.

## Evidence-Based Current Roadmap Position

```text
Pre-Alpha
→ Phase 6 — Compiler
→ Package Pipeline Hardening
→ Artifact Provenance
→ Package-Format Stabilization
```

## Remaining Platform Work

The following capabilities remain future work:

- Dedicated Validation Engine
- Remote package registry and package acquisition
- Package-format stabilization
- Compiler optimization and production-grade execution semantics
- Runtime loader
- Scheduler
- Remote package resolution
- Advanced dependency and version resolution
- AI Runtime

# Long-Term Goal

Forge 1.0 will provide a complete AI-native development platform that enables developers to build modern applications through declarative manifests, modular runtimes, and intelligent automation.
