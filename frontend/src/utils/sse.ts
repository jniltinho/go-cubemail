import type { useMailStore } from '../stores/mail'
import type { useAuthStore } from '../stores/auth'

type MailStore = ReturnType<typeof useMailStore>
type AuthStore = ReturnType<typeof useAuthStore>

let pollTimer: ReturnType<typeof setInterval> | null = null
const POLL_INTERVAL_MS = 10 * 60 * 1000

export function startSSE(mail: MailStore, auth: AuthStore): void {
  stopSSE()
  pollTimer = setInterval(async () => {
    if (auth.isAuthenticated) await mail.fetchFolderMessages('inbox')
  }, POLL_INTERVAL_MS)
}

export function stopSSE(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}
