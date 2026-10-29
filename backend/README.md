# Huginn Backend

**Market Intelligence + Lead Discovery + Lead Qualification + First Contact Platform**

Huginn is the pre-CRM and sales-intelligence platform of **Tafakor Pardazan**.
It ingests raw market data, extracts potential leads with an LLM, deduplicates
and scores them, assigns them to sales-development agents, prepares first
contact actions, enforces mandatory contact reports, qualifies leads, and
transfers **only qualified leads** to **Muninn CRM**.

Huginn is **not** a CRM. It ends where Muninn begins (qualified sales
opportunities, pipeline, negotiations, contracts).

---

## Architecture

Modular monolith with asynchronous workers. Two binaries share the same domain
packages:

```
cmd/
    api/        # REST API (huginn-api)
    worker/     # RabbitMQ consumers (huginn-worker)
    cli/        # migrations + seed (huginn-cli)

internal/
    auth/          JWT access tokens, rotating refresh tokens, sessions, RBAC
    rbac/          roles + permission map
    users/         user management
    imports/       import batches, documents, LLM extraction service
    documents/     Apache Tika extraction
    leads/         lead model, strict state machine, repository, work queue
    contacts/      contact model + normalization
    normalize/     Iranian phone + email normalization
    evidence/      per-field extraction evidence
    duplicates/    deterministic + pg_trgm fuzzy dedup, merge
    scoring/       deterministic explainable scoring
    qualification/ configurable qualification rules, disqualification reasons
    interactions/  interaction records
    reports/       mandatory contact reports + manager override
    followups/     scheduled follow-ups
    messaging/     contact actions, suggested messages, deep links, vCard
    muninn/        CRM export boundary, outbox pattern, idempotency
    timeline/      chronological lead timeline
    notes/         notes with edit history
    audit/         immutable audit log
    dashboard/     manager + agent KPIs
    llm/           OpenAI-compatible provider, structured output, repair, audit
    queue/         RabbitMQ producer/consumer with DLX + backoff
    storage/       MinIO object storage with presigned URLs
    cache/         Redis cache, locks, rate limiting
    database/      pgx pool + migrations
    httpapi/       chi router, middleware, handlers, consistent error format
    worker/        job handler wiring
    seed/          development seed data
migrations/        SQL migrations (pg_trgm enabled)
api/openapi.yaml   OpenAPI 3.x specification
docker/            Dockerfile + prometheus config
```

### Data flow

```
Raw Information → Import Batch → MinIO → RabbitMQ → Worker → Tika
  → Minimal lossless cleanup (svg/\@/zero-width/digit artifacts removed)
  → LLM structured extraction (LLM is the authority for contact association)
  → Deterministic validation + canonical normalization
  → Lead creation → Dedup → Deterministic scoring → Assignment
  → Contact action (prepared, never sent) → Mandatory report → Follow-ups
  → Qualification (configurable rules) → Outbox → Muninn CRM (idempotent)
```

**Contact extraction architecture:** the LLM is the sole authority for semantic
extraction and company-to-contact association. Deterministic logic only (a)
cleans scrape artifacts so explicit data stays visible to the model, (b)
validates and canonically normalizes LLM-extracted phone/email, and (c) emits a
warning (`candidate may be missing source phone/email`, `action=recovery_signal_only`)
when the cleaned source block appears to contain contact data the LLM omitted.
Deterministic detection never assigns or overrides contact values.

Backfill (`make backfill-contacts`) is LLM-driven: for leads missing contact
data it re-sends the company's source block to the LLM and fills **only empty**
fields from the LLM's output — never overwriting manual corrections, never
creating duplicate leads.

---

## Quick start

```bash
cp .env.example .env
# set LLM_BASE_URL + LLM_API_KEY in .env (OpenAI-compatible, default model deepseek-v4-flash)

docker compose up -d            # postgres, redis, rabbitmq, minio, tika
make migrate-up                 # apply SQL migrations
make seed                       # dev users + sample data
make dev                        # run huginn-api locally (or: make worker)
```

