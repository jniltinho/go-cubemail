<script setup lang="ts">
/**
 * @component AppSidebar
 * @description The collapsible/resizable left-hand folders sidebar.
 * Displays user folders (system & custom), offers the "New Folder" context trigger,
 * and visualizes the active user's storage quota progress bar.
 */

import { computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import FolderRow from './FolderRow.vue'
import Icon from './Icon.vue'

/** Authentication store instance */
const auth = useAuthStore()
/** Mail store instance */
const mail = useMailStore()

/**
 * Formats a raw number of bytes to a human-readable size string (MB or GB).
 * 
 * @param bytes - The raw byte size value.
 * @returns The formatted string representation (e.g. "256 MB", "1.24 GB").
 */
function formatBytes(bytes) {
  if (bytes >= 1024 ** 3) return (bytes / 1024 ** 3).toFixed(2) + ' GB'
  return (bytes / 1024 ** 2).toFixed(0) + ' MB'
}

/** Computed string representation of used bytes */
const quotaUsedLabel  = computed(() => formatBytes(auth.currentUser.quotaUsedBytes))
/** Computed string representation of total bytes */
const quotaTotalLabel = computed(() => formatBytes(auth.currentUser.quotaTotalBytes))
/** Computed percentage of total quota currently consumed */
const quotaPercent    = computed(() => {
  if (!auth.currentUser.quotaTotalBytes) return 0
  return Math.min(auth.currentUser.quotaUsedBytes / auth.currentUser.quotaTotalBytes * 100, 100).toFixed(1)
})
</script>

<template>
  <aside class="bg-white border-r border-line flex flex-col min-h-0">
    <!-- Header -->
    <div class="h-10 px-3 flex items-center justify-between bg-panel-2 border-b border-line flex-shrink-0">
      <span class="text-[11px] uppercase tracking-wider text-ink-sub font-bold">Folders</span>
      <button
        type="button"
        class="bg-white border border-line h-[22px] px-2 text-[11.5px] text-ink hover:bg-accent-soft inline-flex items-center gap-1"
        @click="mail.onFolderMenu('new', null)"
      >
        <Icon name="folder-plus" :size="12" /> New Folder
      </button>
    </div>

    <!-- Folder list -->
    <div class="flex-1 overflow-auto scroll-y py-1.5">
      <FolderRow
        v-for="f in mail.folders"
        :key="f.id"
        :folder="f"
        :active="mail.folder === f.id"
        @click="mail.setFolder(f.id)"
        @menu="(action, fl) => mail.onFolderMenu(action, fl)"
      />
    </div>

    <!-- Quota -->
    <div v-if="auth.currentUser.quotaTotalBytes > 0" class="px-3.5 py-2.5 text-[11px] text-ink-mute border-t border-line-soft flex-shrink-0">
      Quota
      <b class="text-ink-sub">{{ quotaUsedLabel }} / {{ quotaTotalLabel }}</b>
      <div class="h-1.5 bg-line-soft mt-1 border border-line">
        <div class="h-full bg-accent" :style="{ width: quotaPercent + '%' }"></div>
      </div>
    </div>
  </aside>
</template>
