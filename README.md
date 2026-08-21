# Forge

> **Build AI-Native Applications with Structured Manifests**

![Status](https://img.shields.io/badge/status-pre--alpha-orange)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

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
- Deterministic artifact bundles and ZIP packages.
- Package integrity metadata and verification.
- Optional Ed25519 package signatures.
- Trust-store-backed signature verification and configurable verification policy.
- Artifact source provenance preserved through package read-back.
- Configuration, structured logging, dependency injection, application lifecycle,
  HTTP, middleware, and plugin foundations.

The implemented top-level CLI commands are `forge version`, `forge doctor`,
`forge config`, and `forge build`.

Forge remains **Pre-Alpha**. The compiler and package pipeline are a tested
foundation, not a stable production package format.

## Known Limitations

- The artifact bundle/package format does not yet have an explicit schema version.
- Compatibility with packages created before artifact provenance was introduced is
  not guaranteed.
- Remote package registry and remote package resolution are not implemented.
- Repeated builds in one application composition have unresolved package-source
  registration semantics.
- Default integrity verification proves package consistency. Without strict trusted
  signatures, it does not prove publisher authenticity.

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

MIT License

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

The core engineering foundation is now substantially established. Active
development is focused on package pipeline hardening and package-format
stabilization while registry, validation, and runtime capabilities continue to
evolve.

AI-First Engineering Operating System
