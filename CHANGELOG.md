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

## Changed

- Manifest module `import_path` is now the package source of truth.
- Artifacts and bundle documents preserve import-path provenance.
- Compiler composition is available through application bootstrap.

## Tests

- Added multi-module dependency-first build coverage.
- Added invalid dependency rejection coverage.
- Added integrity, signature, trust-store, and verification-policy coverage.
- Added provenance serialization and package read-back coverage.

## Known Limitations

- A direct ToolchainExecutor result provenance test is missing.
- A direct Engine-to-Artifact provenance test is missing.
- Two comment-only placeholder tests remain in `internal/cli/build_test.go` and
  do not count as coverage.
- The package schema/version and backward-compatibility policy remain undefined.
- Repeated package-source registration semantics remain unresolved.

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
- Compiler package-format stabilization and optimization
- Production-grade compiler execution semantics
- Runtime loader and scheduler
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
