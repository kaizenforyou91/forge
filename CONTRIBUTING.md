# Contributing to Forge

First of all, thank you for your interest in contributing to Forge.

Forge is an AI-native application platform built with a strong focus on engineering quality, maintainability, and long-term sustainability.

Although the project is currently in the Pre-Alpha stage, every contribution should follow the engineering standards defined in this repository.

---

# Development Philosophy

Every change should make Forge:

- Easier to understand
- Easier to maintain
- Easier to extend
- More reliable
- Better documented

We value clarity over cleverness.

---

# Engineering Workflow

Every contribution follows this workflow:

Specification
→ Implementation
→ Review
→ Testing
→ Documentation
→ Commit
→ Merge

No implementation should be merged without documentation.

---

# Branch Strategy

Use the following branch naming convention.

Feature development

feature/<feature-name>

Examples

feature/manifest-parser

feature/runtime-loader

---

Bug fixes

bugfix/<description>

Examples

bugfix/schema-validation

---

Documentation

docs/<topic>

Examples

docs/readme

docs/roadmap

---

Refactoring

refactor/<component>

Examples

refactor/logger

---

Release

release/<version>

Examples

release/v0.1.0

---

# Commit Convention

Forge follows Conventional Commits.

Examples:

feat(cli): add validate command

feat(runtime): implement loader

fix(parser): handle invalid manifest

docs: update roadmap

docs: improve README

refactor(logger): simplify API

test(runtime): add loader tests

chore: update dependencies

---

# Pull Request Checklist

Before opening a Pull Request, ensure:

- Code builds successfully
- Tests pass
- Documentation updated
- No unnecessary files included
- Commit messages follow conventions

---

# Code Style

General principles:

- Keep functions small.
- Prefer readability.
- Avoid duplicated code.
- Write meaningful names.
- Keep packages focused.

---

# Project Structure

The repository is organized as follows:

cmd/
Command-line applications.

internal/
Private implementation.

pkg/
Reusable public packages.

runtime/
Runtime engine.

schemas/
Schema definitions.

docs/
Project documentation.

examples/
Example projects.

test/
Integration and testing assets.

---

# Documentation

Documentation is considered part of the implementation.

Every significant architectural decision should eventually be documented.

---

# Testing

Every new feature should include appropriate tests whenever practical.

Testing is an essential part of Forge engineering.

---

# Reporting Issues

When reporting a bug, please include:

- Description
- Expected behavior
- Actual behavior
- Reproduction steps
- Environment
- Logs (if available)

---

# Engineering Principles

Forge follows these principles:

- Simplicity
- Consistency
- Maintainability
- Extensibility
- Reliability
- Transparency

---

# Code Review

Every change should answer these questions:

- Is the code easy to understand?
- Is it maintainable?
- Is documentation updated?
- Are tests sufficient?
- Does it follow project conventions?

---

# Thank You

Thank you for helping build Forge.

Every improvement, no matter how small, helps move the project forward.