# Go CubeMail

A modern, lightweight, self-hosted webmail + groupware client written in Go with a Vue 3 + TypeScript frontend.

Go CubeMail connects directly to your existing mail servers using standard IMAP (reading) and SMTP (sending) protocols. It provides a clean, responsive web interface inspired by the classic Roundcube "Larry" theme, delivered as a single compiled binary with the frontend embedded at build time.

In addition to mail, it includes a full calendar with CalDAV/CardDAV servers and an ActiveSync (EAS) server for synchronization with Apple Calendar, Thunderbird, Outlook and mobile devices.

**No email data is stored** — everything happens in real time against your IMAP server. The database is used only for contacts, identities, user settings, calendar events, and sessions.

## Key Features

**Mail Client**

- **Direct IMAP + SMTP** — Works with any standards-compliant mail server (no email data stored in the app DB)
- **Modern Vue 3 SPA** — Fast, reactive 3-column layout inspired by Roundcube "Larry"
- **Full email workflow** — Rich-text compose (TipTap), reply / reply-all, forward, move, flag, delete, drafts, attachments, drag-and-drop to folders
- **Contacts** — Complete CRUD with CSV import/export and autocomplete in the composer
- **Search** — Server-side IMAP SEARCH
- **Keyboard shortcuts** — Vim-style navigation (`j`/`k`, `r`, `c`, `#`, etc.)
- **Real-time notifications** — Server-Sent Events (SSE) for new mail with heartbeats + optional browser Web Push (VAPID configurable)
- **Customization & Identities** — Accent color, date formats, rows per page, multiple sender identities with signatures, OOF settings

**Groupware & Sync**

- **Calendar** — Month, week and day views, RRULE recurrence, free/busy, iCal import/export, sharing, RSVP via iMIP
- **CalDAV server** (RFC 4791) — Compatible with Apple Calendar, Thunderbird and other clients
- **CardDAV server** (RFC 6352) — Full vCard address book synchronization
- **ActiveSync / EAS server** — Mail, calendar (vevent), contacts (vcard) and tasks (vtodo) synchronization (EAS 16.1)
- **Remote subscriptions** — Subscribe to external .ics calendars with background refresh

**Developer & Operations**

- **Interactive API documentation** — Swagger UI (toggleable via `swagger_enable`) with comprehensive OpenAPI annotations
- **Single binary deployment** — Vue frontend is built and embedded via `//go:embed`
- **Easy configuration** — Built-in `init` command generates ready-to-use TOML config
- **Production ready** — CSRF protection, security headers, rate limiting, TLS, systemd example, configurable session cleanup

## Architecture Highlights

- **Backend**: Go 1.26 + Echo v5 + GORM (SQLite / MariaDB / PostgreSQL) + emersion/go-imap
- **Frontend**: Vue 3 (Composition API) + TypeScript + Vite + Pinia + Tailwind CSS v4 + TipTap
- **No Node.js at runtime** — Vite is used only for building
- **Authentication**: IMAP login with AES-GCM encrypted credentials in secure cookies
- **Real-time / Notifications**: Server-Sent Events (SSE) for new-mail push (with 55s heartbeats for proxy compatibility) + Web Push notifications via VAPID. Background workers for calendar subscription refresh.
- **Groupware servers**: Full CalDAV (RFC 4791), CardDAV (RFC 6352) and Microsoft ActiveSync/EAS (mail + calendar + contacts + tasks) implementations.

## Tech Stack

| Layer            | Technology                                              |
|------------------|---------------------------------------------------------|
| Language         | Go 1.26+                                                |
| Web / API        | Echo v5, GORM, Cobra, Viper, swaggo/swag (OpenAPI)      |
| Email            | IMAP (emersion/go-imap), SMTP (wneessen/go-mail)        |
| Frontend         | Vue 3 + TS, Pinia, Vite, Tailwind CSS v4, TipTap        |
| Real-time        | SSE (Server-Sent Events) + Web Push (VAPID)             |
| Sync Protocols   | CalDAV (RFC 4791), CardDAV (RFC 6352), ActiveSync/EAS   |
| Calendar         | rrule-go (recurrence), iCalendar handling               |
| Database         | SQLite (dev) / MariaDB / PostgreSQL (prod)              |

## Quick Start

### Prerequisites
- Go 1.26+
- Node.js + npm (for frontend build only)

### Build & Run

