# 📬 Cubemail Frontend — Vue 3 + TypeScript + Vite

A modern, responsive webmail interface built as the frontend for **go-cubemail-vue**. It delivers a clean three-column experience inspired by the classic Roundcube "Larry" theme, written as a fully typed Vue 3 + TypeScript SPA that is compiled and embedded into the Go binary at build time.

---

## 🚀 Key Features

- **⚡ Blazing Fast Build**: Powered by [Vite+](https://viteplus.dev) (`vp`): one toolchain for the dev server with HMR, the Rolldown build, the formatter (Oxfmt) and the linter (Oxlint).
- **🔒 Strict Type Safety**: Fully written in TypeScript with comprehensive interface modeling for all mail entities, contacts, and state.
- **📦 Clean State Management**: Pinia stores with a deliberately split mail module (`stores/mail/`) to keep the main store readable while isolating API, folder, composer, mail, and contact actions.
- **💡 Keyboard Shortcuts**: Vim-style navigation (`j`/`k`), compose (`c`), reply (`r`), archive (`e`), delete (`#` or `Delete`). Implemented globally in `App.vue`.
- **📧 Isolated Mail Rendering**: Secure read-pane framing utilizing sandboxed iframes to completely isolate external HTML contents and block unauthorized scripts.
- **🎨 Themes**: Eleven themes, all WCAG AAA, built on one contract of role tokens (`src/styles/contract.css`); a theme is one CSS file of role values under `src/themes/`, created with `npm run theme:new`. Chosen per user in Settings → Appearance, applied without a reload. See `DOCUMENTS/docs/THEMES.md`.
- **🔔 New Mail Notifications**: Client-side polling (10-minute interval) that detects new messages in the inbox and plays an audio alert. True server-push (SSE) is planned but not yet implemented — see `DOCUMENTS/docs/SDD.md` §5.4 for details.

---

## ⚠️ Important Implementation Notes

- **Polling, not Push**: The module `utils/sse.ts` (and functions `startNewMailPolling` / `stopNewMailPolling`) implements **client-side polling**. The original names were kept for historical reasons but have been clarified in code and docs.
- **No backend session storage for mail**: All messages are fetched live via IMAP. The only persistent data is contacts, identities, user settings, and encrypted session credentials (GORM + SQLite/Postgres/MariaDB).
- **Embedded SPA**: The entire built frontend (`web/dist/`) is compiled into the Go binary via `//go:embed`. There is no separate Node.js server at runtime.
- **Larry Theme Fidelity**: The UI deliberately follows the classic Roundcube "Larry" square aesthetic (navy accent, 3-column layout, minimal chrome).

See the full audit for documentation and code quality findings: `DOCUMENTS/docs/CODE_AUDIT_AND_IMPROVEMENTS.md`.

For the complete system architecture, backend details, and configuration, start with the project root [README.md](../README.md) and the [Software Design Document](../DOCUMENTS/docs/SDD.md).

---

## 📖 Code Quality & Documentation

The core of the frontend is written in English and extensively documented:

- All TypeScript modules (Pinia stores, composables, utils, and types) use comprehensive JSDoc/TSDoc.
- Major Vue components include `@component` and `@description` documentation plus JSDoc on key functions and reactive state.

This enables rich IDE tooltips, precise autocomplete, and reliable auto-typing. A full documentation audit was performed in June 2026 (see `DOCUMENTS/docs/CODE_AUDIT_AND_IMPROVEMENTS.md`).

```typescript
/**
 * Builds a month-view grid array containing exactly 42 calendar day cells (CalCell),
 * incorporating dimmed trailing and leading days from adjacent months.
 *
 * @param events - Map of scheduled events, keyed by day of the month numbers.
 * @returns Array with 42 CalCell instances representing the month's layout.
 */
export function buildCalCells(events: Record<number, CalEvent[]> = {}): CalCell[] { ... }
```

---

## 📂 Project Structure

```bash
frontend/
├── index.html             # Loads /theme-boot.js before the app, so a reload starts in the user's theme
├── public/
│   ├── theme-boot.js      # Applies the remembered theme before the first paint (a file: the CSP forbids inline scripts)
│   └── favicon.svg, sw.js
├── scripts/
│   ├── themes.mjs         # `npm run theme:new` and `npm run themes` (registry generation)
│   └── style-dump.js      # Console snippet to compare a screen before and after a stylesheet change
├── tests/unit/            # Theme tests: file structure, color lint, registry, WCAG AAA contrast
├── src/
│   ├── components/        # 🎨 Vue 3 SFCs (Single File Components)
│   │   ├── AppBar.vue          # Top bar: brand mark, view tabs, global search, sign-out
│   │   ├── AppSidebar.vue      # Folder tree and quota gauge
│   │   ├── AppToolbar.vue      # Mail actions (compose, reply, delete, move)
│   │   ├── MailList.vue        # Message list with checkboxes and sorting
│   │   ├── ReadingPane.vue     # Headers, sandboxed HTML body (iframe), attachments, calendar invitations
│   │   ├── ComposerModal.vue   # Composer: recipients with autocomplete, attachments, the rich-text editor
│   │   ├── editor/             # TipTap rich-text editor and its toolbar
│   │   ├── ContactsPane.vue    # Address book with import/export
│   │   ├── CalendarPane.vue    # Month, week and day views
│   │   ├── SettingsPane.vue    # Identities, preferences and the Appearance tab (theme picker)
│   │   ├── BrandMark.vue       # The logo, drawn with currentColor
│   │   └── ...                 # Dialogs, toasts, source viewer, contact and event modals, icons
│   │
│   ├── stores/            # 🍍 Pinia State Modules
│   │   ├── auth.ts        # Session, CSRF token, theme registry and the active theme
│   │   ├── dialog.ts      # Alert, confirm and prompt dialogs
│   │   ├── toast.ts       # Toast notification queue
│   │   └── mail/          # Mail store, deliberately split for maintainability
│   │       ├── index.ts           # Store assembly, getters, cross-cutting concerns
│   │       ├── api.ts             # REST calls (folders, messages, bodies)
│   │       ├── mailActions.ts     # Reply/forward, flag, move, delete, archive, select
│   │       ├── folderActions.ts   # Create/rename/delete folders, context menu
│   │       ├── composerActions.ts # Compose, reply, forward, draft, source viewer
│   │       ├── contactActions.ts  # Contacts CRUD, import/export, autocomplete
│   │       ├── calendarActions.ts # Calendars and events
│   │       └── settingsActions.ts # Preferences, identities, theme save
│   │
│   ├── styles/            # 🎨 Stylesheets
│   │   ├── contract.css   # The role contract: every color, radius, shadow and font, with Larry's values
│   │   ├── base.css       # Document defaults, large-screen scaling, scrollbars, keyboard focus
│   │   └── components/    # One file per group of component classes (buttons, sidebar, mail-list, ...)
│   ├── themes/            # One file per theme (role values only) + contrast.json; loaded automatically
│   ├── style.css          # Entry: Tailwind and the files of styles/
│   │
│   ├── utils/             # 🛠️ Utility Services
│   │   ├── helpers.ts     # Date formatting, ink over calendar colors, calendar grid, raw source builder
│   │   ├── theme.ts       # Applies a theme to the document and remembers it
│   │   ├── sse.ts         # Client-side new-mail polling (legacy "SSE" module name)
│   │   └── webpush.ts     # Web Push subscription
│   │
│   ├── main.ts            # 🚀 Application entry: Inter font, style.css, every theme file
│   └── types.ts           # 🏷️ TypeScript models
│
├── vite.config.ts         # ⚙️ Vite+ config: dev server and proxy, build, tests, formatter and linter
└── tsconfig.json          # 🔧 TypeScript compiler options
```

---

## 🛠️ Development & Commands

Ensure you have [Node.js](https://nodejs.org/) installed, then navigate into the `frontend/` directory to run:

### 📦 Install Dependencies

```bash
npm install
```

### 🏎️ Spin Up Local Dev Server

```bash
npm run dev
```

> [!NOTE]
> The development server uses Vite proxies configured in `vite.config.ts` to redirect backend API calls seamlessly.

### 🧪 Format, Lint and Type-Check

```bash
npm run check
```

> Runs `vp check` (format check, lint and type-check of the `.ts` files) and then `vue-tsc` for the Vue SFCs.
> `npm run fmt` formats the sources; `npx vp check --fix` also applies the lint autofixes.

### 🏗️ Build Production Assets

```bash
npm run build
```

> Compiles, minifies, and bundles assets into the backend target folder (`../web/dist/`) for seamless static delivery.

---

## 💻 Recommended IDE Setup

For the absolute best developer experience, we strongly recommend:

- **IDE**: [VS Code](https://code.visualstudio.com/)
- **Extensions**:
  - [Vue - Official](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (Volar) for rich component syntax support.
  - [TypeScript Vue Plugin](https://marketplace.visualstudio.com/items?itemName=Vue.vscode-typescript-vue-plugin) for enhanced auto-imports.

---

## 📚 Main Project Documentation

This README focuses on the frontend. For the full project (including backend, architecture, and contribution guidelines), see the main documentation:

- [Main README](../../README.md)
- [Development Guide](../../DOCUMENTS/docs/DEVELOPMENT.md) — Local development setup
- [Contributing Guide](../../DOCUMENTS/docs/CONTRIBUTING.md) — How to contribute to the project
- [Software Design Document (SDD)](../../DOCUMENTS/docs/SDD.md) — Technical architecture and design

All project documentation is located in the `DOCUMENTS/` directory.
