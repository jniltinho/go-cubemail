import type { Folder, MailMessage, Contact, CalEvent } from '../../types'

export const FOLDER_ID_MAP: Record<string, string> = {
  'Inbox': 'inbox', 'Starred': 'starred', 'Sent Items': 'sent', 'Sent': 'sent',
  'Drafts': 'drafts', 'Archive': 'archive', 'Junk Mail': 'junk', 'Junk': 'junk',
  'Deleted Items': 'trash', 'Trash': 'trash', 'INBOX': 'inbox',
}

export const MOCK_FOLDERS: Folder[] = [
  { id: 'inbox',   label: 'Inbox',        count: '5/12' },
  { id: 'starred', label: 'Starred',       count: '2' },
  { id: 'sent',    label: 'Sent Items',    count: '38' },
  { id: 'drafts',  label: 'Drafts',        count: '2' },
  { id: 'archive', label: 'Archive',       count: '214' },
  { id: 'junk',    label: 'Junk Mail',     count: '7' },
  { id: 'trash',   label: 'Deleted Items', count: '19' },
]

export const MOCK_MAIL: MailMessage[] = [
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

export const MOCK_CONTACTS: Contact[] = [
  { email: 'h.vargas@northbridge-co.com',  name: 'Helena Vargas', title: 'Finance Operations · Northbridge & Co.' },
  { email: 'marcus@strataworks.io',        name: 'Marcus Ahn',    title: 'Strategic Partnerships · Strataworks' },
  { email: 'priya.desai@meridian-lab.org', name: 'Priya Desai',   title: 'Engineering Manager · Meridian Lab' },
  { email: 'owen@halfmoon-studio.test',    name: 'Owen Carlisle', title: 'Design Lead · Halfmoon Studio' },
  { email: 'renata.lopes@oakfield-hr.test',name: 'Renata Lopes',  title: 'People Operations · Oakfield' },
  { email: 'd.okafor@portico-legal.test',  name: 'Diana Okafor',  title: 'Counsel · Portico Legal' },
  { email: 'sven@northshore.test',         name: 'Sven Holt',     title: 'Partner · Northshore Advisors' },
  { email: 'lena.wirth@kestrel-inc.test',  name: 'Lena Wirth',    title: 'VP Product · Kestrel Inc.' },
  { email: 'aiko.t@meridian-lab.org',      name: 'Aiko Tanabe',   title: 'Researcher · Meridian Lab' },
  { email: 'fbauer@kestrel-inc.test',      name: 'Felix Bauer',   title: 'Operations · Kestrel Inc.' },
]

export const CAL_EVENTS: Record<number, CalEvent[]> = {
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
