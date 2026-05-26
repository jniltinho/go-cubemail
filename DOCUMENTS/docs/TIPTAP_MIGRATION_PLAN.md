# TinyMCE to TipTap Migration Plan
**Project:** go-cubemail-vue (Vue 3 + TypeScript frontend)  
**Goal:** Replace TinyMCE with TipTap as the rich text editor for the email composer while preserving full functionality, visual style (Larry theme), and email HTML output quality.  
**Date:** 2026-06 (current state)  
**Status:** Draft for review and approval  
**Related:** [SDD.md](../../docs/SDD.md) (sections 2, 9.1, 9.5, 11.1, 15), [english_only.md](../../specs/english_only.md)

## 1. Executive Summary
The frontend currently integrates TinyMCE 6.8.6 exclusively through a thin wrapper (`frontend/src/components/TinyEditor.vue`) used only by `ComposerModal.vue` for HTML email composition (send, draft, reply/forward with quoted content).

**Key findings from exploration:**
- Single integration point: `ComposerModal.vue` passes `v-model="body"` (HTML string) and calls no other methods beyond the exposed `getContent()`.
- Heavy TinyMCE footprint: full distribution in `frontend/public/tinymce/`, copied to `web/dist/tinymce/` on build, large number of files and JS (~hundreds of KB minified + CSS/skins).
- Strict project style: Tailwind CSS v4 + design tokens (CSS vars: `--accent: #1B3A6B`, `--ink`, panel colors), zero border-radius, `.tbtn`/`.composer-*` classes, heavy JSDoc comments on every component, all source in English.
- English-only policy (see `DOCUMENTS/specs/english_only.md`): All new code, comments, documentation, error messages, and plan artifacts **must** be written in English. No Portuguese strings or identifiers.

**Recommendation:** Proceed with a direct replacement (no runtime dual-impl toggle, per clarification). Create a new canonical `<RichTextEditor>` component in a dedicated `rich-text/` subdirectory. Implement full toolbar and feature parity using TipTap's headless, Vue-native architecture. This enables complete control over markup and styling to match the Larry theme exactly, while dramatically reducing bundle size and eliminating the static asset directory.

**Benefits:**
- Smaller, tree-shakeable editor (TipTap + selected extensions vs. full TinyMCE).
- Native Vue 3 Composition API (no iframe or heavy global init).
- Headless = pixel-perfect integration with existing `.composer-shell`, accent colors, no `.tox` classes.
- Easier long-term maintenance and future editor swaps (the wrapper becomes the stable contract).
- Modern ecosystem with excellent TypeScript support.

**Risk level:** Medium (HTML fidelity for emails is critical; quoted replies + complex formatting must survive sanitization and render consistently in major clients).

## 2. Current State Analysis

### 2.1 TinyMCE Usage Inventory
- **Only consumer:** `frontend/src/components/ComposerModal.vue` (line ~13 import, line ~289 usage: `<TinyEditor v-model="body" />`).
- **Wrapper:** `frontend/src/components/TinyEditor.vue` (fully self-contained, ~117 lines).
  - Props: `modelValue?: string`
  - Emits: `update:modelValue`
  - Exposed: `getContent(): string`
  - Initialization: `tinymce.init({...})` in `onMounted`, careful suppress flag for two-way binding, `onBeforeUnmount` cleanup.
  - Hard-coded config (see 2.2).
- No other references in `frontend/src/` (confirmed via grep; node_modules noise excluded).
- No usage in contacts, calendar, or reading pane (sanitized HTML is rendered in sandboxed iframe in `ReadingPane.vue`).

### 2.2 Current TinyMCE Configuration (key excerpts from TinyEditor.vue:50-96)
- `base_url: '/tinymce'`, `suffix: '.min'`, skin `oxide`, `content_css: 'default'`.
- `menubar: false`, `statusbar: false`, `branding/promotion: false`, fixed `height: 433`.
- Plugins: `autolink lists link image table code emoticons charmap searchreplace wordcount`.
- Toolbar (exact):
  ```
  fontfamily fontsize | bold italic underline strikethrough | forecolor backcolor |
  alignleft aligncenter alignright | bullist numlist outdent indent |
  link image table emoticons | removeformat | code
  ```
- Custom `font_family_formats` (Arial, Comic Sans, Courier, Georgia, Segoe UI, Tahoma, Times New Roman, Trebuchet, Verdana).
- `content_style`: Segoe UI / Helvetica Neue, 13.5px, `#1A1F2A` ink, `line-height:1.6`, specific p margins. Matches project `--font-sans` and `--color-ink`.
- Placeholder support.
- Two-way binding via `ed.on('input change keyup undo redo')` + `ed.getContent()` / `setContent()`.

