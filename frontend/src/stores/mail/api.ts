/**
 * @file api.ts
 * @description Integration module between the mail store and the backend APIs.
 * Handles folder reloading, querying messages, and retrieving complete email bodies from the server.
 */

import axios from 'axios'
import type { Ref } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'
import { formatDate } from '../../utils/helpers'
import { FOLDER_ID_MAP } from './constants'

type AuthStore = ReturnType<typeof useAuthStore>

/**
 * Shared context containing reactive states and dependencies
 * needed for executing mail API network requests.
 */
interface MailApiContext {
  /** Authentication store instance */
  auth: AuthStore
  /** Reactive list reference of folders on the frontend */
  folders: Ref<Folder[]>
  /** Reactive list reference of all loaded email messages in memory */
  mails: Ref<MailMessage[]>
  /** Reactive identifier of the active/selected folder */
  folder: Ref<string>
  /** Reactive UID of the currently selected/open email */
  selectedId: Ref<string | null>
}

/**
 * Composable wrapping HTTP requests for backend IMAP mailbox integration.
 * 
 * @param context - The context containing reactive stores and references.
 * @returns Object containing API communication methods.
 */
export function useMailApi({ auth, folders, mails, folder, selectedId }: MailApiContext) {
  
  /**
   * Fetches messages from a specific folder on the backend IMAP server.
   * If successful, merges the fetched emails into local memory, updates
   * total and unread email counters for the folder, and initiates message body
   * fetches for the first email if the active folder changed.
   * 
   * @param folderId - Folder identifier to fetch (e.g. "inbox").
   * @param onlyIfNew - If true, evaluates if new UIDs exist before updating the state.
   * @returns Promise resolving to `true` if new messages were found/loaded, `false` otherwise.
   */
  async function fetchFolderMessages(folderId: string, onlyIfNew = false): Promise<boolean> {
    if (!auth.isApiOnline) return false
    const folderDef = folders.value.find(f => f.id === folderId)
    const imapName  = folderDef?.name || folderDef?.label
    if (!imapName) return false
    try {
      const mailRes = await axios.get(`${API_BASE}/mail/${encodeURIComponent(imapName)}`)
      const fetched: MailMessage[] = (mailRes.data.messages || []).map((m: Record<string, unknown>) => ({
        id:       String(m.uid),
        folder:   folderId,
        from:     { name: m.from || '', addr: m.from_email || '' },
        to:       String(m.to || ''),
        subject:  m.subject || '(No Subject)',
        rawDate:  m.date || '',
        date:     formatDate(String(m.date || ''), auth.datetimeFormat),
        fullDate: m.date || '',
        snippet:  m.subject || '',
        unread:   !m.seen,
        starred:  !!m.flagged,
        size:     Number(m.size) || 0,
        attachments: [],
        htmlBody: '',
        body: [],
      }))

      if (onlyIfNew) {
        const currentIds = new Set(mails.value.filter(m => m.folder === folderId).map(m => m.id))
        if (!fetched.some(m => !currentIds.has(m.id))) return false
      }

      mails.value = [...mails.value.filter(m => m.folder !== folderId), ...fetched]

      if (folder.value === folderId) {
        if (!fetched.some(m => m.id === selectedId.value))
          selectedId.value = fetched[0]?.id ?? null

        const sel = selectedId.value ? fetched.find(m => m.id === selectedId.value) : null
        if (sel && !sel.htmlBody && !sel.body?.length)
          await fetchMessageBody(sel.id)
      }

      const folderObj = folders.value.find(f => f.id === folderId)
      if (folderObj) {
        const total  = mailRes.data.total ?? fetched.length
        const unread = fetched.filter(m => m.unread).length
        folderObj.count = unread > 0 ? `${unread}/${total}` : String(total)
      }
      return true
    } catch (e) {
      console.error('fetchFolderMessages failed', e)
      return false
    }
  }

  /**
   * Performs the initial data load on connection.
   * Resolves logged in profile parameters, fetches active IMAP folders,
   * maps folder labels to frontend routes, and fetches messages for the active folder.
   */
  async function loadFromApi(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const userRes = await axios.get(`${API_BASE}/auth/me`)
      auth.currentUser.email = userRes.data.username || auth.currentUser.email

      const foldersRes = await axios.get(`${API_BASE}/folders`)
      folders.value = foldersRes.data.map((f: Record<string, unknown>) => {
        const id     = FOLDER_ID_MAP[String(f.Name)] || String(f.Name).toLowerCase().replace(/\s+/g, '-')
        const unread = Number(f.Unseen)   || 0
        const total  = Number(f.Messages) || 0
        return { id, label: f.DisplayName || f.Name, name: f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: !f.IsSystem }
      })

      await fetchFolderMessages(folder.value)
    } catch (e) {
      console.error('API load failed', e)
    }
  }

  /**
   * Fetches the detailed HTML/plain body and attachments of a specific message
   * from the IMAP server and updates the message entry in the local store list.
   * 
   * @param msgId - Unique IMAP UID of the message.
   */
  async function fetchMessageBody(msgId: string): Promise<void> {
    const msg = mails.value.find(m => m.id === msgId)
    if (!msg || !auth.isApiOnline) return
    try {
      const fd    = folders.value.find(f => f.id === msg.folder)
      const label = fd?.name || fd?.label || 'INBOX'
      const res   = await axios.get(`${API_BASE}/mail/${encodeURIComponent(label)}/${msgId}`)
      msg.htmlBody           = res.data.html_body  || ''
      msg.body               = res.data.plain_body ? res.data.plain_body.split('\n') : []
      msg.isCalendarRequest  = res.data.is_calendar_request || false
      msg.calendarInfo       = res.data.calendar_info || null
      msg.attachments = (res.data.attachments || []).map((a: Record<string, unknown>) => ({
        name:         a.filename,
        size:         a.size_label || '',
        ext:          (String(a.filename || '')).split('.').pop()?.toUpperCase() ?? '',
        part:         a.part,
        content_type: a.content_type || '',
      }))
    } catch {}
  }

  return { fetchFolderMessages, loadFromApi, fetchMessageBody }
}
