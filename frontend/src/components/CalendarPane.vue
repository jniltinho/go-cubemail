<script setup lang="ts">
import { computed, watch, onMounted } from 'vue'
import Icon from './Icon.vue'
import { useMailStore } from '../stores/mail'
import type { CalendarEvent } from '../types'

const mail = useMailStore()

const MONTHS = ['January','February','March','April','May','June','July','August','September','October','November','December']
const DOW    = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat']

// ── Title for toolbar ──────────────────────────────────────────────────────
const viewTitle = computed(() => {
  const d = mail.calCurrentDate
  if (mail.calView === 'month') return `${MONTHS[d.getMonth()]} ${d.getFullYear()}`
  if (mail.calView === 'week')  {
    const sun = new Date(d); sun.setDate(d.getDate() - d.getDay())
    const sat = new Date(sun); sat.setDate(sun.getDate() + 6)
    return `${MONTHS[sun.getMonth()]} ${sun.getDate()} – ${sat.getDate()}, ${sat.getFullYear()}`
  }
  return `${MONTHS[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
})

// ── Month grid cells ───────────────────────────────────────────────────────
interface GridCell {
  date:   Date
  dim:    boolean
  today:  boolean
  events: CalendarEvent[]
}

const today = new Date()
today.setHours(0, 0, 0, 0)

const monthCells = computed((): GridCell[] => {
  const d = mail.calCurrentDate
  const y = d.getFullYear(), m = d.getMonth()
  const first = new Date(y, m, 1)
  const startOffset = first.getDay()
  const daysInMonth = new Date(y, m + 1, 0).getDate()
  const daysInPrev  = new Date(y, m, 0).getDate()

  const cells: GridCell[] = []

  for (let i = 0; i < startOffset; i++) {
    const day = daysInPrev - startOffset + 1 + i
    cells.push({ date: new Date(y, m - 1, day), dim: true, today: false, events: [] })
  }
  for (let day = 1; day <= daysInMonth; day++) {
    const date = new Date(y, m, day)
    const t = new Date(date); t.setHours(0,0,0,0)
    cells.push({ date, dim: false, today: t.getTime() === today.getTime(), events: [] })
  }
  let next = 1
  while (cells.length < 42) {
    cells.push({ date: new Date(y, m + 1, next++), dim: true, today: false, events: [] })
  }

  // Place events into cells
  for (const ev of mail.calEvents) {
    const evStart = new Date(ev.start_at)
    evStart.setHours(0,0,0,0)
    for (const cell of cells) {
      const cellDate = new Date(cell.date); cellDate.setHours(0,0,0,0)
      if (cellDate.getTime() === evStart.getTime()) {
        cell.events.push(ev)
        break
      }
    }
  }
  return cells
})

// ── Day/week grid hours for day/week views ─────────────────────────────────
const HOURS = Array.from({ length: 24 }, (_, i) => i)

const weekDays = computed(() => {
  const d = mail.calCurrentDate
  const sun = new Date(d); sun.setDate(d.getDate() - d.getDay()); sun.setHours(0,0,0,0)
  return Array.from({ length: 7 }, (_, i) => {
    const day = new Date(sun); day.setDate(sun.getDate() + i)
    return day
  })
})

function eventsForDayHour(date: Date, hour: number): CalendarEvent[] {
  return mail.calEvents.filter(ev => {
    const s = new Date(ev.start_at)
    return s.getFullYear() === date.getFullYear() &&
           s.getMonth()    === date.getMonth()    &&
           s.getDate()     === date.getDate()     &&
           s.getHours()    === hour
  })
}

function formatHour(h: number): string {
  return h === 0 ? '12 AM' : h < 12 ? `${h} AM` : h === 12 ? '12 PM' : `${h-12} PM`
}

function formatEventTime(iso: string): string {
  const d = new Date(iso)
  const h = d.getHours(), min = d.getMinutes()
  const suffix = h < 12 ? 'AM' : 'PM'
  const hh = h % 12 || 12
  return `${hh}:${String(min).padStart(2,'0')} ${suffix}`
}

function isToday(date: Date): boolean {
  const t = new Date(date); t.setHours(0,0,0,0)
  return t.getTime() === today.getTime()
}

// ── Load data on mount and when switching to calendar view ─────────────────
onMounted(async () => {
  await mail.fetchCalendars()
  await mail.fetchEvents()
})

watch(() => mail.view, async (v) => {
  if (v === 'calendar') {
    await mail.fetchCalendars()
    await mail.fetchEvents()
  }
})
</script>

<template>
  <div class="bg-white overflow-hidden flex flex-col">
    <!-- Toolbar -->
    <div class="h-10 px-4 bg-panel-2 border-b border-line flex items-center gap-1.5 flex-shrink-0">
      <button class="tbtn" type="button" @click="mail.navigatePrev()">
        <Icon name="chevron-left" :size="13" />
      </button>
      <button class="tbtn" type="button" @click="mail.navigateNext()">
        <Icon name="chevron-right" :size="13" />
      </button>
      <button class="tbtn" type="button" @click="mail.navigateToday()">Today</button>
      <span class="text-[15px] font-bold text-accent-bar ml-2.5">{{ viewTitle }}</span>
      <div class="ml-auto flex gap-1.5">
        <button
          :class="['tbtn', mail.calView === 'day' ? 'tbtn-primary' : '']"
          type="button"
          @click="mail.calView = 'day'"
        >Day</button>
        <button
          :class="['tbtn', mail.calView === 'week' ? 'tbtn-primary' : '']"
          type="button"
          @click="mail.calView = 'week'"
        >Week</button>
        <button
          :class="['tbtn', mail.calView === 'month' ? 'tbtn-primary' : '']"
          type="button"
          @click="mail.calView = 'month'"
        >Month</button>
        <button class="tbtn" type="button" @click="mail.openNewEvent()">
          <Icon name="plus" :size="13" /> Event
        </button>
      </div>
    </div>

    <!-- ── Month view ─────────────────────────────────────────────────────── -->
    <template v-if="mail.calView === 'month'">
      <div
        class="flex-1 grid gap-px bg-line border-t border-line overflow-hidden"
        style="grid-template-columns:repeat(7,1fr);grid-template-rows:auto repeat(6,1fr)"
      >
        <!-- Day-of-week headers -->
        <div v-for="d in DOW" :key="d" class="cal-cell head">{{ d }}</div>

        <!-- Calendar cells -->
        <div
          v-for="(cell, i) in monthCells"
          :key="i"
          :class="['cal-cell', { dim: cell.dim, today: cell.today }]"
          @click="mail.openNewEvent(cell.date)"
        >
          <div class="num">{{ cell.date.getDate() }}</div>
          <div
            v-for="(ev, j) in cell.events.slice(0, 3)"
            :key="j"
            class="cal-evt"
            :style="{ backgroundColor: ev.color || '#3788d8' }"
            :title="ev.summary"
            @click.stop="mail.openEditEvent(ev)"
          >{{ ev.summary }}</div>
          <div v-if="cell.events.length > 3" class="cal-evt text-ink-sub" style="background:transparent">
            +{{ cell.events.length - 3 }} more
          </div>
        </div>
      </div>
    </template>

    <!-- ── Week view ──────────────────────────────────────────────────────── -->
    <template v-else-if="mail.calView === 'week'">
      <div class="flex-1 overflow-auto">
        <!-- Header row -->
        <div class="grid sticky top-0 bg-panel-2 border-b border-line z-10" style="grid-template-columns:50px repeat(7,1fr)">
          <div class="border-r border-line" />
          <div
            v-for="day in weekDays"
            :key="day.toISOString()"
            class="text-center py-1.5 text-xs border-r border-line last:border-r-0"
            :class="{ 'font-bold text-accent': isToday(day) }"
          >
            {{ DOW[day.getDay()] }} {{ day.getDate() }}
          </div>
        </div>
        <!-- Hour rows -->
        <div class="grid" style="grid-template-columns:50px repeat(7,1fr)">
          <template v-for="hour in HOURS" :key="hour">
            <div class="text-[10px] text-ink-sub text-right pr-1.5 border-r border-line h-12 flex items-start pt-0.5">
              {{ formatHour(hour) }}
            </div>
            <div
              v-for="day in weekDays"
              :key="day.toISOString() + hour"
              class="border-r border-b border-line last:border-r-0 h-12 relative cursor-pointer hover:bg-accent-soft/30"
              @click="() => { const d = new Date(day); d.setHours(hour); mail.openNewEvent(d) }"
            >
              <div
                v-for="ev in eventsForDayHour(day, hour)"
                :key="ev.id"
                class="absolute inset-x-0.5 top-0.5 rounded text-[10px] text-white px-1 truncate cursor-pointer z-10"
                :style="{ backgroundColor: ev.color || '#3788d8' }"
                :title="ev.summary"
                @click.stop="mail.openEditEvent(ev)"
              >{{ formatEventTime(ev.start_at) }} {{ ev.summary }}</div>
            </div>
          </template>
        </div>
      </div>
    </template>

    <!-- ── Day view ───────────────────────────────────────────────────────── -->
    <template v-else>
      <div class="flex-1 overflow-auto">
        <div class="grid" style="grid-template-columns:50px 1fr">
          <template v-for="hour in HOURS" :key="hour">
            <div class="text-[10px] text-ink-sub text-right pr-1.5 border-r border-line h-16 flex items-start pt-0.5">
              {{ formatHour(hour) }}
            </div>
            <div
              class="border-b border-line h-16 relative cursor-pointer hover:bg-accent-soft/30"
              @click="() => { const d = new Date(mail.calCurrentDate); d.setHours(hour); mail.openNewEvent(d) }"
            >
              <div
                v-for="ev in eventsForDayHour(mail.calCurrentDate, hour)"
                :key="ev.id"
                class="absolute inset-x-1 top-0.5 rounded text-xs text-white px-2 py-0.5 truncate cursor-pointer z-10"
                :style="{ backgroundColor: ev.color || '#3788d8' }"
                @click.stop="mail.openEditEvent(ev)"
              >{{ formatEventTime(ev.start_at) }} – {{ ev.summary }}</div>
            </div>
          </template>
        </div>
      </div>
    </template>
  </div>
</template>