**Styling overrides** (in `frontend/src/style.css:239-252`):
- `.composer-rich .tox.tox-tinymce { border:0; border-top:1px solid var(--color-line-soft); }`
- Toolbar background forced to `#F5F7FA` / panel colors, hover states using `--accent-soft` etc.
- All components enforce `* { border-radius: 0 !important; }`.

### 2.3 Composer Integration & Data Flow
- `body` ref initialized via `buildInitHtml()` (plain text → `<p>`, or quoted reply with `<blockquote>` + inline styles + header).
- On send: HTML sent as `body_html`; plain-text fallback derived by crude tag stripping.
- Quoted content can contain original message HTML (from `ReadingPane` sanitized output or prefill).
- Attachments handled separately (not embedded via editor image plugin in current flows).

### 2.4 Assets & Build Impact
- `frontend/public/tinymce/` — full static copy served at `/tinymce` (dev + prod via Vite copy).
- `web/dist/tinymce/` — result of build (embedded in Go binary via `//go:embed`).
- `frontend/package.json`: `"tinymce": "^6.8.6"`.
- No Vite plugin or special handling beyond public dir.
- Removing this eliminates thousands of files and significant JS weight.

### 2.5 Project Conventions (must be followed exactly)
- All Vue SFCs: `<script setup lang="ts">`, extensive JSDoc (`@component`, `@description`, per-prop/func docs).
- Imports: relative paths, `Icon` from `./Icon.vue` (Lucide wrapper, kebab-case names), Pinia stores where needed.
- Styling: Tailwind v4 + `@theme` + `:root` CSS vars + custom component classes (`.tbtn`, `.composer-field`, `.modal-wrap`).
- No rounded corners anywhere.
- English only (identifiers, strings, comments, docs).
- Example components for patterns: `ContactModal.vue`, `DialogModal.vue`, `ComposerModal.vue` (backdrop click, reactive forms, loading flags, error display).

## 3. Goals & Non-Goals

### Goals
- 100% functional parity with current TinyMCE toolbar and behavior in the first TipTap release.
- Pixel-perfect visual match inside `.composer-shell` (toolbar colors, content typography, borders, no extra padding/frames).
- Clean, email-client-compatible HTML output (basic tags, limited inline styles, lists, links, tables, blockquotes).
- `v-model` + `getContent()` API compatibility so `ComposerModal` change is minimal (ideally 1-line swap).
- Significant bundle size reduction and removal of `public/tinymce/`.
- Full TypeScript + strict English + JSDoc compliance.
- Easy future editor replacement (wrapper + composable abstraction).

### Non-Goals (for v1)
- Real-time collaborative editing.
- Markdown source mode as primary (code view toggle is ok).
- Full custom font-size numeric input (presets or CSS classes sufficient).
- AI writing features.
- Changing the composer UX or adding new fields.

## 4. Why TipTap?
- Headless + extension system: full control over toolbar (reuse `.tbtn` + `Icon`) and content styles (Tailwind + CSS vars).
- Excellent Vue 3 support via official `@tiptap/vue-3` (Composition API friendly with `useEditor` or `new Editor`).
- Much smaller payload than TinyMCE; tree-shakeable.
- Active maintenance, great TS types, ProseMirror foundation (reliable operations).
- Proven in production for content editors; easy to style with Tailwind (see community examples).
- MIT license, no self-hosting skin hassles.

**Installation (from official docs):** `npm install @tiptap/vue-3 @tiptap/pm @tiptap/starter-kit`.

## 5. Required Feature Parity (First Release)
From clarification session — **all** current capabilities must be present:

- Core: bold, italic, underline, strikethrough, text align (left/center/right), bullet/numbered lists, indent/outdent.
- Links (insert/edit with href, optional target/title).
- Font family dropdown (same list or reasonable modern subset) + font size (presets or commands).
- Text color (forecolor) + background/highlight color.
- Insert image (URL prompt or basic upload hook; integrate with existing attachment flow where possible).
- Tables (insert basic table with rows/cols; header row optional).
- Emoticons / emoji picker + special characters (charmap).
- Code / source view toggle (raw HTML editing + sync back).
- Search & replace within editor.
- Word count (status or toolbar indicator).

