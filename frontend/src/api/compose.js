import api from './axios.js'

export function sendEmail(formData) {
  return api.post('/compose/send', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function saveDraft(payload) {
  return api.post('/compose/draft', payload)
}

export function uploadAttachment(file) {
  const fd = new FormData()
  fd.append('file', file)
  return api.post('/compose/upload', fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
