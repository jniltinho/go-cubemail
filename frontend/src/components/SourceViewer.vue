<script setup lang="ts">
import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

const mail = useMailStore()

function copy() { mail.copySource(mail.sourceRaw) }
function backdrop(e) { if (e.target === e.currentTarget) mail.closeSource() }

function download() {
  const blob = new Blob([mail.sourceRaw], { type: 'message/rfc822' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const subject = mail.sourceMail?.subject?.replace(/[^a-z0-9_\-\. ]/gi, '_').trim() || 'message'
  a.href = url
  a.download = `${subject}.eml`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="modal-wrap" @click="backdrop">
    <div class="composer-shell" style="width:820px" role="dialog" aria-label="View Source">
      <!-- Header -->
      <div class="bg-accent-bar text-white py-2 px-3 flex items-center gap-2 text-[13px] font-semibold">
        <Icon name="code-2" :size="14" />
        <span class="truncate">View Source — {{ mail.sourceMail?.subject }}</span>
        <button
          type="button"
          class="ml-auto flex-shrink-0 cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978]"
          @click="mail.closeSource()"
        >
          <Icon name="x" :size="12" />
        </button>
      </div>

      <!-- Raw body -->
      <div class="bg-[#0E1A2E] text-[#D5E0F2] font-mono text-[12px] leading-snug py-3.5 px-4 max-h-[calc(100vh-200px)] overflow-auto whitespace-pre-wrap break-words scroll-y">
        <span v-if="!mail.sourceRaw" class="text-[#6A82A0] italic">Loading…</span>
        <template v-else>{{ mail.sourceRaw }}</template>
      </div>

      <!-- Footer -->
      <div class="py-2 px-2.5 bg-panel-2 border-t border-line flex items-center gap-1.5">
        <span class="text-[12px] text-ink-sub px-1.5">Raw RFC 822 source · read-only</span>
        <div class="ml-auto flex gap-1.5">
          <button class="tbtn" type="button" @click="download" :disabled="!mail.sourceRaw">
            <Icon name="download" :size="13" /> Download .eml
          </button>
          <button class="tbtn" type="button" @click="copy">
            <Icon name="copy" :size="13" /> Copy
          </button>
          <button class="tbtn tbtn-primary" type="button" @click="mail.closeSource()">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>
