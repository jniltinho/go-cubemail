# 📬 Cubemail Frontend — Vue 3 + TypeScript + Vite

A premium, modern, and highly responsive webmail interface built as the frontend client for **Cubemail**. Designed with focus on raw performance, rich aesthetics (harmonious dark themes, glassmorphism, responsive grids), and smooth user interaction.

---

## 🚀 Key Features

- **⚡ Blazing Fast Build**: Powered by Vite and Rolldown/ESbuild for instant Hot Module Replacement (HMR).
- **🔒 Strict Type Safety**: Fully written in TypeScript with comprehensive interface modeling for all mail entities, contacts, and state.
- **📦 Clean State Management**: Modular state architecture handled by Pinia with atomic sub-stores (Auth, Mail, Toast, Dialog).
- **💡 Rich Keyboard Hotkeys**: Vim-like navigation (`J`/`K` to browse), quick actions (`C` to compose, `R` to reply, `#` to delete, `E` to archive) for a professional workflow.
- **📧 Isolated Mail Rendering**: Secure read-pane framing utilizing sandboxed iframes to completely isolate external HTML contents and block unauthorized scripts.
- **💬 Live Synchronization**: Real-time polling via Server-Sent Events (SSE) featuring browser notifications and auditory alerts.

---

## 📖 Code Quality & Documentation

To maintain the highest level of code quality and team cooperation, **100% of the frontend codebase has been fully documented in English** using rich JSDoc/TSDoc standards. 

This enables rich, instant IDE tooltips, precise autocomplete recommendations, and auto-typing capabilities across your workspace:

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
│   │   ├── MailList.vue   # Dynamic email table with bulk operations
│   │   ├── ReadingPane.vue# Sandboxed rich email content frame
│   │   ├── ComposerModal.vue # Premium mail composer (TinyMCE integration)
│   │   └── ...            # Utility icons and specialized modal overlays
│   │
│   ├── stores/            # 🍍 Pinia State Modules
│   │   ├── auth.ts        # Session, CSRF tokens, and credentials authorization
│   │   ├── dialog.ts      # Global interactive pop-ups stack (Alert, Confirm)
│   │   ├── toast.ts       # Visual temporary notifications queues
│   │   └── mail/          # Unified Mail Store Architecture
│   │       ├── api.ts     # Server REST API calls & pagination
│   │       ├── mailActions.ts # Bulk mail deletions, moves, and state toggles
│   │       └── ...        # Composer, folders, and local address book actions
│   │
│   ├── utils/             # 🛠️ Utility Services
│   │   ├── helpers.ts     # Data adapters, extensions colors, calendar cells
│   │   └── sse.ts         # Server-Sent Events live polling loop
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
