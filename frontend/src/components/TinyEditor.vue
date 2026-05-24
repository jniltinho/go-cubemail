<script setup lang="ts">
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

const props = defineProps<{
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const taRef = ref<HTMLTextAreaElement | null>(null)
let destroyEditor: (() => void) | null = null

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

onBeforeUnmount(() => destroyEditor?.())

defineExpose({
  getContent: () => tinymce.activeEditor?.getContent() ?? ''
})
</script>

<template>
  <div class="composer-field composer-rich" style="grid-template-columns:1fr">
    <textarea ref="taRef" placeholder="Write your message…"></textarea>
  </div>
</template>
