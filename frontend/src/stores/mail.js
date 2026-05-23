import { defineStore } from 'pinia'
import { ref, computed, watch, watchEffect, nextTick } from 'vue'
import axios from 'axios'
import { useAuthStore } from './auth'
import { useDialogStore } from './dialog'
import { formatDate, applyAccent, buildCalCells } from '../utils/helpers'

const API_BASE = '/api/v1'

const FOLDER_ID_MAP = {
  'Inbox': 'inbox', 'Starred': 'starred', 'Sent Items': 'sent', 'Sent': 'sent',
  'Drafts': 'drafts', 'Archive': 'archive', 'Junk Mail': 'junk', 'Junk': 'junk',
  'Deleted Items': 'trash', 'Trash': 'trash', 'INBOX': 'inbox',
}

const MOCK_FOLDERS = [
  { id: 'inbox',   label: 'Inbox',        count: '5/12' },
  { id: 'starred', label: 'Starred',       count: '2' },
  { id: 'sent',    label: 'Sent Items',    count: '38' },
  { id: 'drafts',  label: 'Drafts',        count: '2' },
  { id: 'archive', label: 'Archive',       count: '214' },
  { id: 'junk',    label: 'Junk Mail',     count: '7' },
  { id: 'trash',   label: 'Deleted Items', count: '19' },
]

const MOCK_MAIL = [
  {
    id: 'm1', folder: 'inbox',
    from: { name: 'Helena Vargas', addr: 'h.vargas@northbridge-co.com' }, to: 'me@go-webmail.test',
    subject: 'Q3 budget review — please confirm Thursday slot',
    rawDate: '2026-05-21T10:42:00', date: '21/05/2026 10:42', fullDate: '2026-05-21T10:42:00',
    snippet: 'Hi — sharing the latest spreadsheet ahead of Thursday. Two line items still need your sign-off…',
    unread: true, starred: true,
    attachments: [{ name: 'Q3-budget-v4.xlsx', size: '184 KB', ext: 'XLS' }],
    body: ['Hi,', "Sharing the latest spreadsheet ahead of Thursday's review.", 'Talk soon,'],
    signature: { name: 'Helena Vargas', role: 'Finance Operations, Northbridge & Co.' },
  },
  {
    id: 'm2', folder: 'inbox',
    from: { name: 'IT Helpdesk', addr: 'helpdesk@go-webmail.test' }, to: 'all-staff@go-webmail.test',
    subject: 'Scheduled maintenance window — Saturday 02:00–04:00 UTC',
    rawDate: '2026-05-21T09:18:00', date: '21/05/2026 09:18', fullDate: '2026-05-21T09:18:00',
    snippet: 'Mail and calendar services will be briefly unavailable during the maintenance window…',
    unread: true, starred: false, attachments: [],
    body: ['Hello,', 'We will be performing scheduled maintenance on the mail and calendar tier this Saturday between 02:00 and 04:00 UTC.'],
    signature: { name: 'Infrastructure Team', role: 'IT Operations' },
  },
  {
    id: 'm3', folder: 'inbox',
    from: { name: 'Marcus Ahn', addr: 'marcus@strataworks.io' }, to: 'me@go-webmail.test',
    subject: 'Re: Re: Vendor contract — redlines back from legal',
    rawDate: '2026-05-20T17:52:00', date: '20/05/2026 17:52', fullDate: '2026-05-20T17:52:00',
    snippet: 'Legal sent the redlines back this afternoon. Most of it is clean…',
    unread: false, starred: true,
    attachments: [{ name: 'Vendor-MSA-redline.pdf', size: '412 KB', ext: 'PDF' }, { name: 'schedule-A.docx', size: '62 KB', ext: 'DOC' }],
    body: ['Hi,', "Legal sent the redlines back this afternoon. Most of it is clean — they flagged the indemnity clause."],
    signature: { name: 'Marcus Ahn', role: 'Strategic Partnerships' },
  },
]

