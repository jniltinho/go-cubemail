import axios from 'axios'
import type { Ref } from 'vue'
import type { Calendar, CalendarEvent, CalendarView } from '../../types'
import type { useAuthStore } from '../auth'
import type { useToastStore } from '../toast'

type AuthStore  = ReturnType<typeof useAuthStore>
type ToastStore = ReturnType<typeof useToastStore>

interface CalendarActionsContext {
  auth:            AuthStore
  toast:           ToastStore
  calendars:       Ref<Calendar[]>
  events:          Ref<CalendarEvent[]>
  calView:         Ref<CalendarView>
  calCurrentDate:  Ref<Date>
  selectedEvent:   Ref<CalendarEvent | null>
  editorOpen:      Ref<boolean>
}

export function useCalendarActions({
  auth, toast, calendars, events, calView, calCurrentDate, selectedEvent, editorOpen,
}: CalendarActionsContext) {

  function calRangeForView(date: Date, view: CalendarView): { start: Date; end: Date } {
    const y = date.getFullYear(), m = date.getMonth(), d = date.getDate()
    if (view === 'month') {
      const start = new Date(y, m, 1)
      // include up to 6 weeks: shift back to Sunday of week containing day 1
      start.setDate(start.getDate() - start.getDay())
      const end = new Date(y, m + 1, 1)
      end.setDate(end.getDate() + (6 - end.getDay()) + 1)
      return { start, end }
    }
    if (view === 'week') {
      const start = new Date(y, m, d - date.getDay())
      const end = new Date(start)
      end.setDate(end.getDate() + 7)
      return { start, end }
    }
    // day
    const start = new Date(y, m, d)
    const end   = new Date(y, m, d + 1)
    return { start, end }
  }

  async function fetchCalendars(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/calendar`)
      calendars.value = (res.data.calendars ?? []) as Calendar[]
    } catch {}
  }

  async function fetchEvents(): Promise<void> {
    if (!auth.isApiOnline) return
    const { start, end } = calRangeForView(calCurrentDate.value, calView.value)
    const activeIds = calendars.value.filter(c => c.is_active).map(c => c.id)
    const params: Record<string, string> = {
      start: start.toISOString(),
      end:   end.toISOString(),
    }
    if (activeIds.length > 0) {
      params.calendar_ids = activeIds.join(',')
    }
    try {
      const res = await axios.get(`${API_BASE}/events`, { params })
      events.value = (res.data.events ?? []) as CalendarEvent[]
    } catch {}
  }

  async function createEvent(data: Partial<CalendarEvent>): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/events`, data)
      events.value = [...events.value, res.data as CalendarEvent]
      toast.success('Event created.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to create event.')
      throw e
    }
  }

  async function updateEvent(id: number, data: Partial<CalendarEvent>): Promise<void> {
    try {
      const res = await axios.put(`${API_BASE}/events/${id}`, data)
      const updated = res.data as CalendarEvent
      events.value = events.value.map(e => e.id === id ? updated : e)
      toast.success('Event updated.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to update event.')
      throw e
    }
  }

  async function deleteEvent(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/events/${id}`)
      events.value = events.value.filter(e => e.id !== id)
      if (selectedEvent.value?.id === id) selectedEvent.value = null
      toast.success('Event deleted.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to delete event.')
    }
  }

  async function moveEvent(id: number, start_at: string, end_at: string, calendar_id?: number): Promise<void> {
    try {
      const payload: Record<string, unknown> = { start_at, end_at }
      if (calendar_id !== undefined) payload.calendar_id = calendar_id
      const res = await axios.post(`${API_BASE}/events/${id}/move`, payload)
      const updated = res.data as CalendarEvent
      events.value = events.value.map(e => e.id === id ? updated : e)
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to move event.')
    }
  }

  async function createCalendar(name: string, color: string): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/calendar`, { name, color })
      calendars.value = [...calendars.value, res.data as Calendar]
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to create calendar.')
    }
  }

  async function updateCalendar(id: number, data: Partial<Calendar>): Promise<void> {
    try {
      const res = await axios.put(`${API_BASE}/calendar/${id}`, data)
      const updated = res.data as Calendar
      calendars.value = calendars.value.map(c => c.id === id ? updated : c)
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to update calendar.')
    }
  }

  async function deleteCalendar(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/calendar/${id}`)
      calendars.value = calendars.value.filter(c => c.id !== id)
      await fetchEvents()
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to delete calendar.')
    }
  }

  async function toggleCalendar(id: number, active: boolean): Promise<void> {
    try {
      await axios.post(`${API_BASE}/calendar/activation`, { ids: [id], active })
      calendars.value = calendars.value.map(c => c.id === id ? { ...c, is_active: active } : c)
      await fetchEvents()
    } catch {}
  }

  function openNewEvent(startAt?: Date): void {
    const start = startAt ?? calCurrentDate.value
    const end   = new Date(start.getTime() + 60 * 60 * 1000)
    const defaultCal = calendars.value.find(c => c.is_default) ?? calendars.value[0]
    selectedEvent.value = {
      id: 0,
      calendar_id: defaultCal?.id ?? 0,
      uid: '',
      summary: '',
      start_at: start.toISOString(),
      end_at:   end.toISOString(),
      is_all_day: false,
      status: 'CONFIRMED',
    }
    editorOpen.value = true
  }

  function openEditEvent(event: CalendarEvent): void {
    selectedEvent.value = { ...event }
    editorOpen.value = true
  }

  function closeEditor(): void {
    editorOpen.value = false
    selectedEvent.value = null
  }

  function navigatePrev(): void {
    const d = new Date(calCurrentDate.value)
    if (calView.value === 'month') d.setMonth(d.getMonth() - 1)
    else if (calView.value === 'week') d.setDate(d.getDate() - 7)
    else d.setDate(d.getDate() - 1)
    calCurrentDate.value = d
  }

  function navigateNext(): void {
    const d = new Date(calCurrentDate.value)
    if (calView.value === 'month') d.setMonth(d.getMonth() + 1)
    else if (calView.value === 'week') d.setDate(d.getDate() + 7)
    else d.setDate(d.getDate() + 1)
    calCurrentDate.value = d
  }

  function navigateToday(): void {
    calCurrentDate.value = new Date()
  }

  return {
    fetchCalendars, fetchEvents,
    createEvent, updateEvent, deleteEvent, moveEvent,
    createCalendar, updateCalendar, deleteCalendar, toggleCalendar,
    openNewEvent, openEditEvent, closeEditor,
    navigatePrev, navigateNext, navigateToday,
  }
}
