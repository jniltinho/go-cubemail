<script setup lang="ts">
/**
 * @component ComposerModal
 * @description The modal viewport for writing and sending email messages.
 * Integrates with the rich text editor (TipTap) to support styled HTML emails,
 * implements dynamic contact autocomplete suggestions as the user types in the "To" input,
 * handles multi-file attachments uploads with size formatting, and displays API errors.
 */

import { ref, computed } from 'vue'
import axios from 'axios'
import Icon from './Icon.vue'
import RichTextEditor from './editor/RichTextEditor.vue'
import { extIcon, extColor } from '../utils/helpers'
import { useMailStore } from '../stores/mail'
import { useToastStore } from '../stores/toast'

/** Data structures to prefill the email composer modal */
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

/** Component properties with standard default values */
const props = withDefaults(defineProps<{
  prefill?: ComposerPrefill
}>(), { prefill: () => ({}) })

/** Custom events emitted by the composer */
const emit = defineEmits<{ close: [] }>()

/** Mail store instance containing autocomplete contact targets */
const mailStore   = useMailStore()
/** Toast notifications store instance */
const toastStore  = useToastStore()

/** Active sender identity (default or first available) */
const activeIdentity = computed(() => mailStore.defaultIdentity())

/** Primary recipient input model string */
const to          = ref(props.prefill?.to   || '')
/** Carbon Copy (Cc) recipient input model string */
const cc          = ref('')
/** Subject line input model string */
const subj        = ref(props.prefill?.subj || '')
/** Visibility status of the CC field row */
const showCc      = ref(false)
/** True if the mail payload is currently uploading */
const sending     = ref(false)
/** Holds backend transmission error details, if any */
const sendError   = ref('')
/** Reference element for the primary To input field */
const toRef       = ref<HTMLInputElement | null>(null)
/** Reference element for the hidden multi-file input picker */
const fileInputRef = ref<HTMLInputElement | null>(null)
/** List of selected files queued to be attached */
const attachments  = ref<File[]>([])

/** Visibility of autocomplete contact suggestions dropdown list */
const showSuggestions = ref(false)
/** Index of the highlighted contact suggestion in the dropdown list */
const activeSuggestion = ref(-1)

/**
 * Extracts and returns the last partially typed email address query segment.
 */
const currentTerm = computed(() => {
  const parts = to.value.split(',')
  return parts[parts.length - 1].trim().toLowerCase()
})

/**
 * Filters the master contacts roster using the parsed search term.
 * Limits the results list to a maximum of 8 suggestions.
 */
const suggestions = computed(() => {
  const term = currentTerm.value
  if (!term) return []
  return mailStore.contacts
    .filter(c => c.name.toLowerCase().includes(term) || c.email.toLowerCase().includes(term))
    .slice(0, 8)
})

/** Shows the autocomplete recommendations and resets active suggestion focus indices */
function onToInput() {
  showSuggestions.value = true
  activeSuggestion.value = -1
}

/** Hides the autocomplete recommendations after a slight delay to allow click events */
function onToBlur() {
  setTimeout(() => { showSuggestions.value = false }, 150)
}

/**
 * Handles ArrowUp/ArrowDown, Enter, and Escape navigation inside autocomplete listings.
 * 
 * @param e - The keydown keyboard event.
 */
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

/**
 * Appends the chosen contact email address to the end of the recipients input field.
 * 
 * @param contact - The highlighted contact item.
 */
function selectSuggestion(contact: { name: string; email: string }) {
  const parts = to.value.split(',').map(p => p.trim()).filter(Boolean)
  parts.pop()
  parts.push(contact.email)
  to.value = parts.join(', ')
  showSuggestions.value = false
  activeSuggestion.value = -1
  toRef.value?.focus()
}

/**
 * Formats file sizes to a short human-readable string (B, KB, MB).
 * 
 * @param bytes - Size in bytes.
 * @returns Formatted size representation.
 */
function fmtSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

/** Extracts the extension segment from a filename */
function fileExt(name) { return name.split('.').pop() }

/** Simulates click on the hidden file input element */
function openFilePicker() { fileInputRef.value?.click() }

/** Appends selected files to the attachments list queue and clears the picker input */
function onFilesSelected(e) {
  for (const f of e.target.files) attachments.value.push(f)
  e.target.value = ''
}

/** Removes an attachment item from the file queue by index */
function removeAttachment(i) { attachments.value.splice(i, 1) }

/**
 * Generates initial rich HTML editor content from composer prefill definitions.
 * Wraps plain text blocks in paragraph tags and blockquotes quoted message sources.
 * 
 * @returns Initial HTML content string.
 */
function buildSignatureHtml(): string {
  const sig = activeIdentity.value?.signature
  if (!sig) return ''
  const sigPos = mailStore.settings?.signature_pos ?? 'below'
  const html = `<p>--&nbsp;</p><p>${sig.replace(/\n/g, '<br>')}</p>`
  return sigPos === 'above' ? html + '<p><br></p>' : '<p><br></p>' + html
}

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
  const bodyHtml = rawBody
    .split('\n')
    .map(l => l.trim() === '' ? '<p>&nbsp;</p>' : `<p>${esc(l)}</p>`)
    .join('')
  return bodyHtml + buildSignatureHtml()
}

/** The bound HTML body editor model content */
const body = ref(buildInitHtml())

/**
 * Transmits the email payload. Asserts that recipients exist, strips HTML tags
 * to generate a plain-text fallback segment, compiles a FormData payload, and posts
 * the content to the backend. Closes the composer upon transmission success.
 */
async function send() {
  sendError.value = ''
  if (!to.value.trim()) { sendError.value = 'Please enter a recipient.'; return }

  const plainText = body.value.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()

  const identity = activeIdentity.value
  const fd = new FormData()
  fd.append('to',         to.value.trim())
  fd.append('cc',         cc.value.trim())
  fd.append('subject',    subj.value)
  fd.append('body_html',  body.value)
  fd.append('body_plain', plainText)
  if (identity) {
    fd.append('from_email', identity.email)
    fd.append('from_name',  identity.display_name || '')
    fd.append('reply_to',   identity.reply_to || '')
  }
  for (const f of attachments.value) fd.append('attachments', f, f.name)

  sending.value = true
  try {
    await axios.post(`${API_BASE}/compose/send`, fd)
    toastStore.success('Message sent successfully.')
    emit('close')
  } catch (e: any) {
    sendError.value = e.response?.data?.error || 'Failed to send. Please try again.'
  } finally {
    sending.value = false
  }
}

/** Closes the modal if user clicks on the outer grey background mask */
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

      <!-- From identity row -->
      <div v-if="mailStore.identities.length > 1" class="composer-field">
        <label>From</label>
        <select
          class="border-0 bg-transparent text-[13px] text-ink focus:outline-none w-full"
          @change="(e) => { const sel = mailStore.identities.find(i => String(i.id) === (e.target as HTMLSelectElement).value); if (sel) mailStore.setDefaultIdentity(sel.id!) }"
          :value="String(activeIdentity?.id ?? '')"
        >
          <option v-for="id in mailStore.identities" :key="id.id" :value="String(id.id)">
            {{ id.display_name ? id.display_name + ' <' + id.email + '>' : id.email }}
          </option>
        </select>
      </div>
      <div v-else-if="activeIdentity" class="composer-field">
        <label>From</label>
        <span class="text-[13px] text-ink-sub">
          {{ activeIdentity.display_name ? activeIdentity.display_name + ' <' + activeIdentity.email + '>' : activeIdentity.email }}
        </span>
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

        <RichTextEditor v-model="body" placeholder="Write your message…" :min-height="433" />

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
