import type { useMailStore } from '../stores/mail'
import type { useAuthStore } from '../stores/auth'

type MailStore = ReturnType<typeof useMailStore>
type AuthStore = ReturnType<typeof useAuthStore>

let pollTimer: ReturnType<typeof setInterval> | null = null
const POLL_INTERVAL_MS = 10 * 60 * 1000

function playNotificationSound(): void {
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
}

export function startSSE(mail: MailStore, auth: AuthStore): void {
  stopSSE()
  pollTimer = setInterval(async () => {
    if (!auth.isAuthenticated) return
    const hasNew = await mail.fetchFolderMessages('inbox', true)
    if (hasNew) playNotificationSound()
  }, POLL_INTERVAL_MS)
}

export function stopSSE(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}
