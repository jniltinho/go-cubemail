import axios from 'axios'
import type { Ref } from 'vue'
import type { Identity, UserSettings } from '../../types'
import type { useAuthStore } from '../auth'
import type { useToastStore } from '../toast'

type AuthStore  = ReturnType<typeof useAuthStore>
type ToastStore = ReturnType<typeof useToastStore>

interface SettingsActionsContext {
  auth:       AuthStore
  toast:      ToastStore
  identities: Ref<Identity[]>
  settings:   Ref<UserSettings | null>
}

const DEFAULT_SETTINGS: UserSettings = {
  rows_per_page: 50,
  timezone: 'UTC',
  compose_html: true,
  date_format: '02/01/2006 15:04',
  signature_pos: 'below',
  oof_enabled: false,
  oof_message: '',
}

export function useSettingsActions({ auth, toast, identities, settings }: SettingsActionsContext) {

  async function fetchSettings(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/settings`)
      settings.value = res.data as UserSettings
    } catch {
      settings.value = { ...DEFAULT_SETTINGS }
    }
  }

  async function saveSettings(data: Partial<UserSettings>): Promise<void> {
    try {
      const res = await axios.put(`${API_BASE}/settings`, data)
      settings.value = res.data as UserSettings
      toast.success('Settings saved.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to save settings.')
    }
  }

  async function fetchIdentities(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/identities`)
      identities.value = (res.data.identities ?? []) as Identity[]
    } catch {}
  }

  async function createIdentity(data: Omit<Identity, 'id'>): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/identities`, data)
      identities.value = [...identities.value, res.data as Identity]
      toast.success('Identity created.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to create identity.')
      throw e
    }
  }

  async function updateIdentity(id: number, data: Omit<Identity, 'id'>): Promise<void> {
    try {
      const res = await axios.put(`${API_BASE}/identities/${id}`, data)
      const updated = res.data as Identity
      identities.value = identities.value.map(i => i.id === id ? updated : i)
      toast.success('Identity updated.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to update identity.')
      throw e
    }
  }

  async function deleteIdentity(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/identities/${id}`)
      identities.value = identities.value.filter(i => i.id !== id)
      toast.success('Identity deleted.')
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to delete identity.')
    }
  }

  async function setDefaultIdentity(id: number): Promise<void> {
    try {
      await axios.post(`${API_BASE}/identities/${id}/default`)
      identities.value = identities.value.map(i => ({ ...i, is_default: i.id === id }))
    } catch {}
  }

  const defaultIdentity = () =>
    identities.value.find(i => i.is_default) ?? identities.value[0] ?? null

  return {
    fetchSettings, saveSettings,
    fetchIdentities, createIdentity, updateIdentity, deleteIdentity, setDefaultIdentity,
    defaultIdentity,
  }
}