`docker compose up` also builds `huginn-api` and `huginn-worker` containers.
Optional observability: `docker compose --profile observability up -d`.

### Development credentials (seed only — never production)

| username | password      | role                       |
|----------|---------------|----------------------------|
| admin    | Admin@12345   | super_admin                |
| manager  | Manager@12345 | sales_development_manager  |
| agent1   | Agent@12345   | sales_agent                |
| agent2   | Agent@12345   | sales_agent                |

---

## Make targets

```bash
make help          # list all targets
make dev           # run API locally
make worker        # run worker locally
make build         # build bin/huginn-api, bin/huginn-worker, bin/huginn-cli
make test          # unit tests
make lint          # go vet (+ staticcheck if installed)
make migrate-up    # apply migrations
make migrate-down  # roll back the last migration
make migrate-status
make seed          # development seed data
make logs          # tail api + worker logs
make compose-up    # start infrastructure
make compose-down
```

> `make` targets source `.env` automatically. Running binaries directly requires
> the environment variables to be exported.

---

## Environment variables

All configuration is via environment variables (see `.env.example`). Key ones:

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | API port |
| `POSTGRES_*` | `huginn` | PostgreSQL connection |
| `REDIS_ADDR` | `localhost:6379` | Redis (cache/locks/rate limit) |
| `RABBITMQ_URL` | `amqp://huginn:huginn@localhost:5672/` | Job queue |
| `MINIO_ENDPOINT` / `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` | `huginn` | Object storage |
| `TIKA_BASE_URL` | `http://localhost:9998` | Document extraction |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | dev defaults | **change in production** |
| `LLM_BASE_URL` | — | OpenAI-compatible base URL |
| `LLM_API_KEY` | — | Provider key |
| `LLM_MODEL` | `deepseek-v4-flash` | Model name |
| `LLM_TIMEOUT_SECONDS` | `60` | LLM call timeout |
| `QUALIFICATION_SCORE_THRESHOLD` | `70` | Minimum score for qualification |
| `MUNINN_BASE_URL` / `MUNINN_API_KEY` | — | Muninn CRM API |
| `MAX_UPLOAD_BYTES` | `26214400` | Max upload size |
| `RATE_LIMIT_PER_MIN` | `300` | Per-IP rate limit |

---

## Security & privacy model

- **LLM safety**: imported documents are treated as untrusted data. The system
  prompt forbids following in-document instructions and fabricating facts.
  Output is validated against a strict schema with a repair retry limit;
  malformed output is never accepted silently.
- **No automatic sending**: contact actions only *prepare* a message and return
  deep links. Huginn never reads the employee's private contacts, messages, or
  mailboxes. Contacts are imported via vCard using OS-level Intents.
- **Mandatory reports**: a completed contact action requires a report. The
  backend enforces this (`MANDATORY_REPORT_PENDING`); only a manager override
  with a reason (audited) can void it.
- **Secrets**: passwords are bcrypt-hashed, refresh tokens stored hashed
  (sha256) and rotated, JWT access tokens are short-lived, and API keys/LLM
  prompts are never logged by default (`LLM_PERSIST_PROMPTS` off).
- **Audit**: user changes, merges, assignments, qualification/disqualification,
  manager overrides, and Muninn exports are immutable audit records.
- **MinIO** is not exposed publicly; downloads use presigned URLs.

---

## API

OpenAPI 3.x spec: [`api/openapi.yaml`](api/openapi.yaml).

REST under `/api/v1`. Errors use a machine-readable envelope:

```json
{
  "error": {
    "code": "MANDATORY_REPORT_PENDING",
    "message": "A previous interaction report must be completed.",
    "details": { "interaction_id": "..." }
  }
}
```

Lead detail responses include `permissions` and `workflow` hints so Android,
PWA and admin clients never reimplement business rules.

Main endpoints:

