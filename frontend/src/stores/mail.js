import { defineStore } from 'pinia'
import { ref, computed, watch, watchEffect, nextTick } from 'vue'
import axios from 'axios'
import { useAuthStore } from './auth'
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
  const auth = useAuthStore()

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
        return { id, label: f.DisplayName || f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: false }
      })

      const label   = folders.value.find(f => f.id === folder.value)?.label || 'Inbox'
      const mailRes = await axios.get(`${API_BASE}/mail/${label}`)
      mails.value = (mailRes.data.messages || []).map(m => ({
        id:       String(m.uid),
        folder:   folder.value,
        from:     { name: m.from || '', addr: m.from_email || '' },
        to:       auth.currentUser.email,
        subject:  m.subject || '(No Subject)',
        rawDate:  m.date || '',
        date:     formatDate(m.date),
        fullDate: m.date || '',
        snippet:  m.subject || '',
        unread:   !m.seen,
        starred:  !!m.flagged,
        attachments: [],
        htmlBody: '',
        body: [],
      }))

      if (mails.value.length && !selectedId.value)
        selectedId.value = mails.value[0].id
    } catch (e) {
      console.error('API load failed', e)
    }
  }

  async function fetchMessageBody(msgId) {
    const msg = mails.value.find(m => m.id === msgId)
    if (!msg || !auth.isApiOnline) return
    try {
      const label = folders.value.find(f => f.id === msg.folder)?.label || 'Inbox'
      const res   = await axios.get(`${API_BASE}/mail/${label}/${msgId}`)
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

  function toggleRead() {
    const m = mails.value.find(x => x.id === selectedId.value)
    if (m) m.unread = !m.unread
  }

  function archiveMail() {
    const s = selected.value; if (!s) return
    const idx = visibleMails.value.findIndex(m => m.id === s.id)
    const m   = mails.value.find(x => x.id === s.id)
    if (m) m.folder = 'archive'
    const next = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]
    selectedId.value = next?.id ?? null
  }

  function deleteMail() {
    const s = selected.value; if (!s) return
    const idx = visibleMails.value.findIndex(m => m.id === s.id)
    const m   = mails.value.find(x => x.id === s.id)
    if (m) m.folder = 'trash'
    const next = visibleMails.value[idx + 1] || visibleMails.value[idx - 1]
    selectedId.value = next?.id ?? null
  }

  function reply() {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   s.from?.addr,
      subj: 'Re: ' + s.subject,
      body: `\n\n— On ${s.fullDate}, ${s.from?.name} wrote —\n${(s.body || []).join('\n')}`,
    }
  }

  function forward() {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   '',
      subj: 'Fwd: ' + s.subject,
      body: `\n\n— Forwarded from ${s.from?.name} —\n${(s.body || []).join('\n')}`,
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
        const label = folders.value.find(f => f.id === m.folder)?.label || 'Inbox'
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
  function setFolder(id) { folder.value = id; view.value = 'mail' }

  function onFolderMenu(action, f) {
    if (action === 'new' || action === 'subfolder') {
      const label = action === 'subfolder' && f ? `New subfolder inside "${f.label}":` : 'New folder name:'
      const name  = window.prompt(label)
      if (!name?.trim()) return
      const id     = 'f-' + Date.now()
      const prefix = action === 'subfolder' && f ? `${f.label} / ` : ''
      folders.value.push({ id, label: prefix + name.trim(), count: '0', custom: true })
      folder.value = id
      return
    }
    if (!f) return
    if (action === 'rename') {
      const next = window.prompt('Rename folder:', f.label)
      if (!next?.trim()) return
      const x = folders.value.find(x => x.id === f.id)
      if (x) x.label = next.trim()
    } else if (action === 'read-all') {
      mails.value.forEach(m => { if (m.folder === f.id) m.unread = false })
    } else if (action === 'empty') {
      if (!window.confirm(`Empty "${f.label}"? Messages will be moved to Deleted Items.`)) return
      mails.value.forEach(m => { if (m.folder === f.id) m.folder = 'trash' })
    } else if (action === 'delete') {
      if (!window.confirm(`Delete folder "${f.label}"?`)) return
      folders.value = folders.value.filter(x => x.id !== f.id)
      if (folder.value === f.id) folder.value = 'inbox'
    } else if (action === 'properties') {
      window.alert(`Folder properties\n\nName: ${f.label}\nType: ${f.custom ? 'User folder' : 'System folder'}`)
    }
  }

  return {
    // state
    accent, mails, folders, contacts, calCells, calDow,
    view, folder, selectedId, selectedIds, query, composer, sourceMail, sourceRaw,
    // computed
    visibleMails, counts, selected, currentFolderLabel,
    // actions
    loadFromApi, fetchMessageBody,
    selectMsg, toggleSelect, toggleRead, archiveMail, deleteMail,
    reply, forward, compose, closeComposer, showSource, closeSource, copySource,
    setFolder, onFolderMenu,
  }
})
