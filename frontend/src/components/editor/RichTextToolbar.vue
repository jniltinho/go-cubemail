<script setup lang="ts">
/**
 * @component RichTextToolbar
 * @description Custom toolbar for the TipTap-based rich text editor.
 * Uses the project's existing `.tbtn` buttons, `Icon` component, accent colors,
 * and zero-radius design language. Provides all controls from the legacy
 * TinyMCE toolbar (formatting, links, lists, colors, tables, emoji, source, etc.).
 * All strings, comments, and identifiers are in English.
 */

import { computed, ref, nextTick } from 'vue'
import type { Editor } from '@tiptap/core'
import { NodeSelection } from '@tiptap/pm/state'
import Icon from '../Icon.vue'

/** Props */
const props = defineProps<{
  /** TipTap Editor instance (or ref.value). May be null during init. */
  editor: Editor | null
}>()

/** Reactive editor for template use */
const ed = computed(() => props.editor)

/** Current font family at the cursor position (empty string = default) */
const currentFontFamily = computed<string>(() =>
  ed.value?.getAttributes('textStyle').fontFamily ?? ''
)

/** Current font size at the cursor position (empty string = default) */
const currentFontSize = computed<string>(() =>
  ed.value?.getAttributes('textStyle').fontSize ?? ''
)

/** Apply or clear font family on the selected text */
function onFontFamilyChange(e: Event) {
  const value = (e.target as HTMLSelectElement).value
  if (!ed.value) return
  if (!value) {
    ed.value.chain().focus().unsetFontFamily().run()
  } else {
    ed.value.chain().focus().setFontFamily(value).run()
  }
}

/** Apply or clear font size on the selected text */
function onFontSizeChange(e: Event) {
  const value = (e.target as HTMLSelectElement).value
  if (!ed.value) return
  if (!value) {
    ed.value.chain().focus().unsetFontSize().run()
  } else {
    ed.value.chain().focus().setFontSize(value).run()
  }
}

/** Simple local state for popovers (emoji, color, table, link, etc.) */
const showEmoji = ref(false)
const showColor = ref(false)
const showBgColor = ref(false)
const showTable = ref(false)
const showLink = ref(false)
const linkUrl = ref('')
const showImage = ref(false)
const imageUrl = ref('')
const linkInputRef = ref<HTMLInputElement | null>(null)
const imageInputRef = ref<HTMLInputElement | null>(null)

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

/** Alignment — handles both regular text selection and NodeSelection (e.g. clicking an image).
 *  When a NodeSelection is active (inline image clicked), ProseMirror selects the node
 *  itself rather than the surrounding paragraph. selectParentNode() moves up to the <p>
 *  so setTextAlign has a matching type to update.
 */
function applyAlign(alignment: string) {
  if (!ed.value) return
  const isNodeSel = ed.value.state.selection instanceof NodeSelection
  const chain = ed.value.chain().focus()
  if (isNodeSel) {
    chain.selectParentNode().setTextAlign(alignment).run()
  } else {
    chain.setTextAlign(alignment).run()
  }
}
function alignLeft()   { applyAlign('left') }
function alignCenter() { applyAlign('center') }
function alignRight()  { applyAlign('right') }

/** Lists & structure */
function toggleBullet() { ed.value?.chain().focus().toggleBulletList().run() }
function toggleOrdered() { ed.value?.chain().focus().toggleOrderedList().run() }

function isInList(): boolean {
  if (!ed.value) return false
  const { from, to } = ed.value.state.selection
  let found = false
  ed.value.state.doc.nodesBetween(from, to, (node) => {
    if (node.type.name === 'listItem') { found = true; return false }
  })
  return found
}

function indent() {
  if (!ed.value) return
  if (isInList()) {
    ed.value.chain().focus().sinkListItem('listItem').run()
  } else {
    ed.value.chain().focus().indent().run()
  }
}

function outdent() {
  if (!ed.value) return
  if (isInList()) {
    ed.value.chain().focus().liftListItem('listItem').run()
  } else {
    ed.value.chain().focus().outdent().run()
  }
}

