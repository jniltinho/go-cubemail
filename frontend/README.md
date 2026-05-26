# 📬 Cubemail Frontend — Vue 3 + TypeScript + Vite

A modern, responsive webmail interface built as the frontend for **go-cubemail-vue**. It delivers a clean three-column experience inspired by the classic Roundcube "Larry" theme, written as a fully typed Vue 3 + TypeScript SPA that is compiled and embedded into the Go binary at build time.

---

## 🚀 Key Features

- **⚡ Blazing Fast Build**: Powered by Vite and Rolldown/ESbuild for instant Hot Module Replacement (HMR).
- **🔒 Strict Type Safety**: Fully written in TypeScript with comprehensive interface modeling for all mail entities, contacts, and state.
- **📦 Clean State Management**: Pinia stores with a deliberately split mail module (`stores/mail/`) to keep the main store readable while isolating API, folder, composer, mail, and contact actions.
- **💡 Keyboard Shortcuts**: Vim-style navigation (`j`/`k`), compose (`c`), reply (`r`), archive (`e`), delete (`#` or `Delete`). Implemented globally in `App.vue`.
- **📧 Isolated Mail Rendering**: Secure read-pane framing utilizing sandboxed iframes to completely isolate external HTML contents and block unauthorized scripts.
- **🎨 Runtime Theming**: In-app accent color picker that updates CSS custom properties live (no reload).
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
├── src/
│   ├── components/        # 🎨 Vue 3 SFCs (Single File Components)
│   │   ├── AppBar.vue     # Top global header & global search bar
│   │   ├── AppSidebar.vue # Expandable mailbox navigation tree & quota gauge
│   │   ├── AppToolbar.vue # Mail actions toolbar (Compose, Delete, Move, Reply)
│   │   ├── MailList.vue        # Scrollable message list with checkboxes, sorting, bulk actions
│   │   ├── ReadingPane.vue     # Headers + sandboxed HTML body (iframe) + attachments + calendar invites
│   │   ├── ComposerModal.vue   # Full composer with TinyMCE, autocomplete, attachments
│   │   ├── ContactsPane.vue    # Address book with import/export
│   │   ├── CalendarPane.vue    # Monthly grid view (demo data)
│   │   └── ...                 # AppBar, AppSidebar, AppToolbar, modals (Dialog, Toast, SourceViewer, ContactModal, etc.)
│   │
│   ├── stores/            # 🍍 Pinia State Modules
│   │   ├── auth.ts        # Session, CSRF tokens, and credentials authorization
│   │   ├── dialog.ts      # Global interactive pop-ups stack (Alert, Confirm)
│   │   ├── toast.ts       # Visual temporary notifications queues
│   │   └── mail/          # Unified Mail Store Architecture (deliberately split for maintainability)
│   │       ├── index.ts   # Main store assembly + getters + cross-cutting concerns
│   │       ├── api.ts     # All REST calls (fetchFolderMessages, loadFromApi, fetchMessageBody…)
│   │       ├── mailActions.ts    # Reply/forward, flag, move, delete, archive, select
│   │       ├── folderActions.ts  # Create/rename/delete folders + context menu handling
│   │       ├── composerActions.ts # Compose, reply, forward, draft, source viewer
│   │       ├── contactActions.ts  # Contacts CRUD + CSV/VCard import/export + autocomplete
│   │       ├── constants.ts & mockData.ts # Folder name mapping + demo calendar events
│   │
│   ├── utils/             # 🛠️ Utility Services
│   │   ├── helpers.ts     # Date formatting, accent color derivation, calendar grid, raw source builder
│   │   └── sse.ts         # Client-side new-mail polling (10 min) + Web Audio notifications (legacy "SSE" module name)
│   │
│   ├── main.ts            # 🚀 Application entry bootstrap
│   └── types.ts           # 🏷️ Unified static TypeScript models
│
├── vite.config.ts         # ⚙️ Vite & proxy routing configurations
└── tsconfig.json          # 🔧 Strictly configured TypeScript compiler options
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

### 🧪 Perform Strict Type Checking
```bash
npm run type-check
```
> Runs `vue-tsc` in non-emitting mode to double-check structural safety across all Vue SFCs and TypeScript modules.

### 🏗️ Build Production Assets
```bash
npm run build
```
> Compiles, minifies, and bundles assets into the backend target folder (`../web/dist/`) for seamless static delivery.

---

## 💻 Recommended IDE Setup

For the absolute best developer experience, we strongly recommend:
* **IDE**: [VS Code](https://code.visualstudio.com/)
* **Extensions**:
  * [Vue - Official](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (Volar) for rich component syntax support.
  * [TypeScript Vue Plugin](https://marketplace.visualstudio.com/items?itemName=Vue.vscode-typescript-vue-plugin) for enhanced auto-imports.

---

## 📚 Main Project Documentation

This README focuses on the frontend. For the full project (including backend, architecture, and contribution guidelines), see the main documentation:

- [Main README](../../README.md)
- [Development Guide](../../DOCUMENTS/docs/DEVELOPMENT.md) — Local development setup
- [Contributing Guide](../../DOCUMENTS/docs/CONTRIBUTING.md) — How to contribute to the project
- [Software Design Document (SDD)](../../DOCUMENTS/docs/SDD.md) — Technical architecture and design

All project documentation is located in the `DOCUMENTS/` directory.
