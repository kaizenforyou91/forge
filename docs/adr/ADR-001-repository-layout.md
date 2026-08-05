# ADR-001: Repository Layout

- **Status:** Accepted
- **Date:** 2026-08-05
- **Decision Makers:** Forge Project Maintainer
- **Version:** 0.1.0

---

# Context

Forge is intended to become an AI-native application platform composed of multiple independent components.

As the project grows, maintaining a clear separation of responsibilities is essential to ensure scalability, maintainability, and ease of contribution.

A repository layout must therefore be established early in the project lifecycle.

---

# Decision

Forge adopts a layered repository structure based on Go community best practices.

```
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

Each top-level directory has a clearly defined responsibility.

---

# Responsibilities

## cmd/

Contains executable applications.

Only entry points should exist here.

---

## internal/

Contains implementation details that must remain private to the project.

Packages inside `internal/` are not intended for external reuse.

---

## pkg/

Contains reusable libraries that may be imported by external applications.

Only stable APIs should be placed here.

---

## runtime/

Contains runtime execution components.

Examples include:

- Loader
- Scheduler
- Runtime Engine
- Plugin Loader

---

## schemas/

Contains schema definitions used by Forge.

Examples:

- JSON Schema
- YAML Schema

---

## docs/

Contains project documentation.

Including:

- ADR
- Architecture
- Roadmap
- Specifications

---

## examples/

Contains example projects.

These examples should demonstrate recommended usage.

---

## scripts/

Contains automation scripts.

Examples:

- build
- release
- generate

---

## tools/

Contains development utilities.

These are helper programs used only during development.

---

## assets/

Contains static project resources.

Examples:

- logos
- icons
- diagrams

---

## test/

Contains integration tests and testing assets.

---

# Rationale

The chosen layout provides several advantages.

- Clear package ownership
- Better scalability
- Easier onboarding
- Better test organization
- Consistent engineering practices
- Alignment with Go community conventions

---

# Alternatives Considered

## Flat Repository

Rejected.

Reason:

Difficult to scale beyond small projects.

---

## Monolithic Package Structure

Rejected.

Reason:

Creates strong coupling between unrelated components.

---

# Consequences

Positive:

- Better modularity
- Easier maintenance
- Improved readability
- Easier code review
- Clear dependency boundaries

Negative:

- Slightly more directories to manage
- Requires discipline when creating new packages

---

# Dependency Rules

Dependencies should follow these principles.

```
cmd
│
▼
internal
│
▼
pkg
```

Higher layers may depend on lower layers.

Lower layers must never depend on higher layers.

Runtime components may depend on `pkg`, but `pkg` must never depend on runtime.

---

# Future Evolution

The repository structure may evolve as Forge grows.

However, major structural changes should always be documented through a new ADR.

---

# References

- Go Project Layout
- Architecture Decision Records (ADR)
- Semantic Versioning
- Keep a Changelog