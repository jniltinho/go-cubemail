<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()

const newCalName  = ref('')
const newCalColor = ref('#3788d8')
const showNew     = ref(false)

async function addCalendar() {
  const name = newCalName.value.trim()
  if (!name) return
  await mail.createCalendar(name, newCalColor.value)
  newCalName.value = ''
  showNew.value    = false
}
</script>

<template>
  <div class="flex flex-col h-full text-sm select-none">
    <div class="px-3 py-2 font-semibold text-ink flex items-center justify-between">
      <span>My Calendars</span>
      <button class="tbtn p-0.5" title="Add calendar" @click="showNew = !showNew">
        <Icon name="plus" :size="12" />
      </button>
    </div>

    <!-- New calendar form -->
    <div v-if="showNew" class="px-3 pb-2 space-y-1.5">
      <input
        v-model="newCalName"
        class="w-full border border-line rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-accent"
        placeholder="Calendar name"
        @keyup.enter="addCalendar"
        @keyup.escape="showNew = false"
        autofocus
      />
      <div class="flex gap-1.5 items-center">
        <input type="color" v-model="newCalColor" class="h-6 w-8 rounded border border-line cursor-pointer" />
        <button class="tbtn tbtn-primary text-xs py-0.5" @click="addCalendar">Add</button>
        <button class="tbtn text-xs py-0.5" @click="showNew = false">Cancel</button>
      </div>
    </div>

    <!-- Calendar list -->
    <div class="flex-1 overflow-y-auto">
      <div
        v-for="cal in mail.calendars"
        :key="cal.id"
        class="flex items-center gap-2 px-3 py-1.5 hover:bg-accent-soft cursor-pointer group"
      >
        <!-- Color dot / checkbox -->
        <button
          class="flex-shrink-0 w-3.5 h-3.5 rounded border-2 flex items-center justify-center transition-opacity"
          :style="{ borderColor: cal.color, backgroundColor: cal.is_active ? cal.color : 'transparent' }"
          :title="cal.is_active ? 'Hide calendar' : 'Show calendar'"
          @click="mail.toggleCalendar(cal.id, !cal.is_active)"
        >
          <Icon v-if="cal.is_active" name="check" :size="8" class="text-white" />
        </button>

        <span
          class="flex-1 truncate text-ink"
          :class="{ 'opacity-50': !cal.is_active }"
        >{{ cal.name }}</span>

        <!-- Actions shown on hover -->
        <div class="hidden group-hover:flex gap-0.5">
          <button
            class="p-0.5 text-ink-sub hover:text-ink rounded"
            title="Delete calendar"
            @click.stop="!cal.is_default && mail.deleteCalendar(cal.id)"
            :class="{ 'opacity-30 cursor-not-allowed': cal.is_default }"
          >
            <Icon name="trash-2" :size="11" />
          </button>
        </div>
      </div>

      <div v-if="mail.calendars.length === 0" class="px-3 py-2 text-xs text-ink-sub italic">
        No calendars yet.
      </div>
    </div>
  </div>
</template>
