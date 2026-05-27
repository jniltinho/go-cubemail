# Software Design Document — go-cubemail-vue

**Version:** 0.2.1 (implementation review)  
**Date:** 2026-06 (current codebase state)  
**Based on:** [Roundcube Webmail](https://github.com/roundcube/roundcubemail)  
**Theme:** Larry (square layout — adapted for Vue 3 SPA)

> **Note:** This document describes both the intended architecture and the actual implementation state.  
> Some planned features (true SSE push notifications) are not yet implemented — see section 5.4.

**Related Documentation:**

- [Development Guide](DEVELOPMENT.md) — Local setup and contribution workflow
- [Contributing Guide](CONTRIBUTING.md) — How to contribute to the project
- [Code Audit Report](CODE_AUDIT_AND_IMPROVEMENTS.md) — Recent documentation and code quality review

---

## 1. Overview

**go-cubemail-vue** is a webmail client written in Go that reimplements the visual and functional experience of Roundcube (Larry theme) using a modern stack. The frontend is a fully migrated **Vue 3 + TypeScript SPA** (Single Page Application) embedded into the Go binary, communicating with the backend exclusively via a REST API at `/api/v1`.

There is no email session database — all message reading/writing occurs in real-time via IMAP. GORM is used exclusively for application data persistence (identities, contacts, user settings). New mail notifications are currently delivered via lightweight **client-side polling** (10 min interval) with audio alerts; true SSE push is planned but not yet implemented.

---

## 2. Tech Stack

### Backend

| Layer | Technology | Version |
|---|---|---|
| Language | Go | 1.26+ |
| HTTP Framework | Echo | v5.x |
| CLI / bootstrap | Cobra | v1.9+ |
| Configuration | Viper | v2.x |
| ORM / App DB | GORM | v2.x |
| App Database | SQLite (dev) / MariaDB / PostgreSQL (prod) | — |
| Email reading protocol | IMAP (`emersion/go-imap/v2`) | v2.0-beta |
| Email sending protocol | SMTP (`wneessen/go-mail`) | v0.7+ |
| HTML sanitization | bluemonday | v1.0+ |
| Web authentication | Cookie session + AES-GCM password encryption | — |
| Middleware | CSRF, Rate limiting, Security headers | custom |
| Config file | TOML | — |
| Real-time / notifications | Client-side polling (10 min) + planned SSE | — |

### Frontend

| Layer | Technology | Version |
|---|---|---|
| Framework | Vue 3 (Composition API) | ^3.5 |
| Language | TypeScript | ^6.0 |
| Build tool | Vite | ^8.0 |
| State management | Pinia | ^3.0 |
| CSS framework | Tailwind CSS v4 (Vite plugin) | ^4.3 |
| HTTP client | Axios | ^1.16 |
| Rich text editor | TipTap (via RichTextEditor) | 2.x |
| Icons | Lucide Vue | ^1.16 |
| **No jQuery / No Node server** | — | — |

---

## 3. Configuration — `config.toml`

```toml
[server]
host        = "0.0.0.0"
port        = 8080
debug       = false
secret_key  = "change-this-secret-key-32-chars!!"
base_url    = "http://localhost:8080"
# Optional HTTPS
tls_cert    = ""
tls_key     = ""

[imap]
host            = "mail.example.com"
port            = 993
tls             = true
timeout_sec     = 30
show_host_input = false     # allow user to type custom IMAP host at login

[smtp]
host        = "mail.example.com"
port        = 587
starttls    = true
timeout_sec = 30

[database]
driver = "sqlite"          # "sqlite" (default) | "mariadb" | "mysql" | "postgres"
dsn    = "./data/app.db"
debug  = false             # true = log all SQL queries (dev only)

# mariadb example:
# dsn   = "user:pass@tcp(localhost:3306)/go_cubemail?charset=utf8mb4&parseTime=True&loc=Local"

# postgres example:
# dsn   = "host=localhost user=go_cubemail password=xxx dbname=go_cubemail port=5432 sslmode=disable TimeZone=America/Sao_Paulo"

[session]
name      = "gorc_session"
max_age   = 86400
secure    = false
http_only = true

[ui]
theme           = "larry"
rows_per_page   = 50
timezone        = "America/Sao_Paulo"
date_format     = "02/01/2006"
datetime_format = "02/01/2006 15:04"
compose_html    = true

[upload]
max_size_mb = 25
temp_dir    = "./tmp/uploads"
```

---

## 4. Directory Structure

```
go-cubemail-vue/
├── main.go                    # Entry: embeds web/dist into binary (//go:embed)
├── go.mod / go.sum
├── Makefile
├── config.toml
├── cmd/
│   ├── root.go               # Cobra root, loads config via Viper
│   ├── init.go               # cobra: generates default config.toml
│   ├── serve.go              # cobra: starts Echo server
│   ├── migrate.go            # cobra: runs GORM migrations
│   └── version.go
├── internal/
│   ├── config/
│   │   └── config.go         # Config structs from TOML + Viper
│   ├── server/
│   │   ├── server.go         # Echo bootstrap, middleware, routes, graceful shutdown, session cleanup
│   │   ├── routes.go         # All /api/v1 routes + SPA fallback handler
│   │   ├── render.go
│   │   └── middleware/
│   │       ├── auth.go           # RequireAuth (IMAP session cookie)
│   │       ├── csrf.go           # CSRF token validation
│   │       ├── ratelimit.go      # Simple rate limiter for auth
│   │       └── security_headers.go
│   ├── handler/
│   │   ├── handler.go        # Central Handlers aggregator
│   │   ├── auth.go           # login, logout, /me, /quota
│   │   ├── mailbox.go        # list folders, list messages, unread count
│   │   ├── message.go        # common message types
│   │   ├── message_read.go   # Read, Raw, Download, Attachment
│   │   ├── message_write.go  # Flag, Move, Delete, EmptyTrash
│   │   ├── compose.go        # compose, send, draft, upload
│   │   ├── contacts.go       # contacts CRUD + import/export
│   │   ├── settings.go       # user preferences (handler exists, not fully routed)
│   │   └── search.go         # IMAP search
│   ├── imap/
│   │   ├── client.go         # IMAP connection management per session
│   │   ├── mailbox.go        # folder ops (LIST, SELECT, SUBSCRIBE)
│   │   ├── message.go        # FETCH, STORE flags, COPY, MOVE, EXPUNGE
│   │   ├── search.go         # IMAP SEARCH
│   │   └── parse.go          # MIME parse, attachment extraction, HTML sanitization
│   ├── smtp/
│   │   └── sender.go         # Send via SMTP (wneessen/go-mail), MIME composition
│   ├── model/                # GORM entities (app data only)
│   │   ├── user.go
│   │   ├── identity.go
│   │   ├── contact.go
│   │   ├── contact_group.go
│   │   ├── draft.go
│   │   ├── session.go
│   │   └── user_settings.go
│   ├── repository/
│   │   ├── contact.go
│   │   ├── identity.go
│   │   └── settings.go
│   ├── database/
│   │   └── database.go       # Centralized DB connection + GORM query logger (sqlite, mariadb, mysql, postgres)
│   └── session/
│       └── imap_session.go   # Wrapper: credentials + in-memory IMAP conn + cleanup goroutine
├── frontend/                 # Vue 3 SPA source (not served directly)
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── package.json
│   └── src/
│       ├── main.ts
│       ├── App.vue           # Root: layout, SSE, keyboard shortcuts
│       ├── types.ts          # TypeScript interfaces
│       ├── style.css         # Tailwind import + design tokens
│       ├── env.d.ts
│       ├── components/       # 16 Vue SFC components
│       ├── stores/           # Pinia stores
│       └── utils/
│           └── helpers.ts    # Date formatting, color utilities
├── web/
│   ├── dist/                 # Vite build output — embedded in binary
│   ├── templates/            # Legacy Go templates (inactive)
│   └── static/               # Static assets (editor fonts, icons, etc.)
├── data/                     # Runtime (SQLite DB, sessions) — gitignored
├── tmp/
└── DOCUMENTS/                # Setup docs, systemd service file
```

---

## 5. Architecture

### 5.1 Authentication Flow

```
Browser → POST /api/v1/auth/login (user + pass + imap_host)
    └─► handler/auth.go
            └─► imap/client.go  ──► IMAP LOGIN (TLS)
                    ├── ERROR → 401 JSON { error: "..." }
                    └── OK    → stores {host, user, pass_enc} in cookie session
                                returns 200 JSON { email, quota }
```

- The password is encrypted with AES-GCM using `server.secret_key` before entering the session.
- Each authenticated API request opens (or reuses from the pool) an IMAP connection.
- Logout revokes the session and closes the connection.

### 5.2 Layers

```
[Browser — Vue 3 SPA]
        │
        │  REST API (/api/v1)    SSE (/api/v1/events)
        ▼
[Echo Handlers]
        │
   ┌────┼────────────┐
   ▼    ▼            ▼
[imap] [smtp]  [repository]
   │                 │
[IMAP]         [GORM → SQLite/MariaDB]
```

### 5.3 IMAP Connection Management

- `session.ImapSession` is stored in an in-memory map indexed by session ID.
- A housekeeping goroutine (in `server.Start`) closes idle sessions after 30 minutes (ticker every 10 minutes).
- SMTP connections are opened per request (no pool).
- Global middleware stack: Recover, RequestID, SecurityHeaders, CSRF, structured RequestLogger, and rate limiter on auth routes.

### 5.4 Real-time / New Mail Notifications (Current Implementation)

**Current state (as of latest code):**  
There is **no true Server-Sent Events (SSE) push** implementation yet.

- The frontend module `frontend/src/utils/sse.ts` (despite the name) implements **client-side polling** every 10 minutes.
- On each tick it calls `fetchFolderMessages('inbox', true)` and plays a short Web Audio beep if new messages are detected.
- No `/api/v1/events` endpoint exists in `routes.go`.
- No `internal/poll` package or SSE hub is present.
- The server comment in `server.go` still references "SSE poller" but it is not wired.

**Planned (documented intent):**  
A future implementation should provide real Server-Sent Events at `GET /api/v1/events` for instant new-mail notifications without polling.

The misleading function names `startSSE()` / `stopSSE()` are legacy and should ideally be renamed to `startNewMailPolling()` / `stopNewMailPolling()` in a future cleanup.

### 5.5 SPA Hosting

- Vite builds the frontend into `web/dist/`.
- The Go binary embeds `web/dist` at compile time via `//go:embed all:web/dist`.
- Echo serves `index.html` as a fallback for all unmatched routes (Vue Router history mode).
- Asset files (`*.js`, `*.css`) are served with long-lived cache headers.

---

## 6. HTTP Routes (Echo v5)

### Public
| Method | Route | Handler | Description |
|---|---|---|---|
| POST | `/api/v1/auth/login` | auth.DoLogin | Authenticates via IMAP, sets session cookie |
| GET | `/version` | — | App version JSON |
| GET | `/*` | SPA handler | Serves embedded Vue SPA (fallback to index.html) |

### Protected (middleware: auth + CSRF + security headers)

**Auth** (some public, some protected)
| Method | Route | Handler | Description |
|---|---|---|---|
| POST | `/api/v1/auth/login` | auth.DoLogin | (public) IMAP login + session cookie |
| POST | `/api/v1/auth/logout` | auth.DoLogout | (protected) Ends session |
| GET | `/api/v1/auth/me` | auth.Me | Current user info |
| GET | `/api/v1/auth/quota` | auth.Quota | Mailbox storage quota |

**Folders**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/folders` | mailbox.FoldersJSON | Lists all folders |
| POST | `/api/v1/folders` | mailbox.CreateSubfolder | Creates subfolder |
| POST | `/api/v1/folders/rename` | mailbox.RenameFolder | Renames folder |
| POST | `/api/v1/folders/delete` | mailbox.DeleteFolder | Deletes folder |
| GET | `/api/v1/folders/:name/count` | mailbox.UnreadCountJSON | Unread count |

**Mail**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/mail/:mailbox` | mailbox.List | Lists messages (paginated) |
| GET | `/api/v1/mail/:mailbox/:uid` | message.Read | Reads full message + body |
| GET | `/api/v1/mail/:mailbox/:uid/raw` | message.Raw | Raw email source |
| GET | `/api/v1/mail/:mailbox/:uid/download` | message.Download | Download as .eml |
| POST | `/api/v1/mail/:mailbox/:uid/flag` | message.Flag | Set/unset flags (seen, starred) |
| POST | `/api/v1/mail/:mailbox/:uid/move` | message.Move | Move to another folder |
| DELETE | `/api/v1/mail/:mailbox/:uid` | message.Delete | Move to Trash / expunge |
| DELETE | `/api/v1/mail/:mailbox` | message.EmptyTrash | Empty trash folder |
| GET | `/api/v1/mail/:mailbox/:uid/attachment/:part` | message.Attachment | Download attachment |

**Compose**
| Method | Route | Handler | Description |
|---|---|---|---|
| POST | `/api/v1/compose/send` | compose.Send | Sends via SMTP |
| POST | `/api/v1/compose/draft` | compose.SaveDraft | Saves draft (IMAP APPEND) |
| POST | `/api/v1/compose/upload` | compose.UploadAttachment | Attachment upload |

**Contacts**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/contacts` | contacts.Index | Lists contacts |
| POST | `/api/v1/contacts` | contacts.Create | Creates contact |
| PUT | `/api/v1/contacts/:id` | contacts.Update | Updates contact |
| DELETE | `/api/v1/contacts/:id` | contacts.Delete | Deletes contact |
| POST | `/api/v1/contacts/import` | contacts.Import | Import from CSV |
| GET | `/api/v1/contacts/export` | contacts.Export | Export to CSV |

**Search**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/search` | search.Results | IMAP SEARCH (subject, from, to) |

**Real-time / Notifications**
| Method | Route | Handler | Description |
|---|---|---|---|
| (none) | — | — | No `/events` SSE endpoint yet. Frontend uses 10-minute client-side polling (see 5.4) |

---

## 7. GORM Models (application data)

```go
// model/user.go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    ImapUser  string    `gorm:"uniqueIndex;not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Settings  UserSettings
    Identities []Identity
}

