<script setup lang="ts">
/**
 * @component ReadingPane
 * @description The main email reading viewport. Shows the email header details
 * (subject, sender, recipients, dates), lists and links attachments, and renders
 * the email body inside a sandboxed iframe if HTML exists, falling back to clean
 * plain text with support for automatic signature sections.
 */

import { computed } from 'vue'
import { useMailStore } from '../stores/mail'
import { extIcon, extColor } from '../utils/helpers'
import Icon from './Icon.vue'
import SpinnerIcon from './SpinnerIcon.vue'

/**
 * Parses a calendar date string in DD/MM/YYYY or DD/MM/YYYY HH:MM format
 * (as produced by the Go backend iCalendar parser) into a JS Date.
 * Returns null for empty or unparseable input.
 *
 * @param s - The raw date/time string from calendarInfo.
 * @returns Date instance or null.
 */
function parseCalDate(s: string): Date | null {
  if (!s) return null
  let m = s.match(/^(\d{2})\/(\d{2})\/(\d{4}) (\d{2}):(\d{2})$/)
  if (m) return new Date(+m[3], +m[2] - 1, +m[1], +m[4], +m[5])
  m = s.match(/^(\d{2})\/(\d{2})\/(\d{4})$/)
  if (m) return new Date(+m[3], +m[2] - 1, +m[1])
  return null
}

/**
 * Formats a calendar event start/end pair into a human friendly string
 * e.g. "Monday, May 25, 2026 15:00 to 16:00".
 * Falls back to the raw start string if parsing fails.
 *
 * @param start - Formatted start date/time.
 * @param end - Optional formatted end date/time.
 * @returns Localized display string for the invitation bar.
 */
function formatCalDateTime(start: string, end: string): string {
  const d = parseCalDate(start)
  if (!d) return start
  const dayName = d.toLocaleDateString('en-US', { weekday: 'long' })
  const date    = d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
  const startTime = start.match(/(\d{2}:\d{2})$/)?.[1] ?? ''
  const endTime   = end?.match(/(\d{2}:\d{2})$/)?.[1] ?? ''
  let result = `${dayName}, ${date}`
  if (startTime) {
    result += ` ${startTime}`
    if (endTime) result += ` to ${endTime}`
  }
  return result
}

/** Mail store instance containing selection and action API calls */
const mail = useMailStore()
/** Computed alias of the currently selected email message */
const m    = computed(() => mail.selected)
/** True while the message body is being fetched from the backend */
const bodyLoading = computed(() => mail.bodyLoading)

/**
 * Formats an ISO raw date string to a full localized display layout.
 * 
 * @param raw - The raw date string.
 * @returns The formatted date/time string (e.g. "24/05/2026 19:57").
 */
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

