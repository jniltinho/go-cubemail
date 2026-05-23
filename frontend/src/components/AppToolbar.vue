<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'
import { useAuthStore } from '../stores/auth'
import Icon from './Icon.vue'

const mail = useMailStore()
const auth = useAuthStore()

const moveOpen = ref(false)
const moveBtn  = ref(null)

const moveFolders = computed(() =>
  mail.folders.filter(f => f.id !== mail.folder)
)

function refresh() {
  mail.fetchFolderMessages(mail.folder)
  auth.fetchQuota()
}

function toggleMove() { moveOpen.value = !moveOpen.value }

function pickFolder(folderId) {
  mail.moveMail(folderId)
  moveOpen.value = false
}

function onDocClick(e) {
  if (moveBtn.value && !moveBtn.value.contains(e.target)) moveOpen.value = false
}

onMounted(()  => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))
</script>

<template>
  <div class="h-9 bg-panel-2 border-b border-line flex items-center px-2 gap-1 flex-shrink-0">
    <button class="tbtn tbtn-primary" type="button" @click="mail.compose()">
      <Icon name="pencil-line" :size="14" /> New Message
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>
    <button class="tbtn" type="button" @click="refresh()">
      <Icon name="refresh-cw" :size="14" /> Refresh
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.reply()">
      <Icon name="reply" :size="14" /> Reply
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.reply()">
      <Icon name="reply-all" :size="14" /> Reply All
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.forward()">
      <Icon name="forward" :size="14" /> Forward
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>
    <button class="tbtn tbtn-danger" type="button" :disabled="!mail.selected && !mail.selectedIds.size" @click="mail.deleteMail()">
      <Icon name="trash-2" :size="14" /> Delete
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected && !mail.selectedIds.size" @click="mail.toggleRead()">
      <Icon name="circle-dot" :size="14" /> Mark read/unread
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>

    <!-- Move to dropdown -->
    <div class="relative" ref="moveBtn">
      <button
        class="tbtn"
        type="button"
        :disabled="!mail.selected && !mail.selectedIds.size"
        @click.stop="toggleMove()"
      >
        <Icon name="folder-input" :size="14" /> Move to…
        <Icon name="chevron-down" :size="12" class="ml-0.5" />
      </button>
      <div
        v-if="moveOpen"
        class="absolute left-0 top-full mt-0.5 z-50 min-w-[160px] bg-white border border-line shadow-md py-1"
      >
        <button
          v-for="f in moveFolders"
          :key="f.id"
          class="w-full text-left px-3 py-1.5 text-[12.5px] text-ink hover:bg-accent hover:text-white flex items-center gap-2"
          @click="pickFolder(f.id)"
        >
          <Icon name="folder" :size="13" />
          {{ f.label }}
        </button>
        <div v-if="!moveFolders.length" class="px-3 py-1.5 text-[12px] text-ink-sub">No folders</div>
      </div>
    </div>

    <div class="flex-1"></div>
    <span class="text-[12px] text-ink-sub px-1.5">
      {{ mail.visibleMails.length }} items · {{ mail.counts.inboxUnread }} unread
    </span>
  </div>
</template>
