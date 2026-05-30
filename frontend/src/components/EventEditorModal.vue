<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Icon from './Icon.vue'
import { useMailStore } from '../stores/mail'
import type { CalendarEvent } from '../types'

const mail = useMailStore()

const form = ref({
  summary:        '',
  description:    '',
  location:       '',
  start_at:       '',
  end_at:         '',
  is_all_day:     false,
  calendar_id:    0,
  rrule:          '',
  organizer_name: '',
  organizer_email:'',
})

const saving = ref(false)

function toLocalInput(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fromLocalInput(val: string): string {
  if (!val) return ''
  return new Date(val).toISOString()
}

watch(() => mail.selectedEvent, (ev) => {
  if (!ev) return
  const defaultCal = mail.calendars.find(c => c.is_default) ?? mail.calendars[0]
  form.value = {
    summary:         ev.summary ?? '',
    description:     ev.description ?? '',
    location:        ev.location ?? '',
    start_at:        toLocalInput(ev.start_at),
    end_at:          toLocalInput(ev.end_at),
    is_all_day:      ev.is_all_day ?? false,
    calendar_id:     ev.calendar_id || defaultCal?.id || 0,
    rrule:           ev.rrule ?? '',
    organizer_name:  ev.organizer_name ?? '',
    organizer_email: ev.organizer_email ?? '',
  }
}, { immediate: true })

const isNew = computed(() => !mail.selectedEvent?.id)
const title = computed(() => isNew.value ? 'New Event' : 'Edit Event')

async function save() {
  if (!form.value.summary.trim()) return
  saving.value = true
  try {
    const payload: Partial<CalendarEvent> = {
      calendar_id:     form.value.calendar_id,
      summary:         form.value.summary.trim(),
      description:     form.value.description,
      location:        form.value.location,
      start_at:        fromLocalInput(form.value.start_at),
      end_at:          fromLocalInput(form.value.end_at),
      is_all_day:      form.value.is_all_day,
      rrule:           form.value.rrule || undefined,
      organizer_name:  form.value.organizer_name || undefined,
      organizer_email: form.value.organizer_email || undefined,
    }
    if (isNew.value) {
      await mail.createEvent(payload)
    } else {
      await mail.updateEvent(mail.selectedEvent!.id, payload)
    }
    await mail.fetchEvents()
    mail.closeEditor()
  } catch {
    // toast shown by action
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!mail.selectedEvent?.id) return
  if (!confirm('Delete this event?')) return
  await mail.deleteEvent(mail.selectedEvent.id)
  mail.closeEditor()
}
</script>

<template>
  <div
    v-if="mail.editorOpen"
    class="fixed inset-0 z-50 flex items-center justify-center"
    @click.self="mail.closeEditor()"
  >
    <div class="absolute inset-0 bg-black/40" />
    <div class="relative bg-white rounded-lg shadow-xl w-full max-w-lg mx-4 z-10 flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-line">
        <span class="font-semibold text-ink">{{ title }}</span>
        <button class="tbtn p-1" @click="mail.closeEditor()">
          <Icon name="x" :size="14" />
        </button>
      </div>

      <!-- Body -->
      <div class="overflow-y-auto px-5 py-4 space-y-3">
        <!-- Title -->
        <div>
          <label class="block text-xs text-ink-sub mb-1">Title *</label>
          <input
            v-model="form.summary"
            class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="Event title"
            autofocus
          />
        </div>

        <!-- Calendar -->
        <div>
          <label class="block text-xs text-ink-sub mb-1">Calendar</label>
          <select
            v-model="form.calendar_id"
            class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
          >
            <option v-for="cal in mail.calendars" :key="cal.id" :value="cal.id">{{ cal.name }}</option>
          </select>
        </div>

        <!-- All day -->
        <label class="flex items-center gap-2 text-sm cursor-pointer">
          <input type="checkbox" v-model="form.is_all_day" class="checkbox" />
          All day
        </label>

        <!-- Start / End -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs text-ink-sub mb-1">Start</label>
            <input
              v-model="form.start_at"
              :type="form.is_all_day ? 'date' : 'datetime-local'"
              class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label class="block text-xs text-ink-sub mb-1">End</label>
            <input
              v-model="form.end_at"
              :type="form.is_all_day ? 'date' : 'datetime-local'"
              class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>

        <!-- Location -->
        <div>
          <label class="block text-xs text-ink-sub mb-1">Location</label>
          <input
            v-model="form.location"
            class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="Optional location"
          />
        </div>

        <!-- Description -->
        <div>
          <label class="block text-xs text-ink-sub mb-1">Description</label>
          <textarea
            v-model="form.description"
            rows="3"
            class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent resize-none"
            placeholder="Optional notes"
          />
        </div>

        <!-- Recurrence -->
        <div>
          <label class="block text-xs text-ink-sub mb-1">Recurrence (RRULE)</label>
          <select
            v-model="form.rrule"
            class="w-full border border-line rounded px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
          >
            <option value="">No recurrence</option>
            <option value="FREQ=DAILY">Daily</option>
            <option value="FREQ=WEEKLY">Weekly</option>
            <option value="FREQ=MONTHLY">Monthly</option>
            <option value="FREQ=YEARLY">Yearly</option>
            <option value="FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR">Weekdays (Mon–Fri)</option>
          </select>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between px-5 py-3 border-t border-line">
        <button
          v-if="!isNew"
          class="tbtn text-red-600 hover:bg-red-50"
          @click="remove"
        >
          <Icon name="trash-2" :size="13" class="mr-1" /> Delete
        </button>
        <div v-else />
        <div class="flex gap-2">
          <button class="tbtn" @click="mail.closeEditor()">Cancel</button>
          <button
            class="tbtn tbtn-primary"
            :disabled="saving || !form.summary.trim()"
            @click="save"
          >
            <Icon v-if="saving" name="loader-circle" :size="13" class="animate-spin mr-1" />
            {{ isNew ? 'Create' : 'Save' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