```bash
# 1. Build the Vue frontend
make frontend

# 2. Build the Go binary (embeds web/dist)
make build

# 3. (Recommended) Generate a configuration file
./bin/go-cubemail init

# 4. Run migrations (creates tables)
./bin/go-cubemail migrate

# 5. Start the server
./bin/go-cubemail serve
```

Access at http://localhost:8080

> **Tip:** Run `make swagger` (requires Go) to (re)generate the OpenAPI docs served by the integrated Swagger UI when `swagger_enable = true`.

### Development

```bash
make dev          # runs Go (watch) + Vite dev server with HMR
# or
make frontend-dev # Vite only (proxies /api to :8080)
```

## Configuration

The recommended way to create a configuration file is using the built-in `init` command:

```bash
./bin/go-cubemail init
```

This will generate a file named `config_<timestamp>.toml` (example: `config_1750945822.toml`) in the current directory. You can then rename or copy it to `config.toml`.

Alternatively, you can copy the example manually:

```bash
cp config.toml.example config.toml
```

Then edit `config.toml` with your IMAP/SMTP credentials and other settings.

All settings can also be overridden using environment variables with the `GORC_` prefix (example: `GORC_SERVER_PORT=9000`).

Full production installation guide (MariaDB / PostgreSQL + systemd) is available in `DOCUMENTS/setup/README.md`.

## Project Structure

- `cmd/` — Cobra CLI commands:
  - `init` — Generate a default configuration file
  - `serve` — Start the web server
  - `migrate` — Run database migrations
  - `version` — Show version information
- `internal/` — Core Go packages:
  - `handler/` — API handlers (mail, compose, contacts, calendar, settings, identities, push, SSE, CalDAV, CardDAV, ActiveSync)
  - `calendar/`, `activesync/` — Calendar logic and EAS server
  - `imap/`, `smtp/` — Protocol clients
  - `model/`, `repository/`, `session/` — App data (GORM) and IMAP session management
  - `server/`, `config/` — Echo setup, routing, middleware, configuration
- `frontend/` — Vue 3 + TypeScript source (built to `web/dist/`)
- `web/` — Embedded assets (`dist/` for the SPA and `files/` for other embedded resources including default config)
- `docs/` — Auto-generated OpenAPI (Swagger) specs
- `DOCUMENTS/` — Documentation index, setup guides, SDD, development & contributing guides, DAV/ActiveSync guides, and releasing process
- `scripts/` — Testing helpers for CalDAV, CardDAV and ActiveSync protocols

## Documentation

For a complete overview of all available documentation, see the **[Documentation Index](DOCUMENTS/README.md)**.

Key documents include:

- [Software Design Document (SDD)](DOCUMENTS/docs/SDD.md) — Architecture, design decisions, and technical overview.
- [Development Guide](DOCUMENTS/docs/DEVELOPMENT.md) — Local setup, build process, and development workflow.
- [Contributing Guide](DOCUMENTS/docs/CONTRIBUTING.md) — How to contribute code, report issues, and follow project standards.
- [Releasing Guide](DOCUMENTS/docs/RELEASING.md) — How version tags and polished GitHub Releases are produced.
- [DAV & Sync Setup](DOCUMENTS/docs/DAV_AND_SYNC_SETUP.md) — Configuration and testing of CalDAV, CardDAV and ActiveSync.
- [Code Audit Report](DOCUMENTS/docs/CODE_AUDIT_AND_IMPROVEMENTS.md) — Recent documentation and code quality review.
- [Production Setup Guide](DOCUMENTS/setup/README.md) — Step-by-step installation for production environments (Ubuntu + MariaDB).

The API is also documented live via Swagger UI (when `swagger_enable = true` in config) at `/swagger/`.

## License

MIT License

## Contributing

We welcome contributions! Before getting started, please read the following guides:

- [Contributing Guide](DOCUMENTS/docs/CONTRIBUTING.md) — How to report issues, submit pull requests, and coding standards.
- [Development Guide](DOCUMENTS/docs/DEVELOPMENT.md) — How to set up your local environment, build the project, and run it in development mode.
- [English Only Policy](DOCUMENTS/specs/english_only.md) — All code, comments, documentation, and commit messages **must** be written in English.

### Quick Links

- Found a bug? → Open an issue
- Want to propose a feature? → Open an issue or start a discussion
- Ready to contribute code? → Read the [Contributing Guide](DOCUMENTS/docs/CONTRIBUTING.md) and [Development Guide](DOCUMENTS/docs/DEVELOPMENT.md)

All pull requests must follow the English-only policy and pass the project's contribution guidelines.
