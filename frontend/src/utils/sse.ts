/**
 * @file sse.ts
 * @description Module responsible for periodic polling checks for new email messages on the server
 * and playing audio notification sound alerts upon delivery.
 */

import type { useMailStore } from '../stores/mail'
import type { useAuthStore } from '../stores/auth'

type MailStore = ReturnType<typeof useMailStore>
type AuthStore = ReturnType<typeof useAuthStore>

/**
 * Periodic interval timer ID.
 */
let pollTimer: ReturnType<typeof setInterval> | null = null

/**
 * Polling check interval defined in milliseconds (2 minutes).
 */
const POLL_INTERVAL_MS = 2 * 60 * 1000

/**
 * Synthesizes and plays a short alert notification sound using the Web Audio API.
 * Triggers a short 880Hz sine wave tone with exponential gain decay.
 */
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

/**
 * Starts the periodic polling loop to fetch new emails from the backend.
 * 
 * @param mail - Mail store instance (`useMailStore`).
 * @param auth - Authentication store instance (`useAuthStore`).
 */
export function startSSE(mail: MailStore, auth: AuthStore): void {
  stopSSE()
  pollTimer = setInterval(async () => {
    if (!auth.isAuthenticated) return
    const hasNew = await mail.fetchFolderMessages('inbox', true)
    if (hasNew) playNotificationSound()
  }, POLL_INTERVAL_MS)
}

/**
 * Stops the active polling loop and clears the associated interval timer.
 */
export function stopSSE(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}
