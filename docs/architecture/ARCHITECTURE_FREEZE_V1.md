# Forge Architecture Freeze v1.0

**Status:** FROZEN  
**Version:** v1.0  
**Milestone:** v0.2.0-core-runtime  
**Scope:** Forge Core Runtime  
**Baseline Commit:** f139a5f  
**Baseline Tag:** v0.2.0-core-runtime

---

## 1. Purpose

This document defines the architectural rules for the Forge framework.

The purpose of the Architecture Freeze is to establish a stable foundation before
the framework enters HTTP Host and higher-level application framework development.

The architecture defined here MUST be treated as the baseline for all future
development unless an explicit Architecture Decision Record (ADR) approves a change.

---

# 2. Architectural Vision

Forge is designed as a modular Go application framework.

The architecture follows these major layers:

    Application
        |
        v
    Application Host
        |
        v
    Module System
        |
        v
    Dependency Injection
        |
        v
    Core Infrastructure
        |
        +---- Config
        +---- Logger
        +---- Errors

Future framework capabilities MUST be built on top of these foundations.

---

# 3. Core Architecture Layers

## Layer 0 — Go Runtime

The lowest layer is the Go runtime and standard library.

Forge SHOULD prefer the Go standard library whenever it provides an adequate
implementation.

External dependencies MUST have a clear architectural justification.

---

## Layer 1 — Core Infrastructure

Current packages:

    pkg/config
    pkg/logger
    pkg/errors

Responsibilities:

- configuration
- logging
- framework errors
- foundational utilities

These packages MUST remain lightweight.

They MUST NOT depend on higher-level application features.

---

## Layer 2 — Dependency Injection

Current package:

    pkg/container

Responsibilities:

- service registration
- singleton lifetime
- transient lifetime
- factories
- constructors
- dependency resolution
- recursive dependency resolution
- auto wiring
- dependency graph
- cycle detection
- generic resolution
- container freezing

The DI container is the primary dependency mechanism of Forge.

Higher-level packages MAY depend on the container.

The container MUST NOT depend on HTTP, application controllers,
routing, persistence, or business logic.

---

## Layer 3 — Application Host

Current package:

    pkg/app

Responsibilities:

- application lifecycle
- module registration
- module management
- startup
- shutdown
- runtime context
- application bootstrap
- application execution

The Application Host coordinates the framework.

It MUST NOT contain HTTP-specific implementation details.

---

## Layer 4 — Framework Hosts

Future packages:

    pkg/http
    pkg/grpc
    pkg/ws

Responsibilities:

- transport
- server lifecycle
- request handling
- middleware
- routing
- protocol-specific concerns

Transport implementations MUST depend on the Application Host and
Dependency Injection infrastructure rather than the reverse.

---

## Layer 5 — Application Framework Features

Future packages MAY include:

    pkg/router
    pkg/middleware
    pkg/controller
    pkg/validation
    pkg/openapi
    pkg/orm
    pkg/auth

These features MUST remain independent from transport-specific implementations
whenever practical.

---

# 4. Dependency Direction

The dependency direction is:

    Core Infrastructure
            ^
            |
    Dependency Injection
            ^
            |
    Application Host
            ^
            |
    Framework Hosts
            ^
            |
    Application Features

Dependencies MUST flow toward lower-level abstractions.

Lower layers MUST NOT import higher layers.

---

# 5. Package Dependency Rules

## Rule 1

`pkg/config` MUST NOT import:

    pkg/app
    pkg/container
    pkg/http

---

## Rule 2

`pkg/logger` MUST NOT import:

    pkg/app
    pkg/container
    pkg/http

---

## Rule 3

`pkg/errors` MUST remain independent from application-level packages.

---

## Rule 4

`pkg/container` MUST NOT import:

    pkg/http
    pkg/router
    pkg/controller

---

## Rule 5

`pkg/app` MAY depend on:

    pkg/container
    pkg/config
    pkg/logger
    pkg/errors

but MUST NOT contain transport-specific logic.

---

## Rule 6

`pkg/http` MAY depend on:

    pkg/app
    pkg/container
    pkg/config
    pkg/logger
    pkg/errors

but these packages MUST NOT depend on `pkg/http`.

---

# 6. Dependency Injection Rules

Forge uses constructor-based dependency injection as the preferred mechanism.

Preferred:

    func NewService(repo Repository) *Service

Avoid:

    func NewService() *Service {
        repo := globalRepository
        ...
    }

Global mutable service state SHOULD NOT be used.

Dependency resolution SHOULD happen through the container.

---

# 7. Service Lifetime Rules

Forge currently supports:

    Singleton
    Transient

Singleton:

- one instance per container
- reused during application lifetime

Transient:

- a new instance MAY be created for every resolution

Future lifetimes MAY be introduced, but they MUST NOT change the semantics
of existing lifetimes.

---

# 8. Constructor Rules

Constructors SHOULD:

- return exactly one value
- declare dependencies through parameters
- avoid hidden global dependencies
- remain deterministic
- avoid unnecessary side effects

Preferred:

    func NewUserService(repo UserRepository) *UserService

