# Panel Provenance

This directory contains a vendored copy of the upstream
[Cli-Proxy-API-Management-Center](https://github.com/router-for-me/Cli-Proxy-API-Management-Center)
panel, rebranded as the Playful Proxy API Panel (PPAP) and extended with
PPAP-specific pages and configuration sections.

## Upstream reference

- Upstream repository: `router-for-me/Cli-Proxy-API-Management-Center`
- Vendored tag: `v1.18.2`
- Vendored commit: `79589155` (`feat: add passthrough mode for disable image generation functionality and update related tests`)
- Vendored on: 2026-07-12
- Vendored by: PPAP maintainers

## Re-vendor checklist

Whenever a newer upstream tag is pulled in, replay the modifications
below before landing the update. Grep for `PPAP` or `Playful Proxy API`
across `src/` to double-check nothing regressed.

### Branding replacements

- `package.json`: `name` = `ppap-management-panel`, `version` follows
  `<upstream-version>-ppap.<n>`.
- `index.html`: `<title>` = `Playful Proxy API Panel`.
- `src/components/layout/MainLayout.tsx`: `fullBrandName` constant is
  `Playful Proxy API Panel`.
- `src/assets/logoInline.ts`: PPAP wordmark placeholder (SVG or reused
  upstream inline). No proprietary logo asset ships with this fork.
- `src/i18n/locales/*.json`: `CLI Proxy API` / `CLIProxyAPI` / `CPAMC`
  / `Cli Proxy` → `Playful Proxy API` / `PPAP`. Technical tokens such
  as `Codex`, `OAuth`, `MCP`, `Gemini`, `Vertex`, `Anthropic`, `xAI`,
  `Kimi`, `Antigravity` remain unchanged.
- GitHub links point at `daishuge/playful-proxy-api-panel`.

### Feature removals

- Upstream "sponsor" providers are pruned: `apikeyFun`, `code0`,
  `fennoAI`, `qiniuCloud`, `claudeApi`.
- The `/quick-start` route and its promo cards on the dashboard are
  removed together with the sponsor providers.

### Feature additions

- Conversation Logs viewer page (`src/pages/ConversationLogsPage.tsx`,
  service `src/services/api/conversationLogs.ts`, nav entry under
  `observe`).
- PPAP-specific ConfigPage sections: Preset Prompts, Per-Key Controls,
  Immersive Translate Guard, Upstream Concurrency Gate, Log Controls.
- Codex provider sheet: fast mode, thinking aliases, service tier
  fields feeding `serializeProviderKey()` in
  `src/services/api/providers.ts`.
- Dashboard tiles for PPAP-specific per-model / per-key aggregates.

### Backend contract

Every request in `src/services/api/*` maps to a matching route on the
PPAP backend (`internal/api/server.go`). Adapter code is intentionally
zero. Confirm this each re-vendor by grepping upstream services and
diffing against the backend router before shipping.
