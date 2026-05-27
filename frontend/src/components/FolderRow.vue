<script setup lang="ts">
/**
 * @component FolderRow
 * @description Single folder item display row inside the left side sidebar.
 * Displays counts, stars/unread/system icons, and triggers context menus
 * (kebab option list) for creating subfolders, renaming, deleting, marking all
 * as read, and showing properties.
 */

import { ref, computed, watch, onBeforeUnmount } from 'vue'
import type { Folder } from '../types'
import Icon from './Icon.vue'

/** Component properties */
const props = defineProps<{
  /** Folder model structure */
  folder: Folder
  /** True if this folder is currently active/rendered */
  active?: boolean
}>()

/** Emitted event triggers */
const emit = defineEmits<{
  /** Emitted upon row click selection */
  click: []
  /** Emitted upon context option choice */
  menu: [action: string, folder: Folder]
  /** Emitted when mails are dropped on this folder */
  'drop-mail': [ids: string[], folderId: string]
}>()

/** Maps folder IDs to specific Lucide icons */
const FOLDER_ICON_MAP = {
  inbox: 'inbox', starred: 'star', sent: 'send', drafts: 'file-edit',
  archive: 'archive', junk: 'shield-alert', trash: 'trash-2',
}

/** Visibility status of kebab options menu */
const menu   = ref(false)
/** Reference element for outer component container */
const rootEl = ref<HTMLElement | null>(null)

/** Reactive state tracking whether a mail item is actively dragged over this row */
const isDragOver = ref(false)

/**
 * Handles HTML5 dragover event to allow dropping.
 * 
 * @param e - The drag event.
 */
function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move'
  }
}

/** Sets hover highlight active state when dragging entry pointer overlaps */
function onDragEnter() {
  isDragOver.value = true
}

/** Resets hover status when drag leaves the component bounds */
function onDragLeave() {
  isDragOver.value = false
}

/**
 * Handles the drop event, parsing serialized message IDs and triggering emissions.
 * 
 * @param e - The drop event.
 */
function onDrop(e: DragEvent) {
  isDragOver.value = false
  const data = e.dataTransfer?.getData('text/plain')
  if (!data) return
  try {
    const ids = JSON.parse(data)
    if (Array.isArray(ids)) {
      emit('drop-mail', ids, props.folder.id)
    }
  } catch (err) {
    if (data) {
      emit('drop-mail', [data], props.folder.id)
    }
  }
}

/**
 * Closes the kebab options dropdown if clicking outside the component.
 * 
 * @param e - The document mouse click event.
 */
function onDocClick(e: MouseEvent) {
  if (rootEl.value && !rootEl.value.contains(e.target as Node)) menu.value = false
}

/**
 * Closes the kebab options dropdown when the Escape key is pressed.
 * 
 * @param e - The keyboard event.
 */
function onEsc(e: KeyboardEvent) { if (e.key === 'Escape') menu.value = false }

/** Registers or tears down mouse and keyboard escape listeners depending on dropdown visibility state */
watch(menu, v => {
  if (v) {
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onEsc)
  } else {
    document.removeEventListener('mousedown', onDocClick)
    document.removeEventListener('keydown', onEsc)
  }
})

/** Cleans up document listeners before the component gets unmounted */
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClick)
  document.removeEventListener('keydown', onEsc)
})

/**
 * Closes the dropdown menu and bubble chosen menu options to parent triggers.
 * 
 * @param name - The action item name (ex: 'rename').
 */
function act(name: string) { menu.value = false; emit('menu', name, props.folder) }

/** Computes the target Lucide icon name with a default fallback */
const iconName = computed(() => FOLDER_ICON_MAP[props.folder.id] || 'folder')
</script>

<template>
  <div ref="rootEl"
       :class="['side-item', { active, 'menu-open': menu, 'drag-over': isDragOver }]"
       @click="$emit('click')"
       @dragover="onDragOver"
       @dragenter.prevent="onDragEnter"
       @dragleave="onDragLeave"
       @dragend="onDragLeave"
       @drop="onDrop">
    <Icon :name="iconName" :size="14" class="text-accent-2" />
    <span class="lbl">{{ folder.label }}</span>
    <span class="count">{{ folder.count }}</span>
    <button class="kebab" type="button" title="Folder options" @click.stop="menu = !menu">
      <Icon name="more-vertical" :size="14" />
    </button>

    <div v-if="menu" class="kebab-menu" @click.stop>
      <div class="kmi" @click="act('new')">
        <span class="ic-wrap"><Icon name="plus" :size="13" /></span><span>New Folder…</span>
      </div>
      <div class="kmi" @click="act('subfolder')">
        <span class="ic-wrap"><Icon name="folder-plus" :size="13" /></span><span>New Subfolder…</span>
      </div>
      <div class="kdiv"></div>
      <div :class="['kmi', { disabled: !folder.custom }]"
           @click="folder.custom && act('rename')">
        <span class="ic-wrap"><Icon name="pencil" :size="13" /></span><span>Rename…</span>
      </div>
      <div class="kmi" @click="act('read-all')">
        <span class="ic-wrap"><Icon name="mail-check" :size="13" /></span><span>Mark all as read</span>
      </div>
      <div class="kmi" @click="act('empty')">
        <span class="ic-wrap"><Icon name="eraser" :size="13" /></span><span>Empty folder</span>
      </div>
      <div class="kdiv"></div>
      <div :class="['kmi', { disabled: !folder.custom }]"
           @click="folder.custom && act('delete')">
        <span class="ic-wrap"><Icon name="trash-2" :size="13" /></span><span>Delete folder</span>
      </div>
      <div class="kmi" @click="act('properties')">
        <span class="ic-wrap"><Icon name="settings" :size="13" /></span><span>Properties…</span>
      </div>
    </div>
  </div>
</template>
