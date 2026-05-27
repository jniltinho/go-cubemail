<script setup lang="ts">
/**
 * @component TipTapEditor
 * @description Internal TipTap-powered rich text editor.
 * Handles editor lifecycle via the useTipTapEditor composable, two-way v-model
 * synchronization (with loop suppression), source view toggle, and basic
 * image URL insertion. Renders the project-styled toolbar.
 * This component is not intended to be used directly outside RichTextEditor.vue.
 * All documentation and identifiers are in English.
 */

import { ref, watch, computed } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import { useTipTapEditor } from './composables/useTipTapEditor'
import RichTextToolbar from './RichTextToolbar.vue'
import type { TipTapEditorOptions } from './types'

/** Props (controlled by the public RichTextEditor wrapper) */
const props = defineProps<{
  modelValue?: string
  placeholder?: string
  minHeight?: number | string
  maxHeight?: number | string
  disabled?: boolean
  readonly?: boolean
}>()

/** Emits for v-model and optional events */
const emit = defineEmits<{
  'update:modelValue': [value: string]
  focus: []
  blur: []
  update: [value: string]
}>()

/** Source (raw HTML) view toggle */
const showSource = ref(false)

/** Local HTML buffer for source editing (synced on toggle) */
const sourceHtml = ref('')

/** Options passed to the composable */
const editorOptions = computed<TipTapEditorOptions>(() => ({
  content: props.modelValue || '',
  placeholder: props.placeholder,
  onUpdate: (html: string) => {
    if (suppressUpdate.value) return
    emit('update:modelValue', html)
    emit('update', html)
  },
  onFocus: () => emit('focus'),
  onBlur: () => emit('blur'),
  editable: !(props.disabled || props.readonly),
}))

/** The reactive editor + helpers from the composable */
const {
  editor: editorRef,
  setContent: setEditorContent,
  getHTML,
  getWordCount,
} = useTipTapEditor(editorOptions.value)

/** Raw instance for components that expect Editor (not Ref).
 *  Cast as any to bridge ShallowRef<Editor|undefined> vs typed prop expectations.
 */
const editor = computed<any>(() => editorRef.value ?? null)

/** Suppress flag to prevent update loops when parent pushes content */
const suppressUpdate = ref(false)

/** Current content height style (matches legacy 433px editor area) */
const contentStyle = computed(() => ({
  minHeight: typeof props.minHeight === 'number' ? `${props.minHeight}px` : (props.minHeight || '380px'),
  maxHeight: typeof props.maxHeight === 'number' ? `${props.maxHeight}px` : (props.maxHeight || '520px'),
}))

/**
 * Watch modelValue from parent (quoted replies, initial body, etc.).
 * Only push when it differs to avoid unnecessary transactions.
 */
watch(
  () => props.modelValue,
  (newVal) => {
    if (!editor.value) return
    const current = getHTML()
    if (newVal && newVal !== current) {
      suppressUpdate.value = true
      setEditorContent(newVal, false)
      suppressUpdate.value = false
      // If source view is open, keep it in sync
      if (showSource.value) sourceHtml.value = newVal
    }
  },
  { immediate: false }
)

/** Toggle between rich editor and raw HTML source */
function toggleSource() {
  if (!showSource.value) {
    // Entering source: snapshot current HTML
    sourceHtml.value = getHTML()
    showSource.value = true
  } else {
    // Exiting source: push edited HTML back into the TipTap doc
    suppressUpdate.value = true
    setEditorContent(sourceHtml.value, true)
    suppressUpdate.value = false
    showSource.value = false
  }
}

/** Expose the same surface as the legacy TinyEditor for compatibility */
defineExpose({
  getContent: () => getHTML(),
  setContent: (html: string, emitUpdate = true) => setEditorContent(html, emitUpdate),
  focus: () => editor.value?.commands.focus(),
  getWordCount,
  toggleSource,
})
</script>

<template>
  <div class="composer-field composer-rich" style="grid-template-columns: 1fr">
    <!-- Toolbar (always visible when editable) -->
    <RichTextToolbar
      :editor="editor"
      @toggle-source="toggleSource"
    />

    <!-- Rich contenteditable area -->
    <div v-if="!showSource" class="rte-content" :style="contentStyle">
      <EditorContent
        v-if="editor"
        :editor="editor"
        :style="{ minHeight: contentStyle.minHeight }"
      />
      <div v-else class="text-ink-mute p-3 text-sm">Loading editor…</div>
    </div>

    <!-- Raw HTML source view (syncs with rich editor) -->
    <div v-else class="rte-content" :style="contentStyle">
      <textarea
        v-model="sourceHtml"
        class="w-full h-full font-mono text-[12.5px] border-0 outline-none resize-y bg-white"
        style="min-height: inherit; font-family: var(--font-mono)"
        placeholder="Raw HTML..."
      />
      <div class="text-[10px] text-ink-mute pt-1">
        Editing raw HTML. Use with caution — invalid markup may be sanitized on send.
      </div>
    </div>

  </div>
</template>