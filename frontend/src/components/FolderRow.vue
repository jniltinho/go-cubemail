<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  folder: { type: Object, required: true },
  active: { type: Boolean, default: false },
})
const emit = defineEmits(['click', 'menu'])

const FOLDER_ICON_MAP = {
  inbox: 'inbox', starred: 'star', sent: 'send', drafts: 'file-edit',
  archive: 'archive', junk: 'shield-alert', trash: 'trash-2',
}

const menu   = ref(false)
const rootEl = ref(null)

function onDocClick(e) {
  if (rootEl.value && !rootEl.value.contains(e.target)) menu.value = false
}
function onEsc(e) { if (e.key === 'Escape') menu.value = false }

watch(menu, v => {
  if (v) {
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onEsc)
  } else {
    document.removeEventListener('mousedown', onDocClick)
    document.removeEventListener('keydown', onEsc)
  }
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClick)
  document.removeEventListener('keydown', onEsc)
})

function act(name) { menu.value = false; emit('menu', name, props.folder) }

const iconName = computed(() => FOLDER_ICON_MAP[props.folder.id] || 'folder')
</script>

<template>
  <div ref="rootEl"
       :class="['side-item', { active, 'menu-open': menu }]"
       @click="$emit('click')">
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
