# 0.4.0-alpha.1 — Publication Record and Historical Candidate Evidence

- Target: **0.4.0-alpha.1**
- Status: **PUBLISHED**
- RR-004-PUB: **CLOSED / PASS — PUBLISHED**
- Release authorization: **GRANTED / EXECUTED**
- Tag: **v0.4.0-alpha.1 — CREATED / ANNOTATED**
- Tag object: `c02b81a94c3c7bead6a7fcf4030ce03cc98af52a`
- Tag target: `85d78143db1b8bcf2f96b79d681895b1a0492642`
- Publication tree: `e58a864eb6617811cca1dcb8d2d30bbd72e19298`
- GitHub Release: **PUBLISHED PRERELEASE**
- Release ID: **388084778**
- draft: **false**
- prerelease: **true**
- assets: **0**
- created_at: `2026-09-14T00:40:39Z`
- published_at: `2026-09-14T00:42:10Z`
- Historical Phase 10 status at publication: **NOT DEFINED / NOT AUTHORIZED**

Post-release P10-A0-R1 selected bounded synchronous workflow composition.
Implementation remains **NOT AUTHORIZED** and separately gated.
[ADR-003](architecture/adr/ADR-003-bounded-workflow-composition.md) records the
architecture-only decision; P10-A1 integration and strict push-main CI establish
its canonical definition. No Phase 10 runtime capability is part of this release.

[Published Forge 0.4.0-alpha.1](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1)
is a source-only, non-production prerelease. This record reconciles documentation
after publication in RR-005; it does not change the immutable tag or Release.
The tagged source intentionally preserves the RR-003 pre-publication snapshot.
The historical RR-003 preparation and RR-004 validation evidence below remain
separate from the publication event. Phase 8 and Phase 9 remain CLOSED / PASS.

## Summary

The published non-production prerelease adds explicit bounded AI text and
read-only tool execution to the existing manifest/package/trust/run preview.
It also includes public pre-stable AI contracts and an internal operation
lifecycle foundation. RR-003 prepared evidence and RR-005 reconciles documentation; neither adds runtime capability.

## Highlights

- `forge ai prompt` with explicit network permission, bounded text execution,
  and classified safe failures.
- Optional authority for one read-only runtime metadata tool.
- Public pre-stable `pkg/ai` and `pkg/ai/tool` contracts.
- Internal single-use Run and application-host lifecycle composition.
- No package migration or dependency change relative to published alpha.1.

## AI text execution

```text
forge ai prompt --provider openai --model <model-id> --text "<prompt>" --allow-network [--allow-tools] [--timeout 30s] [--max-output-tokens 1024]
```

This grammar describes capability; it is not an instruction to perform live
validation. RR-003 made no live provider call; RR-005 also makes none. Without `--allow-tools`,
the CLI uses `ai.Executor` for one provider turn and enforces the single-turn
output-token cap. Provider/model/text are explicit; there is no default model,
fallback, streaming, or retry. Unknown/raw failures are sanitized, and mixed
provider/cancellation failures remain failures rather than pure cancellation.

## Authorized read-only tool mode

`--allow-network` and explicit `--allow-tools` permit only the prebuilt,
invocation-local authority for `forge_runtime_info`. Tool access defaults false.
The tool returns read-only version, commit, and build-time metadata.

The CLI uses the existing direct Phase 8 authorized round trip: at most two
provider POSTs and one handler attempt, no retry and no recursive tool loop.
There are no filesystem, shell, subprocess, external-network, or write/mutation
tool handlers, and no arbitrary CLI tool registration. Admission is distinct
from execution; provider output never creates authority.

Each turn is independently bounded. Aggregate `Usage.OutputTokens` may exceed
the per-turn request cap: the 64-cap / 96-aggregate test is valid accounting
regression evidence, not a guaranteed provider response. The final aggregate
is not passed through `ai.Executor` for another single-turn cap check.

