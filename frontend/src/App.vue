<script setup>
import { ref, computed, watch, watchEffect, onMounted, onBeforeUnmount, nextTick } from 'vue'
import axios from 'axios'
import Icon from './components/Icon.vue'
import FolderRow from './components/FolderRow.vue'
import ComposerModal from './components/ComposerModal.vue'
import MailList from './components/MailList.vue'
import ReadingPane from './components/ReadingPane.vue'

// ─── Constants ────────────────────────────────────────────────────────────────
const API_BASE = '/api/v1'

const ACCENT_PRESETS = ['#1B3A6B', '#0B3D91', '#0E4D8C', '#1F4E5F']

const FOLDER_ID_MAP = {
  'Inbox': 'inbox', 'Starred': 'starred', 'Sent Items': 'sent', 'Sent': 'sent',
  'Drafts': 'drafts', 'Archive': 'archive', 'Junk Mail': 'junk', 'Junk': 'junk',
  'Deleted Items': 'trash', 'Trash': 'trash',
}

// ─── Mock data ────────────────────────────────────────────────────────────────
const MOCK_FOLDERS = [
  { id: 'inbox',   label: 'Inbox',         count: '5/12' },
  { id: 'starred', label: 'Starred',        count: '2' },
  { id: 'sent',    label: 'Sent Items',     count: '38' },
  { id: 'drafts',  label: 'Drafts',         count: '2' },
  { id: 'archive', label: 'Archive',        count: '214' },
  { id: 'junk',    label: 'Junk Mail',      count: '7' },
  { id: 'trash',   label: 'Deleted Items',  count: '19' },
]

