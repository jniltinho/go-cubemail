/**
 * @file sse.ts
 * @description Real-time new-mail notifications via Server-Sent Events (SSE).
 * Falls back to 10-minute polling when EventSource is unavailable or the
 * connection fails (e.g. reverse proxy does not support streaming).
 */

import type { useMailStore } from '../stores/mail'
import type { useAuthStore } from '../stores/auth'

type MailStore = ReturnType<typeof useMailStore>
type AuthStore = ReturnType<typeof useAuthStore>

let sseSource: EventSource | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

const POLL_INTERVAL_MS = 10 * 60 * 1000  // 10 min fallback
const SSE_URL = `${typeof API_BASE !== 'undefined' ? API_BASE : '/api/v1'}/events`

function playNotificationSound(): void {
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'sine'
    osc.frequency.setValueAtTime(880, ctx.currentTime)
    gain.gain.setValueAtTime(0.3, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.8)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.8)
  } catch {}
}

function startPollingFallback(mail: MailStore, auth: AuthStore): void {
  if (pollTimer !== null) return
  pollTimer = setInterval(async () => {
    if (!auth.isAuthenticated) return
    const hasNew = await mail.fetchFolderMessages('inbox', true)
    if (hasNew) playNotificationSound()
  }, POLL_INTERVAL_MS)
}

/**
 * Starts the SSE connection. On error or server-sent "error" event,
 * falls back to periodic polling so the app still detects new mail.
 */
export function startNewMailPolling(mail: MailStore, auth: AuthStore): void {
  stopNewMailPolling()

  if (typeof EventSource === 'undefined') {
    startPollingFallback(mail, auth)
    return
  }

  const src = new EventSource(SSE_URL, { withCredentials: true })
  sseSource = src

  src.addEventListener('new-mail', async () => {
    const hasNew = await mail.fetchFolderMessages('inbox', true)
    if (hasNew) playNotificationSound()
  })

  src.addEventListener('error', () => {
    // SSE error (proxy timeout, network) — fall back to polling.
    stopSSE()
    startPollingFallback(mail, auth)
  })

  // Server closed connection (e.g. IMAP error) — reconnect after 30 s via polling fallback.
  src.onmessage = () => { /* keep-alive data messages — no action needed */ }
}

function stopSSE(): void {
  if (sseSource) {
    sseSource.close()
    sseSource = null
  }
}

/**
 * Stops both the SSE connection and the polling fallback timer.
 */
export function stopNewMailPolling(): void {
  stopSSE()
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}
