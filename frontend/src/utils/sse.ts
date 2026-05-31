import type { useMailStore } from '../stores/mail'
import type { useAuthStore } from '../stores/auth'

type MailStore = ReturnType<typeof useMailStore>
type AuthStore = ReturnType<typeof useAuthStore>

// How often to check for new mail (5 minutes).
const POLL_INTERVAL_MS = 5 * 60 * 1000

let pollTimer: ReturnType<typeof setInterval> | null = null

/** Plays a short sine-wave beep using the Web Audio API to alert the user of new mail. */
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
  } catch { }
}

/**
 * Starts a background timer that checks the INBOX for new mail every 5 minutes.
 * Plays a short audio notification when the unread count increases.
 * Calling this while a timer is already running replaces the existing one.
 */
export function startNewMailPolling(mail: MailStore, auth: AuthStore): void {
  stopNewMailPolling()
  pollTimer = setInterval(async () => {
    if (!auth.isAuthenticated) return
    const hasNew = await mail.fetchFolderMessages('inbox', true)
    if (hasNew) {
      playNotificationSound()
    }
  }, POLL_INTERVAL_MS)
}

/** Stops the background polling timer, if one is running. */
export function stopNewMailPolling(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}