**Extensions needed (core + additions):**
- `@tiptap/starter-kit` (bold, italic, strike, code, lists, history, paragraph, headings, blockquote, horizontal rule).
- `@tiptap/extension-underline`, `@tiptap/extension-link`, `@tiptap/extension-text-align`, `@tiptap/extension-color`, `@tiptap/extension-highlight`.
- `@tiptap/extension-placeholder`.
- `@tiptap/extension-image` (for image).
- `@tiptap/extension-table`, `@tiptap/extension-table-row`, etc. (for tables).
- Custom or community for font family/size (common pattern: mark + select with CSS `font-family`).
- Emoji/char: custom menu or popover + `editor.commands.insertContent`.
- Search/replace: implement simple find/replace UI + `editor.state` traversal (or note as advanced and provide basic).
- Word count: `editor.storage` or post-update count.

**Paste handling:** Configure `editorProps.handlePaste` or rely on default + schema to preserve basic formatting from Outlook/Word/Google Docs (lists, bold, links, colors).

## 6. Proposed Architecture & Components

### 6.1 Directory Structure (new)
```
frontend/src/
  components/
    rich-text/                    # Dedicated subdir (preferred per clarification)
      RichTextEditor.vue          # Public API / drop-in replacement (v-model, placeholder, height, disabled, expose getContent/setContent/focus)
      TipTapEditor.vue            # Internal impl (Editor instance, extensions, onUpdate, content sync)
      RichTextToolbar.vue         # Fully custom toolbar (groups of <button class="tbtn"> + <Icon>)
      composables/
        useTipTapEditor.ts        # Reusable logic (createEditor, commands, active states, getHTML/setHTML)
      extensions/                 # (optional) custom marks/nodes (fontSize, fontFamily, email-specific)
        fontSize.ts
        fontFamily.ts
      types.ts                    # Editor-specific TS interfaces
    TinyEditor.vue                # (kept temporarily for reference during dev, deleted in cleanup phase)
  composables/                    # (top-level if shared utils grow)
```

### 6.2 Component API (RichTextEditor.vue — stable contract)
**Props (v-model friendly + extras):**
- `modelValue?: string` (HTML)
- `placeholder?: string`
- `minHeight?: number | string` (default ~420px to match old 433)
- `disabled?: boolean`
- `readonly?: boolean`

**Emits:**
- `update:modelValue`
- `focus`, `blur`, `update` (optional for future)

**Exposed (defineExpose):**
- `getContent(): string`
- `setContent(html: string, emitUpdate?: boolean)`
- `focus()`
- `getWordCount?(): number`

**Usage (after migration):**
```vue
<RichTextEditor
  v-model="body"
  placeholder="Write your message…"
  :min-height="433"
/>
```
(Almost identical to current `<TinyEditor>` — minimal change in ComposerModal.)

**Internal delegation:** RichTextEditor renders the toolbar + the TipTap content area (or a future alternative impl). No `implementation` prop (direct replacement).

### 6.3 Styling Strategy
- Container reuses `.composer-field.composer-rich` (or renders the inner content only and lets parent control).
- New classes (add to `style.css`):
  - `.rte-toolbar` (bg-panel-2, border-b line-soft, flex wrap, gap-0.5, padding)
  - `.rte-content` (bg-white, min-height, padding:10px 12px, font-family/sizing matching old content_style, overflow-auto)
  - Active button states: reuse `.tbtn` + `data-active` or Tailwind `bg-accent-soft` when `editor.isActive(...)`.
- Content styles applied via:
  - Tailwind on the editor div + ProseMirror classes.
  - Or a small `<style>` block / CSS layer inside the component.
  - Match exactly: 13.5px, Segoe/Helvetica, ink color, line-height, p margins, blockquote left border using accent.
- Zero border-radius inherited from global rule.
- Toolbar buttons: use existing `.tbtn` (small height 26px, padding, hover accent-soft) + `Icon` components. Group separators with subtle borders or `|` text.
- Color pickers: native `<input type="color">` or small swatch popovers (keep simple, no heavy UI lib).
- Emoji/char picker: lightweight popover (absolute, Tailwind, grid of buttons) — no external dep.

**Result:** No `.tox` anything. Looks native to the Larry composer.

### 6.4 Data & Lifecycle Handling
- Use `useEditor` (preferred for script-setup) or `new Editor({ onCreate, onUpdate, onDestroy })` + `onBeforeUnmount(() => editor?.destroy())`.
- Two-way sync: `onUpdate` → `emit('update:modelValue', editor.getHTML())`.
- Parent → editor: `watch(() => props.modelValue, (val) => { if (editor && val !== editor.getHTML()) editor.commands.setContent(val || '', false) })`.
- Suppress flag or transaction checks to prevent loops (same pattern as current).
- Init content for quoted replies: supports full HTML (blockquotes, inline styles, p tags) — TipTap parses via DOM parser; test thoroughly.
- Cleanup on unmount.

