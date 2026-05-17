# Software Design Document — go-cubemail

**Version:** 0.1.0  
**Date:** 2026-05-16  
**Based on:** [Roundcube Webmail](https://github.com/roundcube/roundcubemail)  
**Theme:** Larry (square layout)

---

## 1. Overview

**go-cubemail** is a webmail client written in Go that reimplements the visual and functional experience of Roundcube (Larry theme) using a modern stack: Echo v5, TailwindCSS v4, jQuery 4, and authentication via direct IMAP to the email server.

There is no email session database — all message reading/writing occurs in real-time via IMAP. GORM is used exclusively for application data persistence (identities, contacts, user settings, filters).

---

## 2. Tech Stack

| Layer | Technology | Version |
|---|---|---|
| Language | Go | 1.23+ |
| HTTP Framework | Echo | v5.x |
| CLI / bootstrap | Cobra | v1.9+ |
| Configuration | Viper | v2.x |
| ORM / App DB | GORM | v2.x |
| App Database | SQLite (dev) / MariaDB (prod) | — |
| Email reading protocol | IMAP (via `emersion/go-imap`) | v2.x |
| Email sending protocol | SMTP (via `net/smtp` + `jordan-wright/email`) | stdlib |
| Frontend CSS | TailwindCSS (standalone binary) | v4.2.0 |
| Frontend JS | jQuery | 4.0.0 |
| Template engine | Go `html/template` | stdlib |
| Web authentication | Cookie session (`gorilla/sessions`) | v1.4+ |
| Config file | TOML | — |

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
go-cubemail/
├── cmd/
│   ├── root.go          # cobra root, loads config via viper
│   ├── serve.go         # cobra subcommand: starts Echo server
│   ├── migrate.go       # cobra subcommand: runs GORM migrations
│   └── version.go
├── internal/
│   ├── config/
│   │   └── config.go    # config structs mapped from TOML
│   ├── server/
│   │   ├── server.go    # Echo bootstrap, middlewares, routes
│   │   └── middleware/
│   │       ├── auth.go      # checks active IMAP session
│   │       └── logger.go
│   ├── handler/
│   │   ├── auth.go      # login, logout
│   │   ├── mailbox.go   # list folders, list messages
│   │   ├── message.go   # read, flag, move, delete
│   │   ├── compose.go   # compose, send, draft
│   │   ├── contacts.go  # address book
│   │   ├── settings.go  # user preferences
│   │   └── search.go    # IMAP search
│   ├── imap/
│   │   ├── client.go    # IMAP connection pool per session
│   │   ├── mailbox.go   # folder ops (LIST, SELECT, SUBSCRIBE)
│   │   ├── message.go   # FETCH, STORE flags, COPY, MOVE, EXPUNGE
│   │   ├── search.go    # IMAP SEARCH
│   │   └── parse.go     # MIME parse, attachment extraction
│   ├── smtp/
│   │   └── sender.go    # send via SMTP, MIME composition
│   ├── model/           # GORM entities (app data only)
│   │   ├── user.go
│   │   ├── identity.go
│   │   ├── contact.go
│   │   ├── contact_group.go
│   │   ├── draft.go
│   │   └── user_settings.go
│   ├── repository/
│   │   ├── contact.go
│   │   ├── identity.go
│   │   └── settings.go
│   └── session/
│       └── imap_session.go  # wrapper: credentials + in-memory IMAP conn
├── web/
│   ├── templates/
│   │   ├── layout/
│   │   │   ├── base.html       # HTML shell, top nav, sidebar
│   │   │   └── auth.html       # login screen shell
│   │   ├── mailbox/
│   │   │   ├── index.html      # folders list + messages list
│   │   │   └── message.html    # reading pane
│   │   ├── compose/
│   │   │   └── compose.html    # email editor
│   │   ├── contacts/
│   │   │   ├── index.html
│   │   │   └── edit.html
│   │   ├── settings/
│   │   │   └── index.html
│   │   └── auth/
│   │       └── login.html
│   ├── static/
│   │   ├── css/
│   │   │   ├── input.css       # TailwindCSS v4 input (committed)
│   │   │   └── style.css       # TW binary generated output (gitignored)
│   │   ├── js/
│   │   │   ├── jquery-4.0.0.min.js
│   │   │   ├── app.js          # general initialization
│   │   │   ├── mailbox.js      # message list, split pane
│   │   │   ├── compose.js      # editor, attachments
│   │   │   └── contacts.js
│   │   └── icons/              # SVG icons (heroicons)
├── data/                       # generated at runtime (gitignored)
├── tmp/
├── config.toml                 # example / dev
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 5. Architecture

### 5.1 Authentication Flow

```
Browser → POST /login (user + pass + imap_host)
    └─► handler/auth.go
            └─► imap/client.go  ──► IMAP LOGIN (TLS)
                    ├── ERROR → renders login with error message
                    └── OK    → stores {host, user, pass_enc} in cookie session
                                redirects → /mail/INBOX
```

- The password is encrypted with AES-GCM using `server.secret_key` before entering the session.  
- Each authenticated request opens (or reuses from the pool) an IMAP connection.  
- Logout revokes the session and closes the connection.

### 5.2 Layers

```
[Browser] ←HTTP/HTML+JSON→ [Echo Handlers]
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
              [imap pkg]   [smtp pkg]   [repository]
                    │                        │
              [IMAP Server]           [GORM → SQLite/MariaDB]
```

### 5.3 IMAP Connection Management

- `session.ImapSession` is stored in an in-memory map, indexed by session ID.  
- A housekeeping goroutine closes idle sessions after N minutes (configurable).  
- For sending emails, a separate SMTP connection is opened per request (no pool).

---

## 6. HTTP Routes (Echo v5)

### Public
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/` | auth.LoginPage | Redirects to `/login` or `/mail/INBOX` |
| GET | `/login` | auth.LoginPage | Login page |
| POST | `/login` | auth.DoLogin | Authenticates via IMAP |
| POST | `/logout` | auth.DoLogout | Ends session |

### Protected (`middleware/auth.go`)
| Method | Route | Handler | Description |
|---|---|---|---|
| GET | `/mail/:mailbox` | mailbox.List | Lists folder messages |
| GET | `/mail/:mailbox/:uid` | message.Read | Reads message |
| POST | `/mail/:mailbox/:uid/flag` | message.Flag | Flags/unflags (seen, flagged) |
| POST | `/mail/:mailbox/:uid/move` | message.Move | Moves to another folder |
| DELETE | `/mail/:mailbox/:uid` | message.Delete | Moves to Trash / expunge |
| GET | `/compose` | compose.New | New email form |
| GET | `/compose/reply/:mailbox/:uid` | compose.Reply | Reply |
| GET | `/compose/forward/:mailbox/:uid` | compose.Forward | Forward |
| POST | `/compose/send` | compose.Send | Sends via SMTP |
| POST | `/compose/draft` | compose.SaveDraft | Saves draft (IMAP APPEND) |
| GET | `/compose/attachment/:id` | compose.ServeAttachment | Serves uploading attachment |
| POST | `/compose/upload` | compose.UploadAttachment | Attachment upload |
| GET | `/mail/:mailbox/:uid/attachment/:part` | message.Attachment | Attachment download |
| GET | `/contacts` | contacts.Index | Lists contacts |
| GET | `/contacts/new` | contacts.New | New contact form |
| POST | `/contacts` | contacts.Create | Creates contact |
| GET | `/contacts/:id/edit` | contacts.Edit | Edit contact |
| PUT | `/contacts/:id` | contacts.Update | Saves edit |
| DELETE | `/contacts/:id` | contacts.Delete | Removes contact |
| GET | `/settings` | settings.Index | Settings page |
| POST | `/settings` | settings.Save | Saves preferences |
| GET | `/search` | search.Results | IMAP Search |
| GET | `/api/folders` | mailbox.FoldersJSON | Lists folders (JSON, for AJAX sidebar) |
| GET | `/api/folders/:name/count` | mailbox.UnreadCountJSON | Unread count |

---

## 7. GORM Models (application data)

```go
// model/user.go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    ImapUser  string    `gorm:"uniqueIndex;not null"`  // IMAP email/login
    CreatedAt time.Time
    UpdatedAt time.Time
    Settings  UserSettings
    Identities []Identity
}