const MOCK_CONTACTS = [
  { email: 'h.vargas@northbridge-co.com', name: 'Helena Vargas',  title: 'Finance Operations · Northbridge & Co.' },
  { email: 'marcus@strataworks.io',       name: 'Marcus Ahn',     title: 'Strategic Partnerships · Strataworks' },
  { email: 'priya.desai@meridian-lab.org',name: 'Priya Desai',    title: 'Engineering Manager · Meridian Lab' },
  { email: 'owen@halfmoon-studio.test',   name: 'Owen Carlisle',  title: 'Design Lead · Halfmoon Studio' },
  { email: 'renata.lopes@oakfield-hr.test',name: 'Renata Lopes',  title: 'People Operations · Oakfield' },
  { email: 'd.okafor@portico-legal.test', name: 'Diana Okafor',   title: 'Counsel · Portico Legal' },
  { email: 'sven@northshore.test',        name: 'Sven Holt',      title: 'Partner · Northshore Advisors' },
  { email: 'lena.wirth@kestrel-inc.test', name: 'Lena Wirth',     title: 'VP Product · Kestrel Inc.' },
  { email: 'aiko.t@meridian-lab.org',     name: 'Aiko Tanabe',    title: 'Researcher · Meridian Lab' },
  { email: 'fbauer@kestrel-inc.test',     name: 'Felix Bauer',    title: 'Operations · Kestrel Inc.' },
]

const CAL_EVENTS = {
  4:  [{ t: 'Sprint kickoff', k: 'alt' }],
  6:  [{ t: 'Design review' }],
  7:  [{ t: '1:1 Helena', k: 'ghost' }],
  11: [{ t: 'All-hands' }, { t: 'Vendor call', k: 'alt' }],
  13: [{ t: 'Q3 budget prep' }],
  15: [{ t: 'Offsite planning', k: 'warn' }],
  20: [{ t: 'Standup → Atrium 2', k: 'alt' }],
  21: [{ t: '1:1 Helena' }, { t: 'Q3 budget review', k: 'warn' }],
  22: [{ t: 'Legal redlines', k: 'ghost' }],
  25: [{ t: 'Holiday — office closed', k: 'ghost' }],
  27: [{ t: 'Roadmap sync', k: 'alt' }],
  29: [{ t: 'Newsletter ships' }],
}

