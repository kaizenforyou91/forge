# Single-Prompt Execution — Development Workflow

This feature belongs to the development branch `phase8/single-prompt`.
It is **not included in published release `v0.3.0-alpha.1`**, whose tag and
version remain unchanged. This implements one small Phase 8 slice, not all of
the AI Runtime roadmap.

External pilot: **DEFERRED / NON-BLOCKING / NOT RUN**.
Live provider acceptance: **NOT RUN; separate owner approval required**.
The tests described below exercise offline contracts only.

## One explicit prompt

From a binary built from this development checkout, the command grammar is:

```text
forge ai prompt --provider openai --model <model-id> --text "<prompt>" --allow-network [--timeout 30s] [--max-output-tokens 1024]
```

The only provider is exactly `openai`. Provider, model and text are required.
There is no default model, provider alias, fallback or provider routing.
Missing `--allow-network` or `--allow-network=false` fails without a request.

Illustrative command for a future separately approved live run; **not an
instruction to run it during offline acceptance**:

```text
forge ai prompt --provider openai --model gpt-4.1-mini-2025-04-14 --text "Summarize in one sentence: The dummy team agreed to hold a demo on Friday." --allow-network
```

The model is an example, not an access guarantee, default, or allowlist.
Model answers are nondeterministic and are not verified facts.
`--text` can appear in shell history and process arguments. Use dummy,
non-sensitive data for this first slice.

One invocation produces complete text with a final newline, or a classified
error. There is no success banner. Existing final newlines are preserved.
Errors use Forge's existing stderr/exit path: success 0, pure cancellation 130,
timeout and other failures 1. A failed/short output write is a failure; bytes
already accepted by the caller's writer cannot be rolled back.

There are no positional prompts, interactive stdin, `--file`, `--api-key`,
`--endpoint`, `--json`, `--stream`, tool calling, agents, memory, history,
scheduler or background jobs. Model output is plain data and is never passed
to a shell, subprocess, plugin, evaluator or file executor.

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

Only these JSON fields are sent:

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

Authorization is a bearer header. No tools, conversation, previous response,
source files, environment dump or machine identity is included. There is no
automatic input discovery or upload. This client retains environment proxy
selection and system TLS roots, with TLS verification enabled, without changing
global clients, transport, proxy, certificate stores or configuration.

The adapter owns an HTTP/1 transport with keep-alives disabled, a non-replayable
POST body (`GetBody == nil`), and no idempotency header. It issues one `Do`
call per invocation, rejects redirects without following them, and never
retries, polls or falls back. This is a local request-attempt contract, not a
guarantee about provider-side delivery/accounting or network intermediaries.

`--allow-network` is explicit per-invocation consent; organizational egress
permission is still required. A token limit is **not a hard monetary budget**.
Before live acceptance, the owner must approve the account/provider, exact model,
dummy input, number of requests and monetary budget. No key or paid request is
needed for implementation or offline tests.

`store:false` is **not a zero-retention guarantee**. Local cancellation is
**not a guarantee that provider processing or billing stopped**.

## Core API and bounds

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
| Usage | Optional pointer; absent/null stays nil, present counts must be nonnegative integers; output count cannot exceed requested cap |

Each call creates its own timeout context, respects an earlier caller deadline,
validates before calling the provider and checks context again before accepting
success. Providers must honor context and finish owned work before returning.
The coordinator calls synchronously; it cannot forcibly terminate a broken
custom provider and does not hide one behind a goroutine.

Success is a validated `Result` plus nil error. Failure is `Result{}` plus
error, never partial text. Usage is copied to avoid returning a provider-owned
pointer. Input bounds limit small text requests; response bounds limit memory.
They do not guarantee quality, source correctness, a tokenizer-exact input
budget, or production isolation.

## Response and error handling

The adapter accepts exactly one UTF-8 JSON object, with no trailing value or
garbage. It validates required fields and types and ignores additional unused
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
go test ./pkg/ai ./internal/aiprovider/openai ./internal/cli -count=1
go test -race ./pkg/ai ./internal/aiprovider/openai ./internal/cli -count=1
```

The race command requires a supported race/CGO toolchain. Report an unavailable
toolchain as NOT RUN, not PASS. Run the repository's normal format, dependency
cleanliness, list, vet, full tests and build checks for implementation acceptance.

**OFFLINE CONTRACT PASS** means these offline checks have corresponding recorded
evidence. **LIVE PROVIDER ACCEPTANCE PASS** requires a separately approved real
request. Neither fake output nor prior CI establishes live acceptance, hosted
Linux/Windows acceptance, or completion of all Phase 8.

## References and follow-up

Official documentation checked for this implementation on 2026-09-07:

- [Text generation and HTTP endpoint](https://developers.openai.com/api/docs/guides/text)
- [Responses reference](https://developers.openai.com/api/reference/typescript/resources/beta/subresources/responses/methods/create)
- [Structured error codes](https://developers.openai.com/api/docs/guides/error-codes)
- [Provider data controls](https://developers.openai.com/api/docs/guides/your-data)
- [Example model and snapshot](https://developers.openai.com/api/docs/models/gpt-4.1-mini)

`forge build`, `build-runnable`, `inspect` and `run` retain their existing
contracts and package formats. The published First Alpha is preserved.
A later explicitly scoped change should synchronize ROADMAP/CHANGELOG with
accepted development status; neither file is changed by this slice.