// model/identity.go — "From:" aliases
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
    SignaturePos   string `gorm:"default:'below'"` // below | above
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
| Save draft | `APPEND Drafts \Draft` | `AppendMessage(mailbox, msg)` |
| Unread count | `STATUS UNSEEN` | `UnreadCount(mailbox)` |

### 8.2 MIME Parse (`internal/imap/parse.go`)

- Uses `emersion/go-message` for multipart parsing.  
- Extracts: text/plain body, text/html body, attachment list with name, type, and size.  
- Sanitizes HTML body with `microcosm-cc/bluemonday` before rendering.

---

## 9. Visual Interface — Larry Theme (Square Layout)

### 9.1 Larry Principles

- **3-column layout**: folders sidebar | message list | reading pane.  
- Straight borders (`rounded-none` / `border` without radius).  
- Palette: neutral gray for background, navy blue (`#37517e`) for header and selected items, white for panes.  
- Toolbar with small SVG icons + text, without exaggerated shadows.  
- Typography: system `font-sans`, base size 13–14px.

### 9.2 Base HTML Layout

```
┌──────────────────────────────────────────────────────────────┐
│  TOPBAR: logo | global search | user menu | logout           │
├──────────────────────────────────────────────────────────────┤
│  TOOLBAR: New | Reply | Forward | Delete | Move...           │
├──────────┬───────────────────────┬───────────────────────────┤
│          │  MESSAGE LIST         │  READING PANE             │
│ SIDEBAR  │  ─────────────────    │  From: ...                │
│          │  □ ● Subject...  date │  To: ...                  │
│ INBOX(3) │  □   Subject...  date │  Subject: ...             │
│ Sent     │  □ ● Subject...  date │  ─────────────────────    │
│ Drafts   │  □   Subject...  date │  Message body             │
│ Trash    │  □   Subject...  date │  ...                      │
│ Spam     │  ...                  │                           │
│          │                       │  Attachments: [file.pdf]  │
└──────────┴───────────────────────┴───────────────────────────┘
```

