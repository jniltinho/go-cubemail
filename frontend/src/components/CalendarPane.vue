<script setup lang="ts">
/**
 * @component CalendarPane
 * @description The monthly calendar grid view panel. Displays a grid layout 
 * of calendar days (cells), mock events for May 2026, and controls to switch views.
 */

import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

/** Mail store instance containing calendar cell details */
const mail = useMailStore()
</script>

<template>
  <div class="bg-white overflow-hidden flex flex-col" style="grid-column:2/4">
    <!-- Toolbar -->
    <div class="h-10 px-4 bg-panel-2 border-b border-line flex items-center gap-1.5 flex-shrink-0">
      <button class="tbtn" type="button"><Icon name="chevron-left" :size="13" /></button>
      <button class="tbtn" type="button"><Icon name="chevron-right" :size="13" /></button>
      <button class="tbtn" type="button">Today</button>
      <span class="text-[15px] font-bold text-accent-bar ml-2.5">May 2026</span>
      <div class="ml-auto flex gap-1.5">
        <button class="tbtn" type="button">Day</button>
        <button class="tbtn" type="button">Week</button>
        <button class="tbtn tbtn-primary" type="button">Month</button>
        <button class="tbtn" type="button"><Icon name="plus" :size="13" /> Event</button>
      </div>
    </div>

    <!-- Calendar grid -->
    <div
      class="flex-1 grid gap-px bg-line border-t border-line overflow-hidden"
      style="grid-template-columns:repeat(7,1fr);grid-template-rows:auto repeat(6,1fr)"
    >
      <!-- Day-of-week headers -->
      <div v-for="d in mail.calDow" :key="d" class="cal-cell head">{{ d }}</div>

      <!-- Calendar cells -->
      <div
        v-for="(c, i) in mail.calCells"
        :key="i"
        :class="['cal-cell', { dim: c.dim, today: c.today }]"
      >
        <div class="num">{{ c.day }}</div>
        <div
          v-for="(e, j) in c.events"
          :key="j"
          :class="['cal-evt', e.k || '']"
        >{{ e.t }}</div>
      </div>
    </div>
  </div>
</template>
