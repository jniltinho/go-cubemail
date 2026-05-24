# Software Design Document — go-cubemail-vue

**Version:** 0.2.0  
**Date:** 2026-05-24  
**Based on:** [Roundcube Webmail](https://github.com/roundcube/roundcubemail)  
**Theme:** Larry (square layout — adapted for Vue 3 SPA)

---

## 1. Overview

**go-cubemail-vue** is a webmail client written in Go that reimplements the visual and functional experience of Roundcube (Larry theme) using a modern stack. The frontend is a fully migrated **Vue 3 + TypeScript SPA** (Single Page Application) embedded into the Go binary, communicating with the backend exclusively via a REST API at `/api/v1`.

There is no email session database — all message reading/writing occurs in real-time via IMAP. GORM is used exclusively for application data persistence (identities, contacts, user settings, filters). New mail notifications are delivered to the browser via **Server-Sent Events (SSE)**.

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
| App Database | SQLite (dev) / MariaDB (prod) | — |
| Email reading protocol | IMAP (`emersion/go-imap`) | v2.x |
| Email sending protocol | SMTP (`jordan-wright/email`) | — |
| HTML sanitization | bluemonday | v1.0+ |
| Web authentication | Cookie session (`gorilla/sessions`) | v1.4+ |
| Config file | TOML | — |
| Real-time push | Server-Sent Events (SSE) | stdlib |

### Frontend

| Layer | Technology | Version |
|---|---|---|
| Framework | Vue 3 (Composition API) | ^3.5 |
| Language | TypeScript | ^6.0 |
| Build tool | Vite | ^8.0 |
| State management | Pinia | ^3.0 |
| CSS framework | Tailwind CSS v4 (Vite plugin) | ^4.3 |
| HTTP client | Axios | ^1.16 |
| Rich text editor | TinyMCE | ^6.8 |
| Icons | Lucide Vue | ^1.16 |
| **No jQuery / No Node server** | — | — |

---

## 3. Configuration — `config.toml`

```toml
[server]
host        = "0.0.0.0"
port        = 8080
debug       = false
secret_key  = "change-this-key-in-production"
base_url    = "http://localhost:8080"

[imap]
host        = "imap.example.com"
port        = 993
tls         = true
timeout_sec = 30

[smtp]
host        = "smtp.example.com"
port        = 587
starttls    = true
timeout_sec = 30

[database]
driver = "sqlite"          # "sqlite" | "mariadb"
dsn    = "./data/app.db"   # mariadb: "user:pass@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True"

[session]
name       = "gorc_session"
max_age    = 86400         # seconds (24h)
secure     = false         # true in production with HTTPS
http_only  = true

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
│   ├── serve.go              # cobra: starts Echo server
│   ├── migrate.go            # cobra: runs GORM migrations
│   └── version.go
├── internal/
│   ├── config/
│   │   └── config.go         # Config structs from TOML
│   ├── server/
│   │   ├── server.go         # Echo bootstrap, middleware, routes
│   │   └── middleware/
│   │       ├── auth.go       # Checks active IMAP session
│   │       └── logger.go
│   ├── handler/
│   │   ├── auth.go           # login, logout, /me, /quota
│   │   ├── mailbox.go        # list folders, list messages, unread count
│   │   ├── message.go        # read, flag, move, delete, raw, download
│   │   ├── compose.go        # compose, send, draft, upload
│   │   ├── contacts.go       # contacts CRUD + import/export
│   │   ├── settings.go       # user preferences
│   │   └── search.go         # IMAP search
│   ├── imap/
│   │   ├── client.go         # IMAP connection pool per session
│   │   ├── mailbox.go        # folder ops (LIST, SELECT, SUBSCRIBE)
│   │   ├── message.go        # FETCH, STORE flags, COPY, MOVE, EXPUNGE
│   │   ├── search.go         # IMAP SEARCH
│   │   └── parse.go          # MIME parse, attachment extraction
│   ├── smtp/
│   │   └── sender.go         # Send via SMTP, MIME composition
│   ├── model/                # GORM entities (app data only)
│   │   ├── user.go
│   │   ├── identity.go
│   │   ├── contact.go
│   │   ├── contact_group.go
│   │   ├── draft.go
│   │   ├── session.go
│   │   └── user_settings.go
│   ├── poll/
│   │   └── hub.go            # SSE hub: polls IMAP, pushes new-mail events
│   ├── repository/
│   │   ├── contact.go
│   │   ├── identity.go
│   │   └── settings.go
│   └── session/
│       └── imap_session.go   # Wrapper: credentials + in-memory IMAP conn
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
│   └── static/               # Static assets (TinyMCE dist, fonts, icons)
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
- A housekeeping goroutine closes idle sessions after 30 minutes (every 10 minutes tick).
- SMTP connections are opened per request (no pool).

### 5.4 Real-time Notifications (SSE)

- `GET /api/v1/events` — long-lived SSE stream authenticated via session cookie.
- The `poll` package runs a background goroutine per connected user that polls IMAP INBOX for unseen messages at a configurable interval.
- When new mail arrives, the hub pushes an `event: new-mail` message to all active SSE connections for that user.
- The Vue frontend listens with the browser-native `EventSource` API and dispatches the event into the Pinia mail store.

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

### Protected (`middleware/auth.go`)

**Auth**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/auth/me` | auth.Me | Current user info |
| GET | `/api/v1/auth/quota` | auth.Quota | Mailbox storage quota |
| POST | `/api/v1/auth/logout` | auth.DoLogout | Ends session |

**Folders**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/folders` | mailbox.FoldersJSON | Lists all folders |
| POST | `/api/v1/folders` | mailbox.Create | Creates subfolder |
| POST | `/api/v1/folders/rename` | mailbox.Rename | Renames folder |
| POST | `/api/v1/folders/delete` | mailbox.Delete | Deletes folder |
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

**Real-time**
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/api/v1/events` | poll.SSE | Server-Sent Events stream |

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
    ├── AppBar.vue       — top bar: logo, search, user menu, logout
    ├── AppToolbar.vue   — action buttons (reply, delete, move, archive…)
    ├── AppSidebar.vue   — folder tree with unread counts + context menu
    ├── [view === 'mail']
    │   ├── MailList.vue       — scrollable message list with checkboxes
    │   └── ReadingPane.vue    — email headers, body, attachments
    ├── [view === 'contacts']
    │   └── ContactsPane.vue   — contact list + import/export
    └── [view === 'calendar']
      └── CalendarPane.vue   — monthly calendar grid with event indicators

Overlays (always mounted):
├── ComposerModal.vue    — rich email composer (TinyMCE)
├── ContactModal.vue     — add/edit contact form
├── DialogModal.vue      — alert / confirm / prompt dialogs
├── ToastContainer.vue   — toast notification queue
└── SourceViewer.vue     — raw email source viewer
```

### 9.2 Pinia Stores

**`stores/auth.ts`**
- State: `isAuthenticated`, `isApiOnline`, `currentUser` (email, quota), login form fields
- Actions: `checkSession()`, `handleLogin()`, `handleLogout()`, `fetchQuota()`
- Injects CSRF token into all Axios requests via interceptor

**`stores/mail/index.ts`** (main store)
- State: `mails`, `folders`, `contacts`, `view`, `folder`, `selectedId`, `selectedIds`, `composer`, `query`, `calCells`
- Computed: `visibleMails`, `counts`, `selected`, `currentFolderLabel`
- Actions: mail CRUD, folder management, contacts CRUD, API loading

**`stores/mail/api.ts`**
- `fetchFolderMessages(mailbox)` → `GET /api/v1/mail/:mailbox`
- `loadFromApi()` → initial bootstrap (folders → messages → contacts)
- `fetchMessageBody(mailbox, uid)` → `GET /api/v1/mail/:mailbox/:uid`

**`stores/mail/folderActions.ts`**
- Folder create / rename / delete via API
- Context menu handlers for `AppSidebar`

**`stores/toast.ts`**
- Queue of toasts with auto-dismiss; methods: `success()`, `error()`, `warning()`, `info()`

**`stores/dialog.ts`**
- Single-slot dialog state machine; methods: `alert()`, `confirm()`, `prompt()`

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

- [x] Login/logout via IMAP with TLS
- [x] List IMAP folders (inbox, sent, drafts, trash, spam + custom)
- [x] Message listing with pagination
- [x] Read messages (sanitized HTML + plain text fallback)
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
- [x] Contact import from CSV / vCard
- [x] Contact export to CSV
- [x] Contact autocomplete in composer (To / Cc / Bcc fields)
- [x] Real-time new mail notifications via SSE
- [x] Calendar view (monthly grid)
- [x] View raw email source
- [x] Download message as .eml
- [x] Keyboard shortcuts (j/k/r/e/#/c)
- [x] Accent color picker (runtime CSS variable update)
- [x] Toast notifications
- [x] Folder create / rename / delete

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
gorm.io/driver/mysql                 (MariaDB support)
github.com/emersion/go-imap/v2       v2.0.0-beta.8
github.com/emersion/go-message       v0.18.2
github.com/jordan-wright/email       (SMTP sending)
github.com/gorilla/sessions          v1.4+
github.com/microcosm-cc/bluemonday  v1.0.27
golang.org/x/time                    (rate limiting)
```

---

## 15. Frontend Dependencies (`frontend/package.json`)

```json
{
  "dependencies": {
    "axios":           "^1.16.1",
    "lucide-vue-next": "^1.16.0",
    "pinia":           "^3.0.4",
    "tinymce":         "^6.8.6",
    "vue":             "^3.5.34"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.3.0",
    "@vitejs/plugin-vue": "latest",
    "tailwindcss":       "^4.3.0",
    "typescript":        "^6.0.3",
    "vite":              "^8.0.12",
    "vue-tsc":           "latest"
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
10. New mail triggers an SSE event that updates the unread count badge in the sidebar in real time.
11. Contacts can be created, edited, deleted, imported (CSV), and exported (CSV).
12. The Vue SPA loads without console errors; Vite HMR works during development.
