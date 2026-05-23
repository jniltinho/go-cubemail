import axios from 'axios'
import type { Ref } from 'vue'
import type { Folder, MailMessage } from '../../types'
import type { useAuthStore } from '../auth'
import type { useDialogStore } from '../dialog'
import { FOLDER_ID_MAP } from './mockData'

const API_BASE = '/api/v1'

type AuthStore   = ReturnType<typeof useAuthStore>
type DialogStore = ReturnType<typeof useDialogStore>

interface FolderActionsContext {
  auth: AuthStore
  dialog: DialogStore
  folders: Ref<Folder[]>
  mails: Ref<MailMessage[]>
  folder: Ref<string>
  view: Ref<string>
  selectedId: Ref<string | null>
  fetchFolderMessages: (folderId: string) => Promise<void>
}

export function useFolderActions({ auth, dialog, folders, mails, folder, view, selectedId, fetchFolderMessages }: FolderActionsContext) {
  async function reloadFolders(): Promise<void> {
    if (!auth.isApiOnline) return
    const res = await axios.get(`${API_BASE}/folders`)
    folders.value = res.data.map((f: Record<string, unknown>) => {
      const id     = FOLDER_ID_MAP[String(f.Name)] || String(f.Name).toLowerCase().replace(/\s+/g, '-')
      const unread = Number(f.Unseen)   || 0
      const total  = Number(f.Messages) || 0
      return { id, label: f.DisplayName || f.Name, name: f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: !f.IsSystem }
    })
  }

  function setFolder(id: string): void {
    folder.value = id
    view.value   = 'mail'
    selectedId.value = null
    fetchFolderMessages(id)
  }

  async function onFolderMenu(action: string, f: Folder | null): Promise<void> {
    if (action === 'new' || action === 'subfolder') {
      const promptLabel = action === 'subfolder' && f ? `New subfolder inside "${f.label}":` : 'New folder name:'
      const name = await dialog.prompt(promptLabel)
      if (!name || typeof name !== 'string' || !name.trim()) return
      if (auth.isApiOnline) {
        const fd = new FormData()
        fd.append('name', name.trim())
        if (action === 'subfolder' && f) fd.append('parent', f.name || f.label)
        try {
          await axios.post(`${API_BASE}/folders`, fd)
          await reloadFolders()
        } catch (e: unknown) {
          const msg = e && typeof e === 'object' && 'response' in e
            ? (e as { response?: { data?: { error?: string } } }).response?.data?.error
            : (e instanceof Error ? e.message : String(e))
          await dialog.alert('Failed to create folder: ' + msg)
        }
      }
      return
    }
    if (!f) return
    if (action === 'rename') {
      const next = await dialog.prompt('Rename folder:', f.label)
      if (!next || typeof next !== 'string' || !next.trim()) return
      if (auth.isApiOnline) {
        const fd = new FormData()
        fd.append('name', f.name || f.label)
        fd.append('newname', next.trim())
        try {
          await axios.post(`${API_BASE}/folders/rename`, fd)
          await reloadFolders()
          if (folder.value === f.id) fetchFolderMessages(folder.value)
        } catch (e: unknown) {
          const msg = e && typeof e === 'object' && 'response' in e
            ? (e as { response?: { data?: { error?: string } } }).response?.data?.error
            : (e instanceof Error ? e.message : String(e))
          await dialog.alert('Failed to rename folder: ' + msg)
        }
      } else {
        const x = folders.value.find(x => x.id === f.id)
        if (x) x.label = next.trim()
      }
    } else if (action === 'read-all') {
      mails.value.forEach(m => { if (m.folder === f.id) m.unread = false })
      const folderObj = folders.value.find(x => x.id === f.id)
      if (folderObj) {
        const total = mails.value.filter(m => m.folder === f.id).length
        folderObj.count = String(total)
      }
    } else if (action === 'empty') {
      const isTrash = f.id === 'trash'
      const confirmMsg = isTrash
        ? `Permanently delete all messages in "${f.label}"? This cannot be undone.`
        : `Move all messages in "${f.label}" to Trash?`
      if (!await dialog.confirm(confirmMsg)) return
      mails.value = mails.value.filter(m => m.folder !== f.id)
      selectedId.value = null
      const folderObj = folders.value.find(x => x.id === f.id)
      if (folderObj) folderObj.count = '0'
      if (auth.isApiOnline) {
        axios.delete(`${API_BASE}/mail/${encodeURIComponent(f.name || f.label)}`).catch(() => {})
      }
    } else if (action === 'delete') {
      if (!await dialog.confirm(`Delete folder "${f.label}"?`)) return
      if (auth.isApiOnline) {
        const fd = new FormData()
        fd.append('name', f.name || f.label)
        try {
          await axios.post(`${API_BASE}/folders/delete`, fd)
          folders.value = folders.value.filter(x => x.id !== f.id)
          mails.value   = mails.value.filter(m => m.folder !== f.id)
          if (folder.value === f.id) setFolder('inbox')
        } catch (e: unknown) {
          const msg = e && typeof e === 'object' && 'response' in e
            ? (e as { response?: { data?: { error?: string } } }).response?.data?.error
            : (e instanceof Error ? e.message : String(e))
          await dialog.alert('Failed to delete folder: ' + msg)
        }
      } else {
        folders.value = folders.value.filter(x => x.id !== f.id)
        if (folder.value === f.id) folder.value = 'inbox'
      }
    } else if (action === 'properties') {
      await dialog.alert(`Folder properties\n\nName: ${f.label}\nType: ${f.custom ? 'User folder' : 'System folder'}`)
    }
  }

  return { reloadFolders, setFolder, onFolderMenu }
}
