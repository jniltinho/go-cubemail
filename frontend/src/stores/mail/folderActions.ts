/**
 * @file folderActions.ts
 * @description Action modules to manage user-created or system IMAP folders.
 * Handles folder creation, recursive subfolders, deletions, renaming, emptying, and properties modal.
 */

import axios from 'axios'
import type { Ref } from 'vue'
import type { Folder, MailMessage } from '../../types'
import type { useAuthStore } from '../auth'
import type { useDialogStore } from '../dialog'
import { FOLDER_ID_MAP } from './constants'

type AuthStore   = ReturnType<typeof useAuthStore>
type DialogStore = ReturnType<typeof useDialogStore>

/**
 * Context containing reactive states and dependencies needed
 * for managing mail folders.
 */
interface FolderActionsContext {
  /** Authentication store instance */
  auth: AuthStore
  /** Dialogue popup control store */
  dialog: DialogStore
  /** Reactive folders list reference on the frontend */
  folders: Ref<Folder[]>
  /** Reactive in-memory emails registry reference */
  mails: Ref<MailMessage[]>
  /** Active folder identifier */
  folder: Ref<string>
  /** Active application layout panel view (e.g. 'mail', 'calendar', 'contacts') */
  view: Ref<string>
  /** Active open email UID reference */
  selectedId: Ref<string | null>
  /** Set of batch selected email UIDs */
  selectedIds: Ref<Set<string>>
  /** Callback utility to request email bodies or folders sync from API */
  fetchFolderMessages: (folderId: string, onlyIfNew?: boolean) => Promise<boolean>
}

/**
 * Composable wrapping IMAP folders control features.
 * 
 * @param context - The context containing reactive stores and references.
 * @returns Folder action controller methods.
 */
export function useFolderActions({ auth, dialog, folders, mails, folder, view, selectedId, selectedIds, fetchFolderMessages }: FolderActionsContext) {
  
  /**
   * Syncs the directory of folder objects from the server,
   * calculating unread and total email metrics for each folder.
   */
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

  /**
   * Sets the active folder viewed on the sidebar navigation tab,
   * resetting existing checkbox batch selections and requesting email updates.
   * 
   * @param id - Frontend key of the target folder (e.g. "inbox").
   */
  function setFolder(id: string): void {
    folder.value     = id
    view.value       = 'mail'
    selectedId.value = null
    selectedIds.value = new Set()
    fetchFolderMessages(id)
  }

  /**
   * Universal controller processor dispatching context actions on folders.
   * Governs adding folders/subfolders, renaming, deleting, marking all messages
   * as read, emptying folders (moving items to trash), and displaying folder metrics.
   * 
   * @param action - Action command name ('new', 'subfolder', 'rename', 'read-all', 'empty', 'delete', 'properties').
   * @param f - Target folder object structure.
   */
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

      // Retrieve all email messages currently assigned to this folder
      const targets = mails.value.filter(m => m.folder === f.id)
      // Extract count of unread email messages to correctly adjust folder badges
      const unreadCount = targets.filter(m => m.unread).length

      if (isTrash) {
        // If emptying Trash itself, permanently purge the messages from memory
        mails.value = mails.value.filter(m => m.folder !== f.id)
      } else {
        // Otherwise, move messages locally to the 'trash' folder key
        targets.forEach(m => { m.folder = 'trash' })
        
        // Find the 'trash' folder object in the sidebar store to update its count badge
        const trashObj = folders.value.find(x => x.id === 'trash')
        if (trashObj) {
          const parts  = String(trashObj.count).split('/')
          const total  = Math.max(0, (parts.length > 1 ? parseInt(parts[1]) : parseInt(parts[0])) + targets.length)
          const unread = Math.max(0, (parts.length > 1 ? parseInt(parts[0]) : 0) + unreadCount)
          // Format count label: e.g. "unread/total" if there are unread emails, otherwise just "total"
          trashObj.count = unread > 0 ? `${unread}/${total}` : String(total)
        }
      }

      // Reset active selection if the focused email is being emptied
      selectedId.value = null
      
      // Reset the emptied folder's count badge to '0'
      const folderObj = folders.value.find(x => x.id === f.id)
      if (folderObj) folderObj.count = '0'
      
      // Perform the asynchronous background network request to empty the folder on the IMAP server
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
