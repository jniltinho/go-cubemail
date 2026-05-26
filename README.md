# Go CubeMail

A modern, lightweight, self-hosted webmail client written in Go with a Vue 3 + TypeScript frontend.

Go CubeMail connects directly to your existing mail servers using standard IMAP (reading) and SMTP (sending) protocols. It provides a clean, responsive web interface inspired by the classic Roundcube "Larry" theme, delivered as a single compiled binary with the frontend embedded at build time.

**No email data is stored** — everything happens in real time against your IMAP server. The database is used only for contacts, identities, user settings, and sessions.

## Key Features

- **Direct IMAP + SMTP** — Works with any standards-compliant mail server
- **Modern Vue 3 SPA** — Fast, reactive 3-column layout (sidebar, message list, reading pane)
- **Full email workflow** — Compose (TinyMCE), reply, forward, move, flag, delete, drafts, attachments
- **Contacts** — Complete CRUD with CSV import/export and autocomplete in the composer
- **Search** — Server-side IMAP SEARCH
- **Keyboard shortcuts** — Vim-style navigation (`j`/`k`, `r`, `c`, `#`, etc.)
- **New mail notifications** — Lightweight client-side polling (10-minute interval) with audio alert (real-time push via SSE is planned)
- **Customization** — Runtime accent color picker, configurable date formats, rows per page
- **Single binary deployment** — Frontend is compiled and embedded via `//go:embed`
- **Easy configuration** — Built-in `init` command to generate ready-to-use config files
- **Production ready** — CSRF protection, security headers, rate limiting, TLS support, systemd example

## Architecture Highlights

- **Backend**: Go 1.26 + Echo v5 + GORM (SQLite / MariaDB / PostgreSQL) + emersion/go-imap
- **Frontend**: Vue 3 (Composition API) + TypeScript + Vite + Pinia + Tailwind CSS v4 + TinyMCE
- **No Node.js at runtime** — Vite is used only for building
- **Authentication**: IMAP login with AES-GCM encrypted credentials in secure cookies
- **Real-time / Notifications**: Client-side polling (10 min). True server-side push (SSE) is planned for the future.

## Tech Stack

| Layer       | Technology                          |
|-------------|-------------------------------------|
| Language    | Go 1.26+                            |
| Web         | Echo v5, GORM, Cobra, Viper         |
| Email       | IMAP (emersion/go-imap), SMTP (wneessen/go-mail) |
| Frontend    | Vue 3 + TS, Pinia, Vite, Tailwind v4 |
| Editor      | TinyMCE 6                           |
| Database    | SQLite (dev) / MariaDB / PostgreSQL (prod) |

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
- `internal/` — All Go business logic (handlers, IMAP/SMTP, models, repositories)
- `frontend/` — Vue 3 + TypeScript source (built to `web/dist/`)
- `web/` — Embedded assets (`dist/` for the SPA and `files/` for other embedded resources)
- `DOCUMENTS/` — Setup guides, SDD, and code audit report

## License

MIT License

## Contributing

Contributions are welcome. Please follow the English-only policy for all code and documentation (see `DOCUMENTS/specs/english_only.md`).