### 6.5 Toolbar Implementation (RichTextToolbar.vue)
- Receive `editor: Editor` as prop (or via provide/inject from parent).
- Groups matching old toolbar order.
- Buttons call `editor.chain().focus().toggleBold().run()` etc.
- Active state: `editor.isActive('bold')` → apply highlight class.
- Dropdowns (font, size, colors, emoticons, charmap): small custom menus using absolute positioned divs + Tailwind (no floating-ui dep unless already present).
- Image/table: simple prompts or mini forms for now (URL for image, numeric inputs for table dimensions).
- Search/replace: modal-like small panel inside composer or floating (reuse DialogModal patterns?).
- Code view: toggle between EditorContent and a `<textarea>` bound to raw HTML.
- Word count: simple display in toolbar corner or status area.

All labels and tooltips in English.

## 7. Phased Migration Plan

### Phase 0 — Preparation (1-2 days)
- Create branch `feat/tiptap-editor`.
- Update `frontend/package.json`: add TipTap deps, **keep** tinymce temporarily.
- Run `npm install` in frontend/.
- Add new CSS rules for `.rte-*` to `style.css` (non-breaking).
- (Optional) Spike one file to validate basic TipTap + Tailwind toolbar works in the composer shell.

### Phase 1 — Core Components (3-5 days)
- Create `frontend/src/components/rich-text/` directory and files.
- Implement `useTipTapEditor.ts` composable (extensions list, editor factory, command helpers, active states).
- Implement `RichTextToolbar.vue` (all groups; start with core formatting + link + lists, iterate to colors/tables/image/emoji).
- Implement `TipTapEditor.vue` (EditorContent + lifecycle + v-model glue).
- Implement `RichTextEditor.vue` (public API, forwards to TipTap, JSDoc, matches TinyEditor surface).
- Full JSDoc on every export/function.
- Unit-test the composable in isolation if possible (or manual in dev).

**Verification gate:** Mount `<RichTextEditor>` standalone in a test view or via Vite HMR; basic typing, formatting, link, list works; HTML output looks reasonable.

### Phase 2 — Integration & Parity (2-4 days)
- Update `ComposerModal.vue`: swap import + usage to `<RichTextEditor v-model="body" placeholder="..." />`.
- Preserve `buildInitHtml()` logic (it produces the HTML the editor must accept).
- Test all flows:
  - New blank compose.
  - Reply / Reply-all / Forward (quoted content with original HTML + styles).
  - Font family/size/color changes.
  - Lists, indent, align.
  - Links (create, edit, remove).
  - Insert image (URL), table (various sizes).
  - Emoji + special chars.
  - Source view toggle + edit + sync.
  - Search/replace.
  - Word count accuracy.
- Visual regression: side-by-side screenshots (old Tiny vs new) inside the exact `.composer-shell` — toolbar bg, button states, content font/padding/margins, focus rings, disabled states must match.
- Cross-browser: Chrome, Firefox, Safari (editor contenteditable quirks).

**Verification gate:** Send 10+ varied test messages (with all formatting + quote) to Gmail, Outlook.com, Apple Mail, Thunderbird. Compare rendered result + "view source" HTML fidelity. No major breakage.

### Phase 3 — Hardening & Cleanup (1-2 days)
- Remove TinyMCE from `package.json` and `frontend/package-lock.json` (or let npm prune).
- Delete `frontend/public/tinymce/` (and confirm `web/dist/tinymce/` is gone after next build).
- Delete or archive `TinyEditor.vue` (or move to `legacy/` for 30 days).
- Update any references in `DOCUMENTS/docs/SDD.md` (tech stack, component diagram, 9.1, 11.1, 15).
- Run full `npm run build` + `make build` (or equivalent) and verify Go binary serves correctly, no 404 on old /tinymce.
- Add short "Editor Migration" note to `DOCUMENTS/docs/DEVELOPMENT.md` or a new `MIGRATION_NOTES.md`.
- Accessibility spot-check: keyboard navigation in toolbar/content, aria labels on buttons, focus management.

**Verification gate:** `make dev` works cleanly; no console errors; composer fully functional; bundle size measurably smaller (use `npm run build` stats or `du` on dist).

### Phase 4 — Documentation & Polish (0.5-1 day)
- Write or update usage examples in the plan (or a `frontend/src/components/rich-text/README.md`).
- Final English-only review of all new files.
- Update SDD "Rich text editor" row from "TinyMCE" to "TipTap (via RichTextEditor)".
- Close the feature branch with clear PR description referencing this plan.

