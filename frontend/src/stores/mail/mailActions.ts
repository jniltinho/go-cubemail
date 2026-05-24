/**
 * @file mailActions.ts
 * @description Action modules defining batch or individual updates applicable on email messages.
 * Handles archiving, moving, deleting, reading/unread states, and checking rows.
 */

import axios from 'axios'
import { nextTick } from 'vue'
import type { Ref, ComputedRef } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'

type AuthStore = ReturnType<typeof useAuthStore>

/**
 * Context containing reactive states and dependencies needed
 * for executing mail action operations.
 */
interface MailActionsContext {
  /** Authentication store instance */
  auth:             AuthStore
  /** Reactive list reference of all email messages in memory */
  mails:            Ref<MailMessage[]>
  /** Reactive list reference of folders */
  folders:          Ref<Folder[]>
  /** Active folder identifier */
  folder:           Ref<string>
  /** Active email message UID */
  selectedId:       Ref<string | null>
  /** Set of batch selected email UIDs */
  selectedIds:      Ref<Set<string>>
  /** Filtered list of emails visible under the active folder */
  visibleMails:     ComputedRef<MailMessage[]>
  /** Callback utility to request details of specific message body */
  fetchMessageBody: (id: string) => Promise<void>
}

/**
 * Composable defining batch or individual actions on email messages.
 * 
 * @param context - The context containing reactive stores and references.
 * @returns Actions controller methods.
 */
export function useMailActions({
  auth, mails, folders, folder, selectedId, selectedIds, visibleMails, fetchMessageBody,
}: MailActionsContext) {

  /**
   * Recalculates total and unread email metrics for a given folder in-place
   * from local memory and updates the folder count label.
   * 
   * @param folderId - Frontend target folder ID (e.g. "inbox").
   */
  function _updateFolderCount(folderId: string): void {
    const obj = folders.value.find(f => f.id === folderId)
    if (!obj) return
    const msgs   = mails.value.filter(m => m.folder === folderId)
    const unread = msgs.filter(m => m.unread).length
    obj.count = unread > 0 ? `${unread}/${msgs.length}` : String(msgs.length)
  }

  /**
   * Increments or decrements folder email counters using numerical delta offsets
   * to avoid full in-memory list recalculations.
   * 
   * @param folderId - Target folder ID.
   * @param totalDelta - Amount to add to total email counts.
   * @param unreadDelta - Amount to add to unread email counts.
   */
  function _adjustFolderCount(folderId: string, totalDelta: number, unreadDelta = 0): void {
    const obj = folders.value.find(f => f.id === folderId)
    if (!obj) return
    const parts  = String(obj.count).split('/')
    const total  = Math.max(0, (parts.length > 1 ? parseInt(parts[1]) : parseInt(parts[0])) + totalDelta)
    const unread = Math.max(0, (parts.length > 1 ? parseInt(parts[0]) : 0) + unreadDelta)
    obj.count = unread > 0 ? `${unread}/${total}` : String(total)
  }

  /**
   * Evaluates the target email UIDs eligible for actions. Returns all batch checked
   * elements if any exist, falling back to the single active selected email.
   * 
   * @returns Array of target email UIDs.
   */
  function _resolveIds(): string[] {
    return selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
  }

  /**
   * Resolves the original backend IMAP folder name string (e.g. "INBOX")
   * for the active folder viewed on the client.
   * 
   * @returns The resolved backend mailbox folder name.
   */
  function _resolveMailbox(): string {
    const def = folders.value.find(f => f.id === folder.value)
    return def?.name || def?.label || 'INBOX'
  }

  /**
   * Selects an email message, focusing it on the reading panel, and requests
   * detailed plain/HTML body contents from the server.
   * 
   * @param id - The unique email UID.
   */
  async function selectMsg(id: string): Promise<void> {
    selectedId.value = id
    await nextTick()
    if (auth.isApiOnline) await fetchMessageBody(id)
  }

  /**
   * Alternates batch checkbox selections of a specific email row.
   * 
   * @param id - Target email UID.
   */
  function toggleSelect(id: string): void {
    const s = new Set(selectedIds.value)
    s.has(id) ? s.delete(id) : s.add(id)
    selectedIds.value = s
  }

  /**
   * Toggles seen/unseen read flags on the server for all targeted emails,
   * syncing unread counts and updating local variables.
   */
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

  /**
   * Quick action to move selected/active message into the "Archive" folder,
   * adjusting counts and auto-focusing adjacent list entries.
   */
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

  /**
   * Relocates target email messages (batch or single) into a destination IMAP folder.
   * Updates folder properties locally and posts move triggers to the backend APIs.
   * 
   * @param destFolderId - Frontend destination folder ID key (e.g. "archive").
   */
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

  /**
   * Deletes target email messages (batch or single).
   * If already viewing the Lixeira (trash), deletes permanently from local memory.
   * Otherwise, redirects items to "Lixeira" (trash) and notifies backend endpoints.
   */
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
