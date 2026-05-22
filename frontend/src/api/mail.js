import api from './axios.js'

export function listMessages(mailbox, page = 1) {
  return api.get(`/mail/${encodeURIComponent(mailbox)}`, { params: { page } })
}

export function getMessage(mailbox, uid) {
  return api.get(`/mail/${encodeURIComponent(mailbox)}/${uid}`)
}

export function flagMessage(mailbox, uid, flag, value) {
  const form = new URLSearchParams()
  form.append('flag', flag)
  form.append('value', value ? '1' : '0')
  return api.post(`/mail/${encodeURIComponent(mailbox)}/${uid}/flag`, form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export function moveMessage(mailbox, uid, dest) {
  const form = new URLSearchParams()
  form.append('dest', dest)
  return api.post(`/mail/${encodeURIComponent(mailbox)}/${uid}/move`, form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export function deleteMessage(mailbox, uid) {
  return api.delete(`/mail/${encodeURIComponent(mailbox)}/${uid}`)
}

export function emptyTrash(mailbox) {
  return api.delete(`/mail/${encodeURIComponent(mailbox)}`)
}

export function downloadMessage(mailbox, uid) {
  return `/api/mail/${encodeURIComponent(mailbox)}/${uid}/download`
}

export function rawMessage(mailbox, uid) {
  return `/api/mail/${encodeURIComponent(mailbox)}/${uid}/raw`
}

export function getRawMessage(mailbox, uid) {
  return api.get(`/mail/${encodeURIComponent(mailbox)}/${uid}/raw`, { responseType: 'text' })
}

export function getAttachmentUrl(mailbox, uid, part) {
  return `/api/mail/${encodeURIComponent(mailbox)}/${uid}/attachment/${part}`
}

export function searchMessages(q, mailbox = 'INBOX') {
  return api.get('/search', { params: { q, mailbox } })
}
