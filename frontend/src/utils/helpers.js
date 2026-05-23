// ─── Date ─────────────────────────────────────────────────────────────────────
export function formatDate(raw) {
  if (!raw) return ''
  let d = new Date(raw)
  if (isNaN(d.getTime())) d = new Date(raw.replace(/^[A-Za-z]{3},\s*/, ''))
  if (isNaN(d.getTime())) d = new Date(raw.replace(' ', 'T'))
  if (isNaN(d.getTime())) return raw
  const dd  = String(d.getDate()).padStart(2, '0')
  const mm  = String(d.getMonth() + 1).padStart(2, '0')
  const hh  = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${dd}/${mm}/${d.getFullYear()} ${hh}:${min}`
}

// ─── Strings ──────────────────────────────────────────────────────────────────
export function initials(name) {
  return (name || '').split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0].toUpperCase()).join('')
}

// ─── File types ───────────────────────────────────────────────────────────────
export function extIcon(ext) {
  const e = (ext || '').toUpperCase()
  if (['PDF','DOC','DOCX'].includes(e)) return 'file-text'
  if (['XLS','XLSX'].includes(e)) return 'file-spreadsheet'
  if (['ZIP','RAR','7Z'].includes(e)) return 'file-archive'
  if (['PNG','JPG','JPEG','GIF','WEBP'].includes(e)) return 'file-image'
  return 'file'
}

export function extColor(ext) {
  const e = (ext || '').toUpperCase()
  if (e === 'PDF') return 'text-[#B22B2B]'
  if (['DOC','DOCX'].includes(e)) return 'text-accent'
  if (['XLS','XLSX'].includes(e)) return 'text-[#1F7A45]'
  if (['ZIP','RAR','7Z'].includes(e)) return 'text-[#7A4E1F]'
  return 'text-ink-sub'
}

// ─── Theme ────────────────────────────────────────────────────────────────────
export function applyAccent(hex) {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const darken  = k => '#' + [r, g, b].map(v => Math.max(0, Math.round(v * k)).toString(16).padStart(2, '0')).join('')
  const lighten = k => '#' + [r, g, b].map(v => Math.min(255, Math.round(v + (255 - v) * k)).toString(16).padStart(2, '0')).join('')
  const s = document.documentElement.style
  s.setProperty('--accent',        hex)
  s.setProperty('--accent-2',      lighten(0.25))
  s.setProperty('--accent-bar',    darken(0.78))
  s.setProperty('--accent-soft',   lighten(0.88))
  s.setProperty('--accent-soft-2', lighten(0.76))
  s.setProperty('--row-selected',  lighten(0.80))
}

// ─── Email source builder ─────────────────────────────────────────────────────
export function buildRawSource(m) {
  if (!m) return ''
  const msgId    = `<${m.id}.${(m.fullDate || '').replace(/\D/g, '') || Date.now()}@webmail.test>`
  const boundary = `----=_Part_${m.id}`
  const hasAtt   = m.attachments?.length
  const lines = [
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

// ─── Calendar ─────────────────────────────────────────────────────────────────
export function buildCalCells(events = {}) {
  const CAL_FIRST_WEEKDAY = 5, CAL_DAYS = 31, CAL_PREV_DAYS = 30, TODAY = 22
  const cells = []
  for (let i = 0; i < CAL_FIRST_WEEKDAY; i++)
    cells.push({ day: CAL_PREV_DAYS - CAL_FIRST_WEEKDAY + 1 + i, dim: true, events: [] })
  for (let d = 1; d <= CAL_DAYS; d++)
    cells.push({ day: d, dim: false, events: events[d] || [], today: d === TODAY })
  while (cells.length % 7 !== 0 || cells.length < 35)
    cells.push({ day: cells.length - CAL_DAYS - CAL_FIRST_WEEKDAY + 1, dim: true, events: [] })
  return cells.slice(0, 42)
}
