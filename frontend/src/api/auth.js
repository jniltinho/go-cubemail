import api from './axios.js'

export function login(payload) {
  // Send as URLSearchParams so Go form decoder works
  const form = new URLSearchParams()
  form.append('imap_host', payload.imap_host || '')
  form.append('username', payload.username)
  form.append('password', payload.password)
  return api.post('/auth/login', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export function logout() {
  return api.post('/auth/logout')
}

export function me() {
  return api.get('/auth/me')
}
