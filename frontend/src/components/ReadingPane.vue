<script setup lang="ts">
import { computed } from 'vue'
import { useMailStore } from '../stores/mail'
import { initials, extIcon, extColor } from '../utils/helpers'
import Icon from './Icon.vue'

const mail = useMailStore()
const m    = computed(() => mail.selected)

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

const srcdoc = computed(() => {
  const html = m.value?.htmlBody
  if (!html) return ''
  return `<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<style>
  html,body{margin:0;padding:12px 16px;font-family:"Segoe UI","Helvetica Neue",Arial,sans-serif;font-size:13.5px;color:#1a1f2a;line-height:1.6;background:#fff}
  img{max-width:100%;height:auto}
  a{color:#2A5599}
  pre{white-space:pre-wrap;word-break:break-all}
  table{border-collapse:collapse}
</style>
</head><body>${html}</body></html>`
})
</script>

<template>
  <div class="bg-white flex flex-col min-h-0">
    <!-- Empty state -->
    <div v-if="!m" class="flex-1 grid place-items-center text-ink-mute bg-panel-2">
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
          {{ m.subject }}
        </h1>
        <div class="flex items-start gap-3">
          <!-- Sender avatar -->
          <div class="w-9 h-9 bg-accent text-white grid place-items-center font-bold text-[14px] flex-shrink-0 mt-0.5">
            {{ initials(m.from?.name) }}
          </div>
          <!-- From / To -->
          <div class="flex-1 min-w-0 text-[13px] leading-snug">
            <div>
              <b class="text-[#0E1A2E]">{{ m.from?.name }}</b>
              <span class="text-ink-mute text-[12px] ml-1">&lt;{{ m.from?.addr }}&gt;</span>
            </div>
            <div class="text-[12px] text-ink-sub mt-0.5">
              to <b class="text-ink">{{ m.to }}</b>
            </div>
          </div>
          <!-- Date + actions -->
          <div class="flex-shrink-0 text-right">
            <div class="text-[12px] text-ink-sub mb-1.5">
              {{ formatFullDate(m.rawDate || m.fullDate) }}
            </div>
            <div class="flex gap-1 justify-end flex-wrap">
              <button class="tbtn" type="button" @click="mail.reply()">
                <Icon name="reply" :size="13" /> Reply
              </button>
              <button class="tbtn" type="button" @click="mail.forward()">
                <Icon name="forward" :size="13" /> Forward
              </button>
              <button class="tbtn" type="button" title="View source" @click="mail.showSource()">
                <Icon name="code-2" :size="13" />
              </button>
              <button class="tbtn" type="button" title="Archive" @click="mail.archiveMail()">
                <Icon name="archive" :size="13" />
              </button>
              <button class="tbtn tbtn-danger" type="button" title="Delete" @click="mail.deleteMail()">
                <Icon name="trash-2" :size="13" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Attachments bar -->
      <div
        v-if="m.attachments?.length"
        class="flex flex-wrap gap-1.5 px-4 py-2 bg-panel-2 border-b border-line-soft flex-shrink-0"
      >
        <span class="text-[10.5px] uppercase text-ink-mute tracking-wider font-semibold self-center mr-1 flex items-center gap-1">
          <Icon name="paperclip" :size="11" />
          {{ m.attachments.length }} attachment{{ m.attachments.length > 1 ? 's' : '' }}
        </span>
        <a
          v-for="(a, i) in m.attachments"
          :key="i"
          :href="`/api/v1/mail/${m.folder}/${m.id}/attachment/${a.part ?? i}`"
          target="_blank"
          class="inline-flex items-center gap-1 px-1.5 py-0.5 bg-white border border-line text-[11px] text-ink no-underline hover:border-accent-2 hover:bg-accent-soft"
        >
          <Icon :name="extIcon(a.ext)" :size="11" :class="extColor(a.ext)" />
          <span class="font-medium max-w-[110px] overflow-hidden text-ellipsis whitespace-nowrap">{{ a.name }}</span>
          <span v-if="a.size" class="text-ink-mute text-[10px]">· {{ a.size }}</span>
          <Icon name="download" :size="10" class="text-accent-2" />
        </a>
      </div>

      <!-- Body: HTML in sandboxed iframe -->
      <iframe
        v-if="m.htmlBody"
        :srcdoc="srcdoc"
        sandbox="allow-popups allow-popups-to-escape-sandbox allow-same-origin"
        class="flex-1 w-full border-none"
        title="Email content"
      ></iframe>

      <!-- Body: plain text fallback -->
      <div v-else class="flex-1 overflow-auto scroll-y py-4 px-5 text-[13.5px] leading-relaxed text-ink">
        <p v-if="!m.body?.length" class="text-ink-mute text-[12px]">(empty message)</p>
        <p v-for="(p, i) in m.body" :key="i" class="m-0 mb-3 last:mb-0">{{ p }}</p>
        <div v-if="m.signature" class="mt-5 pt-3 border-t border-dashed border-line text-[12px] text-ink-sub leading-snug">
          <b class="text-ink">{{ m.signature.name }}</b><br />{{ m.signature.role }}
        </div>
      </div>
    </template>
  </div>
</template>
