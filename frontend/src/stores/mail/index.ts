import { defineStore } from 'pinia'
import { ref, computed, watch, watchEffect, nextTick } from 'vue'
import axios from 'axios'
import { useAuthStore } from '../auth'
import { useDialogStore } from '../dialog'
import { useToastStore } from '../toast'
import { applyAccent, buildCalCells } from '../../utils/helpers'
import { CAL_EVENTS } from './mockData'
import { useMailApi } from './api'
import { useFolderActions } from './folderActions'
import type { MailMessage, Folder, Contact, CalCell } from '../../types'


export const useMailStore = defineStore('mail', () => {
  const auth   = useAuthStore()
  const dialog = useDialogStore()
  const toast  = useToastStore()

  // ── State ──────────────────────────────────────────────────────────────────
  const accent      = ref('#1B3A6B')
  const loading     = ref(false)
  const mails       = ref<MailMessage[]>([])
  const folders     = ref<Folder[]>([])
  const contacts       = ref<Contact[]>([])
  const contactModal   = ref(false)
  const editingContact = ref<Contact | null>(null)
  const calCells    = ref<CalCell[]>(buildCalCells(CAL_EVENTS))
  const calDow      = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

  const view        = ref('mail')
  const folder      = ref('inbox')
  const selectedId  = ref<string | null>(null)
  const selectedIds = ref<Set<string>>(new Set())
  const query       = ref('')
  const composer    = ref<Record<string, unknown> | null>(null)
  const sourceMail  = ref<MailMessage | null>(null)
  const sourceRaw   = ref('')

  watchEffect(() => applyAccent(accent.value))

  // ── Computed ───────────────────────────────────────────────────────────────
  const visibleMails = computed(() => {
    let xs = folder.value === 'starred'
      ? mails.value.filter(m => m.starred)
      : mails.value.filter(m => m.folder === folder.value)
    const q = query.value.trim().toLowerCase()
    if (q) xs = xs.filter(m =>
      m.from?.name?.toLowerCase().includes(q) ||
      m.from?.addr?.toLowerCase().includes(q) ||
      m.subject?.toLowerCase().includes(q) ||
      m.snippet?.toLowerCase().includes(q) ||
      (m.attachments || []).some(a => a.name?.toLowerCase().includes(q))
    )
    return xs
  })

  const counts = computed(() => {
    const inbox = mails.value.filter(m => m.folder === 'inbox')
    return {
      inboxTotal:  inbox.length,
      inboxUnread: inbox.filter(m => m.unread).length,
      starred:     mails.value.filter(m => m.starred).length,
    }
  })

  const selected = computed(() => mails.value.find(m => m.id === selectedId.value) ?? null)

  const currentFolderLabel = computed(() =>
    folders.value.find(f => f.id === folder.value)?.label || 'Inbox'
  )

  // ── Watchers ───────────────────────────────────────────────────────────────
  watch([folder, visibleMails], () => {
    if (!visibleMails.value.find(m => m.id === selectedId.value))
      selectedId.value = visibleMails.value[0]?.id ?? null
  })

  watch(selectedId, () => {
    const s = selected.value
    if (s?.unread) setTimeout(() => {
      const m = mails.value.find(x => x.id === s.id)
      if (m) m.unread = false
    }, 400)
  })

  // ── Composables ────────────────────────────────────────────────────────────
  const { fetchFolderMessages, loadFromApi: _loadFromApi, fetchMessageBody } = useMailApi({
    auth, folders, mails, folder, selectedId,
  })

  async function loadFromApi(): Promise<void> {
    loading.value = true
    try {
      await _loadFromApi()
      await fetchContacts()
    } finally {
      loading.value = false
    }
  }

  const { reloadFolders, setFolder, onFolderMenu } = useFolderActions({
    auth, dialog, folders, mails, folder, view, selectedId, selectedIds, fetchFolderMessages,
  })

  // ── Helpers ────────────────────────────────────────────────────────────────
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

  // ── Mail actions ───────────────────────────────────────────────────────────
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
    const ids = selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
    if (!ids.length) return

    const targets = ids.map(id => mails.value.find(x => x.id === id)).filter((m): m is MailMessage => !!m)
    if (!targets.length) return

    const anyUnread = targets.some(m => m.unread)
    const nowUnread = !anyUnread

    targets.forEach(m => { m.unread = nowUnread })
    selectedIds.value = new Set()
    _updateFolderCount(folder.value)

    if (auth.isApiOnline) {
      const folderDef = folders.value.find(f => f.id === folder.value)
      const mailbox   = folderDef?.name || folderDef?.label || 'INBOX'
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
    const s = selected.value; if (!s) return
    const idx = visibleMails.value.findIndex(m => m.id === s.id)
    const m   = mails.value.find(x => x.id === s.id)
    const wasUnread = m?.unread ?? false
    if (m) m.folder = 'archive'
    const next = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]
    selectedId.value = next?.id ?? null
    _adjustFolderCount(folder.value, -1, wasUnread ? -1 : 0)
    _adjustFolderCount('archive', 1, wasUnread ? 1 : 0)
  }

  async function moveMail(destFolderId: string): Promise<void> {
    const ids = selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
    if (!ids.length) return

    const srcDef  = folders.value.find(f => f.id === folder.value)
    const srcName = srcDef?.name || srcDef?.label || 'INBOX'
    const dstDef  = folders.value.find(f => f.id === destFolderId)
    const dstName = dstDef?.name || dstDef?.label || destFolderId

    const targets = ids.map(id => mails.value.find(m => m.id === id)).filter((m): m is MailMessage => !!m)
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
    const ids = selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
    if (!ids.length) return

    const folderDef    = folders.value.find(f => f.id === folder.value)
    const currentLabel = folderDef?.name || folderDef?.label || 'INBOX'
    const inTrash      = folder.value === 'trash'

    const firstIdx  = visibleMails.value.findIndex(m => m.id === ids[0])
    const remaining = visibleMails.value.filter(m => !ids.includes(m.id))
    const next      = remaining[firstIdx] || remaining[firstIdx - 1] || null

    const targets = ids.map(id => mails.value.find(m => m.id === id)).filter((m): m is MailMessage => !!m)
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
        ...ids.map(id =>
          axios.delete(`${API_BASE}/mail/${encodeURIComponent(currentLabel)}/${id}`).catch(() => {})
        ),
        ...(next ? [fetchMessageBody(next.id)] : []),
      ])
    }
  }

  function reply(): void {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   s.from?.addr,
      subj: 'Re: ' + s.subject,
      quoted: {
        header: `On ${s.fullDate}, ${s.from?.name} &lt;${s.from?.addr}&gt; wrote:`,
        html:   s.htmlBody || null,
        text:   s.body     || [],
      },
    }
  }

  function forward(): void {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   '',
      subj: 'Fwd: ' + s.subject,
      quoted: {
        header: `---------- Forwarded message ----------<br>From: ${s.from?.name} &lt;${s.from?.addr}&gt;<br>Date: ${s.fullDate}<br>Subject: ${s.subject}`,
        html:   s.htmlBody || null,
        text:   s.body     || [],
      },
    }
  }

  function compose():            void { composer.value = {} }
  function closeComposer():      void { composer.value = null }
  function openContactModal():          void { editingContact.value = null; contactModal.value = true }
  function openEditContact(c: Contact): void { editingContact.value = c;    contactModal.value = true }
  function closeContactModal():         void { contactModal.value = false; editingContact.value = null }

  function sortContacts(list: Contact[]): Contact[] {
    return [...list].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  }

  async function fetchContacts(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/contacts`)
      contacts.value = sortContacts(res.data as Contact[])
    } catch {}
  }

  async function saveContact(data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/contacts`, data)
      contacts.value = sortContacts([...contacts.value, res.data as Contact])
    } catch {}
  }

  async function updateContact(id: number, data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res = await axios.put(`${API_BASE}/contacts/${id}`, data)
      const updated = res.data as Contact
      contacts.value = sortContacts(contacts.value.map(c => c.id === id ? updated : c))
    } catch {}
  }

  async function deleteContact(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/contacts/${id}`)
      contacts.value = contacts.value.filter(c => c.id !== id)
    } catch {}
  }

  async function importContacts(file: File): Promise<void> {
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await axios.post(`${API_BASE}/contacts/import`, fd)
      const { imported, total } = res.data
      await fetchContacts()
      if (imported === 0) {
        toast.warning(`No contacts found in file (${total} rows checked).`)
      } else {
        toast.success(`Imported ${imported} contact${imported !== 1 ? 's' : ''} successfully.`)
      }
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to import contacts.')
    }
  }

  async function showSource(): Promise<void> {
    const m = selected.value
    if (!m) return
    sourceMail.value = m
    sourceRaw.value  = ''
    if (auth.isApiOnline) {
      try {
        const fd    = folders.value.find(f => f.id === m.folder)
        const label = fd?.name || fd?.label || 'INBOX'
        const res   = await axios.get(`${API_BASE}/mail/${encodeURIComponent(label)}/${m.id}/raw`, { responseType: 'text' })
        sourceRaw.value = res.data
      } catch {}
    }
  }

  function closeSource(): void { sourceMail.value = null; sourceRaw.value = '' }

  function copySource(rawText: string): void {
    try { navigator.clipboard.writeText(rawText) } catch {}
  }

  return {
    // state
    accent, loading, mails, folders, contacts, contactModal, editingContact, calCells, calDow,
    view, folder, selectedId, selectedIds, query, composer, sourceMail, sourceRaw,
    // computed
    visibleMails, counts, selected, currentFolderLabel,
    // actions
    loadFromApi, fetchFolderMessages, reloadFolders, fetchMessageBody,
    selectMsg, toggleSelect, toggleRead, archiveMail, deleteMail,
    reply, forward, compose, closeComposer, showSource, closeSource, copySource,
    moveMail, setFolder, onFolderMenu,
    openContactModal, openEditContact, closeContactModal, fetchContacts, saveContact, updateContact, deleteContact, importContacts,
  }
})