// model/identity.go
type Identity struct {
    ID          uint   `gorm:"primaryKey"`
    UserID      uint   `gorm:"index;not null"`
    DisplayName string
    Email       string `gorm:"not null"`
    ReplyTo     string
    Signature   string
    IsDefault   bool
}

// model/contact.go
type Contact struct {
    ID        uint   `gorm:"primaryKey"`
    UserID    uint   `gorm:"index;not null"`
    FirstName string
    LastName  string
    Email     string `gorm:"not null"`
    Phone     string
    Company   string
    Notes     string
    GroupID   *uint
    CreatedAt time.Time
    UpdatedAt time.Time
}

type ContactGroup struct {
    ID     uint   `gorm:"primaryKey"`
    UserID uint   `gorm:"index;not null"`
    Name   string `gorm:"not null"`
}

// model/user_settings.go
type UserSettings struct {
    UserID         uint   `gorm:"primaryKey"`
    RowsPerPage    int    `gorm:"default:50"`
    Timezone       string `gorm:"default:'UTC'"`
    ComposeHTML    bool   `gorm:"default:true"`
    DateFormat     string
    SignaturePos   string `gorm:"default:'below'"`
}

// model/session.go
type Session struct {
    ID        string    `gorm:"primaryKey"`
    UserID    uint      `gorm:"index"`
    ExpiresAt time.Time
}
```

---

## 8. IMAP Package (`internal/imap`)

### 8.1 Implemented operations

| Operation | IMAP Command | Go Function |
|---|---|---|
| List folders | `LIST "" "*"` | `ListMailboxes()` |
| Select folder | `SELECT` | `SelectMailbox(name)` |
| Search UIDs | `UID SEARCH` | `SearchMessages(criteria)` |
| Fetch envelope | `UID FETCH ... (ENVELOPE FLAGS)` | `FetchEnvelopes(uids)` |
| Fetch full body | `UID FETCH ... BODY[]` | `FetchMessage(uid)` |
| Fetch MIME part | `UID FETCH ... BODY[n]` | `FetchPart(uid, part)` |
| Mark as read | `UID STORE +FLAGS \Seen` | `MarkSeen(uid)` |
| Mark flagged | `UID STORE +FLAGS \Flagged` | `MarkFlagged(uid)` |
| Move message | `UID MOVE` / `COPY + STORE \Deleted + EXPUNGE` | `MoveMessage(uid, dest)` |
| Delete | `UID STORE +FLAGS \Deleted` + `EXPUNGE` | `DeleteMessage(uid)` |
| Create folder | `CREATE` | `CreateMailbox(name)` |
| Rename folder | `RENAME` | `RenameMailbox(old, new)` |
| Delete folder | `DELETE` | `DeleteMailbox(name)` |
| Save draft | `APPEND Drafts \Draft` | `AppendMessage(mailbox, msg)` |
| Unread count | `STATUS UNSEEN` | `UnreadCount(mailbox)` |

### 8.2 MIME Parse (`internal/imap/parse.go`)

- Uses `emersion/go-message` for multipart parsing.
- Extracts: text/plain body, text/html body, attachment list with name, type, and size.
- Sanitizes HTML body with `microcosm-cc/bluemonday` before returning to frontend.

---

## 9. Frontend — Vue 3 SPA

### 9.1 Architecture Overview

```
App.vue (root)
├── LoginView.vue        — shown when !isAuthenticated
└── Authenticated layout
    ├── AppBar.vue       — top bar: logo, global search, user menu, logout, accent picker
    ├── AppToolbar.vue   — action buttons (reply, delete, move, archive, compose…)
    ├── AppSidebar.vue   — folder tree with unread counts + context menu + quota
    ├── [view === 'mail']
    │   ├── MailList.vue       — scrollable message list with checkboxes + sorting
    │   └── ReadingPane.vue    — email headers, sanitized body (sandboxed), attachments
    ├── [view === 'contacts']
    │   └── ContactsPane.vue   — contact list + import/export
    └── [view === 'calendar']
      └── CalendarPane.vue   — monthly calendar grid with event indicators

