# Forge 0.4.0-alpha.1 AI Prompt Workflow

AI prompt capability is included in [published v0.4.0-alpha.1](https://github.com/kaizenforyou91/forge/releases/tag/v0.4.0-alpha.1),
a source-only GitHub prerelease with zero uploaded assets. It remains
non-production with pre-stable APIs and formats. RR-005 reconciles documentation
after publication; it introduces no new runtime capability.
Phase 8 bounded AI/tool foundation is **CLOSED / PASS**, including accepted
real-provider validation **PASS**. Phase 9 is **CLOSED / PASS — BOUNDED AGENT
EXECUTION LIFECYCLE**, internal/pre-stable and separate from CLI execution.
At the v0.4.0-alpha.1 publication checkpoint, Phase 10 was not yet defined or
authorized. Post-release P10-A0-R1 selected bounded synchronous workflow
composition; implementation remains **NOT AUTHORIZED** and separately gated.
[ADR-003](architecture/adr/ADR-003-bounded-workflow-composition.md) records the
architecture-only decision; P10-A1 integration and strict push-main CI establish
its canonical definition. This does not add a workflow command or change this CLI.

These Phase 8/9 additions are **not included in published `v0.3.0-alpha.1`**.
Its annotated tag and published prerelease remain at release commit
`5d836931216203aeea0737fc54de9e95091a62ef`, with zero uploaded binary assets.
The [Alpha workflow](ALPHA_WORKFLOW.md) describes that historical product boundary.
The original PR #1 text-only checkpoint below is historical evidence; the text
mode contract remains valid within the current two-path CLI.

External pilot: **DEFERRED / NON-BLOCKING / NOT RUN**. RR-005 performs zero
live OpenAI calls and no API-key access. Previously accepted live validation
remained sufficient for publication because the production AI path was unchanged.

## Acceptance evidence

Publication source:
`85d78143db1b8bcf2f96b79d681895b1a0492642`, tree
`e58a864eb6617811cca1dcb8d2d30bbd72e19298`.
[Strict push-main CI 34786373253](https://github.com/kaizenforyou91/forge/actions/runs/34786373253)
completed successfully on that exact SHA: Ubuntu acceptance **PASS**, Windows
acceptance **PASS**, and Ubuntu race **PASS**. Acceptance covers dependency
metadata/cleanliness, listing, vet, full tests, and build. Canonical race covers
`pkg/compiler`, `runtime`, `internal/cli`, `pkg/ai`, `internal/aiprovider/openai`,
`pkg/ai/tool`, and `internal/agent`. This is engineering evidence, not production
certification or permission to make another live request.

Accepted Phase 8 Owner-provided real-provider evidence is **PASS**: authentication,
direct text, a real function proposal, authorized local handler execution,
`function_call_output`, a stateless second provider turn, and final response.
Returned version/commit/build-time matched local runtime identity 3/3. The
[roadmap](../ROADMAP.md#phase-8--ai-runtime) and
[ADR-002](architecture/adr/ADR-002-agent-run-ownership.md) preserve the accepted
scope and history. No new live request was made for publication or RR-005.

### Historical RR-001 checkpoint

RR-001 reconciled documentation against source
`b8e1a2306e98b238b631f1eb7cc373c7ada5ce36`, tree
`5e88273ef7a6753dfe6758f614ba9082f8524853`, with
[CI 34763493724](https://github.com/kaizenforyou91/forge/actions/runs/34763493724)
PASS. That pre-publication checkpoint did not authorize or run new live validation.

### Historical PR #1 checkpoint

[PR #1](https://github.com/kaizenforyou91/forge/pull/1) merged the accepted
text-only implementation and error-sanitization remediation into main at
`bef4874020403e154680f0e682c1b79aed0b937d`. That historical actual merge
commit differs from the earlier synthetic PR merge, with an identical
tree. [Main workflow 34099866050](https://github.com/kaizenforyou91/forge/actions/runs/34099866050)
was triggered by a push to `refs/heads/main`; all three jobs checked out
`main` from `refs/remotes/origin/main` at that actual merge SHA.

| Evidence | Status |
|---|---|
| Implementation and remediation code review | ACCEPTED |
| Offline contract | PASS |
| Main acceptance (ubuntu-latest) | PASS |
| Main acceptance (windows-latest) | PASS |
| Main focused race (ubuntu-latest) | PASS; `pkg/compiler`, `runtime`, `internal/cli`, `pkg/ai`, `internal/aiprovider/openai` |
| Historical local Windows focused race | NOT RUN in the session without CGO/compiler support |
| Historical live provider acceptance at PR #1 | NOT AUTHORIZED / NOT RUN; superseded by accepted Phase 8 validation PASS |

These are historical source-baseline results, not current Phase 8/9 status or
tests rerun by RR-001. The historical local NOT RUN remains unchanged.
Here **offline** means no request to a real AI provider API. Go and CI may
still download normal dependencies; it does not mean the entire engineering
workflow runs without network access.

## One explicit prompt

For v0.4.0-alpha.1, the public command grammar is:

```text
forge ai prompt --provider openai --model <model-id> --text "<prompt>" --allow-network [--allow-tools] [--timeout 30s] [--max-output-tokens 1024]
```

The only provider is exactly `openai`. Provider, model and text are required.
There is no default model, provider alias, fallback or provider routing.
Missing `--allow-network` or `--allow-network=false` fails without a request.

Illustrative text-mode command requiring separate authorization for any new live
run; **not an instruction to run it during RR-005 or offline acceptance**:

```text
forge ai prompt --provider openai --model gpt-4.1-mini-2025-04-14 --text "Summarize in one sentence: The dummy team agreed to hold a demo on Friday." --allow-network
```

The model is an example, not an access guarantee, default, or allowlist.
Model answers are nondeterministic and are not verified facts.
`--text` can appear in shell history and process arguments. Use dummy,
non-sensitive data for these bounded modes.

One invocation produces complete text with a final newline, or a classified
error. There is no success banner. Existing final newlines are preserved.
Errors use Forge's existing stderr/exit path: success 0, pure cancellation 130,
timeout and other failures 1. A failed/short output write is a failure; bytes
already accepted by the caller's writer cannot be rolled back.

There are no positional prompts, interactive stdin, `--file`, `--api-key`,
`--endpoint`, `--json`, or `--stream` flags. There is no agent command, memory,
durable history, scheduler, queue, worker pool, or background agent execution.
Model output never grants execution authority or becomes shell/subprocess code.

## Two execution paths

### Path A — text only (default)

`--allow-tools` omitted or false selects the single-turn `ai.Executor` path.
It makes at most one provider request; `Usage.OutputTokens` cannot exceed the
request's `MaxOutputTokens`. Unsupported function proposals are rejected by
the text decoder. The original single-prompt contract below applies to this path.

### Path B — authorized tool round trip

Both `--allow-network` and explicit `--allow-tools=true` are required. The CLI
supplies a prebuilt invocation-local immutable authority for exactly one
read-only tool, `forge_runtime_info`. It returns version, commit, and build-time
metadata. There is no arbitrary CLI tool registration, filesystem, subprocess,
shell, external-network, or write/mutation handler. Provider output cannot
create authority; admission remains separate from authorized execution.

The direct Phase 8 round trip allows a text answer or one authorized function
call followed by final text: at most two provider POSTs and one handler attempt.
There is no retry, recursive tool loop, or provider-controlled authority.
Each turn independently obeys the requested output-token cap. Final Usage may
aggregate both turns and legitimately exceed `request.MaxOutputTokens`.
The source regression uses a 64-token per-turn cap and 96 aggregate output
tokens and succeeds; this is accounting regression evidence, not a guaranteed
production/provider response. The final aggregate is not routed through
`ai.Executor` for another single-turn cap check. Result validation and defensive
Usage copying remain in force; usage is not printed by this text-output CLI.

Tool requests preserve `store:false`, `stream:false`, `background:false`, and
`parallel_tool_calls:false`. POST #2 uses `tool_choice:"none"`, owned user/call/
output data, and no `previous_response_id`, `conversation`, or `response_id`.
No reasoning replay or `reasoning.encrypted_content` include is sent.
Only the terminal second response supports the accepted leading opaque reasoning
metadata plus message shape; another function call is not a terminal result.

`--diagnostic-stage` is development-only, hidden/default-off, and omitted from
the normal grammar. It reports only a fixed closed failure-stage vocabulary,
never raw responses, reasoning, arguments, or credentials. It does not introduce
Run lifecycle diagnostics.

### Phase 9 separation

The CLI still uses direct Phase 8 composition, not `internal/agent.Run` or
RunHost. Phase 9 supplies internal/pre-stable single-use text/tool Run ownership,
explicit cancellation, stable Done, and cooperative application-host shutdown,
drain, and restart composition. It is not a public `pkg/agent` API, autonomous
planning, multi-agent orchestration, persistence, memory, or workflow engine.

## Credential and network boundaries

The CLI reads **only `FORGE_OPENAI_API_KEY`** for provider authentication, after
request validation, timeout validation, exact provider selection, network
permission and context checks. Supply it as a process-scoped environment value
using an organization-approved secret mechanism. Do not paste a real key into
command arguments, shell history, documentation, the repository, or chat.

The credential must be nonempty printable ASCII without whitespace/control
characters, at most 16 KiB. Forge does not persist it or auto-generate config
keys. Library callers supply it to the adapter constructor. It never enters
the core request, manifest, ZIP, registries, or a native executable child.

The production endpoint is fixed:

```text
POST https://api.openai.com/v1/responses
```

In text mode only, these JSON fields are sent:

```json
{
  "model": "<explicit-model>",
  "input": "<explicit-text>",
  "max_output_tokens": 1024,
  "stream": false,
  "background": false,
  "store": false
}
```

Authorization is a bearer header. Text mode includes no tools, conversation,
previous response, source files, environment dump, or machine identity.
Opted-in tool mode can return the explicitly authorized runtime metadata above;
there is no automatic file/input discovery or upload. This client retains
environment proxy selection and system TLS roots, with TLS verification enabled, without changing
global clients, transport, proxy, certificate stores or configuration.

The adapter owns an HTTP/1 transport with keep-alives disabled, a non-replayable
POST body (`GetBody == nil`), and no idempotency header. Text mode issues one `Do`
call; tool mode issues at most two. It rejects redirects without following them,
and never retries, polls or falls back. This is a local request-attempt contract, not a
guarantee about provider-side delivery/accounting or network intermediaries.

`--allow-network` is explicit per-invocation consent; organizational egress
permission is still required. A token limit is **not a hard monetary budget**.
Any new live validation requires separate Owner authorization for its provider,
model, input, request count, and budget. Accepted historical validation is not
blanket authorization for another call. No key or paid request is needed for
implementation or offline tests.

`store:false` is **not a zero-retention guarantee**. Local cancellation is
**not a guarantee that provider processing or billing stopped**.

## Text-mode core API and shared data bounds

`pkg/ai` uses constructor injection:

```go
type Provider interface {
    Execute(context.Context, Request) (Result, error)
}
```

Call `NewExecutor(provider, timeout)` and `executor.Execute(ctx, request)`.
Core callers must supply a nonnil provider/context, concrete positive timeout
and concrete token limit. **The core never substitutes defaults**.

| Value | Contract |
|---|---|
| Request text | Nonblank valid UTF-8, at most 16,384 bytes; preserved exactly |
| Model | 1–128 ASCII bytes matching `[A-Za-z0-9][A-Za-z0-9._:-]*` |
| Output token limit | 16–2,048; CLI default 1,024 only when flag omitted |
| Timeout | Positive and at most 120 seconds; CLI default 30s only when omitted |
| Response headers | 16 KiB ceiling in the owned transport |
| Response body | 1 MiB for success and error bodies; limit+1 overflow detection |
| Aggregated result text | Nonblank valid UTF-8, at most 64 KiB |
| Terminal controls | Reject C0 except TAB/LF, DEL and all C1; CR is rejected too |
| Usage | Optional pointer; absent/null stays nil, present counts must be nonnegative integers; text-mode output count cannot exceed requested cap; tool aggregate may exceed it as described above |

Each call creates its own timeout context, respects an earlier caller deadline,
validates before calling the provider and checks context again before accepting
success. Providers must honor context and finish owned work before returning.
The executor calls synchronously; it cannot forcibly terminate a broken
custom provider and does not hide one behind a goroutine.

Success is a validated `Result` plus nil error. Failure is `Result{}` plus
error, never partial text. Usage is copied to avoid returning a provider-owned
pointer. Input bounds limit small text requests; response bounds limit memory.
They do not guarantee quality, source correctness, a tokenizer-exact input
budget, or production isolation.

## Text-mode response and shared error handling

The text-mode adapter accepts exactly one UTF-8 JSON object, with no trailing
value or garbage. It validates required fields and types and ignores additional unused
metadata. It reads `output` in order; top-level `output_text` is not a wire
authority.

Only `completed` responses with null `error` and `incomplete_details` can
succeed. Text comes from `output_text` parts in completed assistant messages,
concatenated in order without invented separators. Unsupported items, including
tool calls, fail. Refusal fails explicitly. Incomplete, queued and in-progress
responses are errors and are never polled.

Bodies are read within bounds and closed on owned response paths. Read or close
failure cannot become success. There is no unlimited draining for reuse.
Raw provider errors, bodies and transport error chains are not exposed.
Messages never include prompt, key or Authorization. A literal credential
reflected into successful text/model fields is rejected as well; this is not
a general encoded-secret/DLP detector.

Errors support `errors.Is`:

- `ErrInvalidRequest`, `ErrAuthentication`, `ErrAuthorization`.
- `ErrRateLimited`, `ErrQuotaExceeded`, `ErrTransport`.
- `ErrMalformedResponse`, `ErrResponseTooLarge`, `ErrIncompleteResponse`.
- `ErrRefused`, `ErrProvider`.
- `context.Canceled` and `context.DeadlineExceeded`.

Sanitization preserves unknown siblings in joined/nested errors as `ErrProvider`
without retaining raw messages or causes. Pure cancellation remains exit 130;
mixed cancellation remains failure exit 1. Ordinary wrappers do not invent an
extra failure, and repeated sanitization preserves the safe categories.

HTTP 401 maps to authentication; 403 to authorization. For 429, the exact
structured `error.code` values `insufficient_quota`,
`credit_balance_exhausted`, `organization_spend_limit_exceeded`,
`project_spend_limit_exceeded` and `organization_usage_limit_exceeded` map
to quota. Other 429 responses map to rate limit. Human-readable messages are
not searched for quota substrings. Other non-200 statuses map to provider failure.
Local transport/deadline/read/close failures can take precedence over HTTP status.

## Offline validation

No test reads an actual environment key or requests the provider internet.
Tests use fake providers, dummy credentials, injected transports, and explicitly
routed loopback servers with no production fallback. Fake providers are not a
user-facing CLI mode. Cancellation/deadline unit tests use channels and
`testing/synctest` virtual time, not sleeps or retries to hide flakiness.

```text
go test ./pkg/ai ./pkg/ai/tool ./internal/aiprovider/openai ./internal/agent ./internal/cli -count=1
```

Hosted `race (ubuntu-latest)` remains canonical and executes:

```text
go test -race ./pkg/compiler ./runtime ./internal/cli ./pkg/ai ./internal/aiprovider/openai ./pkg/ai/tool ./internal/agent -count=1
```

Local race is not required for RR-005. Report an unavailable local race/CGO
toolchain as NOT RUN, not PASS. Run the repository's normal format, dependency
cleanliness, list, vet, full tests and build checks for implementation acceptance.

**OFFLINE CONTRACT PASS** means these offline checks have corresponding recorded
evidence. The accepted **LIVE PROVIDER ACCEPTANCE PASS** above comes from the
separately authorized Phase 8 real-provider validation, not fake test output or
hosted CI. RR-005 performs no live call. Phase 8 and Phase 9 closure are recorded
in the roadmap; offline checks alone do not create new live acceptance evidence.

## References and follow-up

Historical reference links checked for the original text-only implementation
on 2026-09-07 (not a new provider-specification verification by RR-001):

- [Text generation and HTTP endpoint](https://developers.openai.com/api/docs/guides/text)
- [Responses reference](https://developers.openai.com/api/reference/typescript/resources/beta/subresources/responses/methods/create)
- [Structured error codes](https://developers.openai.com/api/docs/guides/error-codes)
- [Provider data controls](https://developers.openai.com/api/docs/guides/your-data)
- [Example model and snapshot](https://developers.openai.com/api/docs/models/gpt-4.1-mini)

`forge build`, `build-runnable`, `inspect` and `run` retain their existing
contracts and package formats. The published First Alpha is preserved.
Publication status is aligned in the [README](../README.md),
[ROADMAP](../ROADMAP.md), and [0.4.0-alpha.1 changelog](../CHANGELOG.md#040-alpha1---2026-09-14).
RR-004-PUB is **CLOSED / PASS — PUBLISHED**; RR-005 reconciles documentation
without changing the immutable release. Phase 10 synchronous-sequence architecture
is selected; implementation remains **NOT AUTHORIZED**. Any new live call also
requires separate authorization. No tool/provider expansion, autonomous agents,
AI memory/durable history, persistence, workflow, scheduler, queues/workers,
public agent API, Beta/production readiness, sandboxing, process-tree containment,
or persistent trust rotation/revocation is claimed.
