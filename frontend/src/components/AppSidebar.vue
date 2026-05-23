<script setup>
import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import FolderRow from './FolderRow.vue'
import Icon from './Icon.vue'

const auth = useAuthStore()
const mail = useMailStore()
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
    <div class="px-3.5 py-2.5 text-[11px] text-ink-mute border-t border-line-soft flex-shrink-0">
      Quota
      <b class="text-ink-sub">{{ auth.currentUser.quotaUsed }} / {{ auth.currentUser.quotaTotal }} GB</b>
      <div class="h-1.5 bg-line-soft mt-1 border border-line">
        <div
          class="h-full bg-accent"
          :style="{ width: (auth.currentUser.quotaUsed / auth.currentUser.quotaTotal * 100).toFixed(1) + '%' }"
        ></div>
      </div>
    </div>
  </aside>
</template>