Overlays (always mounted):
├── ComposerModal.vue    — rich email composer (TipTap via RichTextEditor)
├── ContactModal.vue     — add/edit contact form
├── DialogModal.vue      — alert / confirm / prompt dialogs
├── ToastContainer.vue   — toast notification queue
├── SourceViewer.vue     — raw email source viewer
├── FolderRow.vue        — reusable folder tree row with context menu
├── Icon.vue / SpinnerIcon.vue
└── RichTextEditor.vue   — TipTap rich text editor (editor/ directory)
```

**Current component count:** 17 Vue SFCs.

### 9.2 Pinia Stores (modular architecture)

**`stores/auth.ts`**
- State: `isAuthenticated`, `isApiOnline`, `currentUser` (email, quota), login form fields
- Actions: `checkSession()`, `handleLogin()`, `handleLogout()`, `fetchQuota()`
- Axios interceptor injects CSRF token on mutating requests

**`stores/mail.ts` + `stores/mail/` sub-module** (highly modular)
- `index.ts` — Main store assembly, state, computed getters, watchEffect for accent
- `api.ts` — All REST calls (`fetchFolderMessages`, `loadFromApi`, `fetchMessageBody`, etc.)
- `mailActions.ts` — Reply, forward, delete, flag, move, archive, compose
- `folderActions.ts` — Create/rename/delete folders + context menu
- `composerActions.ts` — Draft/save/send logic
- `contactActions.ts` — Contact CRUD + import/export
- `constants.ts` / `mockData.ts` — Calendar events and constants

**`stores/toast.ts`**
- Queue of toasts with auto-dismiss; methods: `success()`, `error()`, `warning()`, `info()`

**`stores/dialog.ts`**
- Single-slot dialog state machine; methods: `alert()`, `confirm()`, `prompt()`

The mail store is deliberately split to keep the main `index.ts` readable while actions live in focused files.

### 9.3 TypeScript Interfaces (`types.ts`)

```typescript
interface MailMessage {
  id: string
  folder: string
  from: string
  to: string
  subject: string
  date: string
  snippet: string
  unread: boolean
  starred: boolean
  attachments: Attachment[]
  body?: string
  htmlBody?: string
  signature?: string
}