### 9.3 TailwindCSS v4 Classes by Region

**Topbar:**
```html
<header class="flex items-center h-10 bg-[#37517e] text-white px-3 gap-4 border-b border-[#2a3d5e]">
```

**Action Toolbar:**
```html
<div class="flex items-center h-9 bg-[#f0f2f5] border-b border-gray-300 px-2 gap-1">
```

**Folders Sidebar:**
```html
<aside class="w-44 shrink-0 bg-[#f8f9fa] border-r border-gray-300 overflow-y-auto">
```

**Active folder item:**
```html
<a class="flex items-center justify-between px-3 py-1 text-sm bg-[#37517e] text-white font-semibold">
```

**Inactive folder item:**
```html
<a class="flex items-center justify-between px-3 py-1 text-sm text-gray-700 hover:bg-gray-200">
```

**Message list:**
```html
<section class="w-72 shrink-0 border-r border-gray-300 overflow-y-auto">
```

**Unread message row:**
```html
<div class="flex items-start gap-2 px-2 py-2 border-b border-gray-200 bg-white font-semibold cursor-pointer hover:bg-blue-50">
```

**Read message row:**
```html
<div class="flex items-start gap-2 px-2 py-2 border-b border-gray-200 bg-[#f8f9fa] text-gray-600 cursor-pointer hover:bg-blue-50">
```

**Selected row:**
```html
<div class="... bg-[#d4e4f7] border-l-2 border-[#37517e]">
```

**Reading pane:**
```html
<main class="flex-1 overflow-y-auto bg-white px-6 py-4">
```

