<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  message: { type: Object, default: null },
})

const emit = defineEmits(['reply', 'forward', 'source', 'archive', 'delete'])

function initials(name) {
  return (name || '').split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0].toUpperCase()).join('')
}

function extIcon(ext) {
  const e = (ext || '').toUpperCase()
  if (e === 'PDF') return 'file-text'
  if (e === 'DOC' || e === 'DOCX') return 'file-text'
  if (e === 'XLS' || e === 'XLSX') return 'file-spreadsheet'
  if (e === 'ZIP' || e === 'RAR' || e === '7Z') return 'file-archive'
  if (e === 'PNG' || e === 'JPG' || e === 'JPEG' || e === 'GIF') return 'file-image'
  return 'file'
}

function extColor(ext) {
  const e = (ext || '').toUpperCase()
  if (e === 'PDF') return 'text-[#B22B2B]'
  if (e === 'DOC' || e === 'DOCX') return 'text-accent'
  if (e === 'XLS' || e === 'XLSX') return 'text-[#1F7A45]'
  if (e === 'ZIP' || e === 'RAR' || e === '7Z') return 'text-[#7A4E1F]'
  return 'text-ink-sub'
}

function formatFullDate(raw) {
  if (!raw) return ''
  let d = new Date(raw)
  if (isNaN(d.getTime())) d = new Date(raw.replace(/^[A-Za-z]{3},\s*/, ''))
  if (isNaN(d.getTime())) d = new Date(raw.replace(' ', 'T'))
  if (isNaN(d.getTime())) return raw
  const dd  = String(d.getDate()).padStart(2, '0')
  const mm  = String(d.getMonth() + 1).padStart(2, '0')
  const hh  = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${dd}/${mm}/${d.getFullYear()} ${hh}:${min}`
}

// Build a safe srcdoc that resets the iframe styles so the email renders
// inside its own box without bleeding into the parent page.
const srcdoc = computed(() => {
  const m = props.message
  if (!m) return ''
  const html = m.htmlBody
  if (!html) return ''
  return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
  html,body{margin:0;padding:12px 16px;font-family:"Segoe UI","Helvetica Neue",Arial,sans-serif;font-size:13.5px;color:#1a1f2a;line-height:1.6;background:#fff;}
  img{max-width:100%;height:auto;}
  a{color:#2A5599;}
  pre{white-space:pre-wrap;word-break:break-all;}
  table{border-collapse:collapse;}
</style>
</head>
<body>${html}</body>
</html>`
})

const hasHtml = computed(() => !!props.message?.htmlBody)
</script>

<template>
  <div class="bg-white flex flex-col min-h-0">
    <!-- Empty state -->
    <div v-if="!message" class="flex-1 grid place-items-center text-ink-mute bg-panel-2">
      <div class="text-center py-6 px-8 border border-dashed border-line bg-white max-w-[320px]">
        <Icon name="mail" :size="40" class="text-[#D2DCEB] mb-2" />
        <div class="font-bold text-ink mb-1.5">No message selected</div>
        <div class="text-[12px]">Pick a message from the list on the left to read it here.</div>
      </div>
    </div>

    <template v-else>
      <!-- Message header -->
      <div class="py-3.5 px-4 border-b border-line bg-white flex-shrink-0">
        <h1 class="m-0 mb-2.5 text-[17px] text-accent-bar font-bold tracking-tight leading-snug">
          {{ message.subject }}
        </h1>
        <div class="flex items-start gap-3">
          <!-- Avatar -->
          <div class="w-9 h-9 bg-accent text-white grid place-items-center font-bold text-[14px] flex-shrink-0 mt-0.5">
            {{ initials(message.from?.name) }}
          </div>
          <!-- From / To -->
          <div class="flex-1 min-w-0 text-[13px]">
            <div class="text-ink leading-snug">
              <b class="text-[#0E1A2E]">{{ message.from?.name }}</b>
              <span class="text-ink-mute text-[12px] ml-1">&lt;{{ message.from?.addr }}&gt;</span>
            </div>
            <div class="text-[12px] text-ink-sub mt-0.5">
              to <b class="text-ink">{{ message.to }}</b>
            </div>
          </div>
          <!-- Date + actions -->
          <div class="flex-shrink-0 text-right">
            <div class="text-[12px] text-ink-sub mb-1.5">{{ formatFullDate(message.rawDate || message.fullDate) }}</div>
            <div class="flex gap-1 justify-end flex-wrap">
              <button class="tbtn" @click="$emit('reply')"><Icon name="reply" :size="13" /> Reply</button>
              <button class="tbtn" @click="$emit('forward')"><Icon name="forward" :size="13" /> Forward</button>
              <button class="tbtn" title="View source" @click="$emit('source')"><Icon name="code-2" :size="13" /></button>
              <button class="tbtn" title="Archive" @click="$emit('archive')"><Icon name="archive" :size="13" /></button>
              <button class="tbtn tbtn-danger" title="Delete" @click="$emit('delete')"><Icon name="trash-2" :size="13" /></button>
            </div>
          </div>
        </div>
      </div>

      <!-- Attachments bar -->
      <div v-if="message.attachments && message.attachments.length"
           class="flex flex-wrap gap-1.5 px-4 py-2 bg-panel-2 border-b border-line-soft flex-shrink-0">
        <span class="text-[10.5px] uppercase text-ink-mute tracking-wider font-semibold self-center mr-1 flex items-center gap-1">
          <Icon name="paperclip" :size="11" />
          {{ message.attachments.length }} attachment{{ message.attachments.length > 1 ? 's' : '' }}
        </span>
        <a v-for="(a, i) in message.attachments" :key="i"
           :href="`/api/v1/mail/${message.folder}/${message.id}/attachment/${a.part ?? i}`"
           target="_blank"
           class="inline-flex items-center gap-1.5 px-2 py-1 bg-white border border-line text-[11.5px] text-ink no-underline hover:border-accent-2 hover:bg-accent-soft"
           :title="(a.size || a.size_label || '') + ' · Download'">
          <Icon :name="extIcon(a.ext || a.content_type)" :size="12" :class="extColor(a.ext)" />
          <span class="font-medium max-w-[180px] overflow-hidden text-ellipsis whitespace-nowrap">{{ a.name || a.filename }}</span>
          <span v-if="a.size || a.size_label" class="text-ink-mute text-[10.5px]">· {{ a.size || a.size_label }}</span>
          <Icon name="download" :size="11" class="text-accent-2 ml-0.5" />
        </a>
      </div>

      <!-- Body: HTML email in sandboxed iframe -->
      <iframe
        v-if="hasHtml"
        :srcdoc="srcdoc"
        sandbox="allow-popups allow-popups-to-escape-sandbox allow-same-origin"
        class="flex-1 w-full border-none"
        title="Email content"
      ></iframe>

      <!-- Body: plain text fallback -->
      <div v-else class="flex-1 overflow-auto scroll-y py-4 px-5 text-[13.5px] leading-relaxed text-ink bg-white">
        <p v-if="!message.body || message.body.length === 0"
           class="text-ink-mute text-[12px]">(empty message)</p>
        <p v-for="(p, i) in message.body" :key="i" class="m-0 mb-3 last:mb-0">{{ p }}</p>
        <div v-if="message.signature" class="mt-5 pt-3 border-t border-dashed border-line text-[12px] text-ink-sub leading-snug">
          <b class="text-ink">{{ message.signature.name }}</b><br />
          {{ message.signature.role }}
        </div>
      </div>
    </template>
  </div>
</template>
