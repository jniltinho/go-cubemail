<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import Icon from './Icon.vue'
import { extIcon, extColor } from '../utils/helpers'
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

interface ComposerPrefill {
  to?: string
  subj?: string
  body?: string
  quoted?: {
    header: string
    html?: string | null
    text?: string[]
  }
}

const props = withDefaults(defineProps<{
  prefill?: ComposerPrefill
}>(), { prefill: () => ({}) })
const emit = defineEmits<{ close: [] }>()

const to          = ref(props.prefill?.to   || '')
const cc          = ref('')
const subj        = ref(props.prefill?.subj || '')
const body        = ref(props.prefill?.body || '')
const showCc      = ref(false)
const sent        = ref(false)
const sending     = ref(false)
const sendError   = ref('')
const taRef       = ref(null)
const fileInputRef = ref(null)
const attachments  = ref([])
let destroyEditor  = null

function fmtSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
function fileExt(name) { return name.split('.').pop() }

function openFilePicker() { fileInputRef.value?.click() }

function onFilesSelected(e) {
  for (const f of e.target.files) attachments.value.push(f)
  e.target.value = ''
}

function removeAttachment(i) { attachments.value.splice(i, 1) }

function buildInitHtml() {
  const q = props.prefill?.quoted
  if (q) {
    const esc = s => String(s).replace(/&(?![a-z#]+;)/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    const quotedContent = q.html
      ? q.html
      : (q.text || []).map(l => l.trim() ? `<p>${esc(l)}</p>` : '<p>&nbsp;</p>').join('')
    return (
      '<p><br></p>' +
      `<p style="margin:12px 0 4px;font-size:13px;color:#555;">${q.header}</p>` +
      `<blockquote style="margin:0 0 1em 0;padding:6px 12px;border-left:3px solid #C8D4E8;color:#444;">${quotedContent}</blockquote>`
    )
  }
  const esc = s => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return body.value
    .split('\n')
    .map(l => l.trim() === '' ? '<p>&nbsp;</p>' : `<p>${esc(l)}</p>`)
    .join('')
}

onMounted(() => {
  if (!taRef.value) return

  const html = buildInitHtml()

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
      'bold italic underline strikethrough | forecolor backcolor | ' +
      'alignleft aligncenter alignright | bullist numlist outdent indent | ' +
      'link image table emoticons | removeformat | code',
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
        ed.setContent(html)
        suppress = false
      })
      ed.on('input change keyup undo redo', () => {
        if (!suppress) body.value = ed.getContent()
      })
    }
  })

  destroyEditor = () => { try { editor?.remove() } catch {} }
})

onBeforeUnmount(() => destroyEditor?.())

async function send() {
  sendError.value = ''
  if (!to.value.trim()) { sendError.value = 'Please enter a recipient.'; return }

  // Ensure TinyMCE content is synced to body.value
  const activeEditor = tinymce.activeEditor
  if (activeEditor) body.value = activeEditor.getContent()

  const plainText = body.value.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()

  const fd = new FormData()
  fd.append('to',         to.value.trim())
  fd.append('cc',         cc.value.trim())
  fd.append('subject',    subj.value)
  fd.append('body_html',  body.value)
  fd.append('body_plain', plainText)
  for (const f of attachments.value) fd.append('attachments', f, f.name)

  sending.value = true
  try {
    await axios.post('/api/v1/compose/send', fd)
    sent.value = true
    setTimeout(() => emit('close'), 1800)
  } catch (e) {
    sendError.value = e.response?.data?.error || 'Failed to send. Please try again.'
  } finally {
    sending.value = false
  }
}

function backdrop(e) {
  if (e.target === e.currentTarget) emit('close')
}
</script>