## Public pre-stable Go APIs

`pkg/ai` adds Request, Result, Usage, Provider, Executor, validation, bounded
contracts, and SafeError classifications. `pkg/ai/tool` adds declarations,
catalog/admission, explicit execution bindings, and immutable authority.
These are externally importable, pre-stable library contracts. Caller-supplied
library handlers do not expand the production CLI's one-tool authority.

## Internal Phase 9 lifecycle foundation

The OpenAI adapter remains internal. `internal/agent` provides a single-use Run,
atomic execution claim, cancellation, stable Done, terminal reference release,
authorized-tool Run composition, and RunHost application cancellation/drain/
restart composition. Execution is synchronous; Run/RunHost create no worker
goroutine, queue, scheduler, persistence, or result/error history.

RunHost Register/Start execute zero Runs; Host.Execute is explicit caller action.
Stop closes admission, cancels active work and waits for cooperative completion.
`pkg/app` remains application lifecycle owner. The CLI has not migrated to
Run/RunHost. This is not a public agent API, agent CLI, or autonomous capability.

## Compatibility

Git comparison against `5d836931216203aeea0737fc54de9e95091a62ef` confirms no
changes to `pkg/manifest`, `pkg/compiler`, `runtime`, `pkg/registry`, `pkg/app`,
`go.mod`, or `go.sum`. Source inspection confirms unchanged:

- Manifest structure/format.
- Package format v1/v2 support and bundle schema v1/v2.
- Integrity schema v2 and signature schema v1.
- Runnable v2 contract and compiler/runtime compatibility boundary.

No data/package migration is introduced. Existing CLI commands are preserved;
the AI command tree is additive. APIs and formats remain pre-stable: this is
not a promise of future format/API stability. Go remains 1.26; dependencies
are unchanged.

## Security and authority boundaries

Network access requires explicit `--allow-network`; `--allow-tools` is a
separate default-false opt-in. The one-tool and two-POST/one-handler bounds above
remain mandatory. Tool requests preserve `store:false`, `stream:false`,
`background:false`, and `parallel_tool_calls:false`; POST #2 uses
`tool_choice:"none"`. No provider conversation state or reasoning replay is
introduced. Terminal reasoning compatibility does not authorize reasoning replay.

The credential name is `FORGE_OPENAI_API_KEY`: process-scoped input, not
persisted by Forge, and never supplied as a real secret in these examples.
Prompt arguments can appear in shell history/process listings. `store:false`
is not a zero-retention guarantee; local cancellation is not a guarantee that
provider processing or billing stopped. Output-token limits are not hard
monetary budgets.

`--diagnostic-stage` is development-only, hidden/default-off, with fixed stage
labels and no raw-response logging; it is not part of normal public grammar.

## Important limitations

This remains a **NON-PRODUCTION PRERELEASE / PRE-STABLE** release. Trusted
native code executes with the invoking user's authority. Forge does not provide:

- Sandboxing, filesystem/network isolation, privilege drop, process-tree
  containment, or CPU/memory/process quotas.
- Persistent trust management, key rotation/revocation, or complete provenance/SBOM.
- Remote registry/distribution or dynamic plugin loading.
- Scheduler, workflow engine, queues, worker pools, or background agent jobs.
- AI memory, durable history, autonomous agents, or multi-agent orchestration.
- Public agent API, Beta readiness, or production readiness.

Existing same-user package mutation, validation-to-execution binding, and
Windows ACL/reparse/share-mode hardening remain disclosed accepted debt.

## CI / validation evidence