```
POST /auth/login | /auth/refresh | /auth/logout
GET  /me | /me/work-queue | /me/pending-reports
POST /users (manager) | GET /users | PATCH /users/{id}
POST /imports | GET /imports | GET /imports/{id}
POST /imports/{id}/queue | POST /imports/{id}/documents | POST /imports/{id}/raw-text
GET  /documents/{id}
GET  /leads | POST /leads | GET /leads/{id} | PATCH /leads/{id}
GET  /leads/{id}/timeline | /duplicates | /notes | /contacts
GET  /leads/{id}/contacts/{cid}/vcard
POST /leads/{id}/assign | /assign-bulk | /merge | /contact-actions | /followups
POST /leads/{id}/qualify | /disqualify | /export/muninn | /tags | /archive
POST /interactions/{id}/report | /open | /void (manager)
GET  /dashboard | /dashboard/me
GET  /health | /ready | /metrics
```

---

## Database

Source of truth is PostgreSQL. Migrations are explicit SQL files in
`migrations/` (`pg_trgm` is enabled for fuzzy dedup). No ORM auto-migration.

Key tables: `users`, `sessions`, `audit_log`, `import_batches`,
`source_documents`, `leads`, `lead_evidence`, `contacts`, `possible_duplicates`,
`tags`/`lead_tags`, `assignments`, `interactions`, `contact_reports`,
`follow_ups`, `notes`/`note_edits`, `score_history`, `disqualifications`,
`timeline_events`, `lead_status_history`, `llm_runs`, `crm_exports`,
`outbox_events`.

---

## Testing

```bash
make test                                   # unit tests (no infra required)
HUGINN_INTEGRATION=1 go test ./internal/integration/...   # against live infra
```

Unit coverage: Iranian phone normalization, email normalization, deterministic
scoring, qualification rules, lead state machine, mandatory-report enforcement,
deep-link generation, LLM JSON validation + hallucination-protection behavior.

Integration coverage (needs `docker compose up -d`): auth lifecycle (login →
refresh rotation → reuse rejection → logout), lead state transitions, mandatory
report default, MinIO roundtrip + presigning, Redis, Muninn export idempotency.

---

## Known limitations

- `make migrate-down` removes the schema-version marker; forward-only SQL
  migrations are not automatically reversed.
- LLM message generation falls back to a deterministic template when the
  provider is unavailable or returns invalid output.
- Call-scenario generation also falls back to a deterministic template when the
  LLM fails, but the degradation is **explicit**, never silent: the response
  carries `generation_status: "degraded"`, `fallback_used: true` and a
  `generation_error` reason (e.g. `value_proposition_empty`). The endpoint keeps
  returning HTTP 200 so the agent still receives a scenario before a call, but
  clients MUST read `generation_status`/`fallback_used` and never present
  degraded template content as an AI-generated scenario. A personalized LLM
  generation is `generation_status: "generated"`, `fallback_used: false`.
- DeepSeek's `deepseek-v4-flash` rejects `response_format: json_schema`
  (HTTP 400 "This response_format type is unavailable now"), so structured
  output uses `json_object` mode (`LLM_STRUCTURED_OUTPUT_MODE=json_object`) with
  an explicit JSON contract in the prompt and robust structural validation on
  the parsed result.
- The work-queue priority weights are fixed in SQL; configurable weights are a
  future enhancement.
- Email sending is prepared but no SMTP transport exists (by design — Huginn
  opens the user's mail client).

## Next recommended phase

1. Wire the `lead.score`, `lead.dedupe` and `message.generate` RabbitMQ jobs as
   asynchronous consumers (currently synchronous within processing flows).
2. Add pagination/totals API for timeline, notes, and interactions.
3. Expose `report_status` blocking configurably and add a manager dashboard for
   report compliance.
4. Add PDF page-number preservation metadata from Tika for stronger evidence.
5. Add CI (GitHub Actions) running unit + integration tests against ephemeral
   services.