<template>
  <div class="modal-wrap" @click="backdrop">
    <div class="composer-shell" role="dialog" aria-label="New message">
      <!-- Header -->
      <div class="bg-accent-bar text-white py-2 px-3 flex items-center gap-2 text-[13px] font-semibold">
        <Icon name="pencil-line" :size="14" />
        <span>{{ sent ? 'Message sent' : (prefill?.subj ? 'Re: ' + (subj || 'Message') : 'New Message') }}</span>
        <button class="ml-auto cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978] text-[11px]"
                type="button" @click="$emit('close')">
          <Icon name="x" :size="12" />
        </button>
      </div>

      <!-- Sent confirmation -->
      <div v-if="sent" class="py-10 px-7 text-center">
        <Icon name="check-circle-2" :size="38" class="text-success" />
        <div class="text-[14px] text-ink mt-2 font-semibold">Sent</div>
        <div class="text-[12px] text-ink-sub mt-1">A copy has been saved to Sent Items.</div>
      </div>

      <template v-else>
        <div class="composer-field">
          <label>To</label>
          <input v-model="to" placeholder="recipient@example.com" />
        </div>
        <div v-if="showCc" class="composer-field">
          <label>Cc</label>
          <input v-model="cc" />
        </div>
        <div v-else class="py-1 px-2.5 border-b border-line-soft bg-panel-2">
          <a href="#" @click.prevent="showCc = true"
             class="text-[11px] text-accent-2 no-underline hover:underline">+ Add Cc / Bcc</a>
        </div>
        <div class="composer-field">
          <label>Subject</label>
          <input v-model="subj" placeholder="Subject" />
        </div>
        <!-- TinyMCE rich editor -->
        <div class="composer-field composer-rich" style="grid-template-columns:1fr">
          <textarea ref="taRef" placeholder="Write your message…"></textarea>
        </div>

        <!-- Attachments bar -->
        <div v-if="attachments.length" class="flex flex-wrap gap-1.5 px-3 py-2 bg-panel-2 border-t border-line-soft">
          <span class="text-[10.5px] uppercase text-ink-mute tracking-wider font-semibold self-center mr-1 flex items-center gap-1">
            <Icon name="paperclip" :size="11" />
            {{ attachments.length }} attachment{{ attachments.length > 1 ? 's' : '' }}
          </span>
          <div
            v-for="(f, i) in attachments"
            :key="i"
            class="inline-flex items-center gap-1.5 px-2 py-1 bg-white border border-line text-[11.5px] text-ink"
          >
            <Icon :name="extIcon(fileExt(f.name))" :size="12" :class="extColor(fileExt(f.name))" />
            <span class="font-medium max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">{{ f.name }}</span>
            <span class="text-ink-mute text-[10.5px]">· {{ fmtSize(f.size) }}</span>
            <button
              type="button"
              class="ml-0.5 text-ink-mute hover:text-[#B22B2B] cursor-pointer bg-transparent border-0 p-0 leading-none"
              @click="removeAttachment(i)"
            ><Icon name="x" :size="11" /></button>
          </div>
        </div>

        <!-- Hidden file input -->
        <input ref="fileInputRef" type="file" multiple class="hidden" @change="onFilesSelected" />

        <!-- Send error -->
        <div v-if="sendError" class="px-3 py-1.5 bg-[#FFF0F0] border-t border-[#F5C0C0] text-[11.5px] text-[#B22B2B]">
          {{ sendError }}
        </div>

        <!-- Footer actions -->
        <div class="py-2 px-2.5 bg-panel-2 border-t border-line flex items-center gap-1.5">
          <button class="tbtn" type="button" @click="openFilePicker"><Icon name="paperclip" :size="13" /> Attach</button>
          <div class="ml-auto flex gap-1.5">
            <button class="tbtn" type="button" :disabled="sending" @click="$emit('close')">Discard</button>
            <button class="tbtn tbtn-primary" type="button" :disabled="sending" @click="send">
              <Icon :name="sending ? 'loader-2' : 'send'" :size="13" :class="{ 'animate-spin': sending }" />
              {{ sending ? 'Sending…' : 'Send' }}
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
