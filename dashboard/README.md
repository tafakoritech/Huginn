# Huginn Admin Dashboard

Management dashboard for **Huginn**, Tafakor Pardazan's pre-CRM sales intelligence
system. Used by `super_admin` and `sales_development_manager` roles to import
market data, review extracted leads, assign work, monitor mandatory contact
reports, qualify leads and export them to the Muninn CRM.

Built with **React + TypeScript + Vite**, **TanStack Query / Table / Virtual**,
**React Hook Form + Zod**, and a hand-rolled, RTL-aware SVG/CSS chart layer.

---

## Table of contents

- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Scripts](#scripts)
- [Environment variables](#environment-variables)
- [Mock development mode](#mock-development-mode)
- [Docker](#docker)
- [Backend API contract](#backend-api-contract)
- [Project structure](#project-structure)
- [Implemented routes](#implemented-routes)
- [Design system](#design-system)
- [Internationalization & RTL](#internationalization--rtl)
- [Testing](#testing)
- [Security notes](#security-notes)
- [Known limitations](#known-limitations)

---

## Requirements

- Node.js ≥ 20
- npm ≥ 10
- (Optional) Docker ≥ 24 for container builds

## Quick start

```bash
cd dashboard
npm install
cp .env.example .env.local   # adjust VITE_API_BASE_URL if needed
npm run dev
```

Open http://localhost:5173 and sign in. With the real backend running on
`:8080` (see the Huginn Go `backend/`), the dashboard talks to it directly.

## Scripts

```bash
npm run dev        # Vite dev server (:5173)
npm run build      # type-check (tsc -b) + production build to dist/
npm run preview    # preview the production build locally
npm run lint       # ESLint (flat config, zero warnings allowed)
npm run test       # Vitest component + integration tests (jsdom)
npm run test:watch # watch mode
```

## Environment variables

| Variable               | Default                       | Description                                              |
| ---------------------- | ----------------------------- | -------------------------------------------------------- |
| `VITE_API_BASE_URL`    | `http://localhost:8080/api/v1`| Base URL of the Huginn backend API. Never hardcoded.     |
| `VITE_APP_NAME`        | `Huginn`                      | Product name shown in the UI.                            |
| `VITE_DEFAULT_LOCALE`  | `fa`                          | `fa` (Persian RTL, default) or `en` (English LTR).       |
| `VITE_DEFAULT_THEME`   | `light`                       | `light` or `dark`. Switchable at runtime and persisted.  |
| `VITE_USE_MOCK_API`    | `false`                       | Use the explicit in-memory dev mock instead of the API.  |

**No secrets ever belong in frontend environment variables.** API keys are
managed server-side only.

## Mock development mode

When the backend is unavailable, set `VITE_USE_MOCK_API=true`. This activates a
clearly separated mock layer under `src/mocks/` that implements the exact same
`ApiEndpoints` interface with realistic, deterministic data. Production code
never contains hardcoded demo records; the mock is a build-time/run-time opt-in
only.

```bash
VITE_USE_MOCK_API=true npm run dev
```

## Docker

Build and run the dashboard as a self-contained static site served by nginx:

```bash
docker build -t huginn-dashboard ./dashboard
docker run -d -p 8081:80 --name huginn-dashboard huginn-dashboard
```

The nginx image proxies `/api/` to the backend, so build with the same-origin
API path and the container needs no backend URL baked in:

```bash
docker build \
  --build-arg VITE_API_BASE_URL=/api/v1 \
  -t huginn-dashboard ./dashboard

docker run -d -p 8081:80 \
  -e BACKEND_UPSTREAM=http://backend:8080 \
  -e DASHBOARD_PORT=80 \
  --network huginn-net \
  --name huginn-dashboard huginn-dashboard
```

Environment knobs:

- `BACKEND_UPSTREAM` — where nginx forwards `/api/` (default `http://backend:8080`).
- `DASHBOARD_PORT` — nginx listen port (default `80`).

For docker-compose with the backend, add a service:

```yaml
dashboard:
  build: ./dashboard
  ports: ["8081:80"]
  environment:
    BACKEND_UPSTREAM: http://backend:8080
  depends_on: [backend]
```

The production image runs **nginx only** — the Vite dev server is never used in
production.

## Backend API contract

A single centralized client in `src/api/` defines the REST contract consumed by
the dashboard. Routes follow the domain model in `backend/migrations/001_initial_schema.sql`.

| Area          | Endpoints                                                                                 |
| ------------- | ----------------------------------------------------------------------------------------- |
| Auth          | `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /me`                  |
| Dashboard     | `GET /dashboard` (KPI metrics + by_* maps), `GET /dashboard/funnel`, `GET /dashboard/qualification-by-source`, `GET /dashboard/agent-performance`, `GET /dashboard/leads-by-city`, `GET /dashboard/rejection-reasons`, `GET /dashboard/lead-trend?period=day|week|month` |
| Leads         | `GET /leads`, `GET/PATCH /leads/:id`, `/leads/:id/contacts`, `/leads/:id/evidence`, `/leads/:id/score`, `/leads/:id/timeline`, `/leads/:id/interactions`, `/leads/:id/reports`, `/leads/:id/followups`, `/leads/:id/notes`, `POST /leads/assign`, `/leads/:id/qualify`, `/leads/:id/disqualify`, `/leads/:id/export-muninn`, `/leads/:id/archive` |
| Imports       | `GET /imports`, `POST /imports`, `GET /imports/:id`, `POST /imports/:id/files`, `GET /imports/:id/documents` |
| Documents     | `GET /documents`, `GET /documents/:id`, `GET /documents/:id/candidates`                   |
| Duplicates    | `GET /duplicates`, `POST /duplicates/:id/merge\|distinct\|later`                          |
| Assignments   | `GET /assignments/overview`, `GET /assignments/employees`, `POST /assignments`, `POST /assignments/round-robin` |
| Agents        | `GET /agents`, `GET /agents/:id`                                                          |
| Reports       | `GET /reports`, `GET /reports/:id`, `POST /reports/:id/override`                          |
| Follow-ups    | `GET /followups?view=today\|overdue\|upcoming\|completed`, `POST /followups/:id/complete` |
| Users         | `GET/POST /users`, `PATCH /users/:id`, `/users/:id/activate\|deactivate`, `/users/:id/reset-password`, `GET /users/:id/sessions`, `DELETE /users/:id/sessions/:sid` |
| Settings      | `/settings/scoring`, `/settings/qualification`, `/settings/llm`, `/settings/llm/test`, `/settings/muninn`, `/settings/system` |
| Audit         | `GET /audit`                                                                              |
| Misc          | `GET /notifications`, `POST /notifications/read`, `GET /search`, `GET /health` (root, no prefix), `GET /jobs/pending` |

### Error handling

Errors use a machine-readable envelope — `{ "error": { code, message, details } }`
— that drives the UI. Codes such as `MANDATORY_REPORT_PENDING`,
`INVALID_STATE_TRANSITION` and `DUPLICATE_EXPORT` map to contextual Persian
messages. The HTTP client implements:

- automatic `Authorization: Bearer` injection,
- a single-flight `401 → refresh` flow (no refresh storms) that sends the
  refresh token via the backend's `X-Refresh-Token` header,
- logout on unrecoverable auth failure,
- network-failure detection and friendly messages.

The Backend is the single source of truth for the API contract — every
dashboard request in `src/api/real.ts` maps to an endpoint registered in
`backend/internal/httpapi/router.go`. Run both together and the dashboard talks
to the real Huginn API; `VITE_USE_MOCK_API=true` swaps in the isolated dev mock
only.

## Project structure

```
dashboard/
├── src/
│   ├── api/            # typed REST client, query keys, endpoint interface (real + mock)
│   ├── auth/           # AuthProvider, protected routes, permission map
│   ├── components/
│   │   ├── charts/     # RTL-aware funnel, bar, line, donut, qualified-source charts
│   │   ├── layout/     # AppShell, Sidebar, TopBar, GlobalSearch, Notifications
│   │   └── ui/         # Button, Badge, Field, Dialog, Drawer, DataTable, states…
│   ├── i18n/           # fa/en dictionaries + locale/dir provider
│   ├── mocks/          # explicit dev mock (env-gated)
│   ├── pages/          # dashboard, leads, imports, documents, duplicates,
│   │                   # assignments, agents, reports, followups, settings, audit
│   ├── styles/         # design tokens (TP light/dark) + component CSS
│   ├── test/           # vitest test helpers + component/integration tests
│   ├── theme/          # light/dark provider with persistence
│   ├── utils/          # formatting, error mapping
│   └── config.ts       # typed env access
├── Dockerfile          # multi-stage build → nginx
├── nginx.conf          # SPA fallback + /api proxy
└── vite.config.ts      # dev + vitest configuration
```

## Implemented routes

| Route                    | Page                              |
| ------------------------ | --------------------------------- |
| `/login`                 | Sign-in                           |
| `/dashboard`             | KPI dashboard + management charts |
| `/imports`               | Import batch list                 |
| `/imports/new`           | Create import (drop/paste)        |
| `/imports/:id`           | Batch detail, progress, timeline  |
| `/documents`             | Source document list              |
| `/documents/:id`         | Document detail, text, candidates |
| `/leads`                 | Lead table (server-side pagination, filters, URL state) |
| `/leads/qualified`       | Qualified leads + Muninn export   |
| `/leads/:id`             | Lead profile (9 tabs)             |
| `/duplicates`            | Duplicate review + merge          |
| `/assignments`           | Manual / bulk / round-robin       |
| `/agents` / `/agents/:id`| Agent performance                 |
| `/reports` / `/reports/:id`| Mandatory contact reports       |
| `/followups`             | Today / overdue / upcoming / done |
| `/settings/users`        | Users, roles, sessions            |
| `/settings/scoring`      | Scoring weights                   |
| `/settings/qualification`| Qualification requirements        |
| `/settings/llm`          | LLM provider (no secrets shown)   |
| `/settings/muninn`       | Muninn connection                 |
| `/settings/system`       | System info                       |
| `/audit`                 | Audit log with JSON diffs         |

Lead table state (page, sort, status, city, score, search, …) lives in the URL
so managers can share filtered views, e.g. `/leads?status=QUALIFIED&score_min=70`.

## Design system

Inspired by Adobe Spectrum principles: clear hierarchy, consistent spacing,
accessible focus states, dense but readable enterprise tables, excellent
keyboard navigation. No CSS framework is used — the UI is built on semantic
**CSS custom properties** defined in `src/styles/tokens.css` with two full
Tafakor Pardazan palettes:

- **TP Dark** — `#021C25` background, `#0E3338` cards, `#165A54` borders,
  `#69B437` primary green, `#91DA73` highlight.
- **TP Light** — `#F5F9F7` background, white cards, `#69B437` brand, semantic
  success/info/warning/error tokens.

Themes are switchable from the header and persisted in `localStorage`. The
Huginn raven identity is used subtly (logo, empty states) — the product stays a
serious enterprise tool.

## Internationalization & RTL

Persian is the default locale and RTL is first-class — never faked with ad-hoc
CSS overrides:

- `document.dir` / `document.lang` are set from the active locale.
- Sidebar, tables, pagination, forms, dialogs, menus, tabs, timelines and
  charts all render correctly in RTL (charts mirror the x-axis in RTL).
- Persian digits are used for metrics while phone numbers, URLs, IDs and code
  values stay readable in Latin digits.
- Typography uses a web-safe Persian-friendly stack (`Vazirmatn` via CDN with
  system fallbacks — no proprietary fonts are packaged).

## Testing

`npm run test` runs 28 component/integration tests (Vitest + Testing Library)
covering authentication, theme switching, RTL/language switching, lead filters,
lead detail (evidence + score tabs), duplicate review + merge, import creation,
mandatory report rendering (incl. manager override), Muninn export states and
role-based permission gating. Tests run against the explicit mock layer; no
network is required.

## Security notes

- Frontend permissions are **never** the source of truth — actions are still
  gated server-side — but unavailable actions are hidden per the backend RBAC
  map (mirroring `backend/internal/rbac/rbac.go`).
- Untrusted imported content is sanitized/plain-text rendered; no raw HTML is
  injected.
- Tokens, API keys, passwords and internal secrets are never displayed or
  committed. LLM/API keys are write-only in settings.

## Known limitations

- The dashboard consumes the REST contract defined in `src/api/interface.ts`;
  the Go backend's HTTP layer is being built out in parallel, so endpoints that
  are not yet registered server-side will surface as backend-unavailable errors
  (handled with friendly messages and retry).
- Charts are custom-rendered (SVG/CSS) to guarantee RTL correctness and keep
  the bundle lean; very large time-series datasets should be aggregated
  server-side.
- Upload/processing progress is driven by the batch status endpoint (polled
  while `PROCESSING`); per-file byte progress for large multi-file uploads is
  staged by chunk.
- Sessions inspector lists active sessions; full "all sessions" management is
  delegated to the user detail drawer.
