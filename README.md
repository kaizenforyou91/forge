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

# Planned Features

The following components are planned for Forge.

| Component | Status |
|-----------|--------|
| Workspace | ✅ Bootstrap |
| Documentation | 🚧 In Progress |
| CLI | ⏳ Planned |
| Manifest Engine | ⏳ Planned |
| Validation Engine | ⏳ Planned |
| Registry | ⏳ Planned |
| Compiler | ⏳ Planned |
| Runtime | ⏳ Planned |
| Plugin System | ⏳ Planned |
| AI Runtime | ⏳ Planned |

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
░░░░░░░░░░░░░░░░░░░░ 0%

Package Registry
░░░░░░░░░░░░░░░░░░░░ 0%

Compiler
░░░░░░░░░░░░░░░░░░░░ 0%

AI Runtime
░░░░░░░░░░░░░░░░░░░░ 0%
```
> Progress percentages represent the completed foundation scope for each
> engineering area. They do not imply that the entire long-term platform
> capability has been completed.

---

## Project Status

Forge is currently in the **Pre-Alpha** stage.

The core engineering foundation is now substantially established.
Active development is focused on extending the manifest-driven platform
toward validation, registry, compiler, and full runtime capabilities.

AI-First Engineering Operating System
