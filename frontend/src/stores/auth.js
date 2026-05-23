import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

const API_BASE = '/api/v1'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const isApiOnline = ref(false)
  const currentUser    = ref({ email: 'user@webmail.test', quotaUsedBytes: 0, quotaTotalBytes: 0 })
  const datetimeFormat = ref('02/01/2006 15:04')

  const loginUser = ref('')
  const loginPwd = ref('')
  const loginBusy = ref(false)
  const loginErr = ref(null)
  const loginUserBad = ref(false)
  const loginPwdBad = ref(false)

  async function fetchQuota() {
    try {
      const res = await axios.get(`${API_BASE}/auth/quota`)
      //console.log('[quota] raw API response:', res.data)
      currentUser.value.quotaUsedBytes = res.data.used || 0
      currentUser.value.quotaTotalBytes = res.data.limit || 0
      //console.log('[quota] used bytes:', currentUser.value.quotaUsedBytes, '/ total bytes:', currentUser.value.quotaTotalBytes)
    } catch (e) {
      console.error('[quota] fetch failed:', e)
    }
  }

  // Called on app mount to detect an existing session
  async function checkSession() {
    axios.defaults.xsrfCookieName = 'csrf_token'
    axios.defaults.xsrfHeaderName = 'X-CSRF-Token'
    axios.interceptors.request.use(cfg => {
      const val = `; ${document.cookie}`.split('; csrf_token=').pop().split(';').shift()
      if (val) cfg.headers['X-CSRF-Token'] = val
      return cfg
    })
    try {
      const res = await axios.get(`${API_BASE}/auth/me`)
      currentUser.value.email = res.data.username || currentUser.value.email
      if (res.data.datetime_format) datetimeFormat.value = res.data.datetime_format
      isAuthenticated.value = true
      isApiOnline.value = true
      fetchQuota()
    } catch {
      isApiOnline.value = false
    }
  }

  async function handleLogin() {
    loginErr.value = null; loginUserBad.value = false; loginPwdBad.value = false
    if (!loginUser.value.trim()) {
      loginUserBad.value = true
      loginErr.value = 'Please enter your username.'
      return
    }
    if (loginPwd.value.length < 4) {
      loginPwdBad.value = true
      loginErr.value = 'Please enter your password.'
      return
    }
    loginBusy.value = true
    try {
      if (isApiOnline.value) {
        const params = new URLSearchParams()
        params.append('username', loginUser.value)
        params.append('password', loginPwd.value)
        await axios.post(`${API_BASE}/auth/login`, params, {
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        })
        currentUser.value.email = loginUser.value.includes('@')
          ? loginUser.value
          : loginUser.value + '@go-webmail.test'
        isAuthenticated.value = true
        fetchQuota()
        // Fetch datetime_format after login
        axios.get(`${API_BASE}/auth/me`).then(r => {
          if (r.data.datetime_format) datetimeFormat.value = r.data.datetime_format
        }).catch(() => {})
      } else {
        await new Promise(r => setTimeout(r, 700))
        if (loginPwd.value.toLowerCase() === 'wrong') {
          loginPwdBad.value = true
          loginErr.value = 'The username or password you entered is incorrect.'
          return
        }
        currentUser.value.email = loginUser.value.includes('@')
          ? loginUser.value
          : loginUser.value + '@go-webmail.test'
        isAuthenticated.value = true
      }
    } catch {
      loginPwdBad.value = true
      loginErr.value = 'The username or password you entered is incorrect.'
    } finally {
      loginBusy.value = false
    }
  }

  async function handleLogout() {
    if (isApiOnline.value) { try { await axios.post(`${API_BASE}/auth/logout`) } catch { } }
    isAuthenticated.value = false
  }

  return {
    isAuthenticated, isApiOnline, currentUser,
    loginUser, loginPwd, loginBusy, loginErr, loginUserBad, loginPwdBad,
    checkSession, handleLogin, handleLogout, fetchQuota, datetimeFormat,
  }
})
