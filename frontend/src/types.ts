/**
 * @file types.ts
 * @description TypeScript interface and type definitions used throughout the Cubemail frontend.
 */

/**
 * Represents a mail message attachment.
 */
export interface Attachment {
  /** File name with extension */
  name: string
  /** Formatted file size (e.g. "2.4 MB") */
  size: string
  /** Upper-case file extension (e.g. "PDF") */
  ext: string
  /** Identifier of the MIME message part (optional) */
  part?: string
  /** MIME content-type string (optional, e.g. "application/pdf") */
  content_type?: string
}

/**
 * Represents a complete email message.
 */
export interface MailMessage {
  /** Unique message identifier (IMAP UID) */
  id: string
  /** Folder ID where the message is located (e.g. "inbox") */
  folder: string
  /** Sender metadata containing name and email address */
  from: { name: string; addr: string }
  /** Recipient(s) formatted as string */
  to: string
  /** Message subject line */
  subject: string
  /** Raw date string returned by the API */
  rawDate: string
  /** Formatted date string adjusted to user preference (e.g. "02/01/2006 15:04") */
  date: string
  /** Full detailed date used in MIME headers and tooltips */
  fullDate: string
  /** Short snippet preview of the email body text */
  snippet: string
  /** True if the message has not been read yet */
  unread: boolean
  /** True if the message is flagged/starred as important */
  starred: boolean
  /** List of attachments contained within the message */
  attachments: Attachment[]
  /** Plain text body lines (optional) */
  body?: string[]
  /** Rich HTML format body string (optional) */
  htmlBody?: string
  /** Optional automated user signature details (name and role) */
  signature?: { name: string; role: string }
}

/**
 * Represents a folder/mailbox on the IMAP server.
 */
export interface Folder {
  /** Unique frontend folder identifier (e.g. "inbox", "sent") */
  id: string
  /** Text display label (e.g. "Inbox") */
  label: string
  /** Email count formatted as string (e.g. "5/120" for unread/total, or "120" total) */
  count: string
  /** Original folder name on the IMAP server (optional) */
  name?: string
  /** True if this is a custom folder created by the user */
  custom?: boolean
}

/**
 * Represents a contact in the user's address book.
 */
export interface Contact {
  /** Unique numeric identifier (optional for new contacts) */
  id?: number
  /** Full name of the contact */
  name: string
  /** Email address */
  email: string
  /** Job title or professional role (optional) */
  title?: string
  /** Organization or company name (optional) */
  company?: string
  /** Phone number (optional) */
  phone?: string
  /** Additional notes or comments (optional) */
  notes?: string
}

/**
 * Represents an scheduled event in the calendar.
 */
export interface CalEvent {
  /** Descriptive title of the event */
  t: string
  /** Styling key indicator (optional, e.g. "alt", "ghost", "warn") */
  k?: string
}

/**
 * Represents a single day (cell) in the monthly calendar grid layout.
 */
export interface CalCell {
  /** Day of the month number */
  day: number
  /** True if the day belongs to an adjacent month (dimmed) */
  dim: boolean
  /** True if this represents the current day (today) */
  today?: boolean
  /** List of events scheduled for this day */
  events: CalEvent[]
}

/**
 * Represents a floating visual toast notification.
 */
export interface Toast {
  /** Unique auto-incremented identifier */
  id: number
  /** Notification message text */
  message: string
  /** Status type for styling colors and icons */
  type: 'info' | 'success' | 'error' | 'warning'
}

/**
 * Type of interactive dialog layouts.
 */
export type DialogType = 'prompt' | 'confirm' | 'alert'

/**
 * Represents the active state of an interactive dialog modal.
 */
export interface DialogState {
  /** Type of behavior and styling buttons layout */
  type: DialogType
  /** Question or description text shown to the user */
  message: string
  /** Initial prefill text value for prompt fields (optional) */
  defaultValue?: string
  /** Resolution callback for the active dialog promise containing user answers */
  resolve: (value?: string | boolean | null) => void
}
