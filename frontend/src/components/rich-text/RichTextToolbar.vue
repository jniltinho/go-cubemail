<script setup lang="ts">
/**
 * @component RichTextToolbar
 * @description Custom toolbar for the TipTap-based rich text editor.
 * Uses the project's existing `.tbtn` buttons, `Icon` component, accent colors,
 * and zero-radius design language. Provides all controls from the legacy
 * TinyMCE toolbar (formatting, links, lists, colors, tables, emoji, source, etc.).
 * All strings, comments, and identifiers are in English.
 */

import { computed, ref } from 'vue'
import type { Editor } from '@tiptap/core'
import Icon from '../Icon.vue'

/** Props */
const props = defineProps<{
  /** TipTap Editor instance (or ref.value). May be null during init. */
  editor: Editor | null
}>()

/** Reactive editor for template use */
const ed = computed(() => props.editor)

/** Simple local state for popovers (emoji, color, table, link, etc.) */
const showEmoji = ref(false)
const showColor = ref(false)
const showBgColor = ref(false)
const showTable = ref(false)
const showLink = ref(false)
const linkUrl = ref('')

/** Helper: is the given mark or node active? */
function isActive(name: string, attrs?: Record<string, any>) {
  return ed.value?.isActive(name, attrs) ?? false
}

/** Run a chain command with focus */
function run(cmd: () => void) {
  if (!ed.value) return
  ed.value.chain().focus()[cmd.name as any]?.() || cmd()
  // Note: actual calls below use the chain directly for clarity
}

/** Core formatting */
function toggleBold() { ed.value?.chain().focus().toggleBold().run() }
function toggleItalic() { ed.value?.chain().focus().toggleItalic().run() }
function toggleUnderline() { ed.value?.chain().focus().toggleUnderline().run() }
function toggleStrike() { ed.value?.chain().focus().toggleStrike().run() }

/** Alignment */
function alignLeft() { ed.value?.chain().focus().setTextAlign('left').run() }
function alignCenter() { ed.value?.chain().focus().setTextAlign('center').run() }
function alignRight() { ed.value?.chain().focus().setTextAlign('right').run() }

/** Lists & structure */
function toggleBullet() { ed.value?.chain().focus().toggleBulletList().run() }
function toggleOrdered() { ed.value?.chain().focus().toggleOrderedList().run() }
function indent() { ed.value?.chain().focus().sinkListItem('listItem').run() }
function outdent() { ed.value?.chain().focus().liftListItem('listItem').run() }

/** Link */
function openLinkDialog() {
  if (!ed.value) return
  const previous = ed.value.getAttributes('link').href || ''
  linkUrl.value = previous
  showLink.value = true
}
function applyLink() {
  if (!ed.value) return
  const url = linkUrl.value.trim()
  if (url) {
    ed.value.chain().focus().setLink({ href: url }).run()
  } else {
    ed.value.chain().focus().unsetLink().run()
  }
  showLink.value = false
  linkUrl.value = ''
}
function removeLink() {
  ed.value?.chain().focus().unsetLink().run()
  showLink.value = false
}

/** Colors (simple native inputs for v1) */
function setTextColor(color: string) {
  ed.value?.chain().focus().setColor(color).run()
  showColor.value = false
}
function setHighlight(color: string) {
  ed.value?.chain().focus().setHighlight({ color }).run()
  showBgColor.value = false
}
function clearFormat() {
  ed.value?.chain().focus().clearNodes().unsetAllMarks().run()
}

/** Table */
function insertTable() {
  ed.value?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
  showTable.value = false
}

/** Emoji / special chars (basic set for v1) */
const emojiList = ['😀','😉','🙂','😢','👍','👎','❤️','✅','❌','🔥','📎','📅']
function insertEmoji(e: string) {
  ed.value?.chain().focus().insertContent(e).run()
  showEmoji.value = false
}

/** Source / word count exposed via parent for now */
function toggleSource() {
  // Parent (TipTapEditor) listens for a custom event or we emit
  // For simplicity the parent owns the source toggle state.
  // Here we just emit an event the parent can handle.
  // (Implementation detail kept in TipTapEditor for now.)
}

/** Word count (display only; logic lives in composable/parent) */
const wordCount = computed(() => {
  if (!ed.value) return 0
  const text = ed.value.getText()
  return text.trim().split(/\s+/).filter(Boolean).length
})

/** Disabled state */
const disabled = computed(() => !ed.value || !ed.value.isEditable)
</script>

