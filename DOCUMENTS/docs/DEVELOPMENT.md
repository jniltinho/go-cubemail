# Development Guide

This guide explains how to set up a local development environment for **go-cubemail-vue**.

## Prerequisites

- **Go** 1.26 or higher
- **Node.js** 20+ and npm
- **Make**
- A running database (SQLite for development, MariaDB/PostgreSQL recommended for production-like testing)

Optional but recommended:
- `upx` (for production builds)
- `openssl` (for generating self-signed certificates during development)

## Getting Started

1. Clone the repository:

   ```bash
   git clone https://github.com/jniltinho/go-cubemail-vue.git
   cd go-cubemail-vue
   ```

2. Install dependencies:

   ```bash
   # Go dependencies
   make deps

   # Frontend dependencies
   make deps-frontend
   ```

3. Generate a development configuration:

   ```bash
   ./bin/go-cubemail init -o config.toml
   ```

   Edit `config.toml` with your local IMAP/SMTP settings (or use a test account).

## Project Structure

```
go-cubemail-vue/
├── cmd/                  # CLI entrypoints (init, serve, migrate, version)
├── internal/
│   ├── config/           # Configuration loading (Viper)
│   ├── database/         # GORM connection and logging
│   ├── handler/          # HTTP handlers (Echo)
│   ├── imap/             # IMAP client and operations
│   ├── smtp/             # SMTP sending logic
│   ├── model/            # GORM models
│   ├── repository/       # Data access layer
│   ├── server/           # Echo server, routes, middleware
│   └── session/          # In-memory + DB session management
├── frontend/
│   └── src/              # Vue 3 + TypeScript application
├── web/
│   ├── dist/             # Built frontend (generated)
│   └── files/            # Other embedded assets (e.g. config.default.toml)
├── DOCUMENTS/
│   ├── docs/             # SDD, development guides, audit reports
│   ├── setup/            # Production installation guides
│   └── specs/            # Project rules (e.g. english_only.md)
└── Makefile
```

## Development Workflow

### Running in Development Mode

The fastest way to develop is:

```bash
make dev
```

This starts:
- Go application with file watching (using a watcher or `air` if installed)
- Vite development server on port 5173 (proxies API calls to `:8080`)

Alternatively, you can run them separately:

```bash
# Terminal 1 - Backend
go run . serve

# Terminal 2 - Frontend (with HMR)
make frontend-dev
```

### Building

```bash
# Build everything (frontend + Go binary)
make all

# Production build (with UPX compression)
make build-prod
```

### Database Migrations

After changing models:

```bash
./bin/go-cubemail migrate
```

Or during development:

```bash
make migrate
```

## Useful Makefile Commands

| Command            | Description                                      |
|--------------------|--------------------------------------------------|
| `make all`         | Clean + build frontend + Go binary               |
| `make frontend`    | Build Vue frontend into `web/dist/`              |
| `make frontend-dev`| Start Vite dev server (port 5173)                |
| `make build`       | Build Go binary                                  |
| `make build-prod`  | Production build with UPX                        |
| `make run`         | Run the compiled binary                          |
| `make migrate`     | Run database migrations                          |
| `make clean`       | Remove binary and `web/dist/`                    |
| `make deps`        | Download Go modules                              |
| `make deps-frontend`| Install frontend npm packages                   |
| `make certs`       | Generate self-signed SSL certificates            |
| `make help`        | Show all available commands                      |

## Code Style

- Follow the **English-only policy** strictly. See [DOCUMENTS/specs/english_only.md](../specs/english_only.md).
- Use `gofmt` / `goimports` for Go code.
- Use Prettier + ESLint for the frontend.
- Write clear, English comments and documentation.

## Testing

Currently, the project has limited automated tests. When adding new features:

- Add unit tests for new packages under `internal/`.
- Test CLI commands manually using `./bin/go-cubemail <command>`.
- Test IMAP/SMTP flows with real or mocked accounts when possible.

Run Go tests:

```bash
go test ./...
```

## Working with the CLI

The application uses [Cobra](https://github.com/spf13/cobra). New commands should be added under `cmd/`.

Example:

```bash
./bin/go-cubemail init
./bin/go-cubemail serve --port 9000
./bin/go-cubemail migrate
```

## Debugging

- Enable debug mode: `./bin/go-cubemail serve --debug`
- Set `debug = true` in `[database]` section of `config.toml` to log SQL queries.
- Use `make certs` + configure TLS in `config.toml` for local HTTPS testing.

## Releasing

See the release workflow in `.github/workflows/release.yml`.

Typical release process:

1. Create and push a new tag (`git tag vX.Y.Z && git push origin vX.Y.Z`).
2. The GitHub Action will build, package, and publish the release automatically.

## Additional Resources

- [Software Design Document (SDD)](SDD.md)
- [Code Audit Report](CODE_AUDIT_AND_IMPROVEMENTS.md)
- [Production Setup Guide](../setup/README.md)
- [English Only Policy](../specs/english_only.md)

---

If you find any issues or missing steps in this guide, please open an issue or submit a pull request.