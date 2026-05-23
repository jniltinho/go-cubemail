<script setup>
import Icon from './Icon.vue'

const props = defineProps({
  mails:       { type: Array,  required: true },
  selectedId:  { type: String, default: null },
  selectedIds: { type: Object, default: () => new Set() }, // Set
  folderLabel: { type: String, default: 'Inbox' },
  query:       { type: String, default: '' },
})

const emit = defineEmits(['select', 'toggle-select'])

function formatDate(raw) {
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

const SORT_OPTS = ['Date', 'From', 'Subject', 'Size']
</script>

<template>
  <div class="bg-white border-r border-line flex flex-col min-h-0">
    <!-- Header -->
    <div class="h-10 px-3 bg-panel-2 border-b border-line flex items-center justify-between text-[12px] text-ink-sub flex-shrink-0">
      <h2 class="m-0 text-[13px] text-accent-bar font-bold tracking-tight">{{ folderLabel }}</h2>
      <div class="inline-flex items-center gap-1.5">
        <span class="text-[11.5px]">Sort by</span>
        <select class="bg-white border border-line h-[22px] px-1.5 text-[11.5px] text-ink hover:bg-accent-soft cursor-pointer outline-none">
          <option v-for="o in SORT_OPTS" :key="o">{{ o }}</option>
        </select>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="mails.length === 0"
         class="flex-1 flex items-center justify-center text-ink-mute text-[12px] px-5 py-10 text-center">
      {{ query ? `No messages matching "${query}"` : 'This folder is empty.' }}
    </div>

    <!-- Message rows -->
    <div v-else class="flex-1 overflow-auto scroll-y">
      <div
        v-for="m in mails"
        :key="m.id"
        :class="['mail-row', { unread: m.unread, selected: m.id === selectedId }]"
        @click="$emit('select', m.id)"
      >
        <!-- Checkbox -->
        <div
          :class="['checkbox', { on: selectedIds.has(m.id) }]"
          @click.stop="$emit('toggle-select', m.id)"
        ></div>

        <!-- From / Subject / Snippet -->
        <div class="min-w-0" style="grid-column:2;grid-row:1/4">
          <div class="from text-[12.5px] overflow-hidden whitespace-nowrap text-ellipsis"
               :class="m.unread ? 'text-ink font-semibold' : 'text-ink-sub'">
            {{ m.from?.name || m.from?.addr || '(unknown)' }}
          </div>
          <div class="subject text-[12.5px] text-ink overflow-hidden whitespace-nowrap text-ellipsis mt-px"
               :class="{ 'font-semibold': m.unread }">
            {{ m.subject }}
          </div>
          <div class="snippet text-[11.5px] text-ink-mute overflow-hidden whitespace-nowrap text-ellipsis mt-0.5">
            {{ m.snippet }}
          </div>
        </div>

        <!-- Date -->
        <div class="date text-[11px] text-ink-sub tabular-nums pt-0.5 text-right"
             style="grid-column:3;grid-row:1">
          {{ formatDate(m.rawDate || m.date) }}
        </div>

        <!-- Flags: star + paperclip -->
        <div class="flex flex-col items-end gap-1 mt-0.5" style="grid-column:3;grid-row:2/4">
          <Icon v-if="m.starred" name="star" :size="12" class="text-star fill-current" />
          <Icon v-if="m.attachments && m.attachments.length" name="paperclip" :size="12" class="text-ink-mute" />
        </div>
      </div>
    </div>
  </div>
</template>
