# Huginn — Android

Official Android application for **Huginn**, Tafakor Pardazan's pre-CRM sales
lead intelligence and qualification platform. The app serves daily lead work
for `sales_agent` and `sales_development_manager` roles: lead viewing, calls,
messaging, notes, contact reports, follow-ups and qualification.

```
android/
├── app/
│   ├── src/main/java/ir/tafakoritech/huginn/
│   │   ├── core/        (api, auth, database, datastore, design, external, locale,
│   │   │                  navigation, notifications, network, permissions, theme, util)
│   │   ├── data/        (model DTOs, repositories)
│   │   ├── feature/     (auth, app gate, home, leads, lead_detail, messaging,
│   │   │                  reports, followups, qualification, profile, settings, splash)
│   │   └── work/        (WorkManager workers)
│   ├── src/test/        (JVM unit tests)
│   └── src/androidTest/ (Compose UI tests)
├── build.gradle.kts
├── settings.gradle.kts
├── gradle.properties
├── gradle/libs.versions.toml
└── README.md
```

---

## 1. Requirements

- **Android Studio** (current stable, e.g. Narwhal+ / 2026.x) with AGP 9.x support
- **JDK 17+** (the build's Kotlin toolchain targets JVM 11 bytecode)
- **Android SDK**: platform 37, build-tools 36.x
- Gradle 9.5 (wrapper included; `./gradlew` downloads it automatically)

Verify the toolchain:

```bash
cd android
java -version        # JDK 17+
./gradlew --version  # Gradle 9.5
```

## 2. Environment configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `HUGINN_API_BASE_URL` | `http://10.0.2.2:8080/` | Backend base URL. `10.0.2.2` is the Android emulator's host loopback. |

The client talks **only** to the real Huginn HTTP API (see `../backend`). There is
no mock/demo backend in any runtime build — the bundled mock lives in
`app/src/test` and is used solely by unit tests. The base URL is a Gradle
property, passed at build time — **no source changes needed** to switch
environments:

```bash
./gradlew :app:assembleDebug -PHUGINN_API_BASE_URL=http://10.0.2.2:8080/   # emulator
./gradlew :app:assembleDebug -PHUGINN_API_BASE_URL=http://192.168.1.50:8080/  # physical device on LAN
./gradlew :app:assembleDebug -PHUGINN_API_BASE_URL=https://huginn.example.com/ # production
```

- **Debug** build permits cleartext HTTP so any local host/LAN address works
  (`app/src/debug/res/xml/network_security_config.xml`). Release/staging are
  HTTPS-only.
- No developer-machine IP is hardcoded in the codebase; the host address is
  injected per build via `HUGINN_API_BASE_URL`.

The API contract mirrors `../backend/api/openapi.yaml` and
`../backend/internal/httpapi`. DTOs in `data/model` match the backend Go structs
field-for-field.

## 3. Builds

```bash
./gradlew assembleDebug      # debug APK (real backend)
./gradlew assembleStaging    # staging APK (real backend)
./gradlew assembleRelease    # release APK (R8 + shrink, real backend, HTTPS)
```

Release requires a signing config. The project currently builds unsigned; add a
`signingConfigs { release { ... } }` block pointed at your keystore, or rely on
CI signing.

## 4. Tests

```bash
./gradlew test               # JVM unit tests (64 tests)
./gradlew lintDebug          # Android Lint
./gradlew assembleDebugAndroidTest  # compile Compose UI tests
```

Unit tests cover: auth refresh + single-flight, phone formatting, date/time
(Persian), RTL text protection, score classification, communication channel
state, report validation, deep-link handling, the **mandatory-report critical
flow** (`PendingReportGateTest`), theme tokens, work-queue mapping, and a
manifest permission audit (`ManifestPermissionsTest`).

Compose UI tests (`src/androidTest`) cover: login, home, lead list card, lead
detail badges, communication sheet, suggested-message sheet, mandatory report
conditional form, follow-up card, and the qualification checklist.

Run instrumented UI tests on a connected device/emulator:

```bash
./gradlew connectedDebugAndroidTest
```

## 5. Architecture

```
UI (Compose)  ->  ViewModel  ->  Repository  ->  Retrofit (HuginnApi)
```

- **core/api** — Retrofit contract + `ApiCaller` mapping backend error envelope
  (`{ error: { code, message } }`) to typed `AppError`s. Every endpoint matches
  `../backend/api/openapi.yaml`; the client never talks to DeepSeek, the queue,
  PostgreSQL, MinIO, Redis or Tika directly.
- **core/auth** — `SessionManager` owns JWT state; `RefreshInterceptor`
  transparently refreshes on 401 (refresh token is sent via the backend's
  `X-Refresh-Token` header) with **single-flight** refresh; revoked refresh
  clears the session back to login. Tokens live in Keystore-backed
  `EncryptedSharedPreferences` (`TokenStorage`) and are never logged.
- **core/network** — OkHttp/Retrofit wiring, per-build base URL, debug-only
  cleartext to any local host, HTTPS-only in production (network security
  config), plus a debug-only structured `API_REQUEST` request diagnostic logger
  (method/path/base_url/status/latency — never tokens, passwords or bodies).
- **core/database** — Room: the **offline pending-report queue** only.
- **core/external** — the single place Intents are created: `CallLauncher`,
  `SmsLauncher`, `MessengerLauncher`, `EmailLauncher`, `ContactInsertLauncher`,
  plus `MessengerChannels` package/fallback tables. ViewModels never build Intents.
- **feature/app** — `AppGateViewModel` + `PendingReportGate` implement the
  blocking mandatory-report gate and the session/splash routing.
- **data/mock → test** — `MockHuginnApi`, a test-only in-memory mirror of the
  API used by unit tests. It is not compiled into any runtime build.

### Threading & state

Kotlin Coroutines + StateFlow end to end; ViewModels use `viewModelScope`.
Navigation is single-activity Compose (Navigation Compose, string routes).

## 6. Screens

Splash (raven on TP dark) → Login → Main (bottom nav):

- **خانه (Home)** — greeting, six daily stats, and «کار بعدی» (next task) from
  `GET /me/work-queue` with a primary **تماس** CTA.
- **سرنخها (Leads)** — fast list, search (company/person/mobile/city), filters
  (mine/team, status, score, sort), pull-to-refresh, pagination.
- **پیگیریها (Follow-ups)** — tabs امروز / عقبافتاده / آینده / انجامشده, quick
  actions (call/message/done).
- **گزارشها (Reports)** — pending mandatory reports.
- **حساب (Profile)** — user, settings, logout.
- **Lead detail** — header, primary **تماس** + secondary **پیام**, contact
  selector (primary/decision-maker highlighted), sections: basic info, business,
  last interaction, next follow-up, score breakdown, LLM evidence, notes,
  timeline; qualification/disqualification entry points.
- **Mandatory report** — blocking, non-dismissable form (see below).
- **Settings** — theme (system/light/dark), language (fa/en), notification
  toggle, privacy statement.

## 7. Backend endpoints used

Aligned with the real backend (`../backend/internal/httpapi/router.go` and
`api/openapi.yaml`). The list below was verified live against the running API:

```
POST /api/v1/auth/login              POST /api/v1/auth/logout (X-Refresh-Token header)
POST /api/v1/auth/refresh            (X-Refresh-Token header, via RefreshApi)
GET  /api/v1/me                      GET /api/v1/me/work-queue   GET /api/v1/me/pending-reports
GET  /api/v1/dashboard/me
GET  /api/v1/followups?view=...
GET  /api/v1/leads                   GET /api/v1/leads/{id}
GET  /api/v1/leads/{id}/timeline     GET /api/v1/leads/{id}/notes
POST /api/v1/leads/{id}/notes
GET  /api/v1/leads/{id}/evidence     GET /api/v1/leads/{id}/score
GET  /api/v1/leads/{id}/interactions GET /api/v1/leads/{id}/followups
POST /api/v1/leads/{id}/contact-actions
POST /api/v1/interactions/{id}/report
POST /api/v1/leads/{id}/followups    POST /api/v1/followups/{id}/complete
POST /api/v1/leads/{id}/qualify      POST /api/v1/leads/{id}/disqualify
```

Suggested messages and deep links are **generated server-side** by the backend
(which may use an LLM); the APK never holds an LLM key and never calls an LLM,
DeepSeek, the worker queue or any storage service directly.

## 8. Permissions — full inventory

Three permissions are declared. Every requested permission has a justification;
anything not justifiable was removed.

| Permission | Why |
| --- | --- |
| `android.permission.INTERNET` | Required to talk to the Huginn backend. |
| `android.permission.POST_NOTIFICATIONS` | Android 13+ runtime permission for follow-up / assignment reminders. Requested only when the user enables reminders in Settings. |
| `android.permission.RECORD_AUDIO` | Runtime permission for Persian voice input on the Conversation Summary field. Requested only when the user taps the microphone button; recognition uses the device `SpeechRecognizer` API with `fa-IR`. |

**Explicitly NOT requested** (by design and verified by `ManifestPermissionsTest`):

`READ_CONTACTS`, `WRITE_CONTACTS`, `READ_SMS`, `READ_CALL_LOG`, `WRITE_CALL_LOG`,
`READ_PHONE_STATE`, `READ_MEDIA_IMAGES`, `READ_MEDIA_VIDEO`.

- Huginn never reads employees' personal contacts. "Save to contacts" uses
  `Intent.ACTION_INSERT` so the system Contacts app performs the save.
- Huginn never reads or sends SMS; the system messaging app is opened with a
  pre-filled draft and the employee presses Send.
- Huginn never reads the email inbox; a `mailto:` intent pre-fills recipient,
  subject and body.
- Calls use `ACTION_DIAL` (no `CALL_PHONE` permission). Call audio is never
  captured; `RECORD_AUDIO` is used only for the user-initiated voice-typing
  feature, never for calls.

## 9. Messenger integrations

External app launching is centralized in `core/external`. Availability per channel
is derived from the contact record plus the backend-provided permissions object:

- **Call** — dialer with number pre-filled.
- **SMS** — `smsto:` with suggested text (never auto-sent).
- **WhatsApp** — `wa.me` deep link, message pre-fill where supported.
- **Telegram / Bale / Eitaa / Rubika** — backend-provided deep links; app deep
  link preferred, HTTPS web fallback otherwise. When no username exists the
  backend returns `CONTACT_SAVE_REQUIRED` and the app shows the
  «ذخیره در مخاطبین» flow (system Contacts insert — no contacts permission).
- **Email** — `mailto:` with subject/body.

Messenger package names are listed in the manifest `<queries>` block for
Android 11+ package visibility. A missing app shows «این برنامه روی دستگاه نصب
نیست.» with a browser fallback where suitable.

## 10. Mandatory report workflow

The most important mobile workflow — the app blocks normal navigation until a
mandatory contact report is submitted:

1. Employee opens a lead and taps **تماس**.
2. `POST /leads/{id}/contact-actions` records the interaction (backend is the
   source of truth) and the system dialer opens.
3. On resume, `GET /me/pending-reports` is checked (`PendingReportGate`).
4. If a report is pending, the **blocking overlay** opens: back press and swipe
   are intercepted, tap-outside is impossible.
5. The outcome is chosen from chips; fields are shown conditionally (e.g.
   `NO_ANSWER` → only next-call time + note; `INTERESTED` → interest, need,
   decision maker, next action, follow-up time, summary).
6. `POST /interactions/{id}/report` stores the report; the success card shows
   the score delta, next action and follow-up.
7. Submission survives **process death**: the next launch re-checks the backend,
   which still reports the pending interaction until it is confirmed.

**Offline**: the report is stored in the Room `pending_report_queue` and a
WorkManager worker retries it. Until the backend confirms, the workflow stays
pending — the app never treats a locally-queued report as completed.

## 11. Local persistence

- **Room** — offline pending-report queue only (a submitted report is queued and
  retried by a WorkManager worker; it is never treated as completed until the
  backend confirms it).
- **DataStore** — theme mode, language, notification preference.
- **EncryptedSharedPreferences (Keystore)** — access/refresh tokens and cached
  user. Backups are disabled (`android:allowBackup="false"`).
- **WorkManager** — report sync retries and coarse follow-up reminders
  (no constant polling).

There is deliberately **no local cache fallback for leads/work queue/detail**:
if the backend is unavailable the app shows a typed network error with a retry
state rather than stale or fabricated data.

## 12. Known limitations

- The app has no import/document/candidates screens; those are served by the
  backend API (`/imports`, `/documents`, `/documents/{id}/candidates`) but are
  not part of this client's UI.
- Call outcome relies on the employee's report (start/return time + reported
  outcome). No call audio is captured.
- Two-pane tablet layout is not yet implemented; the UI is responsive but
  single-pane.
- Server push notifications are not wired; local WorkManager reminders use the
  same `NotificationHelper` seam.

## 13. Next recommended phase

1. Implement the two-pane tablet layout for lead list | detail.
2. Add import/document/candidates screens (the backend endpoints already exist:
   `/imports`, `/imports/{id}/documents`, `/imports/{id}/queue`,
   `/documents/{id}/candidates`).
3. Add the default-dialer (`ROLE_DIALER` / `InCallService`) mode behind
   `CallLauncher` when telephony infrastructure is approved.
4. Enable server push notifications through `NotificationHelper`.
