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
import type { useToastStore } from '../toast'
import { FOLDER_ID_MAP } from './constants'

type AuthStore = ReturnType<typeof useAuthStore>
type ToastStore = ReturnType<typeof useToastStore>

/**
 * Context containing reactive states and dependencies needed
 * for executing mail action operations.
 */
interface MailActionsContext {
  /** Authentication store instance */
  auth:             AuthStore
  /** Toast notifications store instance */
  toast:            ToastStore
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
  /** Callback utility to reload folders from API */
  reloadFolders?:   () => Promise<void>
}

/**
 * Composable defining batch or individual actions on email messages.
 * 
 * @param context - The context containing reactive stores and references.
 * @returns Actions controller methods.
 */
export function useMailActions({
  auth, toast, mails, folders, folder, selectedId, selectedIds, visibleMails, fetchMessageBody, reloadFolders,
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
  async function archiveMail(): Promise<void> {
    const s = mails.value.find(m => m.id === selectedId.value)
    if (!s) return

    // 1. Check if Archive folder exists, if not create it
    let archiveFolderObj = folders.value.find(f => f.id === 'archive')
    if (!archiveFolderObj) {
      if (auth.isApiOnline) {
        try {
          const fd = new FormData()
          fd.append('name', 'Archive')
          await axios.post(`${API_BASE}/folders`, fd)
          if (reloadFolders) {
            await reloadFolders()
          } else {
            const res = await axios.get(`${API_BASE}/folders`)
            folders.value = res.data.map((f: Record<string, unknown>) => {
              const id     = FOLDER_ID_MAP[String(f.Name)] || String(f.Name).toLowerCase().replace(/\s+/g, '-')
              const unread = Number(f.Unseen)   || 0
              const total  = Number(f.Messages) || 0
              return { id, label: f.DisplayName || f.Name, name: f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: !f.IsSystem }
            })
          }
        } catch (e: any) {
          toast.error('Failed to create Archive folder on server: ' + (e.response?.data?.error || e.message))
          return
        }
      } else {
        const newFolderObj = { id: 'archive', label: 'Archive', name: 'Archive', count: '0', custom: false }
        folders.value.push(newFolderObj)
      }
    }

    // 2. Select next email before moving
    const idx = visibleMails.value.findIndex(m => m.id === s.id)
    const next = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]

    // 3. Move the mail using the robust moveMail function
    await moveMail('archive', [s.id])

    // 4. Focus and render next email
    selectedId.value = next?.id ?? null
    if (next && auth.isApiOnline) {
      await fetchMessageBody(next.id)
    }

    // 5. Show success toast in English
    toast.success('This message was successfully archived!')
  }

  /**
   * Resolves the original backend IMAP folder name string (e.g. "INBOX")
   * for a given frontend folder ID.
   * 
   * @param folderId - Frontend folder ID.
   * @returns The resolved backend mailbox folder name.
   */
  function _resolveMailboxForFolder(folderId: string): string {
    const def = folders.value.find(f => f.id === folderId)
    return def?.name || def?.label || 'INBOX'
  }

  /**
   * Relocates target email messages (batch or single) into a destination IMAP folder.
   * Updates folder properties locally and posts move triggers to the backend APIs.
   * 
   * @param destFolderId - Frontend destination folder ID key (e.g. "archive").
   * @param customIds - Optional specific array of message UIDs to move.
   */
  async function moveMail(destFolderId: string, customIds?: string[]): Promise<void> {
    const ids = customIds || _resolveIds()
    if (!ids.length) return

    const dstDef  = folders.value.find(f => f.id === destFolderId)
    const dstName = dstDef?.name || dstDef?.label || destFolderId

    // Filter out messages that are already in the destination folder to prevent redundant operations
    const targets = ids.map(id => mails.value.find(m => m.id === id))
      .filter((m): m is MailMessage => !!m && m.folder !== destFolderId)
    if (!targets.length) return

    // Group targets by their source folder to handle counts and API moves properly
    const sourceFolderGroups: Record<string, MailMessage[]> = {}
    targets.forEach(m => {
      if (!sourceFolderGroups[m.folder]) {
        sourceFolderGroups[m.folder] = []
      }
      sourceFolderGroups[m.folder].push(m)
    })

    // If the currently selected message is in the targets list, deselect it
    if (selectedId.value && targets.some(m => m.id === selectedId.value)) {
      selectedId.value = null
    }

    // Remove moved IDs from selection set
    const updatedSelectedIds = new Set(selectedIds.value)
    targets.forEach(m => updatedSelectedIds.delete(m.id))
    selectedIds.value = updatedSelectedIds

    // Process counts and backend API triggers per source folder group
    for (const [srcFolderId, group] of Object.entries(sourceFolderGroups)) {
      const unreadCount = group.filter(m => m.unread).length
      
      // Move in local store state
      group.forEach(m => { m.folder = destFolderId })
      
      // Adjust counter labels
      _adjustFolderCount(srcFolderId, -group.length, -unreadCount)
      _adjustFolderCount(destFolderId, group.length, unreadCount)

      if (auth.isApiOnline) {
        const srcName = _resolveMailboxForFolder(srcFolderId)
        await Promise.all(group.map(m => {
          const fd = new URLSearchParams()
          fd.append('dest', dstName)
          return axios.post(`${API_BASE}/mail/${encodeURIComponent(srcName)}/${m.id}/move`, fd, {
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          }).catch(() => {})
        }))
      }
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
