# Rich Text Editor (TipTap)

This directory contains the new TipTap-based rich text editor that replaces the legacy TinyMCE integration.

## Public API
- `<RichTextEditor v-model="html" placeholder="..." :min-height="433" />`
- Exposed methods: `getContent()`, `setContent(html)`, `focus()`, `getWordCount()`

## Structure
- `RichTextEditor.vue` — stable public wrapper (drop-in for old TinyEditor)
- `TipTapEditor.vue` — internal implementation + source view + lifecycle
- `RichTextToolbar.vue` — fully custom toolbar using project `.tbtn` + `Icon`
- `composables/useTipTapEditor.ts` — editor factory + extensions + helpers
- `types.ts` — shared interfaces

## Styling
Uses new `.rte-toolbar` / `.rte-content` classes defined in `src/style.css`.
Matches the Larry theme (no radius, accent colors, Segoe/Helvetica, panel backgrounds).

## Migration Notes
See the top-level migration plan document (`DOCUMENTS/docs/TIPTAP_MIGRATION_PLAN.md`).

All code and docs are in English per project policy.

## Known Polish Items
- Tighten Editor typing interop between @tiptap/core and @tiptap/vue-3 (current @ts-ignore in TipTapEditor.vue)
- Full font-family/size commands (currently basic presets)
- Image upload integration with the existing attachment flow
- More exhaustive paste handling tests from Outlook/Word

## Usage Example
```vue
<RichTextEditor
  v-model="body"
  placeholder="Write your message…"
  :min-height="433"
/>
```
