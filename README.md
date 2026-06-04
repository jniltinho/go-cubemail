# Go CubeMail

**Self-hosted webmail + calendar that works with your existing mail server.**

Go CubeMail gives you a modern, fast web interface for email and calendar — without moving your data. It talks directly to your IMAP/SMTP server and includes full CalDAV, CardDAV and ActiveSync servers so your calendar and contacts sync natively with Apple Calendar, Thunderbird, Outlook and mobile devices.

- **Single binary** — everything (frontend + backend) in one file. No separate web server or Node.js required at runtime.
- **Privacy by design** — your emails never leave your IMAP server. The database only stores contacts, settings and sessions.
- **Real groupware** — mail + full calendar + contacts with native sync protocols.

[→ Download latest release](https://github.com/jniltinho/go-cubemail/releases) | [Documentation](DOCUMENTS/README.md)

Inspired by the classic Roundcube "Larry" theme, but built with Vue 3 + Go.

## Key Features

- **Works with your existing mail server** — No migration. Uses standard IMAP + SMTP.
- **Modern webmail** — Fast 3-column interface, rich-text composer, drag & drop, reply-all, search, keyboard shortcuts.
- **Real-time new mail** — Instant notifications via SSE + optional browser push notifications.
- **Full calendar** — Month/week/day views, recurring events, free/busy, sharing and RSVP.
- **Native sync for all your devices**:
  - CalDAV (Apple Calendar, Thunderbird)
  - CardDAV (contacts)
  - ActiveSync/EAS (Outlook, iOS, Android, Windows Mail)
- **Multiple identities & signatures** — Send from different addresses with custom signatures and OOF replies.
- **Single binary deployment** — One file to run. Frontend is embedded.
- **Easy to configure** — `./go-cubemail init` generates a ready-to-use config.
- **Privacy focused** — Your emails stay on your IMAP server. Only contacts, settings and sessions are stored locally.

## Quick Start (Self-Hosted)

### Easiest: Use a pre-built release

1. Download the latest Linux binary from the [Releases page](https://github.com/jniltinho/go-cubemail/releases)
2. Extract and run:
   ```bash
   tar -xzf go-cubemail_*.tar.gz
   ./go-cubemail init
   ./go-cubemail migrate
   ./go-cubemail serve
   ```
3. Open http://localhost:8080

### Build from source

Requires Go 1.26+ and Node.js (only for the build step).

```bash
make frontend   # build Vue frontend
make build      # build the single Go binary

./bin/go-cubemail init
./bin/go-cubemail migrate
./bin/go-cubemail serve
```

Access at http://localhost:8080

For development mode (with hot reload), see the [Development Guide](DOCUMENTS/docs/DEVELOPMENT.md).

## Configuration

The easiest way is to run:

```bash
./go-cubemail init
```

This creates a ready-to-edit `config_*.toml` file. Rename it to `config.toml`, fill in your IMAP/SMTP details, and you're done.

You can also copy `config.toml.example`.

All settings can be overridden with `GORC_` environment variables (e.g. `GORC_SERVER_PORT=9000`).

For production (MariaDB/PostgreSQL + systemd), see the [Production Setup Guide](DOCUMENTS/setup/README.md).

## Documentation & Help

- [Production Setup Guide](DOCUMENTS/setup/README.md) — Recommended way to run in production (MariaDB/PostgreSQL + systemd)
- [DAV & Sync Setup](DOCUMENTS/docs/DAV_AND_SYNC_SETUP.md) — How to configure CalDAV, CardDAV and ActiveSync
- Full API reference available via Swagger UI when enabled (`swagger_enable = true`)

For developers and contributors, see the [Documentation Index](DOCUMENTS/README.md).

## License

MIT License

## Contributing

Contributions are welcome! Please read:

- [Contributing Guide](DOCUMENTS/docs/CONTRIBUTING.md)
- [Development Guide](DOCUMENTS/docs/DEVELOPMENT.md)

All code, docs and commits must be in English.

Found a bug or have an idea? [Open an issue](https://github.com/jniltinho/go-cubemail/issues).
