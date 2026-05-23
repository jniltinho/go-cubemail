export interface Attachment {
  name: string
  size: string
  ext: string
  part?: string
  content_type?: string
}

export interface MailMessage {
  id: string
  folder: string
  from: { name: string; addr: string }
  to: string
  subject: string
  rawDate: string
  date: string
  fullDate: string
  snippet: string
  unread: boolean
  starred: boolean
  attachments: Attachment[]
  body?: string[]
  htmlBody?: string
  signature?: { name: string; role: string }
}

export interface Folder {
  id: string
  label: string
  count: string
  name?: string
  custom?: boolean
}

export interface Contact {
  email: string
  name: string
  title: string
}

export interface CalEvent {
  t: string
  k?: string
}

export interface CalCell {
  day: number
  dim: boolean
  today?: boolean
  events: CalEvent[]
}

export interface Toast {
  id: number
  message: string
  type: 'info' | 'success' | 'error' | 'warning'
}

export type DialogType = 'prompt' | 'confirm' | 'alert'

export interface DialogState {
  type: DialogType
  message: string
  defaultValue?: string
  resolve: (value?: string | boolean | null) => void
}