## 8. Risks, Mitigations & Rollback

| Risk | Likelihood | Impact | Mitigation | Rollback |
|------|------------|--------|------------|----------|
| HTML output differs in email clients (esp. tables, colors, lists) | Medium | High | Extensive client testing in Phase 2; conservative extension config; preserve important inline styles | Keep TinyMCE branch alive 7 days post-merge; feature flag in Composer if needed |
| Complex extensions (table + image) add size or bugs | Medium | Medium | Tree-shake; lazy-load heavy UI (emoji picker); implement tables as basic grid first | Disable table/image buttons temporarily; document as "beta" |
| Two-way binding loops or content loss on quoted replies | Low | High | Mirror current suppress pattern + transaction checks; test with real prefill HTML early | Revert single line in ComposerModal to TinyEditor |
| Font family/size exact parity hard in TipTap | Medium | Low | Provide 6-8 common presets; document that advanced fonts may map to system defaults | N/A (acceptable simplification) |
| Team unfamiliar with ProseMirror mental model | Medium | Low | Good JSDoc + composable abstraction; pair review | N/A |

**Rollback path (fast):** Revert the 1-3 line change in `ComposerModal.vue` + restore TinyMCE import if removed. Re-add public assets from git if deleted (they are large — prefer keeping the dir in a backup commit).

## 9. Effort Estimate & Team Impact
- Total: 7-12 developer days (experienced Vue + TipTap).
- Skills required: strong Vue 3 Composition, CSS/Tailwind, basic ProseMirror concepts (docs are excellent).
- No backend changes.
- Low risk to other features (editor is isolated).

## 10. Post-Migration Opportunities
- Replace custom emoji/char picker with a small shared component usable elsewhere.
- Add "signature" insertion button (once user signatures feature exists).
- Experiment with TipTap Collab or Yjs if real-time compose is desired later.
- Consider exposing editor commands to parent for future toolbar customization.

## 11. Implementation Steps / PR Plan (Topological Order)

1. **Prep PR** (or commit): package.json updates + new CSS tokens/classes + directory skeleton + this plan as `DOCUMENTS/docs/TIPTAP_MIGRATION_PLAN.md`.
2. **Core foundation PR**: `useTipTapEditor.ts` + basic `TipTapEditor.vue` + `RichTextEditor.vue` (core formatting + lists + link + placeholder only). Include JSDoc and English strings.
3. **Toolbar & advanced features PR**: Full `RichTextToolbar.vue` + color, font, table, image, emoji, source, search, wordcount. Split if too large.
4. **Integration & test PR**: Swap in `ComposerModal.vue`; add visual + send tests; screenshots in PR description.
5. **Cleanup PR**: Remove tinymce dep + assets + legacy file + SDD/doc updates. Verify production build.
6. **Optional polish PR**: Accessibility, performance, any follow-ups from client testing feedback.

Each PR must pass manual `make dev` + composer smoke test + English-only review.

## 12. References & Citations
- Official TipTap Vue 3 installation & `useEditor`: https://tiptap.dev/docs/editor/getting-started/install/vue3 [web:0]
- Community migration notes & patterns: Tiny blog & GitHub discussions [web:4][web:5]
- Current implementation: `frontend/src/components/TinyEditor.vue:44-109` and `ComposerModal.vue:289`
- Styling targets: `frontend/src/style.css:214-252` (composer + .tox overrides)
- Project rules: `DOCUMENTS/specs/english_only.md`, `DOCUMENTS/docs/SDD.md:57` (tech stack), `9.5` (design tokens)
- Icon usage pattern: `frontend/src/components/Icon.vue`
- Component documentation standard: all existing `.vue` files (JSDoc blocks)

---

**Approval Checklist (for reviewer / author)**
- [ ] Full feature parity confirmed in manual + email client testing.
- [ ] Visual match (screenshots attached).
- [ ] No TypeScript or runtime errors in dev + prod build.
- [ ] All new files 100% English + JSDoc.
- [ ] Bundle size reduced (measure before/after).
- [ ] Assets (`public/tinymce`) removed and no 404s.
- [ ] SDD and this plan updated.
- [ ] Rollback tested (revert 1 line).

**Next action after approval:** Create feature branch and begin Phase 0/1 implementation following the numbered PR plan.

---
*This plan was created following the go-cubemail-vue project conventions and the strict English-only policy. All proposed code and documentation will be in English.*
