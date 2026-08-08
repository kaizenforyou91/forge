# ADR-001: Forge Architecture Freeze v1.0

## Status

Accepted

## Date

2026-08-08

## Context

Forge has completed the Core Runtime milestones:

- FW-021 Dependency Injection Container
- FW-022 Application Host

The framework now has a stable foundation consisting of dependency injection,
module management, application lifecycle, runtime context, startup, shutdown,
and bootstrap capabilities.

Further development will introduce HTTP and higher-level framework features.

Without an architectural baseline, future features may introduce inconsistent
dependencies, duplicated abstractions, or architectural coupling.

## Decision

Freeze the Forge Core Runtime architecture at version 1.0.

The architecture is organized into:

1. Core Infrastructure
2. Dependency Injection
3. Application Host
4. Framework Hosts
5. Application Framework Features

Dependencies must flow from higher-level features toward lower-level
infrastructure.

Lower-level infrastructure must not depend on higher-level framework features.

## Consequences

### Positive

- Clear package boundaries
- Predictable dependency direction
- Lower risk of architectural drift
- Easier future maintenance
- Easier onboarding for contributors
- Safer HTTP Host development
- Better long-term API stability

### Negative

- Some future features may require explicit ADRs
- Architectural changes may become slower
- Some abstractions may need to remain outside lower-level packages

## Baseline

Commit:

f139a5f

Tag:

v0.2.0-core-runtime

## Next Phase

FW-023 — HTTP Host  