/**
 * @file helpers.ts
 * @description Miscellanous utility functions for date formatting, string manipulation,
 * monthly calendar cell generations, and raw RFC 822 MIME mail source building.
 */

import type { CalCell, CalEvent, MailMessage } from '../types'

/**
 * Formats a raw date string using a format string similar to Go's layout reference (02/01/2006 15:04:05).
 *
 * @param raw - The raw date string to parse.
 * @param goFmt - The target Go-style format string (optional, defaults to '02/01/2006 15:04').
 * @returns The formatted date/time representation string, or the raw input if parsing fails.
 */
/**
 * Parses a date string in DD/MM/YYYY HH:MM format (as returned by the backend)
 * into a Unix timestamp in milliseconds for comparison purposes.
 */
export function parseMailDate(s: string): number {
  if (!s) return 0
  const [datePart, timePart = '00:00'] = s.split(' ')
  const [dd, mm, yyyy] = datePart.split('/')
  const [hh, min] = timePart.split(':')
  const d = new Date(+yyyy, +mm - 1, +dd, +hh, +min)
  return isNaN(d.getTime()) ? 0 : d.getTime()
}

export function formatDate(raw: string, goFmt?: string): string {
  if (!raw) return ''
  let d = new Date(raw)
  if (isNaN(d.getTime())) d = new Date(raw.replace(/^[A-Za-z]{3},\s*/, ''))
  if (isNaN(d.getTime())) d = new Date(raw.replace(' ', 'T'))
  if (isNaN(d.getTime())) return raw

  const fmt = goFmt || '02/01/2006 15:04'
  const pad = (n: number) => String(n).padStart(2, '0')

  return fmt
    .replace('2006', String(d.getFullYear()))
    .replace('06', String(d.getFullYear()).slice(-2))
    .replace('01', pad(d.getMonth() + 1))
    .replace('02', pad(d.getDate()))
    .replace('15', pad(d.getHours()))
    .replace('04', pad(d.getMinutes()))
    .replace('05', pad(d.getSeconds()))
}

/**
 * Extracts the initials from a name string (maximum of two uppercase initials).
 *
 * @param name - The full name of the contact.
 * @returns The initials representation (e.g. "Helena Vargas" -> "HV").
 */
export function initials(name: string): string {
  return (name || '')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0].toUpperCase())
    .join('')
}

/**
 * Resolves a Lucide icon component name for a given file extension.
 *
 * @param ext - The file extension (e.g. "pdf", "docx").
 * @returns The target Lucide icon name (e.g. "file-text").
 */
export function extIcon(ext: string): string {
  const e = (ext || '').toUpperCase()
  if (['PDF', 'DOC', 'DOCX'].includes(e)) return 'file-text'
  if (['XLS', 'XLSX'].includes(e)) return 'file-spreadsheet'
  if (['ZIP', 'RAR', '7Z'].includes(e)) return 'file-archive'
  if (['PNG', 'JPG', 'JPEG', 'GIF', 'WEBP'].includes(e)) return 'file-image'
  return 'file'
}

/**
 * Resolves a Tailwind CSS color utility class for a given file extension.
 *
 * @param ext - The file extension.
 * @returns The target CSS color class (e.g. "text-danger").
 */
export function extColor(ext: string): string {
  const e = (ext || '').toUpperCase()
  if (e === 'PDF') return 'text-danger'
  if (['DOC', 'DOCX'].includes(e)) return 'text-accent'
  if (['XLS', 'XLSX'].includes(e)) return 'text-ok'
  if (['ZIP', 'RAR', '7Z'].includes(e)) return 'text-fg-muted'
  return 'text-fg-muted'
}

/** Color of a calendar that has none of its own. Content, not interface: it does not follow the theme. */
export const DEFAULT_CAL_COLOR = '#3788d8'

/**
 * Ink for text drawn over a color the user chose (a calendar's color): black or white, whichever has the higher
 * WCAG contrast with it. Literal on purpose: the background is the user's, so the ink cannot follow the theme.
 *
 * @param color - Background as `#rgb` or `#rrggbb`; anything else is treated as dark.
 */
export function inkOn(color: string): '#000000' | '#ffffff' {
  const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec((color || '').trim())
  if (!m) return '#ffffff'
  const hex = m[1].length === 3 ? m[1].replace(/./g, '$&$&') : m[1]
  const lin = (i: number) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  }
  const luminance = 0.2126 * lin(0) + 0.7152 * lin(2) + 0.0722 * lin(4)
  // black wins when (L + 0.05) / 0.05 > 1.05 / (L + 0.05)
  return luminance > 0.179 ? '#000000' : '#ffffff'
}

/**
 * Compiles structural RFC 822 MIME mail headers and multi-part boundary segments
 * from a given `MailMessage` model definition.
 *
 * @param m - The email message data object.
 * @returns The compiled MIME format raw source string.
 */
export function buildRawSource(m: MailMessage | null): string {
  if (!m) return ''
  const msgId = `<${m.id}.${(m.fullDate || '').replace(/\D/g, '') || Date.now()}@webmail.test>`
  const boundary = `----=_Part_${m.id}`
  const hasAtt = m.attachments?.length
  const lines: string[] = [
    `Return-Path: <${m.from?.addr}>`,
    `Received: from mx-eu-03.webmail.test (mx-eu-03.webmail.test [10.0.0.41])`,
    `        by mail-eu-03.webmail.test with ESMTPS id 4Y2lW8b03Bz3p9k;`,
    `        ${m.fullDate} +0000 (UTC)`,
    `Date: ${m.fullDate} +0000`,
    `From: ${m.from?.name} <${m.from?.addr}>`,
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
  if (hasAtt)
    lines.push(`--${boundary}`, `Content-Type: text/plain; charset="utf-8"`, `Content-Transfer-Encoding: 8bit`, '')
  lines.push(...(m.body || []))
  if (m.signature) lines.push('', '-- ', m.signature.name, m.signature.role)
  if (hasAtt) {
    for (const a of m.attachments) {
      lines.push(
        '',
        `--${boundary}`,
        `Content-Type: application/octet-stream; name="${a.name}"`,
        `Content-Transfer-Encoding: base64`,
        `Content-Disposition: attachment; filename="${a.name}"`,
        '',
        `[binary content — ${a.size}]`,
      )
    }
    lines.push('', `--${boundary}--`)
  }
  return lines.join('\n')
}

/**
 * Builds a month-view grid array containing exactly 42 calendar day cells (CalCell),
 * incorporating dimmed trailing and leading days from adjacent months.
 *
 * @param events - Map of scheduled events, keyed by day of the month numbers.
 * @returns Array with 42 CalCell instances representing the month's layout.
 */
export function buildCalCells(events: Record<number, CalEvent[]> = {}): CalCell[] {
  const CAL_FIRST_WEEKDAY = 5,
    CAL_DAYS = 31,
    CAL_PREV_DAYS = 30,
    TODAY = 22
  const cells: CalCell[] = []
  for (let i = 0; i < CAL_FIRST_WEEKDAY; i++)
    cells.push({ day: CAL_PREV_DAYS - CAL_FIRST_WEEKDAY + 1 + i, dim: true, events: [] })
  for (let d = 1; d <= CAL_DAYS; d++) cells.push({ day: d, dim: false, events: events[d] || [], today: d === TODAY })
  while (cells.length % 7 !== 0 || cells.length < 35)
    cells.push({ day: cells.length - CAL_DAYS - CAL_FIRST_WEEKDAY + 1, dim: true, events: [] })
  return cells.slice(0, 42)
}
