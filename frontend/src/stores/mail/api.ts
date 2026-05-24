import axios from 'axios'
import type { Ref } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'
import { formatDate } from '../../utils/helpers'
import { FOLDER_ID_MAP } from './mockData'


type AuthStore = ReturnType<typeof useAuthStore>

interface MailApiContext {
  auth: AuthStore
  folders: Ref<Folder[]>
  mails: Ref<MailMessage[]>
  folder: Ref<string>
  selectedId: Ref<string | null>
}

export function useMailApi({ auth, folders, mails, folder, selectedId }: MailApiContext) {
  async function fetchFolderMessages(folderId: string): Promise<void> {
    if (!auth.isApiOnline) return
    const folderDef = folders.value.find(f => f.id === folderId)
    const imapName  = folderDef?.name || folderDef?.label
    if (!imapName) return
    try {
      const mailRes = await axios.get(`${API_BASE}/mail/${encodeURIComponent(imapName)}`)
      const fetched: MailMessage[] = (mailRes.data.messages || []).map((m: Record<string, unknown>) => ({
        id:       String(m.uid),
        folder:   folderId,
        from:     { name: m.from || '', addr: m.from_email || '' },
        to:       auth.currentUser.email,
        subject:  m.subject || '(No Subject)',
        rawDate:  m.date || '',
        date:     formatDate(String(m.date || ''), auth.datetimeFormat),
        fullDate: m.date || '',
        snippet:  m.subject || '',
        unread:   !m.seen,
        starred:  !!m.flagged,
        attachments: [],
        htmlBody: '',
        body: [],
      }))
      mails.value = [...mails.value.filter(m => m.folder !== folderId), ...fetched]
      selectedId.value = fetched[0]?.id ?? null

      const folderObj = folders.value.find(f => f.id === folderId)
      if (folderObj) {
        const total  = mailRes.data.total ?? fetched.length
        const unread = fetched.filter(m => m.unread).length
        folderObj.count = unread > 0 ? `${unread}/${total}` : String(total)
      }
    } catch (e) {
      console.error('fetchFolderMessages failed', e)
    }
  }

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


  async function fetchMessageBody(msgId: string): Promise<void> {
    const msg = mails.value.find(m => m.id === msgId)
    if (!msg || !auth.isApiOnline) return
    try {
      const fd    = folders.value.find(f => f.id === msg.folder)
      const label = fd?.name || fd?.label || 'INBOX'
      const res   = await axios.get(`${API_BASE}/mail/${encodeURIComponent(label)}/${msgId}`)
      msg.htmlBody    = res.data.html_body  || ''
      msg.body        = res.data.plain_body ? res.data.plain_body.split('\n') : []
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