interface Folder {
  id: string
  label: string
  count: number
  name: string
  custom: boolean
}

interface Contact {
  id?: number
  name: string
  email: string
  title?: string
  company?: string
  phone?: string
  notes?: string
}

interface Toast {
  id: number
  message: string
  type: 'info' | 'success' | 'error' | 'warning'
}

interface DialogState {
  type: 'prompt' | 'confirm' | 'alert'
  message: string
  defaultValue?: string
  resolve: (value: boolean | string | null) => void
}
```

### 9.4 Keyboard Shortcuts (App.vue)

| Key | Action |
|---|---|
| `j` | Next mail |
| `k` | Previous mail |
| `r` | Reply |
| `e` | Archive |
| `#` / `Delete` | Delete |
| `c` | Compose |

### 9.5 Design System (`style.css`)

Design tokens defined as CSS custom properties:

```css
:root {
  --accent:       #1B3A6B;   /* primary navy blue */
  --accent-soft:  #e8eef7;   /* selected row background */
  --row-selected: #d4e4f7;
  --ink:          #1e2433;
  --ink-soft:     #6b7280;
}
```

Custom component classes: `.tbtn`, `.side-item`, `.mail-row`, `.checkbox`, `.modal-wrap`, `.composer-shell`, `.search-box`.

