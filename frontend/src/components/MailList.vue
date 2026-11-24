<script setup lang="ts">
/**
 * @component MailList
 * @description The middle column mail list panel. Renders the list of emails
 * in the active folder, supports single and batch checkbox selections, lists attachments/stars
 * indicators, handles row clicks to open emails, and implements empty state messages.
 */

import { useMailStore } from '../stores/mail'
import { formatDate } from '../utils/helpers'
import type { MailMessage } from '../types'
import Icon from './Icon.vue'

/** Mail store instance containing visible mails and selection state */
const mail = useMailStore()

/**
 * Handles the drag start event for message rows.
 * Serializes the dragged message ID, or the complete set of selected message UIDs
 * if the dragged message is part of the batch selection.
 *
 * @param event - The HTML5 drag event.
 * @param message - The mail message row being dragged.
 */
function onDragStart(event: DragEvent, message: MailMessage) {
  let idsToMove = [message.id]
  if (mail.selectedIds.has(message.id)) {
    idsToMove = Array.from(mail.selectedIds)
  }
  event.dataTransfer?.setData('text/plain', JSON.stringify(idsToMove))
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}
</script>

<template>
  <div class="bg-surface border-r border-line flex flex-col min-h-0">
    <!-- Header -->
    <div class="strip h-10 px-3 bg-surface-2 border-b border-line flex items-center justify-between flex-shrink-0">
      <h2 class="m-0 text-[13px] text-heading font-bold tracking-tight">
        {{ mail.currentFolderLabel }}
      </h2>
      <div class="inline-flex items-center gap-1.5 text-[12px] text-fg-muted">
        <span class="text-[11.5px]">Sort by</span>
        <select
          v-model="mail.sortBy"
          class="rounded-control bg-action-2 border border-action-2-line h-[22px] px-1.5 text-[11.5px] text-on-action-2 hover:bg-action-2-hover cursor-pointer outline-none"
        >
          <option value="date">Date</option>
          <option value="from">From</option>
          <option value="subject">Subject</option>
          <option value="size">Size</option>
        </select>
        <button
          class="h-[22px] w-[22px] flex items-center justify-center rounded-control border border-action-2-line bg-action-2 hover:bg-action-2-hover text-on-action-2 cursor-pointer outline-none"
          :title="mail.sortDir === 'asc' ? 'Ascending' : 'Descending'"
          @click="mail.sortDir = mail.sortDir === 'asc' ? 'desc' : 'asc'"
        >
          <Icon :name="mail.sortDir === 'asc' ? 'arrow-up' : 'arrow-down'" :size="11" />
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="mail.visibleMails.length === 0"
      class="flex-1 flex items-center justify-center text-fg-subtle text-[12px] px-5 py-10 text-center"
    >
      {{ mail.query ? `No messages matching "${mail.query}"` : 'This folder is empty.' }}
    </div>

    <!-- Message rows -->
    <div v-else class="flex-1 overflow-auto scroll-y">
      <div
        v-for="m in mail.visibleMails"
        :key="m.id"
        :class="['mail-row', { unread: m.unread, selected: m.id === mail.selectedId }]"
        draggable="true"
        @dragstart="onDragStart($event, m)"
        @click="mail.selectMsg(m.id)"
      >
        <!-- Checkbox -->
        <div :class="['checkbox', { on: mail.selectedIds.has(m.id) }]" @click.stop="mail.toggleSelect(m.id)"></div>

        <!-- From / Subject / Snippet -->
        <div class="min-w-0" style="grid-column: 2; grid-row: 1/4">
          <div class="from text-[12.5px] overflow-hidden whitespace-nowrap text-ellipsis">
            {{ m.from?.name || m.from?.addr || '(unknown)' }}
          </div>
          <div class="subject text-[12.5px] overflow-hidden whitespace-nowrap text-ellipsis mt-px">
            {{ m.subject }}
          </div>
          <div class="snippet text-[11.5px] overflow-hidden whitespace-nowrap text-ellipsis mt-0.5">
            {{ m.snippet }}
          </div>
        </div>

        <!-- Date -->
        <div class="date text-[11px] tabular-nums pt-0.5 text-right" style="grid-column: 3; grid-row: 1">
          {{ formatDate(m.rawDate || m.date) }}
        </div>

        <!-- Flags -->
        <div class="flex flex-col items-end gap-1 mt-0.5" style="grid-column: 3; grid-row: 2/4">
          <Icon v-if="m.starred" name="star" :size="12" class="star" />
          <Icon v-if="m.attachments?.length" name="paperclip" :size="12" class="flag" />
        </div>
      </div>
    </div>
  </div>
</template>