**Primary action button:**
```html
<button class="flex items-center gap-1 px-3 py-1 text-sm bg-[#37517e] text-white hover:bg-[#2a3d5e] border border-[#2a3d5e]">
```

**Secondary button:**
```html
<button class="flex items-center gap-1 px-3 py-1 text-sm bg-white text-gray-700 hover:bg-gray-100 border border-gray-300">
```

### 9.4 Login Screen

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│              [ go-cubemail Logo ]              │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  IMAP Server    [imap.example.com        ]  │   │
│  │  Username       [user@example.com        ]  │   │
│  │  Password       [••••••••••••••••••••••• ]  │   │
│  │                                             │   │
│  │                        [ Login →         ] │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

Fields with straight border `border border-gray-400`, without `rounded`.

---

## 10. jQuery 4 Interactivity

### 10.1 `mailbox.js`

```javascript
// Loads reading pane without page reload
$(document).on('click', '.msg-row', function () {
  const uid     = $(this).data('uid');
  const mailbox = $(this).data('mailbox');
  $('.msg-row').removeClass('selected');
  $(this).addClass('selected');
  $('#reading-pane').load(`/mail/${mailbox}/${uid} #message-body`);
  // Mark as read via AJAX
  $.post(`/mail/${mailbox}/${uid}/flag`, { flag: 'seen', value: 1 });
  $(this).removeClass('unread').addClass('read');
});

// Select all
$('#check-all').on('change', function () {
  $('.msg-check').prop('checked', this.checked);
  updateToolbarState();
});

// Infinite pagination (scroll)
let page = 1;
$('#msg-list').on('scroll', function () {
  if (this.scrollTop + this.clientHeight >= this.scrollHeight - 20) {
    page++;
    $.get(location.href, { page }, function (html) {
      const rows = $(html).find('.msg-row');
      if (rows.length) $('#msg-list .msg-rows').append(rows);
    });
  }
});
```

### 10.2 `compose.js`

```javascript
// Attachment upload
$('#attach-input').on('change', function () {
  const fd = new FormData();
  $.each(this.files, (i, f) => fd.append('files[]', f));
  $.ajax({ url: '/compose/upload', method: 'POST', data: fd,
           processData: false, contentType: false,
    success(res) {
      res.files.forEach(f => {
        $('#attachment-list').append(
          `<div class="attach-item flex items-center gap-2 text-sm py-1">
             <span>${f.name} (${f.size})</span>
             <button class="remove-attach text-red-500" data-id="${f.id}">✕</button>
           </div>`
        );
      });
    }
  });
});