Avoid:

    func NewUserService() *UserService {
        loadDatabase()
        loadConfig()
        ...
    }

Initialization that requires lifecycle management SHOULD be handled by Modules.

---

# 9. Module Architecture

Modules are the primary mechanism for composing applications.

A module is responsible for:

- registering dependencies
- initializing resources
- starting runtime services
- stopping runtime services

The lifecycle is:

    Register
       |
       v
    Start
       |
       v
    Running
       |
       v
    Stop

Modules MUST be independently testable.

Modules SHOULD avoid direct knowledge of other modules.

Dependencies between modules SHOULD be expressed through the DI container.

---

# 10. Application Lifecycle

The canonical lifecycle is:

    New
      |
      v
    Configure
      |
      v
    Register Modules
      |
      v
    Build Dependencies
      |
      v
    Start
      |
      v
    Running
      |
      v
    Stop
      |
      v
    Shutdown

Future runtime features MUST preserve this lifecycle model.

---

# 11. Startup Rules

Startup MUST be deterministic.

Modules SHOULD start in registration order unless an explicit dependency
ordering mechanism is introduced.

If startup fails:

- the error MUST be returned
- already-started resources SHOULD be cleaned up when practical
- the application MUST NOT report itself as running

---

# 12. Shutdown Rules

Shutdown SHOULD occur in reverse module order.

Example:

    Module A
    Module B
    Module C

Startup:

    A -> B -> C

Shutdown:

    C -> B -> A

Shutdown MUST be safe to call more than once.

---

# 13. Context Rules

Application runtime SHOULD use `context.Context` for:

- cancellation
- deadlines
- shutdown propagation
- request-scoped operations

Contexts MUST NOT be stored unnecessarily in long-lived global variables.

---

# 14. Error Handling

Forge MUST prefer explicit error returns.

Preferred:

    if err := app.Start(); err != nil {
        return err
    }

Framework internals SHOULD NOT panic for ordinary runtime failures.

Panics MAY be used for:

- programmer errors
- impossible internal states
- explicitly documented Must-style APIs

Public APIs SHOULD provide error-returning alternatives where practical.

---

# 15. Public API Rules

Public APIs MUST be intentionally designed.

Before exporting a type, function, or method, ask:

1. Is this required by application developers?
2. Is the API stable enough to expose?
3. Can the API be changed later without compatibility problems?

Internal implementation details SHOULD remain unexported.

---

# 16. Fluent API Rules

Fluent APIs MAY be used for configuration and bootstrap operations.

Example:

    app := app.New().
        Use(configModule).
        Use(loggerModule)

Fluent methods MUST return the receiver when chaining is intended.

Fluent APIs SHOULD NOT hide critical runtime errors unless the API is explicitly
named as a Must-style API.

---

# 17. Generic API Rules

Generics SHOULD be used when they materially improve:

- type safety
- developer experience
- API clarity

Reflection MUST NOT be replaced by generics where runtime type information
is fundamentally required.

Reflection SHOULD remain isolated inside infrastructure packages where possible.

---

# 18. Reflection Rules

Reflection is permitted inside:

    pkg/container

and other infrastructure components where runtime type inspection is required.

Reflection SHOULD NOT leak into ordinary application code.

Public application APIs SHOULD remain strongly typed whenever possible.

---

# 19. Concurrency Rules

Concurrency MUST be explicit.

Framework components MUST document ownership of goroutines.

Every goroutine started by Forge SHOULD have a defined shutdown mechanism.

No framework goroutine should be permanently orphaned.

Shared mutable state MUST be protected appropriately.

---

# 20. HTTP Architecture

Future HTTP implementation MUST remain layered.

Target architecture:

    HTTP Host
        |
        +-- Server
        |
        +-- Router
        |
        +-- Middleware
        |
        +-- Request
        |
        +-- Response
        |
        +-- Context

HTTP-specific implementation MUST NOT leak into:

    pkg/container
    pkg/config
    pkg/errors

---

# 21. Router Rules

The router MUST be independently testable.

Routing concerns SHOULD include:

- method matching
- path matching
- route parameters
- route groups
- middleware
- handler execution

Router implementation MUST NOT own application lifecycle.

---

# 22. Middleware Rules

Middleware SHOULD follow a composable pipeline model.

Conceptually:

    Request
       |
       v
    Middleware 1
       |
       v
    Middleware 2
       |
       v
    Handler
       |
       v
    Response

Middleware SHOULD remain independent and composable.

---

# 23. Testing Rules

Every new framework capability SHOULD include tests.

Minimum expectations:

- normal behavior
- invalid input
- error behavior
- lifecycle behavior
- edge cases

Infrastructure packages SHOULD maintain high test coverage.

Tests MUST remain deterministic.

Tests MUST NOT depend on external services unless explicitly marked as integration tests.

---

# 24. Benchmark Rules

Performance-sensitive infrastructure SHOULD have benchmarks.

Current benchmark targets include:

    BenchmarkResolve
    BenchmarkMakeGeneric
    BenchmarkConstructor

