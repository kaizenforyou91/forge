# Forge External Alpha Workflow

## Purpose

This document is the canonical, reproducible workflow for exercising Forge's
bounded First Alpha product path from a clean repository checkout. It covers
manifest validation, package creation and inspection, explicit trust, and
native application execution for published `v0.3.0-alpha.1`. It does not add
current-main AI commands to that historical release's workflow.

Current main is ahead of alpha.1 and **UNRELEASED**: Phase 8 bounded AI/tool
foundation is **CLOSED / PASS**, including accepted real-provider validation **PASS**;
Phase 9 is **CLOSED / PASS — BOUNDED AGENT EXECUTION LIFECYCLE**,
internal/pre-stable. See the separate [current-main AI workflow](AI_PROMPT_WORKFLOW.md)
and [roadmap](../ROADMAP.md). The original text-only implementation checkpoint
is historical evidence, not the full current-main capability set.
**Phase 10: NOT DEFINED / NOT AUTHORIZED. RELEASE DEFERRED.**

## Current maturity: First Alpha — 0.3.0-alpha.1

Forge First Alpha is a local, manifest-driven, non-production technical
preview. Formal readiness is approved with non-blocking accepted debt. This
workflow demonstrates the bounded product path; it is not a stable-format or
production-readiness claim. Annotated tag `v0.3.0-alpha.1` exists and resolves
to `5d836931216203aeea0737fc54de9e95091a62ef`. The
[GitHub prerelease](https://github.com/kaizenforyou91/forge/releases/tag/v0.3.0-alpha.1)
is published, not draft, with zero uploaded prebuilt binary assets. Its release
notes correctly exclude AI Runtime; later Phase 8/9 work is not part of alpha.1.

## Prerequisites

- Git and a clean Forge repository checkout.
- Go 1.26 or newer, matching `go.mod`.
- A temporary directory outside the repository for the Forge binary, packages,
  and ephemeral keys.

Run every Forge command below from the repository root. Replace `<temp>` with
an absolute temporary-directory path and `<forge>` with the stamped executable
path (`<temp>/forge` on Unix-like systems or `<temp>\forge.exe` on Windows).
Never place the example private key in the repository.

## Build Forge

For a local development build without release identity:

```text
go build -trimpath -o <forge> ./cmd/forge
```

That build reports the intentional defaults `dev`, `none`, and `unknown`.

## Version stamping

Release identity is supplied at build time; no source file is edited. The
linker variable paths are:

```text
github.com/kaizenforyou91/forge/internal/cli.AppVersion
github.com/kaizenforyou91/forge/internal/cli.Commit
github.com/kaizenforyou91/forge/internal/cli.BuildTime
```

To build with the published First Alpha identity, use a checkout of annotated
tag `v0.3.0-alpha.1` at commit `5d836931216203aeea0737fc54de9e95091a62ef`, version
`0.3.0-alpha.1`, that exact release commit, and a UTC RFC3339 build timestamp.
Do not stamp later current-main additions as though they were published alpha.1.
This reproduces the source/version identity, not a byte-identical official
binary (no binaries were uploaded):

```text
go build -trimpath -ldflags "-X github.com/kaizenforyou91/forge/internal/cli.AppVersion=0.3.0-alpha.1 -X github.com/kaizenforyou91/forge/internal/cli.Commit=<commit> -X github.com/kaizenforyou91/forge/internal/cli.BuildTime=<UTC-RFC3339>" -o <forge> ./cmd/forge
<forge> version
```

Expected fields are the exact supplied values:

```text
Forge CLI
Version : 0.3.0-alpha.1
Commit  : <commit>
Built   : <UTC-RFC3339>
```

The published release identity is AppVersion `0.3.0-alpha.1` and annotated Git
tag `v0.3.0-alpha.1`; the published prerelease exists. The earlier value
`alpha-acceptance` was a TEST-ONLY WP7 acceptance label and must never be treated as a published
version.

## Example application

The canonical application is [`examples/alpha-app/main.go`](../examples/alpha-app/main.go).
It uses only the Go standard library, takes no input, performs no network or
filesystem operations, prints one deterministic line, and exits zero:

```text
Forge Alpha example: OK
```

Its source can be checked independently without leaving a repository artifact:

```text
go build -o <temp>/alpha-app ./examples/alpha-app
```

## Manifest explanation

[`examples/alpha-app/forge.yaml`](../examples/alpha-app/forge.yaml) declares
application identity `forge-alpha-example@v1`, one module `app@v1`, its exact
Go import path, no dependencies, and that module as the application entrypoint.
The entrypoint is build intent; it does not make an ordinary v1 package
runnable and does not grant runtime trust.

Manifest admission is strict. Forge requires valid UTF-8, one document/object,
the documented field types, and no unknown or duplicate fields. YAML aliases,
anchors, merge keys, explicit tags, and tag directives are rejected. The
manifest format currently has no separate `schema_version` field.

## Structural validation

```text
<forge> validate examples/alpha-app/forge.yaml --profile structural
```

Expected output:

```text
Manifest valid: examples/alpha-app/forge.yaml (profile=structural)
```

This profile checks strict decoding and manifest-domain rules.

## Build validation

Build admission is the default profile:

```text
<forge> validate examples/alpha-app/forge.yaml
```

Expected output:

```text
Manifest valid: examples/alpha-app/forge.yaml (profile=build)
```

It additionally checks the current non-mutating compiler-admission boundary.

## Runnable validation

```text
<forge> validate examples/alpha-app/forge.yaml --profile runnable
```

Expected output:

```text
Manifest valid: examples/alpha-app/forge.yaml (profile=runnable)
```

It additionally requires an admitted application entrypoint and its exact
package source.

All three validation profiles are admission checks. `forge validate` does
**not** prove that `go list` will succeed, that the Go package compiles, that a
signing key is valid, that publication or runtime will succeed, or that native
code is safe.

## Build v1 package

Use an explicit output path outside the repository:

```text
<forge> build examples/alpha-app/forge.yaml --output <temp>/forge-alpha-example-v1.zip
```

`forge build` always produces package format 1 / bundle schema 1 for the
identity/provenance workflow. It remains non-runnable even though this manifest
has an entrypoint.

## Inspect v1 package

```text
<forge> inspect <temp>/forge-alpha-example-v1.zip
```

The report must identify format 1, bundle schema 1,
`forge-alpha-example@v1`, `Type: non-runnable`, `Runtime: none`, verified
integrity, an unsigned signature state, and the `app@v1` artifact with its
import path. No trust flags are needed for an unsigned v1 package.

## Generate signing/trust keys

Save this standard-library-only helper as `<temp>/keygen.go`, outside the
repository:

```go
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/x509"
    "encoding/pem"
    "os"
)

func main() {
    if len(os.Args) != 3 {
        panic("usage: keygen <private.pem> <public.pem>")
    }
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        panic(err)
    }
    privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
    if err != nil {
        panic(err)
    }
    publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
    if err != nil {
        panic(err)
    }
    privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
    publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
    if err := os.WriteFile(os.Args[1], privatePEM, 0o600); err != nil {
        panic(err)
    }
    if err := os.WriteFile(os.Args[2], publicPEM, 0o644); err != nil {
        panic(err)
    }
}
```

Run it using absolute paths:

```text
go run <temp>/keygen.go <temp>/alpha-private.pem <temp>/alpha-public.pem
```

This emits an unencrypted PKCS#8 Ed25519 key in a single `PRIVATE KEY` PEM
block and the matching X.509 PKIX SubjectPublicKeyInfo in a single `PUBLIC KEY`
PEM block. The helper requests owner-only private-key permissions; on
Unix-like systems verify them with `chmod 600 <temp>/alpha-private.pem` if the
directory or filesystem applies a broader creation mask.

Forge limits each key file to 16 KiB, rejects PEM headers and trailing data,
rejects symlinks and non-regular files, and enforces owner-only private-key
permissions on Unix. Windows ACL validation is not currently claimed.

Do not commit private keys, reuse example/acceptance keys, pass a private key as
a command argument, or treat a filename as key identity. The TEST-ONLY example
KeyID in this workflow is `alpha-acceptance-key`; it is not a Forge version,
published key, or persistent trust entry.

## Build signed runnable v2 package

```text
<forge> build-runnable examples/alpha-app/forge.yaml --signing-key <temp>/alpha-private.pem --key-id alpha-acceptance-key --output <temp>/forge-alpha-example-runnable.zip
```

`forge build-runnable` produces package format 2 / bundle schema 2 for the
current host, and signing is mandatory. It does not execute the result. The
output path must end in lowercase `.zip` and an existing target is not
overwritten.

## Inspect signed package without trust

```text
<forge> inspect <temp>/forge-alpha-example-runnable.zip
```

Expected semantics include format 2, bundle schema 2, `Type: runnable`, runtime
`application_executable`, the current host target, entrypoint `app@v1`, verified
integrity, and:

```text
Signature: present, trust not verified
Declared KeyID (unverified): alpha-acceptance-key
```

This `SIGNED_UNVERIFIED` state proves only cryptographic consistency with the
public key embedded in the package. It is not authenticity, publisher
verification, or trust.

## Inspect signed package with explicit trust

```text
<forge> inspect <temp>/forge-alpha-example-runnable.zip --trusted-key <temp>/alpha-public.pem --key-id alpha-acceptance-key
```

Successful explicit verification reports:

```text
Signature: trusted
Verified signer: alpha-acceptance-key
```

Both trust flags must be supplied together. Trust is invocation-local and the
KeyID is an exact, non-normalized string.

## Run explicitly trusted package

```text
<forge> run <temp>/forge-alpha-example-runnable.zip --trusted-key <temp>/alpha-public.pem --key-id alpha-acceptance-key
```

`forge run` requires both trust flags. It accepts an existing local lowercase
`.zip`, never performs an implicit build or remote acquisition, and executes
only a strictly verified, host-compatible runnable v2 package.

## Expected output

Successful execution has exit code 0, empty stderr, and exactly this child
stdout; Forge adds no success banner:

```text
Forge Alpha example: OK
```

## Failure expectations

- Unknown or duplicate manifest fields fail strict loading.
- Invalid profiles, unresolved dependencies, or a missing runnable entrypoint
  fail validation.
- Existing output paths are preserved rather than overwritten.
- Missing, malformed, mismatched, or incorrectly identified keys fail closed.
- Inspection with a different Ed25519 public key and the declared KeyID must
  fail signature verification.
- Unsigned, untrusted, tampered, non-runnable, or host-incompatible packages do
  not run.
- Natural child exit codes 1 through 255 are preserved. Clean cancellation maps
  to 130; Forge infrastructure failures map to 1.

## Cleanup

Delete the entire temporary directory after the workflow. It contains the
stamped Forge binary, example build, v1 and v2 packages, key-generation
material, and private key. Confirm that `git status --short` shows only your
intentional repository edits (or nothing on a clean checkout).

## Security boundaries

Package trust authenticates signed bytes relative to the exact public key and
KeyID explicitly supplied for that invocation. It does not establish code
safety, complete publisher provenance, an SBOM, or reproducible source. A
signed runnable package contains native code and `forge run` executes it
directly with the invoking user's authority.

Forge provides no sandbox, filesystem isolation, network isolation, privilege
drop, CPU/memory/process quotas, process-tree containment, or production-safety
guarantee. Persistent trust, multiple configured keys, rotation, and revocation
are not implemented. Package formats remain First Alpha and pre-stable.

## Capabilities excluded from published alpha.1

Deferred work includes `forge init`, `forge fmt`, machine-readable validation
and inspection output, remote and persistent registries, package acquisition
and indexing, version ranges, complete provenance/SBOM, persistent trust and
revocation, arguments/environment/stdin/caller working directory, live output
streaming, descendant lifecycle and graceful shutdown, sandboxing and resource
controls, dynamic plugin loading, scheduling/orchestration, cross-toolchain
reproducibility, build isolation, and the AI Runtime. That last exclusion
describes alpha.1: current-main bounded Phase 8/9 additions are separately
documented and remain unreleased.

Security debt also remains around same-user in-place modification of an opened
package, materialized validation-to-path-execution binding, and deeper Windows
ACL/reparse/share-mode hardening.

## What Alpha does NOT guarantee

This workflow does not guarantee production suitability, stable package or
manifest compatibility, immutable package snapshots, malware safety,
cross-toolchain reproducibility, complete Windows security parity, remote
distribution, persistent trust, sandboxing, orchestration, or AI-native
execution. Passing it means the bounded local First Alpha workflow is
reproducible; it does not strengthen Forge beyond a non-production technical
preview with pre-stable APIs and package formats.

## Publication boundary

**RELEASE DEFERRED.** RR-001 changes documentation only and performs no live
provider call. It does not decide another release, a version number, current-main
release readiness, binary publication, or whether Phase 10 should open.
Autonomous agents, AI memory/durable history, persistence, workflows, schedulers,
queues/workers, multi-agent orchestration, arbitrary side-effect tools, provider
routing, and public agent APIs remain outside the delivered Phase 8/9 boundary.
No Beta/production, sandboxing, process-tree containment, or persistent trust
rotation/revocation claim is added. Phase 10 remains **NOT DEFINED / NOT AUTHORIZED**.