// Recipients auto-complete (search in contacts)
$('#to-field, #cc-field, #bcc-field').on('input', function () {
  const q = $(this).val();
  if (q.length < 2) return;
  $.getJSON('/contacts', { q, format: 'json' }, function (res) {
    // renders suggestions dropdown
  });
});
```

---

## 11. Webmail Features

### 11.1 Mandatory (MVP)

- [x] Login/logout via IMAP with TLS
- [x] List IMAP folders (inbox, sent, drafts, trash, spam + custom folders)
- [x] Message listing with pagination (50 per page)
- [x] Read messages (Sanitized HTML + fallback plain text)
- [x] Download attachments
- [x] Compose and send via SMTP (to, cc, bcc, subject, HTML/text body, attachments)
- [x] Reply / Reply all / Forward
- [x] Mark as read/unread, flagged
- [x] Move message between folders
- [x] Delete (moves to Trash, second delete = expunge)
- [x] Simple search (subject, from, to) via IMAP SEARCH
- [x] Save draft (IMAP APPEND in Drafts folder)
- [x] Inline reading pane (without page reload, via jQuery AJAX)

### 11.2 Secondary

- [ ] Contacts address book (CRUD via GORM)
- [ ] Multiple "From:" identities per user
- [ ] Email signature (HTML or text)
- [ ] Create / rename / delete IMAP folders
- [ ] Flag messages in batch
- [ ] Message filters (server-side Sieve or client-side via DB rules)
- [ ] User preferences (timezone, rows per page, date format)
- [ ] Email printing (`/mail/:mailbox/:uid/print`)
- [ ] Advanced search (by date, attachment, size)

### 11.3 Out of Scope (v0.1)

- Support for multiple simultaneous IMAP accounts
- CalDAV / CardDAV
- Plugins / extensions
- Push notifications (WebSocket)

---

## 12. Security

| Threat | Mitigation |
|---|---|
| XSS in HTML body | `bluemonday` sanitizes HTML before rendering |
| CSRF | CSRF Token in all POST/PUT/DELETE forms (Echo middleware) |
| Credentials in session | AES-GCM encrypted password with server key |
| SMTP header injection | Validation and encoding of To/Cc/Bcc/Subject fields |
| Path traversal in MIME parts | Part ID validated as positive integer |
| Malicious upload | Size limit, MIME type validation, serves in isolated `/tmp` |
| Session hijacking | `HttpOnly`, `Secure` (prod), `SameSite=Lax` cookies |

---

## 13. CLI Commands (Cobra)

```
go-cubemail
├── serve           # starts web server (default)
│   --config        # path to config.toml (default: ./config.toml)
│   --port          # overrides server.port
│   --debug         # Echo debug mode
├── migrate         # runs GORM migrations (AutoMigrate)
│   --rollback      # reverts last migration
├── version         # displays version and build info
└── help
```

Typical usage:
```bash
go-cubemail migrate
go-cubemail serve --config /etc/gorc/config.toml
```

---

## 14. Makefile

```makefile
## Variables for Tailwind CSS and UPX
TAILWIND_VERSION := v4.2.0
TAILWIND_BIN     := /usr/local/bin/tailwindcss
TAILWIND_URL     := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-x64
UPX_VERSION      := 5.1.1
UPX_ARCHIVE      := upx-$(UPX_VERSION)-amd64_linux.tar.xz
UPX_DIR          := upx-$(UPX_VERSION)-amd64_linux
UPX_BIN          := /usr/local/bin/upx
UPX_URL          := https://github.com/upx/upx/releases/download/v$(UPX_VERSION)/$(UPX_ARCHIVE)

## Variables for Go application
APP        := go-cubemail
BIN        := bin/$(APP)
PREFIX     := go-cubemail/cmd
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS    := -ldflags "-s -w -X $(PREFIX).Version=$(VERSION) -X $(PREFIX).BuildDate=$(BUILD_TIME) -X $(PREFIX).GitCommit=$(GIT_COMMIT)"

.PHONY: all build build-prod run clean css watch-css migrate tidy deps \
        install-tailwind install-upx certs help

all: clean css build-prod

build: clean css
	@echo "Building Go application..."
	CGO_ENABLED=0 go build -o $(BIN) $(LDFLAGS) .

build-prod:
	@echo "Building Go application (production)..."
	CGO_ENABLED=0 go build -o $(BIN) $(LDFLAGS) .
	upx --best --lzma $(BIN)

run:
	@echo "Starting application..."
	./$(BIN) serve

css:
	@echo "Building CSS with Tailwind..."
	tailwindcss -i ./web/static/css/input.css -o ./web/static/css/style.css --minify

watch-css:
	@echo "Watching CSS changes..."
	tailwindcss -i ./web/static/css/input.css -o ./web/static/css/style.css --watch

migrate:
	@echo "Running database migrations..."
	./$(BIN) migrate

clean:
	@echo "Cleaning up..."
	rm -f $(BIN)
	rm -f web/static/css/style.css

tidy:
	@echo "Tidying go modules..."
	go mod tidy

deps:
	@echo "Installing Go dependencies..."
	go mod download

certs:
	@echo "Generating self-signed SSL certificates..."
	mkdir -p ssl
	openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
		-keyout ssl/server.key -out ssl/server.crt \
		-subj "/C=BR/ST=SP/L=Sao Paulo/O=Development/CN=localhost"