Future performance-sensitive components SHOULD follow the same pattern.

Benchmarks MUST NOT be used as functional correctness tests.

---

# 25. Formatting and Static Analysis

Every implementation MUST pass:

    go fmt ./...
    go vet ./...
    go test ./...
    go build ./...

A feature is NOT considered complete if any of these commands fail.

---

# 26. Git Rules

Every completed feature SHOULD produce a focused commit.

Commit format:

    feat(scope): description

Examples:

    feat(container): add recursive dependency resolution
    feat(app): complete Application Host v1.0
    feat(http): add router

Bug fixes:

    fix(container): prevent cyclic dependency resolution

Documentation:

    docs(architecture): freeze architecture v1.0

---

# 27. Architecture Change Policy

This document is FROZEN.

Changing a frozen architectural rule requires:

1. Identify the problem.
2. Explain why the current architecture is insufficient.
3. Propose an alternative.
4. Document trade-offs.
5. Create an ADR.
6. Update this document.
7. Increment the architecture version.

No architectural change should be made silently.

---

# 28. ADR Policy

Architecture Decision Records SHOULD be stored under:

    docs/architecture/adr/

Naming:

    ADR-001-title.md
    ADR-002-title.md

Each ADR SHOULD contain:

- Context
- Problem
- Decision
- Alternatives
- Consequences
- Status

---

# 29. Versioning Policy

Forge follows semantic versioning for public releases:

    MAJOR.MINOR.PATCH

Architecture versions are independent:

    Architecture v1.0
    Architecture v1.1
    Architecture v2.0

A minor architecture revision SHOULD remain backward compatible.

A major architecture revision MAY introduce breaking changes.

---

# 30. Backward Compatibility

Once a public API is released, breaking changes SHOULD be avoided.

Breaking changes MUST be:

- documented
- justified
- versioned

Before v1.0, APIs MAY evolve more aggressively.

After v1.0, public API stability becomes a primary requirement.

---

# 31. Rebranding Policy

The current repository/project identifier is temporary.

The framework MAY undergo a public-facing rebrand before its first public
stable release.

The internal architecture MUST remain independent from the final product name.

Repository/package naming changes MUST NOT be allowed to drive architectural
decisions.

---

# 32. Security Principles

Security MUST be considered from the beginning.

Future framework features SHOULD follow:

- secure defaults
- explicit authentication boundaries
- explicit authorization boundaries
- input validation
- safe error exposure
- no accidental secret logging
- no unsafe global state

Security-sensitive features SHOULD receive dedicated tests.

---

# 33. Observability Principles

The framework SHOULD eventually support:

- structured logging
- metrics
- tracing
- request correlation
- health checks
- readiness checks

Observability MUST remain modular.

Applications SHOULD be able to enable only the capabilities they need.

---

# 34. Performance Principles

Performance is important but MUST NOT override architectural correctness.

Optimization priority:

    Correctness
        |
        v
    Maintainability
        |
        v
    Observability
        |
        v
    Performance Optimization

Performance optimizations MUST be supported by benchmarks or profiling data
when practical.

---

# 35. Simplicity Principle

Forge MUST prefer simple designs over unnecessary abstractions.

Do not introduce:

- interfaces without a purpose
- abstractions without multiple implementations
- configuration without a real use case
- reflection where static typing is sufficient
- dependencies where the standard library is sufficient

Every abstraction should solve a demonstrated problem.

---

# 36. Framework Design Philosophy

Forge follows these principles:

1. Go-first
2. Explicit over magical
3. Modular over monolithic
4. Composition over inheritance
5. Strong typing where possible
6. Reflection only where necessary
7. Small public APIs
8. Deterministic lifecycle
9. Test-driven infrastructure
10. Production-oriented defaults

---

# 37. Current Architecture Baseline

At Architecture Freeze v1.0:

    pkg/
    ├── app/
    ├── config/
    ├── container/
    ├── errors/
    └── logger/

The core runtime consists of:

    Dependency Injection
    +
    Module System
    +
    Application Host
    +
    Runtime Context
    +
    Lifecycle Management

This baseline is frozen.

---

# 38. Future Architecture Direction

The planned framework evolution is:

    Core Infrastructure
            |
            v
    Dependency Injection
            |
            v
    Application Host
            |
            v
    HTTP Host
            |
            v
    Router / Middleware
            |
            v
    Controller / API Layer
            |
            v
    Persistence / ORM
            |
            v
    Authentication / Authorization
            |
            v
    Observability
            |
            v
    Enterprise Features

Each layer MUST preserve the dependency direction established in this document.

---

# 39. Architecture Freeze Statement

As of:

    Architecture Freeze v1.0

the Forge Core Runtime architecture is considered stable.

FW-023 and subsequent features MUST build on this architecture rather than
redefine it.

Architectural changes MUST use the ADR process.

This document is the reference point for future architecture decisions.

---

# 40. Status

**Architecture Freeze v1.0: APPROVED**

Baseline:

    Commit: f139a5f
    Tag: v0.2.0-core-runtime

Next major development phase:

    FW-023 — HTTP Host