/**
 * Sandboxes the raw HTML body inside a standardized document block for
 * rendering secure and responsive content in the iframe.
 */
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
  <div class="relative bg-white flex flex-col min-h-0">
    <!-- Empty state -->
    <div v-if="!m" class="flex-1 grid place-items-center text-ink-mute bg-panel-2">
      <div class="text-center py-6 px-8 border border-dashed border-line bg-white max-w-[320px]">
        <Icon name="mail" :size="40" class="text-[#D2DCEB] mb-2" />
        <div class="font-bold text-ink mb-1.5">No message selected</div>
        <div class="text-[12px]">Pick a message from the list on the left to read it here.</div>
      </div>
    </div>

    <template v-else>
      <!-- Body loading overlay -->
      <div
        v-if="bodyLoading"
        class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-white/80 gap-3"
      >
        <SpinnerIcon />
        <span class="text-[12px] text-ink-mute tracking-wide">Loading message…</span>
      </div>

      <!-- Message header -->
      <div class="py-3.5 px-4 border-b border-line bg-white flex-shrink-0">
        <h1 class="m-0 mb-2.5 text-[17px] text-accent-bar font-bold tracking-tight leading-snug">
          {{ m.subject }}
        </h1>
        <div class="flex items-start gap-3">
          <!-- Sender avatar -->
          <div class="w-9 h-9 bg-accent text-white grid place-items-center flex-shrink-0 mt-0.5">
            <Icon name="user" :size="34" />
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
              <button class="tbtn tbtn-sm" type="button" @click="mail.reply()">
                <Icon name="reply" :size="12" /> Reply
              </button>
              <button class="tbtn tbtn-sm" type="button" @click="mail.replyAll()">
                <Icon name="reply-all" :size="12" /> Reply All
              </button>
              <button class="tbtn tbtn-sm" type="button" @click="mail.forward()">
                <Icon name="forward" :size="12" /> Forward
              </button>
              <button class="tbtn tbtn-sm" type="button" title="View source" @click="mail.showSource()">
                <Icon name="code-2" :size="12" />
              </button>
              <button class="tbtn tbtn-sm" type="button" title="Archive" @click="mail.archiveMail()">
                <Icon name="archive" :size="12" />
              </button>
              <button class="tbtn tbtn-sm tbtn-danger" type="button" title="Delete" @click="mail.deleteMail()">
                <Icon name="trash-2" :size="12" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Calendar invitation bar -->
      <div
        v-if="m.isCalendarRequest && m.calendarInfo"
        class="flex-shrink-0 border-b border-line bg-[#f4f7fc]"
      >
        <!-- Event summary row -->
        <div class="px-4 pt-3 pb-2 flex flex-wrap gap-x-5 gap-y-0.5 text-[12.5px] text-ink">
          <span v-if="m.calendarInfo.summary" class="font-semibold text-accent">
            {{ m.calendarInfo.summary }}
          </span>
          <span v-if="m.calendarInfo.start_time" class="text-ink-sub">
            {{ m.calendarInfo.start_time }}<template v-if="m.calendarInfo.end_time"> — {{ m.calendarInfo.end_time }}</template>
          </span>
          <span v-if="m.calendarInfo.location" class="text-ink-sub">
            <Icon name="map-pin" :size="11" class="inline mr-0.5" />{{ m.calendarInfo.location }}
          </span>
          <span v-if="m.calendarInfo.organizer" class="text-ink-mute">
            Organizer: {{ m.calendarInfo.organizer }}
          </span>
        </div>
        <!-- Action buttons: RSVP for REQUEST, remove-only for CANCEL -->
        <div class="px-4 pb-3 flex flex-wrap gap-2">
          <template v-if="m.calendarInfo.method !== 'CANCEL'">
            <button class="px-3 py-1 text-[12px] font-semibold bg-[#1B3A6B] text-white hover:bg-[#14305a] transition-colors" type="button" @click="mail.calendarRsvp('ACCEPTED')">ACCEPT</button>
            <button class="px-3 py-1 text-[12px] font-semibold bg-[#e05a2b] text-white hover:bg-[#c44d22] transition-colors" type="button" @click="mail.calendarRsvp('DECLINED')">DECLINE</button>
            <button class="px-3 py-1 text-[12px] font-semibold border border-[#bbb] bg-white text-ink hover:bg-panel-2 transition-colors" type="button" @click="mail.calendarRsvp('TENTATIVE')">TENTATIVE</button>
            <button class="px-3 py-1 text-[12px] font-semibold border border-[#bbb] bg-white text-ink hover:bg-panel-2 transition-colors" type="button" @click="mail.calendarDelegate()">DELEGATE ...</button>
            <button class="px-3 py-1 text-[12px] font-semibold border border-[#bbb] bg-white text-ink hover:bg-panel-2 transition-colors" type="button" @click="mail.calendarAddToCalendar()">ADD TO CALENDAR</button>
          </template>
          <template v-else>
            <button class="px-3 py-1 text-[12px] font-semibold border border-[#bbb] bg-white text-ink hover:bg-panel-2 transition-colors" type="button" @click="mail.calendarAddToCalendar()">REMOVE FROM CALENDAR</button>
          </template>
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

      <!-- Body: cancellation detail card -->
      <div
        v-else-if="m.isCalendarRequest && m.calendarInfo?.method === 'CANCEL' && !m.body?.length"
        class="flex-1 overflow-auto scroll-y py-5 px-5 text-[13px] text-ink"
      >
        <p class="mb-5 text-[13.5px]">Your invitation or the whole event was canceled.</p>

        <div v-if="m.calendarInfo.start_time" class="mb-4">
          <div class="text-[10.5px] uppercase tracking-wider text-ink-mute font-semibold mb-0.5">Time</div>
          <div>{{ formatCalDateTime(m.calendarInfo.start_time, m.calendarInfo.end_time) }}</div>
        </div>

        <div v-if="m.calendarInfo.location" class="mb-4">
          <div class="text-[10.5px] uppercase tracking-wider text-ink-mute font-semibold mb-0.5">Location</div>
          <div>{{ m.calendarInfo.location }}</div>
        </div>

        <div v-if="m.calendarInfo.attendees?.length">
          <div class="text-[10.5px] uppercase tracking-wider text-ink-mute font-semibold mb-1.5">Attendees</div>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="a in m.calendarInfo.attendees"
              :key="a"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[#f3f4f6] border border-[#e5e7eb] text-[11.5px] text-ink"
            >
              <Icon name="user" :size="12" class="text-ink-mute" />{{ a }}
            </span>
          </div>
        </div>
      </div>

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
