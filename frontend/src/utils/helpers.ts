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
    .replace('06',   String(d.getFullYear()).slice(-2))
    .replace('01',   pad(d.getMonth() + 1))
    .replace('02',   pad(d.getDate()))
    .replace('15',   pad(d.getHours()))
    .replace('04',   pad(d.getMinutes()))
    .replace('05',   pad(d.getSeconds()))
}

/**
 * Extracts the initials from a name string (maximum of two uppercase initials).
 * 
 * @param name - The full name of the contact.
 * @returns The initials representation (e.g. "Helena Vargas" -> "HV").
 */
export function initials(name: string): string {
  return (name || '').split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0].toUpperCase()).join('')
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
 * @returns The target CSS color class (e.g. "text-[#B22B2B]").
 */
export function extColor(ext: string): string {
  const e = (ext || '').toUpperCase()
  if (e === 'PDF') return 'text-[#B22B2B]'
  if (['DOC', 'DOCX'].includes(e)) return 'text-accent'
  if (['XLS', 'XLSX'].includes(e)) return 'text-[#1F7A45]'
  if (['ZIP', 'RAR', '7Z'].includes(e)) return 'text-[#7A4E1F]'
  return 'text-ink-sub'
}

/**
 * Calculates derived dark and light accent colors from a primary hex color
 * and injects them as dynamic CSS custom variables on the document root element.
 * 
 * @param hex - The base hex primary accent color (e.g. "#1B3A6B").
 */
export function applyAccent(hex: string): void {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const darken  = (k: number) => '#' + [r, g, b].map(v => Math.max(0, Math.round(v * k)).toString(16).padStart(2, '0')).join('')
  const lighten = (k: number) => '#' + [r, g, b].map(v => Math.min(255, Math.round(v + (255 - v) * k)).toString(16).padStart(2, '0')).join('')
  const s = document.documentElement.style
  s.setProperty('--accent',        hex)
  s.setProperty('--accent-2',      lighten(0.25))
  s.setProperty('--accent-bar',    darken(0.78))
  s.setProperty('--accent-soft',   lighten(0.88))
  s.setProperty('--accent-soft-2', lighten(0.76))
  s.setProperty('--row-selected',  lighten(0.80))
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
  const msgId    = `<${m.id}.${(m.fullDate || '').replace(/\D/g, '') || Date.now()}@webmail.test>`
  const boundary = `----=_Part_${m.id}`
  const hasAtt   = m.attachments?.length
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
  if (hasAtt) lines.push(`--${boundary}`, `Content-Type: text/plain; charset="utf-8"`, `Content-Transfer-Encoding: 8bit`, '')
  lines.push(...(m.body || []))
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

/**
 * Builds a month-view grid array containing exactly 42 calendar day cells (CalCell),
 * incorporating dimmed trailing and leading days from adjacent months.
 * 
 * @param events - Map of scheduled events, keyed by day of the month numbers.
 * @returns Array with 42 CalCell instances representing the month's layout.
 */
export function buildCalCells(events: Record<number, CalEvent[]> = {}): CalCell[] {
  const CAL_FIRST_WEEKDAY = 5, CAL_DAYS = 31, CAL_PREV_DAYS = 30, TODAY = 22
  const cells: CalCell[] = []
  for (let i = 0; i < CAL_FIRST_WEEKDAY; i++)
    cells.push({ day: CAL_PREV_DAYS - CAL_FIRST_WEEKDAY + 1 + i, dim: true, events: [] })
  for (let d = 1; d <= CAL_DAYS; d++)
    cells.push({ day: d, dim: false, events: events[d] || [], today: d === TODAY })
  while (cells.length % 7 !== 0 || cells.length < 35)
    cells.push({ day: cells.length - CAL_DAYS - CAL_FIRST_WEEKDAY + 1, dim: true, events: [] })
  return cells.slice(0, 42)
}