Supports an in-app **accent color picker** — swatches update `--accent` and related tokens at runtime via inline CSS variable override.

### 9.6 3-Column Layout (Larry theme)

```
┌──────────────────────────────────────────────────────────────┐
│  APPBAR: logo | global search | user menu | logout           │
├──────────────────────────────────────────────────────────────┤
│  APPTOOLBAR: Reply | Forward | Delete | Move | Archive...    │
├──────────┬───────────────────────┬───────────────────────────┤
│          │  MAIL LIST            │  READING PANE             │
│ SIDEBAR  │  ─────────────────    │  From: ...                │
│          │  ☐ ● Subject...  date │  To: ...                  │
│ INBOX(3) │  ☐   Subject...  date │  Subject: ...             │
│ Sent     │  ☐ ● Subject...  date │  ─────────────────────    │
│ Drafts   │  ☐   Subject...  date │  [TinyMCE-rendered body]  │
│ Trash    │  ☐   Subject...  date │                           │
│ Spam     │  ...                  │  Attachments: [file.pdf]  │
└──────────┴───────────────────────┴───────────────────────────┘
```

---

## 10. Build System

### Backend

```bash
go build -o bin/go-cubemail .     # compiles Go + embedded web/dist
```

### Frontend

```bash
cd frontend
npm install                       # install deps (once)
npm run build                     # vite build → ../web/dist
npm run dev                       # vite dev server with HMR + proxy to :8080
```

