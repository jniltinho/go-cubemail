<script setup lang="ts">
/**
 * @component MailList
 * @description The middle column mail list panel. Renders the list of emails
 * in the active folder, supports single and batch checkbox selections, lists attachments/stars
 * indicators, handles row clicks to open emails, and implements empty state messages.
 */

import { useMailStore } from '../stores/mail'
import { formatDate } from '../utils/helpers'
import Icon from './Icon.vue'

/** Mail store instance containing visible mails and selection state */
const mail = useMailStore()
</script>

<template>
  <div class="bg-white border-r border-line flex flex-col min-h-0">
    <!-- Header -->
    <div class="h-10 px-3 bg-panel-2 border-b border-line flex items-center justify-between flex-shrink-0">
      <h2 class="m-0 text-[13px] text-accent-bar font-bold tracking-tight">
        {{ mail.currentFolderLabel }}
      </h2>
      <div class="inline-flex items-center gap-1.5 text-[12px] text-ink-sub">
        <span class="text-[11.5px]">Sort by</span>
        <select class="bg-white border border-line h-[22px] px-1.5 text-[11.5px] text-ink hover:bg-accent-soft cursor-pointer outline-none">
          <option>Date</option>
          <option>From</option>
          <option>Subject</option>
          <option>Size</option>
        </select>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="mail.visibleMails.length === 0"
      class="flex-1 flex items-center justify-center text-ink-mute text-[12px] px-5 py-10 text-center"
    >
      {{ mail.query ? `No messages matching "${mail.query}"` : 'This folder is empty.' }}
    </div>

    <!-- Message rows -->
    <div v-else class="flex-1 overflow-auto scroll-y">
      <div
        v-for="m in mail.visibleMails"
        :key="m.id"
        :class="['mail-row', { unread: m.unread, selected: m.id === mail.selectedId }]"
        @click="mail.selectMsg(m.id)"
      >
        <!-- Checkbox -->
        <div
          :class="['checkbox', { on: mail.selectedIds.has(m.id) }]"
          @click.stop="mail.toggleSelect(m.id)"
        ></div>

        <!-- From / Subject / Snippet -->
        <div class="min-w-0" style="grid-column:2;grid-row:1/4">
          <div
            class="from text-[12.5px] overflow-hidden whitespace-nowrap text-ellipsis"
            :class="m.unread ? 'text-ink font-semibold' : 'text-ink-sub'"
          >{{ m.from?.name || m.from?.addr || '(unknown)' }}</div>
          <div
            class="subject text-[12.5px] text-ink overflow-hidden whitespace-nowrap text-ellipsis mt-px"
            :class="{ 'font-semibold': m.unread }"
          >{{ m.subject }}</div>
          <div class="snippet text-[11.5px] text-ink-mute overflow-hidden whitespace-nowrap text-ellipsis mt-0.5">
            {{ m.snippet }}
          </div>
        </div>

        <!-- Date -->
        <div
          class="date text-[11px] text-ink-sub tabular-nums pt-0.5 text-right"
          style="grid-column:3;grid-row:1"
        >{{ formatDate(m.rawDate || m.date) }}</div>

        <!-- Flags -->
        <div class="flex flex-col items-end gap-1 mt-0.5" style="grid-column:3;grid-row:2/4">
          <Icon v-if="m.starred" name="star" :size="12" class="text-star" style="fill:currentColor" />
          <Icon v-if="m.attachments?.length" name="paperclip" :size="12" class="text-ink-mute" />
        </div>
      </div>
    </div>
  </div>
</template>