export const useMailStore = defineStore('mail', () => {
  const auth   = useAuthStore()
  const dialog = useDialogStore()

  // ── State ──────────────────────────────────────────────────────────────────
  const accent      = ref('#1B3A6B')
  const mails       = ref(MOCK_MAIL.map(m => ({ ...m })))
  const folders     = ref(MOCK_FOLDERS.map(f => ({ ...f })))
  const contacts    = ref(MOCK_CONTACTS)
  const calCells    = ref(buildCalCells(CAL_EVENTS))
  const calDow      = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

  const view        = ref('mail')
  const folder      = ref('inbox')
  const selectedId  = ref(null)
  const selectedIds = ref(new Set())
  const query       = ref('')
  const composer    = ref(null)
  const sourceMail  = ref(null)
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

  // ── API ────────────────────────────────────────────────────────────────────
  async function fetchFolderMessages(folderId) {
    if (!auth.isApiOnline) return
    const folderDef = folders.value.find(f => f.id === folderId)
    const imapName  = folderDef?.name || folderDef?.label
    if (!imapName) return
    try {
      const mailRes = await axios.get(`${API_BASE}/mail/${encodeURIComponent(imapName)}`)
      const fetched = (mailRes.data.messages || []).map(m => ({
        id:       String(m.uid),
        folder:   folderId,
        from:     { name: m.from || '', addr: m.from_email || '' },
        to:       auth.currentUser.email,
        subject:  m.subject || '(No Subject)',
        rawDate:  m.date || '',
        date:     formatDate(m.date, auth.datetimeFormat),
        fullDate: m.date || '',
        snippet:  m.subject || '',
        unread:   !m.seen,
        starred:  !!m.flagged,
        attachments: [],
        htmlBody: '',
        body: [],
      }))
      // Replace messages for this folder, keep others
      mails.value = [...mails.value.filter(m => m.folder !== folderId), ...fetched]
      selectedId.value = fetched[0]?.id ?? null

      // Update folder count badge
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

  async function loadFromApi() {
    if (!auth.isApiOnline) return
    try {
      const userRes = await axios.get(`${API_BASE}/auth/me`)
      auth.currentUser.email = userRes.data.username || auth.currentUser.email

      const foldersRes = await axios.get(`${API_BASE}/folders`)
      folders.value = foldersRes.data.map(f => {
        const id     = FOLDER_ID_MAP[f.Name] || f.Name.toLowerCase().replace(/\s+/g, '-')
        const unread = f.Unseen   || 0
        const total  = f.Messages || 0
        return { id, label: f.DisplayName || f.Name, name: f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: !f.IsSystem }
      })

      await fetchFolderMessages(folder.value)
    } catch (e) {
      console.error('API load failed', e)
    }
  }

  async function fetchMessageBody(msgId) {
    const msg = mails.value.find(m => m.id === msgId)
    if (!msg || !auth.isApiOnline) return
    try {
      const fd    = folders.value.find(f => f.id === msg.folder)
      const label = fd?.name || fd?.label || 'INBOX'
      const res   = await axios.get(`${API_BASE}/mail/${encodeURIComponent(label)}/${msgId}`)
      msg.htmlBody    = res.data.html_body  || ''
      msg.body        = res.data.plain_body ? res.data.plain_body.split('\n') : []
      msg.attachments = (res.data.attachments || []).map(a => ({
        name:         a.filename,
        size:         a.size_label || '',
        ext:          (a.filename || '').split('.').pop().toUpperCase(),
        part:         a.part,
        content_type: a.content_type || '',
      }))
    } catch {}
  }

  // ── Mail actions ───────────────────────────────────────────────────────────
  async function selectMsg(id) {
    selectedId.value = id
    await nextTick()
    if (auth.isApiOnline) await fetchMessageBody(id)
  }

  function toggleSelect(id) {
    const s = new Set(selectedIds.value)
    s.has(id) ? s.delete(id) : s.add(id)
    selectedIds.value = s
  }

  async function toggleRead() {
    const m = mails.value.find(x => x.id === selectedId.value)
    if (!m) return
    const nowUnread = !m.unread
    m.unread = nowUnread

    // Update folder count badge
    const folderObj = folders.value.find(f => f.id === m.folder)
    if (folderObj) {
      const msgs   = mails.value.filter(x => x.folder === m.folder)
      const unread = msgs.filter(x => x.unread).length
      folderObj.count = unread > 0 ? `${unread}/${msgs.length}` : String(msgs.length)
    }

    if (auth.isApiOnline) {
      const fd = folders.value.find(f => f.id === m.folder)
      const mailbox = fd?.name || fd?.label || 'INBOX'
      const params = new URLSearchParams()
      params.append('flag', 'seen')
      params.append('value', nowUnread ? '0' : '1')
      await axios.post(
        `${API_BASE}/mail/${encodeURIComponent(mailbox)}/${m.id}/flag`,
        params,
        { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
      ).catch(() => {
        m.unread = !nowUnread
        if (folderObj) {
          const msgs   = mails.value.filter(x => x.folder === m.folder)
          const unread = msgs.filter(x => x.unread).length
          folderObj.count = unread > 0 ? `${unread}/${msgs.length}` : String(msgs.length)
        }
      })
    }
  }

  function archiveMail() {
    const s = selected.value; if (!s) return
    const idx = visibleMails.value.findIndex(m => m.id === s.id)
    const m   = mails.value.find(x => x.id === s.id)
    if (m) m.folder = 'archive'
    const next = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]
    selectedId.value = next?.id ?? null
  }

  async function moveMail(destFolderId) {
    const ids = selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
    if (!ids.length) return

    const srcDef  = folders.value.find(f => f.id === folder.value)
    const srcName = srcDef?.name || srcDef?.label || 'INBOX'
    const dstDef  = folders.value.find(f => f.id === destFolderId)
    const dstName = dstDef?.name || dstDef?.label || destFolderId

    ids.forEach(id => {
      const m = mails.value.find(m => m.id === id)
      if (m) m.folder = destFolderId
    })
    selectedId.value  = null
    selectedIds.value = new Set()

    const srcObj = folders.value.find(f => f.id === folder.value)
    if (srcObj) {
      const rem   = mails.value.filter(m => m.folder === folder.value)
      const unr   = rem.filter(m => m.unread).length
      srcObj.count = unr > 0 ? `${unr}/${rem.length}` : String(rem.length)
    }

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

  async function deleteMail() {
    // Collect IDs to delete: checked set takes priority, else current selection
    const ids = selectedIds.value.size > 0
      ? [...selectedIds.value]
      : (selectedId.value ? [selectedId.value] : [])
    if (!ids.length) return

    // Capture original IMAP folder name before optimistic update
    const folderDef    = folders.value.find(f => f.id === folder.value)
    const currentLabel = folderDef?.name || folderDef?.label || 'INBOX'

    // Pick next message to show after deletion
    const firstIdx = visibleMails.value.findIndex(m => m.id === ids[0])
    const remaining = visibleMails.value.filter(m => !ids.includes(m.id))
    const next = remaining[firstIdx] || remaining[firstIdx - 1] || null

    // Optimistic: remove from visible list and update folder count
    mails.value = mails.value.filter(m => !ids.includes(m.id))
    selectedId.value  = next?.id ?? null
    selectedIds.value = new Set()
    const folderObj = folders.value.find(f => f.id === folder.value)
    if (folderObj) {
      const remaining2 = mails.value.filter(m => m.folder === folder.value)
      const unread2    = remaining2.filter(m => m.unread).length
      folderObj.count  = unread2 > 0 ? `${unread2}/${remaining2.length}` : String(remaining2.length)
    }

    // API calls
    if (auth.isApiOnline) {
      await Promise.all(ids.map(id =>
        axios.delete(`${API_BASE}/mail/${encodeURIComponent(currentLabel)}/${id}`).catch(() => {})
      ))
    }
  }

  function reply() {
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

  function forward() {
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

  function compose()       { composer.value = {} }
  function closeComposer() { composer.value = null }
  async function showSource() {
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
  function closeSource() { sourceMail.value = null; sourceRaw.value = '' }

  function copySource(rawText) {
    try { navigator.clipboard.writeText(rawText) } catch {}
  }

  // ── Folder actions ─────────────────────────────────────────────────────────
  function setFolder(id) {
    folder.value = id
    view.value   = 'mail'
    selectedId.value = null
    fetchFolderMessages(id)
  }

  async function reloadFolders() {
    if (!auth.isApiOnline) return
    const res = await axios.get(`${API_BASE}/folders`)
    folders.value = res.data.map(f => {
      const id     = FOLDER_ID_MAP[f.Name] || f.Name.toLowerCase().replace(/\s+/g, '-')
      const unread = f.Unseen   || 0
      const total  = f.Messages || 0
      return { id, label: f.DisplayName || f.Name, name: f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: !f.IsSystem }
    })
  }

  async function onFolderMenu(action, f) {
    if (action === 'new' || action === 'subfolder') {
      const promptLabel = action === 'subfolder' && f ? `New subfolder inside "${f.label}":` : 'New folder name:'
      const name = await dialog.prompt(promptLabel)
      if (!name?.trim()) return
      if (auth.isApiOnline) {
        const fd = new FormData()
        fd.append('name', name.trim())
        if (action === 'subfolder' && f) fd.append('parent', f.name || f.label)
        try {
          await axios.post(`${API_BASE}/folders`, fd)
          await reloadFolders()
        } catch (e) {
          await dialog.alert('Failed to create folder: ' + (e.response?.data?.error || e.message))
        }
      }
      return
    }
    if (!f) return
    if (action === 'rename') {
      const next = await dialog.prompt('Rename folder:', f.label)
      if (!next?.trim()) return
      if (auth.isApiOnline) {
        const fd = new FormData()
        fd.append('name', f.name || f.label)
        fd.append('newname', next.trim())
        try {
          await axios.post(`${API_BASE}/folders/rename`, fd)
          await reloadFolders()
          if (folder.value === f.id) fetchFolderMessages(folder.value)
        } catch (e) {
          await dialog.alert('Failed to rename folder: ' + (e.response?.data?.error || e.message))
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
      const msg = isTrash
        ? `Permanently delete all messages in "${f.label}"? This cannot be undone.`
        : `Move all messages in "${f.label}" to Trash?`
      if (!await dialog.confirm(msg)) return
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
        } catch (e) {
          await dialog.alert('Failed to delete folder: ' + (e.response?.data?.error || e.message))
        }
      } else {
        folders.value = folders.value.filter(x => x.id !== f.id)
        if (folder.value === f.id) folder.value = 'inbox'
      }
    } else if (action === 'properties') {
      await dialog.alert(`Folder properties\n\nName: ${f.label}\nType: ${f.custom ? 'User folder' : 'System folder'}`)
    }
  }

  return {
    // state
    accent, mails, folders, contacts, calCells, calDow,
    view, folder, selectedId, selectedIds, query, composer, sourceMail, sourceRaw,
    // computed
    visibleMails, counts, selected, currentFolderLabel,
    // actions
    loadFromApi, fetchFolderMessages, reloadFolders, fetchMessageBody,
    selectMsg, toggleSelect, toggleRead, archiveMail, deleteMail,
    reply, forward, compose, closeComposer, showSource, closeSource, copySource,
    moveMail, setFolder, onFolderMenu,
  }
})
