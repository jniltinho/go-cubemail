# Go CubeMail Documentation

Everything beyond the [project README](../README.md): what the application does in detail, every way to install and
configure it, and the map of the technical guides.

- [Features](#features)
- [Installation](#installation): Docker, pre-built binary, from source, upgrading
- [Configuration](#configuration)
- [Tech stack](#tech-stack)
- [Documentation map](#documentation-map)

## Features

**Mail**

- Three-pane inbox with folders, search, drag and drop, multi-select and keyboard shortcuts
- Rich-text composer with attachments, inline images, tables and an HTML source mode
- Multiple identities, each with its own signature
- Web Push notifications for new mail, even with the tab in the background (needs HTTPS and VAPID keys)
- HTML mail shown in a sandboxed frame; remote content cannot script the interface

**Calendar and contacts**

- Month, week and day views, recurring events, several calendars with their own colors, invitations
- Contacts with import and export (CSV, and vCard through CardDAV)

**Sync with every client**

| Protocol | Works with |
|---|---|
| CalDAV | Apple Calendar, Thunderbird, Evolution, GNOME Calendar |
| CardDAV | Apple Contacts, Thunderbird, Evolution |
| Exchange ActiveSync | iOS Mail and Calendar, Android (Outlook, Gmail), Outlook desktop |

- Every calendar and address book syncs as its own folder, over both DAV and ActiveSync: create one in Thunderbird
  and it appears on the phone
- Delta sync: clients fetch only what changed, deletions included, and a conflicting edit is refused instead of
  silently overwriting
- What a client sends is what it gets back: addresses, photos, birthdays, alarms and custom fields survive a round
  trip

**Operations**

- `init` writes a ready-to-edit config, `migrate` is one idempotent command, `serve` runs it
- Every setting can come from a `GORC_` environment variable
- Versioned REST API with Swagger documentation
- Multi-arch Docker image (amd64, arm64) and a systemd unit example

## Installation

### Docker

Multi-arch images (amd64, arm64) are published to the GitHub Container Registry:
`latest` and `X.Y.Z` on every release, `edge` from `main`.

```bash
docker run -d --name go-cubemail -p 8080:8080 \
  -e GORC_SERVER_SECRET_KEY="$(openssl rand -base64 24)" \
  -e GORC_IMAP_HOST=mail.example.com \
  -e GORC_SMTP_HOST=mail.example.com \
  -v go-cubemail-data:/data \
  ghcr.io/jniltinho/go-cubemail:latest
```

- The container runs `migrate` and then `serve`, as an unprivileged user (uid 10001).
- `/data` holds the SQLite database. With a bind mount instead of a named volume,
  make the directory writable for uid 10001.
- `GORC_SERVER_SECRET_KEY` is required and must stay the same across restarts:
  it encrypts the IMAP password kept in the session, so changing it logs everyone out.
- Any other setting is a `GORC_` variable (`GORC_SMTP_PORT`, `GORC_DATABASE_DRIVER`,
  `GORC_DATABASE_DSN`, …), or mount a full config file at `/etc/go-cubemail/config.toml`.

### Pre-built binary

1. Download the latest Linux (amd64) binary from the [Releases page](https://github.com/jniltinho/go-cubemail/releases)
2. Extract and generate a config file:

   ```bash
   tar -xzf go-cubemail_*.tar.gz
   ./go-cubemail init -o config.toml
   ```

3. Edit `config.toml` with your IMAP/SMTP server details — without it the server
   starts against the built-in defaults and will not reach your mail server.
4. Create the database schema and start:

   ```bash
   ./go-cubemail migrate
   ./go-cubemail serve
   ```

5. Open http://localhost:8080

Pre-built binaries are published on every release. For production setup with MariaDB + systemd, see the [Production Setup Guide](setup/README.md).

### Build from source

Requires Go 1.27+ and Node.js (build time only).

```bash
git clone https://github.com/jniltinho/go-cubemail.git
cd go-cubemail

make all                            # builds Vue 3 frontend + embeds into Go binary
./bin/go-cubemail init -o config.toml   # then edit config.toml
./bin/go-cubemail migrate
./bin/go-cubemail serve
```

Access at http://localhost:8080.

Swagger UI (when enabled): `http://localhost:8080/swagger/`.

### Upgrading

Replace the binary and run the migration again before starting the new version:

```bash
./go-cubemail migrate
```

Migrations are idempotent, so running them when nothing changed is harmless. A release that adds a column (the
user's theme, for example) needs it: without the migration the settings endpoints fail.

## Configuration

The easiest way to start:

```bash
./go-cubemail init -o config.toml
```

Then fill in your IMAP/SMTP and database details. Without `-o`, the file is
named `config_<timestamp>.toml` so an existing config is never overwritten —
rename it to `config.toml` before starting the server.

The config file is looked up as `./config.toml`, then `/etc/go-cubemail/config.toml`,
or wherever `--config` points. All settings can be overridden with `GORC_`
environment variables.

See the [Production Setup Guide](setup/README.md) for MariaDB/PostgreSQL + systemd examples.

## Tech Stack

**Backend**: Go 1.27, Echo v5, GORM, Cobra, Viper, Swaggo (OpenAPI)

**Frontend (embedded)**: Vue 3 + TypeScript + Vite+ (`vp`), Pinia, Tailwind CSS v4, TipTap (rich text), Lucide icons

**Email & Sync**:
- IMAP: `emersion/go-imap/v2`
- SMTP: `wneessen/go-mail`
- CalDAV / CardDAV: custom implementation (RFC 4791 / 6352, with RFC 6578 delta sync)
- ActiveSync (EAS): `remdev/go-activesync` + custom command handlers
- iCalendar (RRULE, VEVENT): custom parser + recurrence engine

**Packaging**: Single binary with embedded SPA, UPX compression support, systemd unit example

## Documentation map

### Folders

| Directory | Purpose |
|-----------|---------|
| `docs/` | Technical documentation, architecture, development guides, and contributor guidelines |
| `setup/` | Production installation guides and deployment examples (systemd, databases, etc.) |
| `specs/` | Project specifications and rules (e.g., coding standards, language policy) |
| `screenshots/` | Screenshots used by the READMEs: the cover, one per theme, and the main views |
| `branding/` | Logo concepts from the design rounds |

### Where to start

- **New contributors** → Start with [Development Guide](docs/DEVELOPMENT.md) and [Contributing Guide](docs/CONTRIBUTING.md)
- **Understanding the project** → Read the [Software Design Document (SDD)](docs/SDD.md)
- **Deploying to production** → See the [Production Setup Guide](setup/README.md)
- **Project rules** → Review the [English Only Policy](specs/english_only.md)

### Main documents

#### In `docs/`

- [Documentation Index](docs/README.md) — Complete list of technical and contributor documentation
- [Software Design Document (SDD)](docs/SDD.md) — Architecture, design decisions, and implementation details
- [Development Guide](docs/DEVELOPMENT.md) — How to set up your local environment and develop the project
- [Contributing Guide](docs/CONTRIBUTING.md) — Guidelines for contributing code and opening pull requests
- [Code Audit & Improvements Report](docs/CODE_AUDIT_AND_IMPROVEMENTS.md) — Results of a recent documentation and code quality review
- [Themes Guide](docs/THEMES.md) — The shipped themes, how to add one, and what the contrast tests guarantee
- [Releasing Guide](docs/RELEASING.md) — How to create version tags and GitHub Releases with polished, structured release notes

#### In `setup/`

- [Production Installation Guide](setup/README.md) — Step-by-step instructions for deploying on Ubuntu with MariaDB (includes systemd configuration)

#### In `specs/`

- [English Only Policy](specs/english_only.md) — Mandatory rule requiring all code, comments, documentation, and messages to be written in English

### Related

- [Main Project README](../README.md) — Overview, features, and quick start for users
- [Frontend README](../frontend/README.md) — Frontend-specific development instructions

---

If you notice any outdated information or missing documentation, please open an issue or submit a pull request. Contributions to improve the docs are very welcome!