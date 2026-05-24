<script setup lang="ts">
/**
 * @component TinyEditor
 * @description The wrapper component for TinyMCE rich text editor integration.
 * Loads themes, plugins, layouts, font maps, and binds model-value updates
 * for seamless HTML mail text editing with proper setup and teardown handlers.
 */

import { ref, onMounted, onBeforeUnmount } from 'vue'
import tinymce from 'tinymce'
import 'tinymce/themes/silver'
import 'tinymce/icons/default'
import 'tinymce/models/dom'
import 'tinymce/plugins/autolink'
import 'tinymce/plugins/lists'
import 'tinymce/plugins/link'
import 'tinymce/plugins/image'
import 'tinymce/plugins/table'
import 'tinymce/plugins/code'
import 'tinymce/plugins/emoticons'
import 'tinymce/plugins/emoticons/js/emojis'
import 'tinymce/plugins/charmap'
import 'tinymce/plugins/searchreplace'
import 'tinymce/plugins/wordcount'

/** Component properties */
const props = defineProps<{
  /** The HTML content model value */
  modelValue?: string
}>()

/** Emitted event triggers */
const emit = defineEmits<{
  /** Triggers two-way binding updates of the rich text model */
  'update:modelValue': [value: string]
}>()

/** Reference element for the raw underlying textarea element */
const taRef = ref<HTMLTextAreaElement | null>(null)
/** Teardown function to cleanly remove the editor instance */
let destroyEditor: (() => void) | null = null

/** Mount hook to initialize TinyMCE with custom toolbar, fonts, and inline styles */
onMounted(() => {
  if (!taRef.value) return

  let editor = null
  let suppress = false

  tinymce.init({
    target: taRef.value,
    base_url: '/tinymce',
    suffix: '.min',
    inline: false,
    menubar: false,
    branding: false,
    promotion: false,
    statusbar: false,
    license_key: 'gpl',
    height: 433,
    placeholder: 'Write your message…',
    plugins: 'autolink lists link image table code emoticons charmap searchreplace wordcount',
    toolbar:
      'fontfamily fontsize | ' +
      'bold italic underline strikethrough | forecolor backcolor | ' +
      'alignleft aligncenter alignright | bullist numlist outdent indent | ' +
      'link image table emoticons | removeformat | code',
    font_family_formats:
      'Arial=arial,helvetica,sans-serif;' +
      'Comic Sans MS=comic sans ms,cursive;' +
      'Courier New=courier new,courier,monospace;' +
      'Georgia=georgia,palatino;' +
      'Segoe UI=segoe ui,helvetica neue,arial,sans-serif;' +
      'Tahoma=tahoma,arial,helvetica,sans-serif;' +
      'Times New Roman=times new roman,times;' +
      'Trebuchet MS=trebuchet ms,geneva;' +
      'Verdana=verdana,geneva;',
    toolbar_mode: 'wrap',
    content_style: [
      'body { font-family: "Segoe UI","Helvetica Neue",Arial,sans-serif;',
      '       font-size: 13.5px; color: #1A1F2A; line-height: 1.6; margin: 10px 12px; }',
      'p { margin: 0 0 10px; }',
    ].join(' '),
    skin: 'oxide',
    content_css: 'default',
    setup(ed) {
      editor = ed
      ed.on('init', () => {
        suppress = true
        ed.setContent(props.modelValue ?? '')
        suppress = false
      })
      ed.on('input change keyup undo redo', () => {
        if (!suppress) emit('update:modelValue', ed.getContent())
      })
    }
  })

  destroyEditor = () => { try { editor?.remove() } catch {} }
})

/** Cleans up the TinyMCE instance when the component is destroyed */
onBeforeUnmount(() => destroyEditor?.())

/** Exposes public component methods */
defineExpose({
  /** Retrieves the active rich editor's HTML code */
  getContent: () => tinymce.activeEditor?.getContent() ?? ''
})
</script>

<template>
  <div class="composer-field composer-rich" style="grid-template-columns:1fr">
    <textarea ref="taRef" placeholder="Write your message…"></textarea>
  </div>
</template>