<template>
  <div class="rte-toolbar" role="toolbar" aria-label="Rich text formatting">
    <!-- Text style -->
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('bold') }" :disabled="disabled" @click="toggleBold" title="Bold">
      <Icon name="bold" :size="13" />
    </button>
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('italic') }" :disabled="disabled" @click="toggleItalic" title="Italic">
      <Icon name="italic" :size="13" />
    </button>
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('underline') }" :disabled="disabled" @click="toggleUnderline" title="Underline">
      <Icon name="underline" :size="13" />
    </button>
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('strike') }" :disabled="disabled" @click="toggleStrike" title="Strikethrough">
      <Icon name="strikethrough" :size="13" />
    </button>

    <span class="rte-separator" />

    <!-- Font / size (basic presets for v1; can be expanded) -->
    <select class="tbtn" :disabled="disabled" title="Font family (future)">
      <option value="">Font</option>
      <option value="Segoe UI, Helvetica, Arial, sans-serif">Segoe UI</option>
      <option value="Arial, Helvetica, sans-serif">Arial</option>
      <option value="'Times New Roman', Times, serif">Times New Roman</option>
      <option value="Courier New, Courier, monospace">Courier New</option>
      <option value="Georgia, serif">Georgia</option>
    </select>
    <select class="tbtn" :disabled="disabled" title="Font size (future)">
      <option value="">Size</option>
      <option value="12px">12</option>
      <option value="13.5px">13.5</option>
      <option value="16px">16</option>
      <option value="18px">18</option>
    </select>

    <span class="rte-separator" />

    <!-- Colors -->
    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showColor = !showColor" title="Text color">
        <Icon name="palette" :size="13" />
      </button>
      <div v-if="showColor" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow" style="min-width:180px">
        <div class="grid grid-cols-6 gap-1">
          <button v-for="c in ['#1A1F2A','#B22B2B','#1F7A45','#1B3A6B','#E0A40C','#6b7280']" :key="c"
                  class="w-6 h-6 border border-line" :style="{background:c}" @click="setTextColor(c)" />
        </div>
        <input type="color" class="mt-2 w-full" @input="(e) => setTextColor((e.target as HTMLInputElement).value)" />
      </div>
    </div>

    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showBgColor = !showBgColor" title="Highlight color">
        <Icon name="highlighter" :size="13" />
      </button>
      <div v-if="showBgColor" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow" style="min-width:180px">
        <div class="grid grid-cols-6 gap-1">
          <button v-for="c in ['#fff59d','#a5f3fc','#bbf7d0','#fecaca','#e0e7ff','#f3e8ff']" :key="c"
                  class="w-6 h-6 border border-line" :style="{background:c}" @click="setHighlight(c)" />
        </div>
        <input type="color" class="mt-2 w-full" @input="(e) => setHighlight((e.target as HTMLInputElement).value)" />
      </div>
    </div>

    <span class="rte-separator" />

    <!-- Alignment -->
    <button type="button" class="tbtn" :disabled="disabled" @click="alignLeft" title="Align left">
      <Icon name="align-left" :size="13" />
    </button>
    <button type="button" class="tbtn" :disabled="disabled" @click="alignCenter" title="Align center">
      <Icon name="align-center" :size="13" />
    </button>
    <button type="button" class="tbtn" :disabled="disabled" @click="alignRight" title="Align right">
      <Icon name="align-right" :size="13" />
    </button>

    <span class="rte-separator" />

    <!-- Lists -->
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('bulletList') }" :disabled="disabled" @click="toggleBullet" title="Bullet list">
      <Icon name="list" :size="13" />
    </button>
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('orderedList') }" :disabled="disabled" @click="toggleOrdered" title="Numbered list">
      <Icon name="list-ordered" :size="13" />
    </button>
    <button type="button" class="tbtn" :disabled="disabled" @click="outdent" title="Decrease indent">
      <Icon name="outdent" :size="13" />
    </button>
    <button type="button" class="tbtn" :disabled="disabled" @click="indent" title="Increase indent">
      <Icon name="indent" :size="13" />
    </button>

    <span class="rte-separator" />

    <!-- Link -->
    <button type="button" class="tbtn" :class="{ 'rte-active': isActive('link') }" :disabled="disabled" @click="openLinkDialog" title="Insert or edit link">
      <Icon name="link" :size="13" />
    </button>

    <!-- Image (basic URL for v1; wired via parent expose or future toolbar emit) -->
    <button type="button" class="tbtn" :disabled="disabled" title="Insert image (future)">
      <Icon name="image" :size="13" />
    </button>

    <!-- Table -->
    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showTable = !showTable" title="Insert table">
        <Icon name="table" :size="13" />
      </button>
      <div v-if="showTable" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow text-[12px]">
        <button class="tbtn w-full" @click="insertTable">Insert 3×3 table</button>
        <!-- Future: row/col inputs -->
      </div>
    </div>

    <!-- Emoji -->
    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showEmoji = !showEmoji" title="Insert emoji">
        <Icon name="smile" :size="13" />
      </button>
      <div v-if="showEmoji" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow grid grid-cols-6 gap-1 text-lg" style="min-width: 220px">
        <button v-for="e in emojiList" :key="e" class="hover:bg-accent-soft p-1" @click="insertEmoji(e)">{{ e }}</button>
      </div>
    </div>

    <span class="rte-separator" />

    <!-- Remove format -->
    <button type="button" class="tbtn" :disabled="disabled" @click="clearFormat" title="Clear formatting">
      <Icon name="remove-formatting" :size="13" />
    </button>

    <!-- Source view (handled by parent for now) -->
    <button type="button" class="tbtn" :disabled="disabled" @click="$emit('toggle-source')" title="Toggle HTML source">
      <Icon name="code" :size="13" />
    </button>

    <!-- Word count (read-only display) -->
    <span class="ml-auto text-[10.5px] text-ink-mute px-2 tabular-nums" title="Word count">
      {{ wordCount }} words
    </span>

    <!-- Link dialog (simple) -->
    <div v-if="showLink" class="absolute z-50 mt-1 left-0 bg-white border border-line p-2 shadow flex gap-2" style="top:100%">
      <input v-model="linkUrl" class="border border-line px-2 text-sm" placeholder="https://..." style="width:260px" />
      <button class="tbtn" @click="applyLink">OK</button>
      <button class="tbtn" @click="removeLink">Remove</button>
      <button class="tbtn" @click="showLink = false">Cancel</button>
    </div>
  </div>
</template>