**Vite config** (`frontend/vite.config.ts`):
- Plugins: `@vitejs/plugin-vue`, `@tailwindcss/vite`
- Dev proxy: `/api` → `http://localhost:8080`
- Build output: `../web/dist` (picked up by `//go:embed`)

### Makefile targets

```makefile
build       # go build (no CSS step needed — Vite handles it)
build-prod  # go build + UPX compress
run         # ./bin/go-cubemail serve
frontend    # cd frontend && npm run build
dev         # runs both Go (watch) and Vite dev server
migrate     # ./bin/go-cubemail migrate
clean       # remove binary
```

---

## 11. Webmail Features

### 11.1 Implemented (MVP complete)

- [x] Login/logout via IMAP with TLS (with optional custom host input)
- [x] List IMAP folders (inbox, sent, drafts, trash, spam + custom)
- [x] Message listing with pagination + client-side sort
- [x] Read messages (sanitized HTML + plain text fallback, sandboxed iframe)
- [x] Download attachments
- [x] Compose and send via SMTP (to, cc, bcc, subject, HTML body via TinyMCE, attachments)
- [x] Reply / Reply all / Forward
- [x] Mark as read/unread, starred
- [x] Move message between folders
- [x] Delete (moves to Trash, second delete = expunge)
- [x] Empty trash folder
- [x] Simple search (subject, from, to) via IMAP SEARCH
- [x] Save draft (IMAP APPEND in Drafts folder)
- [x] Inline reading pane (no page reload — Vue reactivity)
- [x] Contacts CRUD (create, read, update, delete)
- [x] Contact import from CSV
- [x] Contact export to CSV
- [x] Contact autocomplete in composer (To / Cc / Bcc fields)
- [x] Calendar view (monthly grid with mock events)
- [x] View raw email source + download as .eml
- [x] Keyboard shortcuts (j/k/r/e/#/c + Delete)
- [x] Accent color picker (runtime CSS variable update)
- [x] Toast notifications + modal dialogs
- [x] Folder create / rename / delete
- [x] 10-minute background polling for new mail with audio notification (see 5.4)

**Note on real-time:** The "SSE" feature is currently client-side polling. True push notifications are planned but not yet implemented.

### 11.2 Secondary (Planned)

- [ ] Multiple "From:" identities per user
- [ ] Email signature (HTML or text)
- [ ] Message filters (server-side Sieve or DB rules)
- [ ] User preferences page (timezone, rows per page, date format)
- [ ] Email printing (`/mail/:mailbox/:uid/print`)
- [ ] Advanced search (by date, attachment, size)
- [ ] Flag messages in batch

### 11.3 Out of Scope (v0.2)

- Multiple simultaneous IMAP accounts
- CalDAV / CardDAV
- Plugins / extensions
- WebSocket bidirectional push (SSE is sufficient)

---

## 12. Security

| Threat | Mitigation |
|---|---|
| XSS in HTML body | `bluemonday` sanitizes HTML before returning from API |
| CSRF | CSRF Token header required on all POST/PUT/DELETE (Echo middleware + Axios interceptor) |
| Credentials in session | AES-GCM encrypted password with server secret key |
| SMTP header injection | Validation and encoding of To/Cc/Bcc/Subject fields |
| Path traversal in MIME parts | Part ID validated as positive integer |
| Malicious upload | Size limit, MIME type validation, served from isolated `/tmp` |
| Session hijacking | `HttpOnly`, `Secure` (prod), `SameSite=Lax` cookies |
| Rate limiting | Echo rate limiter middleware on auth endpoints |
| Clickjacking | `X-Frame-Options: DENY` security header |

---

## 13. CLI Commands (Cobra)

```
go-cubemail-vue
├── init            # create a default config.toml in the current directory
├── serve           # starts web server (default)
│   --config        # path to config.toml (default: ./config.toml)
│   --port          # overrides server.port
│   --debug         # Echo debug mode
├── migrate         # runs GORM AutoMigrate
├── version         # displays version and build info
└── help
```

---

## 14. Go Dependencies (`go.mod`)

```
github.com/labstack/echo/v5          v5.1.1
github.com/spf13/cobra               v1.9.1
github.com/spf13/viper               v1.21.0
gorm.io/gorm                         v1.31.1
gorm.io/driver/sqlite
gorm.io/driver/mysql
gorm.io/driver/postgres              (PostgreSQL support)
github.com/emersion/go-imap/v2       v2.0.0-beta.8
github.com/emersion/go-message       v0.18.2
github.com/wneessen/go-mail          v0.7.3   (SMTP)
github.com/microcosm-cc/bluemonday  v1.0.27
golang.org/x/time                    (rate limiting)
```

---

## 15. Frontend Dependencies (`frontend/package.json`)

```json
{
  "dependencies": {
    "@lucide/vue": "^1.16.0",
    "axios": "^1.16.1",
    "pinia": "^3.0.4",
    "@tiptap/starter-kit": "^2.11.5",
    "vue": "^3.5.34"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.3.0",
    "@vitejs/plugin-vue": "^6.0.6",
    "autoprefixer": "^10.5.0",
    "postcss": "^8.5.15",
    "tailwindcss": "^4.3.0",
    "typescript": "^6.0.3",
    "vite": "^8.0.12",
    "vue-tsc": "^3.3.1"
  }
}
```

**No Node.js server at runtime** — Vite is a build-time tool only. The compiled `web/dist` is embedded in the Go binary and served by Echo.

---

## 16. Environment Variables (TOML override)

```
GORC_SERVER_PORT=8080
GORC_SERVER_SECRET_KEY=...
GORC_DATABASE_DSN=...
GORC_IMAP_HOST=...
GORC_SMTP_HOST=...
```

Viper loads in order: defaults → `config.toml` → env variables prefixed `GORC_`.

---

## 17. Acceptance Criteria — v0.2

1. User can login with real IMAP credentials and see the INBOX.
2. Clicking a message displays the body in the reading pane without page reload.
3. User can compose and send an email with attachments via TinyMCE.
4. Deleted messages go to Trash; second delete expunges.
5. Search returns correct results via IMAP SEARCH.
6. Layout is visually faithful to the Larry theme: 3 columns, straight borders, navy blue palette.
7. No password travels in plaintext in the session cookie.
8. CSRF header blocks unauthenticated mutations from foreign origins.
9. `go-cubemail-vue serve` starts without errors after `go-cubemail-vue migrate`.
10. New mail is detected via 10-minute client-side polling (with audio notification). True SSE push is planned but not yet implemented.
11. Contacts can be created, edited, deleted, imported (CSV), and exported (CSV).
12. The Vue SPA loads without console errors; Vite HMR works during development.
