<script setup lang="ts">
import { ref, computed } from 'vue'
import axios from 'axios'
import Icon from './Icon.vue'
import TinyEditor from './TinyEditor.vue'
import { extIcon, extColor } from '../utils/helpers'
import { useMailStore } from '../stores/mail'
import { useToastStore } from '../stores/toast'

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

const mailStore   = useMailStore()
const toastStore  = useToastStore()
const to          = ref(props.prefill?.to   || '')
const cc          = ref('')
const subj        = ref(props.prefill?.subj || '')
const showCc      = ref(false)
const sending     = ref(false)
const sendError   = ref('')
const toRef       = ref<HTMLInputElement | null>(null)
const fileInputRef = ref(null)
const attachments  = ref([])

const showSuggestions = ref(false)
const activeSuggestion = ref(-1)

const currentTerm = computed(() => {
  const parts = to.value.split(',')
  return parts[parts.length - 1].trim().toLowerCase()
})

const suggestions = computed(() => {
  const term = currentTerm.value
  if (!term) return []
  return mailStore.contacts
    .filter(c => c.name.toLowerCase().includes(term) || c.email.toLowerCase().includes(term))
    .slice(0, 8)
})

function onToInput() {
  showSuggestions.value = true
  activeSuggestion.value = -1
}

function onToBlur() {
  setTimeout(() => { showSuggestions.value = false }, 150)
}

function onToKeydown(e: KeyboardEvent) {
  if (!showSuggestions.value || !suggestions.value.length) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeSuggestion.value = Math.min(activeSuggestion.value + 1, suggestions.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeSuggestion.value = Math.max(activeSuggestion.value - 1, 0)
  } else if (e.key === 'Enter' && activeSuggestion.value >= 0) {
    e.preventDefault()
    selectSuggestion(suggestions.value[activeSuggestion.value])
  } else if (e.key === 'Escape') {
    showSuggestions.value = false
  }
}

function selectSuggestion(contact: { name: string; email: string }) {
  const parts = to.value.split(',').map(p => p.trim()).filter(Boolean)
  parts.pop()
  parts.push(contact.email)
  to.value = parts.join(', ')
  showSuggestions.value = false
  activeSuggestion.value = -1
  toRef.value?.focus()
}

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
  const rawBody = props.prefill?.body || ''
  const esc = s => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return rawBody
    .split('\n')
    .map(l => l.trim() === '' ? '<p>&nbsp;</p>' : `<p>${esc(l)}</p>`)
    .join('')
}

const body = ref(buildInitHtml())

async function send() {
  sendError.value = ''
  if (!to.value.trim()) { sendError.value = 'Please enter a recipient.'; return }

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
    await axios.post(`${API_BASE}/compose/send`, fd)
    toastStore.success('Message sent successfully.')
    emit('close')
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
        <span>{{ prefill?.subj ? 'Re: ' + (subj || 'Message') : 'New Message' }}</span>
        <button class="ml-auto cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978] text-[11px]"
                type="button" @click="$emit('close')">
          <Icon name="x" :size="12" />
        </button>
      </div>

      <div class="composer-field" style="position:relative">
          <label>To</label>
          <input
            ref="toRef"
            v-model="to"
            placeholder="recipient@example.com"
            autocomplete="off"
            @input="onToInput"
            @blur="onToBlur"
            @keydown="onToKeydown"
          />
          <div
            v-if="showSuggestions && suggestions.length"
            class="absolute left-0 z-50 bg-white border border-line shadow-md py-1 w-full"
            style="top:100%"
          >
            <button
              v-for="(c, i) in suggestions"
              :key="c.email"
              type="button"
              class="w-full text-left px-3 py-1.5 flex items-center gap-2.5 hover:bg-accent hover:text-white"
              :class="i === activeSuggestion ? 'bg-accent text-white' : 'text-ink'"
              @mousedown.prevent="selectSuggestion(c)"
            >
              <span class="w-[26px] h-[26px] bg-accent text-white grid place-items-center font-bold text-[11px] flex-shrink-0 rounded-sm"
                    :class="i === activeSuggestion ? 'bg-white !text-accent' : ''">
                {{ c.name.slice(0,2).toUpperCase() }}
              </span>
              <span class="flex flex-col min-w-0">
                <span class="text-[12.5px] font-semibold truncate">{{ c.name }}</span>
                <span class="text-[11px] opacity-75 truncate">{{ c.email }}</span>
              </span>
            </button>
          </div>
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

        <TinyEditor v-model="body" />

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
            <span class="font-medium max-w-[80px] overflow-hidden text-ellipsis whitespace-nowrap">{{ f.name }}</span>
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
    </div>
  </div>
</template>