const MOCK_MAIL = [
  {
    id: 'm1', folder: 'inbox',
    from: { name: 'Helena Vargas', addr: 'h.vargas@northbridge-co.com' },
    to: 'me@go-webmail.test',
    subject: 'Q3 budget review — please confirm Thursday slot',
    date: '21/05/2026 10:42', fullDate: 'Tue, May 21 2026, 10:42',
    snippet: 'Hi — sharing the latest spreadsheet ahead of Thursday. Two line items still need your sign-off…',
    unread: true, starred: true,
    attachments: [{ name: 'Q3-budget-v4.xlsx', size: '184 KB', ext: 'XLS' }],
    body: [
      'Hi,',
      "Sharing the latest spreadsheet ahead of Thursday's review. Two line items still need your sign-off — the contractor uplift in row 14 and the consolidated travel forecast at the bottom.",
      "I've also pre-filled the variance commentary for the regional splits. If anything looks off, drop a comment in column G and I'll reconcile before the call.",
      'Talk soon,',
    ],
    signature: { name: 'Helena Vargas', role: 'Finance Operations, Northbridge & Co.' },
  },
  {
    id: 'm2', folder: 'inbox',
    from: { name: 'IT Helpdesk', addr: 'helpdesk@go-webmail.test' },
    to: 'all-staff@go-webmail.test',
    subject: 'Scheduled maintenance window — Saturday 02:00–04:00 UTC',
    date: '21/05/2026 09:18', fullDate: 'Tue, May 21 2026, 09:18',
    snippet: 'Mail and calendar services will be briefly unavailable during the maintenance window…',
    unread: true, starred: false, attachments: [],
    body: [
      'Hello,',
      'We will be performing scheduled maintenance on the mail and calendar tier this Saturday between 02:00 and 04:00 UTC.',
      'No action is required on your part. If you experience issues after 04:30 UTC please open a ticket via the support portal.',
    ],
    signature: { name: 'Infrastructure Team', role: 'IT Operations' },
  },
  {
    id: 'm3', folder: 'inbox',
    from: { name: 'Marcus Ahn', addr: 'marcus@strataworks.io' },
    to: 'me@go-webmail.test',
    subject: 'Re: Re: Vendor contract — redlines back from legal',
    date: '20/05/2026 17:52', fullDate: 'Mon, May 20 2026, 17:52',
    snippet: 'Legal sent the redlines back this afternoon. Most of it is clean — they flagged the indemnity clause…',
    unread: false, starred: true,
    attachments: [
      { name: 'Vendor-MSA-redline.pdf', size: '412 KB', ext: 'PDF' },
      { name: 'schedule-A.docx',        size: '62 KB',  ext: 'DOC' },
    ],
    body: [
      'Hi,',
      'Legal sent the redlines back this afternoon. Most of it is clean — they flagged the indemnity clause and want a tighter cap, plus a small change to section 7.3 around audit rights.',
      "I've pulled both documents into this thread. Once you've taken a pass we can send it back over to Strataworks for a final sign-off.",
    ],
    signature: { name: 'Marcus Ahn', role: 'Strategic Partnerships' },
  },
  {
    id: 'm4', folder: 'inbox',
    from: { name: 'Banco Pátria — Statements', addr: 'no-reply@bancopatria.test' },
    to: 'me@go-webmail.test',
    subject: 'Your monthly statement is now available',
    date: '20/05/2026 06:00', fullDate: 'Mon, May 20 2026, 06:00',
    snippet: 'Your statement for the period ending May 19 is ready to view…',
    unread: false, starred: false, attachments: [],
    body: [
      'Your statement for the period ending May 19 is ready to view in the online banking portal.',
      'This is an automated message — please do not reply directly to this address.',
    ],
    signature: { name: 'Banco Pátria', role: 'Customer Communications' },
  },
  {
    id: 'm5', folder: 'inbox',
    from: { name: 'Priya Desai', addr: 'priya.desai@meridian-lab.org' },
    to: 'me@go-webmail.test',
    subject: 'Conference room change — Wed standup moved to Atrium 2',
    date: '20/05/2026 11:04', fullDate: 'Mon, May 20 2026, 11:04',
    snippet: "Facilities pulled our usual room for an investor visit. We're in Atrium 2…",
    unread: true, starred: false, attachments: [],
    body: [
      'Hey team,',
      "Facilities pulled our usual room for an investor visit, so Wednesday standup is in Atrium 2 (third floor, near the kitchenette).",
      "Dial-in details are unchanged. Bring your laptop — we'll be reviewing the staging build live.",
    ],
    signature: { name: 'Priya Desai', role: 'Engineering Manager, Meridian Lab' },
  },
  {
    id: 'm6', folder: 'inbox',
    from: { name: 'TravelDesk', addr: 'bookings@traveldesk.test' },
    to: 'me@go-webmail.test',
    subject: 'E-ticket confirmation — LIS → AMS, June 14',
    date: '19/05/2026 21:33', fullDate: 'Sun, May 19 2026, 21:33',
    snippet: 'Your booking has been confirmed. Reference TDX-4419Q…',
    unread: false, starred: false,
    attachments: [{ name: 'eticket-TDX4419Q.pdf', size: '94 KB', ext: 'PDF' }],
    body: [
      'Your booking has been confirmed. Reference TDX-4419Q.',
      'Outbound: Lisbon (LIS) → Amsterdam (AMS), 14 June, departing 07:25. Seat 11C, carry-on only.',
      'Check-in opens 24 hours before scheduled departure.',
    ],
    signature: { name: 'TravelDesk', role: 'Corporate Travel Services' },
  },
  {
    id: 'm7', folder: 'inbox',
    from: { name: 'Owen Carlisle', addr: 'owen@halfmoon-studio.test' },
    to: 'me@go-webmail.test',
    subject: 'Mockups for the landing redesign — round 2',
    date: '19/05/2026 15:08', fullDate: 'Sun, May 19 2026, 15:08',
    snippet: 'Second pass attached. I leaned into the editorial direction we discussed…',
    unread: false, starred: false,
    attachments: [{ name: 'landing-r2-mockups.zip', size: '8.2 MB', ext: 'ZIP' }],
    body: [
      'Hey,',
      'Second pass attached. I leaned into the editorial direction we discussed and tightened the type scale across the hero.',
      "Let me know if anything jumps out — happy to do another spin before we share with the wider group.",
    ],
    signature: { name: 'Owen Carlisle', role: 'Halfmoon Studio' },
  },
  {
    id: 'm8', folder: 'inbox',
    from: { name: 'Renata Lopes', addr: 'renata.lopes@oakfield-hr.test' },
    to: 'me@go-webmail.test',
    subject: 'Benefits enrollment closes Friday',
    date: '17/05/2026 13:21', fullDate: 'Fri, May 17 2026, 13:21',
    snippet: 'Friendly reminder — the open enrollment window for 2026 benefits closes at end-of-day Friday…',
    unread: false, starred: false, attachments: [],
    body: [
      'Hi,',
      'Friendly reminder — the open enrollment window for 2026 benefits closes at end-of-day Friday.',
      "Your current selections will roll over by default. If you'd like to make changes please complete the form in the HR portal before the deadline.",
    ],
    signature: { name: 'Renata Lopes', role: 'People Operations, Oakfield' },
  },
  {
    id: 'm9', folder: 'inbox',
    from: { name: 'Field Report Weekly', addr: 'letters@fieldreport.test' },
    to: 'me@go-webmail.test',
    subject: 'Issue #214 — On the slow disappearance of small bookstores',
    date: '17/05/2026 06:00', fullDate: 'Fri, May 17 2026, 06:00',
    snippet: 'This week: a long read on neighborhood bookstores in transition…',
    unread: false, starred: false, attachments: [],
    body: [
      "This week's edition leads with a long-form piece on how neighborhood bookstores in three mid-sized European cities are adapting.",
      'Also in this issue: notes from the archive, a short Q&A with a reader who runs a translation imprint.',
    ],
    signature: { name: 'Field Report Weekly', role: 'Independent newsletter' },
  },
  {
    id: 'm10', folder: 'inbox',
    from: { name: 'Diana Okafor', addr: 'd.okafor@portico-legal.test' },
    to: 'me@go-webmail.test',
    subject: 'NDA executed — countersigned copy attached',
    date: '16/05/2026 18:44', fullDate: 'Thu, May 16 2026, 18:44',
    snippet: 'Countersigned copy attached for your records…',
    unread: false, starred: false,
    attachments: [{ name: 'NDA-executed-2026-05.pdf', size: '201 KB', ext: 'PDF' }],
    body: [
      'Hello,',
      'Please find the countersigned copy attached for your records.',
      'Let me know if you need a clean version for distribution or any redactions before sharing externally.',
    ],
    signature: { name: 'Diana Okafor', role: 'Portico Legal' },
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

// ─── Helpers ──────────────────────────────────────────────────────────────────
function applyAccent(hex) {
  const r = parseInt(hex.slice(1,3),16), g = parseInt(hex.slice(3,5),16), b = parseInt(hex.slice(5,7),16)
  const darken  = k => '#' + [r,g,b].map(v => Math.max(0, Math.round(v*k)).toString(16).padStart(2,'0')).join('')
  const lighten = k => '#' + [r,g,b].map(v => Math.min(255, Math.round(v+(255-v)*k)).toString(16).padStart(2,'0')).join('')
  const s = document.documentElement.style
  s.setProperty('--accent',        hex)
  s.setProperty('--accent-2',      lighten(0.25))
  s.setProperty('--accent-bar',    darken(0.78))
  s.setProperty('--accent-soft',   lighten(0.88))
  s.setProperty('--accent-soft-2', lighten(0.76))
  s.setProperty('--row-selected',  lighten(0.80))
}

function initials(name) {
  return (name || '').split(/\s+/).filter(Boolean).slice(0,2).map(p => p[0].toUpperCase()).join('')
}

function formatDate(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const dd   = String(d.getDate()).padStart(2,'0')
  const mm   = String(d.getMonth()+1).padStart(2,'0')
  const yyyy = d.getFullYear()
  const hh   = String(d.getHours()).padStart(2,'0')
  const min  = String(d.getMinutes()).padStart(2,'0')
  return `${dd}/${mm}/${yyyy} ${hh}:${min}`
}

function extIcon(ext) {
  const e = (ext||'').toUpperCase()
  if (e==='PDF') return 'file-text'
  if (e==='DOC'||e==='DOCX') return 'file-text'
  if (e==='XLS'||e==='XLSX') return 'file-spreadsheet'
  if (e==='ZIP'||e==='RAR'||e==='7Z') return 'file-archive'
  if (e==='PNG'||e==='JPG'||e==='JPEG'||e==='GIF') return 'file-image'
  return 'file'
}

function extColor(ext) {
  const e = (ext||'').toUpperCase()
  if (e==='PDF') return 'text-[#B22B2B]'
  if (e==='DOC'||e==='DOCX') return 'text-accent'
  if (e==='XLS'||e==='XLSX') return 'text-[#1F7A45]'
  if (e==='ZIP'||e==='RAR'||e==='7Z') return 'text-[#7A4E1F]'
  return 'text-ink-sub'
}

function buildRawSource(m) {
  if (!m) return ''
  const msgId = `<${m.id}.${(m.fullDate||'').replace(/[^0-9]/g,'')||Date.now()}@webmail.test>`
  const boundary = `----=_Part_${m.id}`
  const hasAtt = m.attachments?.length
  const lines = [
    `Return-Path: <${m.from.addr}>`,
    `Received: from mx-eu-03.webmail.test (mx-eu-03.webmail.test [10.0.0.41])`,
    `        by mail-eu-03.webmail.test with ESMTPS id 4Y2lW8b03Bz3p9k;`,
    `        ${m.fullDate} +0000 (UTC)`,
    `Date: ${m.fullDate} +0000`,
    `From: ${m.from.name} <${m.from.addr}>`,
    `To: <${m.to}>`,
    `Subject: ${m.subject}`,
    `Message-ID: ${msgId}`,
    `MIME-Version: 1.0`,
    hasAtt ? `Content-Type: multipart/mixed; boundary="${boundary}"` : `Content-Type: text/plain; charset="utf-8"`,
    hasAtt ? '' : `Content-Transfer-Encoding: 8bit`,
    `X-Mailer: Webmail v2.4.1`,
    `X-Spam-Status: No, score=-1.2 required=5.0`,
    '',
  ]
  if (hasAtt) lines.push(`--${boundary}`, `Content-Type: text/plain; charset="utf-8"`, `Content-Transfer-Encoding: 8bit`, '')
  lines.push(...m.body)
  if (m.signature) lines.push('', '-- ', m.signature.name, m.signature.role)
  if (hasAtt) {
    for (const a of m.attachments) {
      lines.push('', `--${boundary}`,
        `Content-Type: application/octet-stream; name="${a.name}"`,
        `Content-Transfer-Encoding: base64`,
        `Content-Disposition: attachment; filename="${a.name}"`,
        '', `[binary content — ${a.size}]`)
    }
    lines.push('', `--${boundary}--`)
  }
  return lines.join('\n')
}

function buildCalCells() {
  const CAL_FIRST_WEEKDAY = 5, CAL_DAYS = 31, CAL_PREV_DAYS = 30, TODAY = 22
  const cells = []
  for (let i = 0; i < CAL_FIRST_WEEKDAY; i++)
    cells.push({ day: CAL_PREV_DAYS - CAL_FIRST_WEEKDAY + 1 + i, dim: true, events: [] })
  for (let d = 1; d <= CAL_DAYS; d++)
    cells.push({ day: d, dim: false, events: CAL_EVENTS[d] || [], today: d === TODAY })
  while (cells.length % 7 !== 0 || cells.length < 35)
    cells.push({ day: cells.length - CAL_DAYS - CAL_FIRST_WEEKDAY + 1, dim: true, events: [] })
  return cells.slice(0, 42)
}

// ─── Auth & login state ───────────────────────────────────────────────────────
const isAuthenticated = ref(false)
const isApiOnline     = ref(false)

const loginUser    = ref('')
const loginPwd     = ref('')
const loginBusy    = ref(false)
const loginErr     = ref(null)
const loginUserBad = ref(false)
const loginPwdBad  = ref(false)

const currentUser = ref({ email: 'cmoreira@go-webmail.test', quotaUsed: 3.4, quotaTotal: 25 })

// ─── App state ────────────────────────────────────────────────────────────────
const accent = ref('#1B3A6B')
watchEffect(() => applyAccent(accent.value))

const mails    = ref(MOCK_MAIL.map(m => ({ ...m })))
const folders  = ref(MOCK_FOLDERS.map(f => ({ ...f })))
const contacts = ref(MOCK_CONTACTS)

const view       = ref('mail')   // 'mail' | 'contacts' | 'calendar'
const folder     = ref('inbox')
const selectedId = ref('m3')
const selectedIds = ref(new Set())
const query      = ref('')
const composer   = ref(null)    // null | { to, subj, body }
const sourceMail = ref(null)

const calCells = buildCalCells()
const calDow   = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat']

// ─── Computed ─────────────────────────────────────────────────────────────────
const visibleMails = computed(() => {
  let xs = mails.value
  if (folder.value === 'starred') xs = xs.filter(m => m.starred)
  else xs = xs.filter(m => m.folder === folder.value)
  const q = query.value.trim().toLowerCase()
  if (q) xs = xs.filter(m =>
    m.from.name.toLowerCase().includes(q) ||
    m.from.addr.toLowerCase().includes(q) ||
    m.subject.toLowerCase().includes(q) ||
    m.snippet.toLowerCase().includes(q) ||
    (m.attachments||[]).some(a => a.name.toLowerCase().includes(q))
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

const selected = computed(() => mails.value.find(m => m.id === selectedId.value) || null)

// ─── Watchers ─────────────────────────────────────────────────────────────────
watch([folder, visibleMails], () => {
  if (!visibleMails.value.find(m => m.id === selectedId.value))
    selectedId.value = visibleMails.value[0]?.id || null
})

watch(selectedId, () => {
  const s = selected.value
  if (s?.unread) {
    setTimeout(() => {
      const m = mails.value.find(x => x.id === s.id)
      if (m) m.unread = false
    }, 400)
  }
})

// ─── API ──────────────────────────────────────────────────────────────────────
async function loadFromApi() {
  try {
    const userRes = await axios.get(`${API_BASE}/auth/me`)
    currentUser.value.email = userRes.data.username || currentUser.value.email

    const foldersRes = await axios.get(`${API_BASE}/folders`)
    folders.value = foldersRes.data.map(f => {
      const id = FOLDER_ID_MAP[f.Name] || f.Name.toLowerCase().replace(/\s+/g,'-')
      const unread = f.Unseen || 0
      const total  = f.Messages || 0
      return { id, label: f.DisplayName || f.Name, count: unread > 0 ? `${unread}/${total}` : String(total), custom: false }
    })

    const folderLabel = folders.value.find(f => f.id === folder.value)?.label || 'Inbox'
    const mailRes = await axios.get(`${API_BASE}/mail/${folderLabel}`)
    mails.value = (mailRes.data.messages || []).map(m => ({
      id:       String(m.uid),
      folder:   folder.value,
      from:     { name: m.from || '', addr: m.from_email || '' },
      to:       currentUser.value.email,
      subject:  m.subject || '(No Subject)',
      rawDate:  m.date || '',
      date:     formatDate(m.date),
      fullDate: m.date || '',
      snippet:  m.subject || '',
      unread:   !m.seen,
      starred:  !!m.flagged,
      attachments: [],
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
  if (!msg || !isApiOnline.value) return
  try {
    const folderLabel = folders.value.find(f => f.id === msg.folder)?.label || 'Inbox'
    const res = await axios.get(`${API_BASE}/mail/${folderLabel}/${msgId}`)
    msg.htmlBody  = res.data.html_body  || ''
    msg.body      = res.data.plain_body ? res.data.plain_body.split('\n') : []
    msg.attachments = (res.data.attachments || []).map(a => ({
      name: a.filename,
      filename: a.filename,
      size: a.size_label || '',
      size_label: a.size_label || '',
      ext: (a.filename || '').split('.').pop().toUpperCase(),
      part: a.part,
      content_type: a.content_type || '',
    }))
  } catch {}
}

// ─── Auth actions ─────────────────────────────────────────────────────────────
async function handleLogin() {
  loginErr.value = null; loginUserBad.value = false; loginPwdBad.value = false
  if (!loginUser.value.trim()) { loginUserBad.value = true; loginErr.value = 'Please enter your username.'; return }
  if (loginPwd.value.length < 4) { loginPwdBad.value = true; loginErr.value = 'Please enter your password.'; return }
  loginBusy.value = true
  try {
    if (isApiOnline.value) {
      const params = new URLSearchParams()
      params.append('username', loginUser.value)
      params.append('password', loginPwd.value)
      await axios.post(`${API_BASE}/auth/login`, params, { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } })
      isAuthenticated.value = true
      await loadFromApi()
    } else {
      await new Promise(r => setTimeout(r, 700))
      if (loginPwd.value.toLowerCase() === 'wrong') {
        loginPwdBad.value = true
        loginErr.value = 'The username or password you entered is incorrect.'
        return
      }
      currentUser.value.email = loginUser.value.includes('@') ? loginUser.value : loginUser.value + '@go-webmail.test'
      isAuthenticated.value = true
    }
  } catch {
    loginPwdBad.value = true
    loginErr.value = 'The username or password you entered is incorrect.'
  } finally {
    loginBusy.value = false
  }
}

async function handleLogout() {
  if (isApiOnline.value) { try { await axios.post(`${API_BASE}/auth/logout`) } catch {} }
  isAuthenticated.value = false
  selectedId.value = null
}

// ─── Mail actions ─────────────────────────────────────────────────────────────
async function selectMsg(id) {
  selectedId.value = id
  await nextTick()
  if (isApiOnline.value) await fetchMessageBody(id)
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
  const m = mails.value.find(x => x.id === s.id)
  if (m) m.folder = 'archive'
  const next = visibleMails.value[idx+1] || visibleMails.value[idx-1]
  selectedId.value = next ? next.id : null
}

function deleteMail() {
  const s = selected.value; if (!s) return
  const idx = visibleMails.value.findIndex(m => m.id === s.id)
  const m = mails.value.find(x => x.id === s.id)
  if (m) m.folder = 'trash'
  const next = visibleMails.value[idx+1] || visibleMails.value[idx-1]
  selectedId.value = next ? next.id : null
}

function reply() {
  const s = selected.value; if (!s) return
  composer.value = {
    to: s.from.addr,
    subj: 'Re: ' + s.subject,
    body: `\n\n— On ${s.fullDate}, ${s.from.name} wrote —\n${s.body.join('\n')}`,
  }
}

function forward() {
  const s = selected.value; if (!s) return
  composer.value = {
    to: '',
    subj: 'Fwd: ' + s.subject,
    body: `\n\n— Forwarded message from ${s.from.name} —\n${s.body.join('\n')}`,
  }
}

function compose()       { composer.value = {} }
function closeComposer() { composer.value = null }
function showSource()    { sourceMail.value = selected.value }
function closeSource()   { sourceMail.value = null }

function copySource() {
  try { navigator.clipboard.writeText(buildRawSource(sourceMail.value)) } catch {}
}

// ─── Folder menu ──────────────────────────────────────────────────────────────
function onFolderMenu(action, f) {
  if (action === 'new' || action === 'subfolder') {
    const prompt = action === 'subfolder' && f ? `New subfolder inside "${f.label}":` : 'New folder name:'
    const name = window.prompt(prompt)
    if (!name?.trim()) return
    const id = 'f-' + Date.now()
    const parentLabel = action === 'subfolder' && f ? `${f.label} / ` : ''
    folders.value.push({ id, label: parentLabel + name.trim(), count: '0', custom: true })
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

function setFolder(id) { folder.value = id; view.value = 'mail' }

// ─── Keyboard shortcuts ───────────────────────────────────────────────────────
function onKey(e) {
  const tag = e.target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || composer.value) return
  const list = visibleMails.value
  const i = list.findIndex(m => m.id === selectedId.value)
  if      (e.key === 'j') selectedId.value = list[Math.min(i+1, list.length-1)]?.id || selectedId.value
  else if (e.key === 'k') selectedId.value = list[Math.max(i-1, 0)]?.id || selectedId.value
  else if (e.key === 'r') reply()
  else if (e.key === 'e') archiveMail()
  else if (e.key === '#' || e.key === 'Delete') deleteMail()
  else if (e.key === 'c') compose()
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────
onMounted(async () => {
  window.addEventListener('keydown', onKey)
  axios.defaults.xsrfCookieName = 'csrf_token'
  axios.defaults.xsrfHeaderName = 'X-CSRF-Token'
  axios.interceptors.request.use(cfg => {
    const val = `; ${document.cookie}`.split('; csrf_token=').pop().split(';').shift()
    if (val) cfg.headers['X-CSRF-Token'] = val
    return cfg
  })
  try {
    await axios.get(`${API_BASE}/auth/me`)
    isAuthenticated.value = true
    isApiOnline.value = true
    await loadFromApi()
  } catch {
    isApiOnline.value = false
  }
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <!-- ══════════════════════════════════════════════════════════════════
       LOGIN VIEW
  ════════════════════════════════════════════════════════════════════ -->
  <div v-if="!isAuthenticated"
       class="min-h-full flex flex-col items-center pt-[100px] px-6 pb-10"
       style="background-color:#C9CDD3;background-image:repeating-linear-gradient(0deg,rgba(255,255,255,.18) 0 1px,transparent 1px 3px),repeating-linear-gradient(90deg,rgba(0,0,0,.05) 0 1px,transparent 1px 3px)">
    <form class="w-full max-w-[620px] bg-accent border border-[#0B1F40] text-white login-shadow"
          @submit.prevent="handleLogin" novalidate>
      <!-- card header -->
      <div class="flex items-center gap-3 px-5 py-3.5 bg-accent-bar border-b border-[#2A4978]">
        <div class="w-7 h-7 bg-[#66A0FF] text-accent-bar grid place-items-center font-mono font-extrabold text-[13px]">W</div>
        <div class="text-[17px] font-bold tracking-tight">
          <b>Web</b>mail<span class="ml-1.5 text-[11px] font-normal text-[#BFD0EA] font-mono">v2.4.1</span>
        </div>
      </div>
      <!-- card body -->
      <div class="px-7 pt-6 pb-6 flex flex-col gap-3.5">
        <div v-if="loginErr"
             class="flex items-start gap-2 px-2.5 py-1.5 border border-[#ff8a8a] bg-[#2A1F2F] text-[#FFD9D9] text-[12px] leading-snug">
          <Icon name="alert-triangle" :size="14" class="text-[#ff8a8a] mt-0.5" />
          <span>{{ loginErr }}</span>
        </div>
        <div class="grid items-center gap-3.5" style="grid-template-columns:110px 1fr">
          <label for="lgn-u" class="text-[12.5px] text-[#BFD0EA] tracking-tight">Username</label>
          <input id="lgn-u" type="text" autocomplete="username" autofocus
                 :class="['login-input', { bad: loginUserBad }]"
                 v-model="loginUser" @input="loginUserBad=false;loginErr=null"
                 :disabled="loginBusy" />
        </div>
        <div class="grid items-center gap-3.5" style="grid-template-columns:110px 1fr">
          <label for="lgn-p" class="text-[12.5px] text-[#BFD0EA] tracking-tight">Password</label>
          <input id="lgn-p" type="password" autocomplete="current-password"
                 :class="['login-input', { bad: loginPwdBad }]"
                 v-model="loginPwd" @input="loginPwdBad=false;loginErr=null"
                 :disabled="loginBusy" />
        </div>
        <div class="grid gap-3.5 mt-2" style="grid-template-columns:110px 1fr">
          <div></div>
          <button type="submit"
                  class="min-w-[120px] h-[30px] bg-[#F5F7FA] border border-[#0B1F40] text-ink text-[13px] font-semibold cursor-pointer px-5 inline-flex items-center justify-center gap-2 hover:bg-white hover:border-accent-2 disabled:text-ink-mute disabled:cursor-wait"
                  :disabled="loginBusy">
            <template v-if="loginBusy">
              <span class="inline-block w-3 h-3 border-2 border-[#14305A]/25 border-t-[#14305A] animate-spin"></span>
              Signing in…
            </template>
            <template v-else>Sign In</template>
          </button>
        </div>
      </div>
    </form>
  </div>

  <!-- ══════════════════════════════════════════════════════════════════
       MAIN APP
  ════════════════════════════════════════════════════════════════════ -->
  <template v-else>

    <!-- ── APP BAR ──────────────────────────────────────────────────── -->
    <div class="h-11 flex items-stretch bg-accent-bar text-white border-b border-[#0B1F40] select-none">
      <!-- Brand -->
      <div class="flex items-center gap-2.5 px-4 font-bold text-[14px] tracking-[0.3px] bg-accent border-r border-[#0B1F40] min-w-[220px]">
        <div class="w-[22px] h-[22px] bg-white text-accent font-mono font-extrabold grid place-items-center text-[13px]">W</div>
        <div><b>Web</b>mail</div>
      </div>

      <!-- Tabs -->
      <div class="flex items-stretch">
        <div v-for="tab in [{id:'mail',label:'Mail',icon:'mail'},{id:'contacts',label:'Contacts',icon:'users'},{id:'calendar',label:'Calendar',icon:'calendar'}]"
             :key="tab.id"
             :class="['flex items-center gap-1.5 px-4 text-[12.5px] cursor-pointer border-r border-[#0B1F40]',
                      view===tab.id ? 'bg-accent text-white shadow-[inset_0_-3px_0_#66A0FF]' : 'text-[#D5E0F2] hover:bg-[#102744] hover:text-white']"
             @click="view=tab.id">
          <Icon :name="tab.icon" :size="14" />
          <span>{{ tab.label }}</span>
        </div>
      </div>

      <!-- Search -->
      <div class="flex-1 flex items-center px-3 gap-2">
        <form class="search-box" @submit.prevent>
          <select>
            <option>All Folders</option><option>Inbox</option>
            <option>Sent</option><option>From…</option><option>Subject…</option>
          </select>
          <input type="text" placeholder="Search mail (from, subject, body, attachment name…)"
                 :value="query" @input="query=$event.target.value" />
          <button type="submit">Search</button>
        </form>
      </div>

      <!-- User + Logout -->
      <div class="flex items-center gap-2.5 px-4 text-[12.5px] text-[#D5E0F2] border-l border-[#0B1F40]">
        <div class="text-white text-[12.5px]">{{ currentUser.email }}</div>
        <button class="inline-flex items-center justify-center w-7 h-[26px] bg-transparent border border-[#4A6FA0] text-[#D5E0F2] hover:bg-[#102744] hover:text-white hover:border-[#66A0FF] cursor-pointer"
                title="Logout" @click="handleLogout">
          <Icon name="log-out" :size="14" />
        </button>
      </div>
    </div>

    <!-- ── TOOLBAR ───────────────────────────────────────────────────── -->
    <div v-if="view==='mail'" class="h-9 bg-panel-2 border-b border-line flex items-center px-2 gap-1">
      <button class="tbtn tbtn-primary" @click="compose"><Icon name="pencil-line" :size="14" /> New Message</button>
      <div class="w-px h-[18px] bg-line mx-1"></div>
      <button class="tbtn" @click="loadFromApi"><Icon name="refresh-cw" :size="14" /> Refresh</button>
      <button class="tbtn" :disabled="!selected" @click="reply"><Icon name="reply" :size="14" /> Reply</button>
      <button class="tbtn" :disabled="!selected" @click="reply"><Icon name="reply-all" :size="14" /> Reply All</button>
      <button class="tbtn" :disabled="!selected" @click="forward"><Icon name="forward" :size="14" /> Forward</button>
      <div class="w-px h-[18px] bg-line mx-1"></div>
      <button class="tbtn" :disabled="!selected" @click="archiveMail"><Icon name="archive" :size="14" /> Archive</button>
      <button class="tbtn tbtn-danger" :disabled="!selected" @click="deleteMail"><Icon name="trash-2" :size="14" /> Delete</button>
      <div class="w-px h-[18px] bg-line mx-1"></div>
      <button class="tbtn" :disabled="!selected" @click="toggleRead"><Icon name="circle-dot" :size="14" /> Mark read/unread</button>
      <div class="flex-1"></div>
      <div class="text-[12px] text-ink-sub px-1.5">{{ visibleMails.length }} items · {{ counts.inboxUnread }} unread</div>
    </div>

    <!-- ── 3-COLUMN GRID ─────────────────────────────────────────────── -->
    <div class="grid bg-app-bg"
         :style="{ gridTemplateColumns: '220px 380px 1fr', height: view==='mail' ? 'calc(100vh - 44px - 36px)' : 'calc(100vh - 44px)' }">

      <!-- ── SIDEBAR ──────────────────────────────────────────────────── -->
      <div class="bg-white border-r border-line overflow-auto flex flex-col scroll-y">
        <div class="h-10 px-3 flex items-center justify-between bg-panel-2 border-b border-line">
          <div class="text-[11px] uppercase tracking-wider text-ink-sub font-bold">Folders</div>
          <button class="bg-white border border-line h-[22px] px-2 text-[11.5px] text-ink hover:bg-accent-soft inline-flex items-center gap-1"
                  @click="onFolderMenu('new', null)">
            <Icon name="folder-plus" :size="12" /> New Folder
          </button>
        </div>
        <div class="py-1.5 border-b border-line-soft">
          <FolderRow
            v-for="f in folders" :key="f.id"
            :folder="f"
            :active="folder===f.id"
            @click="setFolder(f.id)"
            @menu="(action, fl) => onFolderMenu(action, fl)"
          />
        </div>
        <div class="flex-1"></div>
        <div class="px-3.5 py-2.5 text-[11px] text-ink-mute border-t border-line-soft">
          Quota <b class="text-ink-sub">{{ currentUser.quotaUsed }} / {{ currentUser.quotaTotal }} GB</b>
          <div class="h-1.5 bg-line-soft mt-1 border border-line">
            <div class="h-full bg-accent"
                 :style="{ width: (currentUser.quotaUsed/currentUser.quotaTotal*100).toFixed(1)+'%' }"></div>
          </div>
        </div>
      </div>

      <!-- ── MAIL: list + reading pane ─────────────────────────────────── -->
      <template v-if="view==='mail'">
        <!-- Mail list (second column) -->
        <MailList
          :mails="visibleMails"
          :selected-id="selectedId"
          :selected-ids="selectedIds"
          :folder-label="folders.find(f=>f.id===folder)?.label || 'Inbox'"
          :query="query"
          @select="selectMsg"
          @toggle-select="toggleSelect"
        />

        <!-- Reading pane (third column) -->
        <ReadingPane
          :message="selected"
          @reply="reply"
          @forward="forward"
          @source="showSource"
          @archive="archiveMail"
          @delete="deleteMail"
        />
      </template>

      <!-- ── CONTACTS PANE ─────────────────────────────────────────────── -->
      <div v-else-if="view==='contacts'"
           class="bg-white overflow-auto flex flex-col scroll-y"
           style="grid-column:2/4">
        <div class="py-3 px-4 bg-panel-2 border-b border-line flex items-center gap-3">
          <h2 class="m-0 text-[15px] text-accent-bar font-bold">Contacts</h2>
          <div class="text-[12px] text-ink-sub">{{ contacts.length }} entries · sorted by name</div>
          <div class="ml-auto flex gap-1.5">
            <button class="tbtn tbtn-primary"><Icon name="user-plus" :size="13" /> New contact</button>
            <button class="tbtn"><Icon name="upload" :size="13" /> Import</button>
          </div>
        </div>
        <div class="grid gap-px bg-line" style="grid-template-columns:repeat(auto-fill,minmax(260px,1fr))">
          <div v-for="c in contacts" :key="c.email" class="bg-white py-3 px-3.5 flex gap-3 items-start">
            <div class="w-[38px] h-[38px] bg-accent text-white grid place-items-center font-bold text-[14px] flex-shrink-0">
              {{ initials(c.name) }}
            </div>
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-semibold text-ink">{{ c.name }}</div>
              <a href="#" @click.prevent="composer={to:c.email}"
                 class="text-[11.5px] mt-0.5 block text-accent-2 no-underline hover:underline">{{ c.email }}</a>
              <div class="text-[11px] text-ink-mute mt-1">{{ c.title }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- ── CALENDAR PANE ─────────────────────────────────────────────── -->
      <div v-else-if="view==='calendar'"
           class="bg-white overflow-auto flex flex-col scroll-y"
           style="grid-column:2/4">
        <div class="py-2 px-4 bg-panel-2 border-b border-line flex items-center gap-1.5">
          <button class="tbtn"><Icon name="chevron-left" :size="13" /></button>
          <button class="tbtn"><Icon name="chevron-right" :size="13" /></button>
          <button class="tbtn">Today</button>
          <div class="text-[15px] font-bold text-accent-bar ml-2.5">May 2026</div>
          <div class="ml-auto flex gap-1.5">
            <button class="tbtn">Day</button>
            <button class="tbtn">Week</button>
            <button class="tbtn tbtn-primary">Month</button>
            <button class="tbtn"><Icon name="plus" :size="13" /> Event</button>
          </div>
        </div>
        <div class="flex-1 grid gap-px bg-line border-t border-line"
             style="grid-template-columns:repeat(7,1fr);grid-template-rows:auto repeat(6,1fr);min-height:0">
          <div v-for="d in calDow" :key="d" class="cal-cell head">{{ d }}</div>
          <div v-for="(c,i) in calCells" :key="i"
               :class="['cal-cell', { dim: c.dim, today: c.today }]">
            <div class="num">{{ c.day }}</div>
            <div v-for="(e,j) in c.events" :key="j" :class="['cal-evt', e.k||'']">{{ e.t }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── COMPOSER MODAL ─────────────────────────────────────────────── -->
    <ComposerModal v-if="composer!==null" :prefill="composer" @close="closeComposer" />

    <!-- ── SOURCE VIEWER ──────────────────────────────────────────────── -->
    <div v-if="sourceMail" class="modal-wrap" @click.self="closeSource">
      <div class="composer-shell" style="width:820px" role="dialog">
        <div class="bg-accent-bar text-white py-2 px-3 flex items-center gap-2 text-[13px] font-semibold">
          <Icon name="code-2" :size="14" />
          <span>View Source — {{ sourceMail.subject }}</span>
          <button class="ml-auto cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978]"
                  type="button" @click="closeSource">
            <Icon name="x" :size="12" />
          </button>
        </div>
        <div class="bg-[#0E1A2E] text-[#D5E0F2] font-mono text-[12px] leading-snug py-3.5 px-4 max-h-[calc(100vh-200px)] overflow-auto whitespace-pre-wrap break-words scroll-y">{{ buildRawSource(sourceMail) }}</div>
        <div class="py-2 px-2.5 bg-panel-2 border-t border-line flex items-center gap-1.5">
          <span class="text-[12px] text-ink-sub px-1.5">Raw RFC 822 source · read-only</span>
          <div class="ml-auto flex gap-1.5">
            <button class="tbtn" @click="copySource"><Icon name="copy" :size="13" /> Copy</button>
            <button class="tbtn tbtn-primary" @click="closeSource">Close</button>
          </div>
        </div>
      </div>
    </div>

  </template>
</template>
