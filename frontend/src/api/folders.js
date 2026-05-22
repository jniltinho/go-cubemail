import api from './axios.js'

export function getFolders() {
  return api.get('/folders')
}

export function getUnreadCount(name) {
  return api.get(`/folders/${encodeURIComponent(name)}/count`)
}

export function createFolder(parent, name, delim = '/') {
  const form = new URLSearchParams()
  form.append('parent', parent)
  form.append('name', name)
  form.append('delim', delim)
  return api.post('/folders', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export function renameFolder(name, newname) {
  const form = new URLSearchParams()
  form.append('name', name)
  form.append('newname', newname)
  return api.post('/folders/rename', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export function deleteFolder(name) {
  const form = new URLSearchParams()
  form.append('name', name)
  return api.post('/folders/delete', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}