Publication source: `85d78143db1b8bcf2f96b79d681895b1a0492642`, tree
`e58a864eb6617811cca1dcb8d2d30bbd72e19298`.
[Strict main CI 34786373253](https://github.com/kaizenforyou91/forge/actions/runs/34786373253)
completed successfully on this exact push/main source, attempt 1, without retry
or waiver. Ubuntu acceptance, Windows acceptance, and Ubuntu race all passed,
including dependency metadata/cleanliness, listing, vet, full tests, and build.

### Historical RR-003 preparation validation

Preparation base: `88c5300e13c92e88b332349943ae17fd91acb501`, tree
`95b2ee6db9fd07b12eb86c2c4e4ee9ac8755261c`.
[Canonical push-main CI 34765766554](https://github.com/kaizenforyou91/forge/actions/runs/34765766554)
is completed/success on that exact head: Ubuntu acceptance, Windows acceptance,
and Ubuntu race all PASS. Acceptance covers dependency metadata/cleanliness,
package listing, vet, full tests and build. The unchanged race command is:

```text
go test -race ./pkg/compiler ./runtime ./internal/cli ./pkg/ai ./internal/aiprovider/openai ./pkg/ai/tool ./internal/agent -count=1
```

RR-003 local offline validation PASS: `git diff --check`, `go list ./...`,
`go test ./... -count=1`, `go vet ./...`, `go build ./...`, and
`go mod tidy -diff`. `go.mod` and `go.sum` remain unchanged. Existing deterministic
agent, AI, tool, OpenAI, CLI, compiler and runtime tests passed, including the
aggregate 96 > 64 regression. Markdown local links/anchors resolve and released
alpha.1 changelog history is unchanged.

At the RR-003 checkpoint, exact-head PR CI was required before integration;
base/local results did not replace it. Local race was not run and was not required.

Historical accepted live evidence is sufficient for this preparation. Git
comparison from accepted C8/C10 source baseline
`ac68a1b3e059f173d9b5c71eadaff9fcffba981f` to preparation HEAD confirms unchanged
`internal/cli`, `pkg/ai`, `internal/aiprovider/openai`, `go.mod`, and `go.sum`.
Wire serialization, HTTP behavior, response decoding, authority, round-trip
execution and diagnostics are unchanged. Accepted text/tool live validation and
runtime identity 3/3 MATCH remain historical evidence, not a new live result.
Any future production-path change requires reopening the freshness decision.

## Historical RR-003 candidate smoke evidence

The smoke used a clean, isolated local clone of the RR-003 branch head before
the documentation commit, outside the canonical repository. Source SHA:
`88c5300e13c92e88b332349943ae17fd91acb501`; source tree:
`95b2ee6db9fd07b12eb86c2c4e4ee9ac8755261c`. This is preparation evidence, not
the final publication identity. The documentation commit cannot embed its own
hash; the draft PR and handoff identify the final RR-003 head separately.

Toolchain: `go version go1.26.5 windows/amd64`. Dependency downloads were disabled
with `GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local`, using existing caches.
All executables, packages and ephemeral test keys were outside the repository.

From that clean clone, the stamped build used the existing linker contract:

```text
go build -trimpath -ldflags "-X github.com/kaizenforyou91/forge/internal/cli.AppVersion=0.4.0-alpha.1 -X github.com/kaizenforyou91/forge/internal/cli.Commit=88c5300e13c92e88b332349943ae17fd91acb501 -X github.com/kaizenforyou91/forge/internal/cli.BuildTime=2026-09-13T16:04:38Z" -o <temp>/forge.exe ./cmd/forge
<temp>/forge.exe version
```

Observed output matched all three supplied fields exactly:

```text
Forge CLI
Version : 0.4.0-alpha.1
Commit  : 88c5300e13c92e88b332349943ae17fd91acb501
Built   : 2026-09-13T16:04:38Z
```

Development defaults in `internal/cli/version.go` remain `dev`, `none`, and
`unknown`. RR-004 later repeated the smoke on the exact publication commit,
as recorded below. No smoke executable is a release asset.

The same temporary stamped binary passed `--help`, `ai --help`, and
`ai prompt --help`. All seven public AI flags were present; `--diagnostic-stage`
was absent. `--allow-tools` remains default false in source and help. This
offline rejection command exited 1 with the network-authorization category:

```text
<forge> ai prompt --provider openai --model offline-smoke --text offline-smoke
ai: authorization denied: --allow-network must be true
```

The missing permission check precedes credential lookup/provider construction
in source; no API-key access or network request occurred.

Clean-clone product smoke commands (all output paths under the isolated temp
directory; the key ID below is test-only):

```text
<forge> validate examples/alpha-app/forge.yaml --profile structural
<forge> validate examples/alpha-app/forge.yaml --profile build
<forge> validate examples/alpha-app/forge.yaml --profile runnable
<forge> build examples/alpha-app/forge.yaml --output <temp>/identity.zip
<forge> inspect <temp>/identity.zip
go build -o <temp>/keygen.exe <temp>/keygen.go
<temp>/keygen.exe <temp>/test-private.pem <temp>/test-public.pem
<forge> build-runnable examples/alpha-app/forge.yaml --signing-key <temp>/test-private.pem --key-id rr003-ephemeral-test --output <temp>/runnable.zip
<forge> inspect <temp>/runnable.zip
<forge> inspect <temp>/runnable.zip --trusted-key <temp>/test-public.pem --key-id rr003-ephemeral-test
<forge> run <temp>/runnable.zip --trusted-key <temp>/test-public.pem --key-id rr003-ephemeral-test
```

`keygen.go` was the standard-library Ed25519 helper from the historical Alpha
workflow, copied only into the temporary directory. Initial `go run` encountered
a Windows executable cleanup lock (`unlinkat ... keygen.exe`); its test keys
were deleted. Building the unchanged helper separately avoids that cleanup
path. No Forge source remediation or workflow edit was made.

All three validation profiles passed. The identity package reported format 1,
bundle schema 1, non-runnable, verified integrity and unsigned status. The
signed package reported format 2, bundle schema 2, `application_executable`,
`windows/amd64`, and `app@v1`. Inspection first reported signed/unverified;
explicit trust then reported `Signature: trusted` and verified signer
`rr003-ephemeral-test`. Trusted execution exited zero with exactly:

```text
Forge Alpha example: OK
```

The temporary clone remained clean. Both ephemeral test keys and helper source
were deleted after the smoke. No repository/user/production key was used.
Canonical worktree ignored inventory remained 487; all four protected ignored
artifacts remained metadata-identical. The protected config key was metadata-only.

## Published alpha.1 relationship

Published `v0.3.0-alpha.1` ultimately resolves to
`5d836931216203aeea0737fc54de9e95091a62ef`. Its GitHub release is published,
draft=false, prerelease=true, with zero uploaded binary assets. That historical
First Alpha excludes later Phase 8/9 additions and is unchanged.

Historical RR-003 branch: `rr003/release-candidate-0.4.0-alpha.1`.
Documentation implementation: `548548804d0c80922bd0c7ee6e33373c6a8c1a26`.
PR #23 merged as `85d78143db1b8bcf2f96b79d681895b1a0492642`, subsequently
selected for publication. The earlier smoke source above remains historical
preparation evidence, not the final release identity.

## Exact-publication RR-004 validation record

RR-004 selected publication topology A: tag the exact integrated RR-003 source
without another pre-publication documentation commit; reconcile documentation
after publication. GitHub Release metadata records the publication event.

The isolated exact-candidate smoke used Go 1.26.5, windows/amd64. Stamped output:

```text
Forge CLI
Version : 0.4.0-alpha.1
Commit  : 85d78143db1b8bcf2f96b79d681895b1a0492642
Built   : 2026-09-14T00:19:08Z
```

CLI help exposed all seven public AI flags, kept diagnostic-stage hidden, and
preserved default-false tools. Missing network authorization failed before
credential lookup/provider work. Structural/build/runnable validation, v1
build/inspect, signed v2 build, untrusted/trusted inspection, and trusted
execution passed with `Forge Alpha example: OK`. Temporary binaries, packages,
isolated checkout, and ephemeral test keys were removed.

Comparison from accepted live baseline
`ac68a1b3e059f173d9b5c71eadaff9fcffba981f` confirmed no production AI path
change. Historical real-provider validation PASS remained sufficient; no fresh
live request was required or made for publication readiness. RR-005 performs
zero live OpenAI calls and no API-key access.

## Final publication record

| Field | Verified value |
|---|---|
| Version / tag | 0.4.0-alpha.1 / v0.4.0-alpha.1 |
| Tag type / object | annotated / `c02b81a94c3c7bead6a7fcf4030ce03cc98af52a` |
| Source commit | `85d78143db1b8bcf2f96b79d681895b1a0492642` |
| Release ID | 388084778 |
| Source-only / assets | YES / 0 |
| Publication status | SUCCESS |
| No force/tag movement | YES |

The Owner authorized publication after RR-004. RR-004-PUB created and pushed
one annotated tag, then created one published prerelease with the approved body.
There was no force, existing tag movement/deletion, release edit, or asset upload.
Historical v0.3.0-alpha.1 remained unchanged.

## Source-only publication surface

Published model: **SOURCE-ONLY GITHUB PRERELEASE**, with zero uploaded binary
assets. Temporary smoke executables are not official distribution artifacts.
Checksums, SBOMs and signatures are not mandatory uploaded assets for this model.
Adding official binary assets would expand platform, provenance, signing and
support obligations and requires a separate authorization gate.

## Rollback/recovery note

Tags are immutable release identities in normal operation. If a candidate is
not approved, do not publish it. Supersede a published prerelease with a new
immutable version rather than moving an old tag. Source rollback may select
an earlier immutable commit/tag but cannot undo provider processing, provider
billing, or native-code side effects already performed. Alpha.1 lacks the new
AI surface. No persistent agent-state migration exists for this release.

## PUBLICATION CHECKLIST — COMPLETED

This checklist records the completed publication transaction and RR-005's
subsequent documentation reconciliation. RR-005 did not precede tag creation;
its documentation changes were integrated through PR #24 and passed strict
push-main CI.

- [x] Exact final candidate main SHA selected.
- [x] Exact tree recorded.
- [x] Target tag absence verified immediately before creation.
- [x] Target GitHub Release absence verified immediately before creation.
- [x] Strict main CI PASS.
- [x] Final exact-candidate version smoke PASS.
- [x] Final documentation reconciled after publication in RR-005.
- [x] CHANGELOG converted/finalized after publication in RR-005.
- [x] Release notes finalized and published.
- [x] Source-only/no-assets policy reconfirmed.
- [x] Owner explicitly authorized tag creation.
- [x] Owner explicitly authorized GitHub prerelease creation.

Historical RR-003/RR-004 collision evidence: local/remote target tags were absent
and the target Release lookup returned 404 before publication. RR-004-PUB
repeated these checks immediately before creating the new identities. They are
historical absence checks, not assertions that the published identities are absent.

## Owner-only actions and subsequent work

RR-003 did not grant publication authorization. The Owner subsequently granted
it explicitly for RR-004-PUB, which completed publication. RR-005 reconciles
documentation only and does not edit the Release, move the tag, or upload assets.
Further releases or assets require separate authorization. Phase 10 architecture
is selected as bounded synchronous sequences; implementation remains
**NOT AUTHORIZED**. The publication identities and historical evidence are unchanged.

## Sources

- [Current README](../README.md)
- [Published changelog](../CHANGELOG.md#040-alpha1---2026-09-14)
- [Roadmap](../ROADMAP.md)
- [ADR-002](architecture/adr/ADR-002-agent-run-ownership.md)
- [AI workflow and accepted evidence](AI_PROMPT_WORKFLOW.md)
- [Historical First Alpha workflow](ALPHA_WORKFLOW.md)
- [Version-stamping source](../internal/cli/version.go)
