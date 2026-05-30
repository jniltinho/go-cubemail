/**
 * @file webpush.ts
 * @description Web Push subscription management.
 * Registers the service worker, requests Notification permission, and
 * subscribes to the push endpoint using the server's VAPID public key.
 */

import axios from 'axios'

const SW_PATH  = '/sw.js'
const API_BASE_PUSH = typeof API_BASE !== 'undefined' ? API_BASE : '/api/v1'

let swReg: ServiceWorkerRegistration | null = null

/**
 * Initialises the service worker and requests push permission.
 * Should be called once after the user is authenticated.
 * Silently does nothing when the browser does not support push.
 */
export async function initWebPush(): Promise<void> {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) return

  try {
    // Fetch VAPID public key — if the server hasn't configured push,
    // the endpoint returns 503 and we skip silently.
    const vapidRes = await axios.get(`${API_BASE_PUSH}/push/vapid-public-key`)
    const vapidKey: string = vapidRes.data.public_key
    if (!vapidKey) return

    swReg = await navigator.serviceWorker.register(SW_PATH)
    await navigator.serviceWorker.ready

    const permission = await Notification.requestPermission()
    if (permission !== 'granted') return

    // Re-use existing subscription or create a new one.
    let sub = await swReg.pushManager.getSubscription()
    if (!sub) {
      sub = await swReg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidKey),
      })
    }
    if (!sub) return

    const subJson = sub.toJSON() as {
      endpoint: string
      keys: { p256dh: string; auth: string }
    }
    await axios.post(`${API_BASE_PUSH}/push/subscribe`, {
      endpoint: subJson.endpoint,
      keys: { p256dh: subJson.keys.p256dh, auth: subJson.keys.auth },
    })
  } catch {
    // Push not supported or server not configured — ignore silently.
  }
}

/**
 * Unsubscribes from Web Push and removes the server-side record.
 */
export async function stopWebPush(): Promise<void> {
  if (!swReg) return
  try {
    const sub = await swReg.pushManager.getSubscription()
    if (!sub) return
    await axios.delete(`${API_BASE_PUSH}/push/unsubscribe`, {
      data: { endpoint: sub.endpoint },
    })
    await sub.unsubscribe()
  } catch {}
}

/** Converts a URL-safe base64 string to a Uint8Array for applicationServerKey. */
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = atob(base64)
  return Uint8Array.from([...rawData].map((c) => c.charCodeAt(0)))
}
