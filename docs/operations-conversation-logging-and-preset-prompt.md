# Conversation Logging And Preset Prompt Operations

PPAP has two operator-only features for production visibility and upstream request shaping:

- `conversation-log`: opt-in full conversation logging with Management API and panel browsing.
- `preset-prompt`: opt-in upstream-only prompt injection for chat-like requests.

Both features are disabled by default. Enable them only in deployments where the operator understands the data and privacy impact.

## Configuration

Start from `config.example.yaml` or `config.docker.example.yaml`.

```yaml
conversation-log:
  enabled: false
  directory: "conversation-logs"
  max-file-size-mb: 16
  max-total-size-mb: 256
  max-entry-bytes: 2097152

preset-prompt:
  enabled: false
  prompt: ""
  max-bytes: 32768

api-key-controls:
  - name: "gpt-prompt-budget"
    api-key: "client-key"
    models:
      - "gpt*"
    max-cost-usd: 30
    preset-prompt:
      enabled: true
      prompt: "operator prompt"
      max-bytes: 32768
```

For Docker, keep the conversation log directory under the mounted `/CLIProxyAPI/data` volume so logs survive container replacement.

## Conversation Logs

When `conversation-log.enabled` is `false`, PPAP does not write full request or response bodies to the conversation log store. Existing application, request, and error logs continue to work as before.

When enabled, each supported request records structured metadata, request and response payload snapshots, status, errors, usage when present, latency, provider, model, and request id. Authorization headers, API keys, cookies, and provider credentials are redacted before storage.

Conversation log storage is bounded by:

- `max-file-size-mb`: rotate each JSONL shard at this size.
- `max-total-size-mb`: delete oldest shards when total storage exceeds this size.
- `max-entry-bytes`: truncate very large per-entry payload fields without changing the client response.

Management access is required. Use the existing management key controls before exposing `/management.html` or `/v0/management/*` outside localhost.

Management API endpoints:

- `GET /v0/management/conversation-logs`
- `GET /v0/management/conversation-logs/tail`
- `GET /v0/management/conversation-logs/{id}`

The bundled management panel exposes the same data in the Logs area under Conversation Logs, including filtering, tailing, detail view, and JSON export.

## Preset Prompt

When `preset-prompt.enabled` is `false`, upstream request payloads are unchanged.

When enabled, PPAP inserts the configured prompt only into the upstream chat-like payload. It supports the OpenAI chat completions shape, OpenAI Responses instructions, Claude system prompts, and Gemini system instructions.

When `api-key-controls[].preset-prompt` is configured for the authenticated client key, that per-key block overrides the global `preset-prompt` block. A per-key block with `enabled: false` disables prompt injection for that key even when the global prompt is enabled.

The injected prompt is not returned to API clients in non-streaming responses, streaming chunks, or normal client-visible metadata. Conversation logs store the original client request and client-facing response, not the private injected prompt. Configuration change summaries redact prompt body changes.

This feature is request shaping, not a hard server-side policy engine. Use smoke tests for the provider/model pair you plan to run, especially if the prompt is meant to enforce an operational policy.

## Verification Checklist

After enabling either feature:

- Confirm `/healthz` returns `200`.
- Confirm `/v1/models` works with the expected production API key.
- Run one non-streaming and one streaming chat request.
- Confirm `/management.html` loads with the expected management key.
- Confirm `GET /v0/management/conversation-logs` and `/tail` return `200` when logging is enabled.
- Inspect one detail record and verify it contains no API key, bearer token, cookie, provider credential, or private preset prompt text.
- Confirm the log directory is under the intended data path and retention is non-zero.

Keep production prompt bodies and management passwords out of README files, issue comments, release notes, and screenshots. Record only feature status, paths, byte length, hashes, and verification results when documenting operations.
