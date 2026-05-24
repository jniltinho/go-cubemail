import axios from 'axios'
import { nextTick } from 'vue'
import type { Ref, ComputedRef } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'


type AuthStore = ReturnType<typeof useAuthStore>

interface MailActionsContext {
  auth:             AuthStore
  mails:            Ref<MailMessage[]>
  folders:          Ref<Folder[]>
  folder:           Ref<string>
  selectedId:       Ref<string | null>
  selectedIds:      Ref<Set<string>>
  visibleMails:     ComputedRef<MailMessage[]>
  fetchMessageBody: (id: string) => Promise<void>
}

export function useMailActions({
  auth, mails, folders, folder, selectedId, selectedIds, visibleMails, fetchMessageBody,
}: MailActionsContext) {

  function _updateFolderCount(folderId: string): void {
    const obj = folders.value.find(f => f.id === folderId)
    if (!obj) return
    const msgs   = mails.value.filter(m => m.folder === folderId)
    const unread = msgs.filter(m => m.unread).length
    obj.count = unread > 0 ? `${unread}/${msgs.length}` : String(msgs.length)
  }

  function _adjustFolderCount(folderId: string, totalDelta: number, unreadDelta = 0): void {
    const obj = folders.value.find(f => f.id === folderId)
    if (!obj) return
    const parts  = String(obj.count).split('/')
    const total  = Math.max(0, (parts.length > 1 ? parseInt(parts[1]) : parseInt(parts[0])) + totalDelta)
    const unread = Math.max(0, (parts.length > 1 ? parseInt(parts[0]) : 0) + unreadDelta)
    obj.count = unread > 0 ? `${unread}/${total}` : String(total)
  }

  function _resolveIds(): string[] {
    return selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
  }

  function _resolveMailbox(): string {
    const def = folders.value.find(f => f.id === folder.value)
    return def?.name || def?.label || 'INBOX'
  }

  async function selectMsg(id: string): Promise<void> {
    selectedId.value = id
    await nextTick()
    if (auth.isApiOnline) await fetchMessageBody(id)
  }

  function toggleSelect(id: string): void {
    const s = new Set(selectedIds.value)
    s.has(id) ? s.delete(id) : s.add(id)
    selectedIds.value = s
  }

  async function toggleRead(): Promise<void> {
    const ids = _resolveIds()
    if (!ids.length) return

    const targets = ids.map(id => mails.value.find(x => x.id === id)).filter((m): m is MailMessage => !!m)
    if (!targets.length) return

    const nowUnread = !targets.some(m => m.unread)
    targets.forEach(m => { m.unread = nowUnread })
    selectedIds.value = new Set()
    _updateFolderCount(folder.value)

    if (auth.isApiOnline) {
      const mailbox = _resolveMailbox()
      await Promise.all(targets.map(async m => {
        const params = new URLSearchParams()
        params.append('flag', 'seen')
        params.append('value', nowUnread ? '0' : '1')
        await axios.post(
          `${API_BASE}/mail/${encodeURIComponent(mailbox)}/${m.id}/flag`,
          params,
          { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
        ).catch(() => { m.unread = !nowUnread })
      }))
      _updateFolderCount(folder.value)
    }
  }

  function archiveMail(): void {
    const s = mails.value.find(m => m.id === selectedId.value)
    if (!s) return
    const idx       = visibleMails.value.findIndex(m => m.id === s.id)
    const wasUnread = s.unread
    s.folder        = 'archive'
    const next      = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]
    selectedId.value = next?.id ?? null
    _adjustFolderCount(folder.value, -1, wasUnread ? -1 : 0)
    _adjustFolderCount('archive', 1, wasUnread ? 1 : 0)
  }

  async function moveMail(destFolderId: string): Promise<void> {
    const ids = _resolveIds()
    if (!ids.length) return

    const srcName = _resolveMailbox()
    const dstDef  = folders.value.find(f => f.id === destFolderId)
    const dstName = dstDef?.name || dstDef?.label || destFolderId

    const targets     = ids.map(id => mails.value.find(m => m.id === id)).filter((m): m is MailMessage => !!m)
    const unreadCount = targets.filter(m => m.unread).length
    targets.forEach(m => { m.folder = destFolderId })
    selectedId.value  = null
    selectedIds.value = new Set()
    _adjustFolderCount(folder.value, -ids.length, -unreadCount)
    _adjustFolderCount(destFolderId, ids.length, unreadCount)

    if (auth.isApiOnline) {
      await Promise.all(ids.map(id => {
        const fd = new URLSearchParams()
        fd.append('dest', dstName)
        return axios.post(`${API_BASE}/mail/${encodeURIComponent(srcName)}/${id}/move`, fd, {
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        }).catch(() => {})
      }))
    }
  }

  async function deleteMail(): Promise<void> {
    const ids = _resolveIds()
    if (!ids.length) return

    const mailbox = _resolveMailbox()
    const inTrash = folder.value === 'trash'

    const firstIdx  = visibleMails.value.findIndex(m => m.id === ids[0])
    const remaining = visibleMails.value.filter(m => !ids.includes(m.id))
    const next      = remaining[firstIdx] || remaining[firstIdx - 1] || null

    const targets     = ids.map(id => mails.value.find(m => m.id === id)).filter((m): m is MailMessage => !!m)
    const unreadCount = targets.filter(m => m.unread).length

    if (inTrash) {
      mails.value = mails.value.filter(m => !ids.includes(m.id))
    } else {
      targets.forEach(m => { m.folder = 'trash' })
      _adjustFolderCount('trash', ids.length, unreadCount)
    }

    selectedId.value  = next?.id ?? null
    selectedIds.value = new Set()
    _adjustFolderCount(folder.value, -ids.length, -unreadCount)

    if (auth.isApiOnline) {
      await Promise.all([
        ...ids.map(id => axios.delete(`${API_BASE}/mail/${encodeURIComponent(mailbox)}/${id}`).catch(() => {})),
        ...(next ? [fetchMessageBody(next.id)] : []),
      ])
    }
  }

  return { selectMsg, toggleSelect, toggleRead, archiveMail, moveMail, deleteMail }
}
