// Service Worker for go-cubemail Web Push notifications.
// Handles push events and notification clicks even when the app is closed.

self.addEventListener('push', (event) => {
  let data = { title: 'New mail', body: 'You have new messages.' }
  try {
    if (event.data) data = event.data.json()
  } catch {}

  event.waitUntil(
    self.registration.showNotification(data.title || 'New mail', {
      body:    data.body || 'You have new messages in your inbox.',
      icon:    '/favicon.svg',
      badge:   '/favicon.svg',
      tag:     'new-mail',
      renotify: true,
      data:    { url: '/' },
    })
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = event.notification.data?.url || '/'
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
      for (const wc of windowClients) {
        if (wc.url.includes(self.location.origin)) {
          wc.focus()
          return
        }
      }
      return clients.openWindow(url)
    })
  )
})