/** Link */
function openLinkDialog() {
  if (!ed.value) return
  linkUrl.value = ed.value.getAttributes('link').href || ''
  showLink.value = true
  nextTick(() => linkInputRef.value?.focus())
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

/** Image */
function openImageDialog() {
  imageUrl.value = ''
  showImage.value = true
  nextTick(() => imageInputRef.value?.focus())
}
function applyImage() {
  const url = imageUrl.value.trim()
  if (url && ed.value) {
    ed.value.chain().focus().setImage({ src: url }).run()
  }
  showImage.value = false
  imageUrl.value = ''
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

    <!-- Font / size -->
    <select
      class="tbtn"
      :disabled="disabled"
      :value="currentFontFamily"
      title="Font family"
      @change="onFontFamilyChange"
    >
      <option value="">Font</option>
      <option value="Segoe UI, Helvetica, Arial, sans-serif">Segoe UI</option>
      <option value="Arial, Helvetica, sans-serif">Arial</option>
      <option value="'Times New Roman', Times, serif">Times New Roman</option>
      <option value="Courier New, Courier, monospace">Courier New</option>
      <option value="Georgia, serif">Georgia</option>
      <option value="Tahoma, Geneva, sans-serif">Tahoma</option>
      <option value="Verdana, Geneva, sans-serif">Verdana</option>
    </select>
    <select
      class="tbtn"
      :disabled="disabled"
      :value="currentFontSize"
      title="Font size"
      @change="onFontSizeChange"
    >
      <option value="">Size</option>
      <option value="10px">10</option>
      <option value="12px">12</option>
      <option value="13.5px">13.5</option>
      <option value="16px">16</option>
      <option value="18px">18</option>
      <option value="24px">24</option>
      <option value="36px">36</option>
    </select>

    <span class="rte-separator" />

    <!-- Colors -->
    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showColor = !showColor" title="Text color">
        <Icon name="palette" :size="13" />
      </button>
      <div v-if="showColor" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow" style="min-width:216px">
        <div class="grid grid-cols-8 gap-1">
          <button v-for="c in [
            '#000000','#434343','#666666','#999999','#CCCCCC','#D9D9D9','#F3F3F3','#FFFFFF',
            '#FF0000','#E53935','#B22B2B','#880E4F','#9C27B0','#673AB7','#3F51B5','#1A237E',
            '#1565C0','#1B3A6B','#0277BD','#006064','#1F7A45','#2E7D32','#33691E','#827717',
            '#F57F17','#E65100','#E0A40C','#BF360C','#795548','#6b7280','#546E7A','#37474F',
          ]" :key="c"
              class="w-6 h-6 border border-line hover:scale-110 transition-transform"
              :style="{background:c}"
              :title="c"
              @click="setTextColor(c)" />
        </div>
        <input type="color" class="mt-2 w-full h-6 cursor-pointer" @input="(e) => setTextColor((e.target as HTMLInputElement).value)" />
      </div>
    </div>

    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="showBgColor = !showBgColor" title="Highlight color">
        <Icon name="highlighter" :size="13" />
      </button>
      <div v-if="showBgColor" class="absolute z-50 mt-1 bg-white border border-line p-2 shadow" style="min-width:216px">
        <div class="grid grid-cols-8 gap-1">
          <button v-for="c in [
            '#FFFF00','#fff59d','#FFE082','#FFCC80','#FFAB91','#EF9A9A','#F48FB1','#CE93D8',
            '#B39DDB','#90CAF9','#a5f3fc','#80DEEA','#A5D6A7','#bbf7d0','#E6EE9C','#FFFFFF',
            '#fecaca','#e0e7ff','#f3e8ff','#DCEDC8','#B2EBF2','#B2DFDB','#F0F4C3','#FCE4EC',
          ]" :key="c"
              class="w-6 h-6 border border-line hover:scale-110 transition-transform"
              :style="{background:c}"
              :title="c"
              @click="setHighlight(c)" />
        </div>
        <input type="color" class="mt-2 w-full h-6 cursor-pointer" @input="(e) => setHighlight((e.target as HTMLInputElement).value)" />
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
    <div class="relative">
      <button type="button" class="tbtn" :class="{ 'rte-active': isActive('link') }" :disabled="disabled" @click="openLinkDialog" title="Insert or edit link">
        <Icon name="link" :size="13" />
      </button>
      <div v-if="showLink" class="absolute z-50 bg-white border border-line p-2 shadow flex gap-2" style="top:100%; left:0; min-width:360px">
        <input
          ref="linkInputRef"
          v-model="linkUrl"
          class="border border-line px-2 text-sm flex-1"
          placeholder="https://..."
          @keydown.enter.prevent="applyLink"
          @keydown.escape.prevent="showLink = false"
        />
        <button class="tbtn" @click="applyLink">OK</button>
        <button class="tbtn" @click="removeLink">Remove</button>
        <button class="tbtn" @click="showLink = false">Cancel</button>
      </div>
    </div>

    <!-- Image -->
    <div class="relative">
      <button type="button" class="tbtn" :disabled="disabled" @click="openImageDialog" title="Insert image">
        <Icon name="image" :size="13" />
      </button>
      <div v-if="showImage" class="absolute z-50 bg-white border border-line p-2 shadow flex gap-2" style="top:100%; left:0; min-width:380px">
        <input
          ref="imageInputRef"
          v-model="imageUrl"
          class="border border-line px-2 text-sm flex-1"
          placeholder="https://example.com/image.png"
          @keydown.enter.prevent="applyImage"
          @keydown.escape.prevent="showImage = false"
        />
        <button class="tbtn" @click="applyImage">Insert</button>
        <button class="tbtn" @click="showImage = false">Cancel</button>
      </div>
    </div>

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

  </div>
</template>