install-tailwind:
	@echo "Installing Tailwind CSS binary $(TAILWIND_VERSION)..."
	curl -ksSL "$(TAILWIND_URL)" -o tailwindcss-linux-x64
	chmod +x tailwindcss-linux-x64
	mv tailwindcss-linux-x64 "$(TAILWIND_BIN)"

install-upx:
	@echo "Installing UPX $(UPX_VERSION)..."
	curl -ksSL "$(UPX_URL)" -o "$(UPX_ARCHIVE)"
	tar -xf "$(UPX_ARCHIVE)"
	chmod +x "$(UPX_DIR)/upx"
	mv "$(UPX_DIR)/upx" "$(UPX_BIN)"
	rm -rf "$(UPX_DIR)" "$(UPX_ARCHIVE)"

help:
	@echo "Makefile commands:"
	@echo "  build            - Build the Go application (with CSS)"
	@echo "  build-prod       - Build + compress with UPX (production)"
	@echo "  run              - Run the application"
	@echo "  css              - Build CSS using Tailwind binary"
	@echo "  watch-css        - Watch for CSS changes"
	@echo "  migrate          - Run database migrations"
	@echo "  clean            - Remove binary and generated CSS"
	@echo "  tidy             - Run go mod tidy"
	@echo "  deps             - Download Go dependencies"
	@echo "  certs            - Generate self-signed SSL certificates"
	@echo "  install-tailwind - Download and install Tailwind CSS binary"
	@echo "  install-upx      - Download and install UPX binary"
```

---

## 15. Go Dependencies (`go.mod` main ones)

```
github.com/labstack/echo/v5
github.com/spf13/cobra
github.com/spf13/viper
gorm.io/gorm
gorm.io/driver/sqlite
gorm.io/driver/mysql        // supports MariaDB via standard MySQL DSN
github.com/emersion/go-imap/v2
github.com/emersion/go-message
github.com/jordan-wright/email      // SMTP sending
github.com/gorilla/sessions
github.com/microcosm-cc/bluemonday  // HTML sanitization
github.com/BurntSushi/toml          // (used internally by viper)
```

---

## 16. Frontend Dependencies

**There is no `package.json` nor Node.js** — TailwindCSS is used via the official standalone binary.

### Installation (once per machine/CI)
```bash
make install-tailwind   # downloads tailwindcss-linux-x64 → /usr/local/bin/tailwindcss
make install-upx        # (optional) Go binary compressor for production
```

### CSS Files
```
web/static/css/
├── input.css    # Tailwind input — committed in repository
└── style.css    # generated output — gitignored, never committed
```

`input.css` contains only the import directive:
```css
@import "tailwindcss";
```

TailwindCSS v4 detects the classes used in Go templates automatically (without a mandatory `tailwind.config.js`). If needed to add theme customizations, CSS variables are used inside `input.css` itself:

```css
@import "tailwindcss";

@theme {
  --color-brand: #37517e;
  --color-brand-dark: #2a3d5e;
}
```

### jQuery and icons
jQuery 4.0.0 and SVG icons (heroicons) are served as static files in
`/web/static/js/` and `/web/static/icons/` — no bundler, no transpilation.

---

## 17. Environment Variables (TOML override)

```
GORC_SERVER_PORT=8080
GORC_SERVER_SECRET_KEY=...
GORC_DATABASE_DSN=...
GORC_IMAP_HOST=...
GORC_SMTP_HOST=...
```

Viper loads in order: default values → `config.toml` → environment variables prefixed with `GORC_`.

---

## 18. Acceptance Criteria — MVP

1. User can login with real IMAP credentials and see the INBOX.
2. Clicking a message displays the body in the right pane without reload.
3. User can send an email with an attachment.
4. Deleted messages go to the Trash; second delete expunges them.
5. Search returns correct results via IMAP SEARCH.
6. Layout is visually faithful to the Larry theme: 3 columns, straight borders, navy blue palette.
7. No password travels in plaintext in the session cookie.
8. CSRF blocks on all forms.
9. `go-cubemail serve` spins up without errors after `go-cubemail migrate`.
10. TailwindCSS v4 and jQuery 4 load correctly, without console errors.
