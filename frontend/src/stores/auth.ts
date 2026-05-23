import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

const API_BASE = '/api/v1'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const isApiOnline     = ref(false)
  const currentUser     = ref({ email: 'user@webmail.test', quotaUsedBytes: 0, quotaTotalBytes: 0 })
  const datetimeFormat  = ref('02/01/2006 15:04')
  const appVersion      = ref('')

  const loginUser    = ref('')
  const loginPwd     = ref('')
  const loginBusy    = ref(false)
  const loginErr     = ref<string | null>(null)
  const loginUserBad = ref(false)
  const loginPwdBad  = ref(false)

  async function fetchQuota(): Promise<void> {
    try {
      const res = await axios.get(`${API_BASE}/auth/quota`)
      currentUser.value.quotaUsedBytes  = res.data.used  || 0
      currentUser.value.quotaTotalBytes = res.data.limit || 0
    } catch (e) {
      console.error('[quota] fetch failed:', e)
    }
  }

  async function checkSession(): Promise<void> {
    axios.defaults.xsrfCookieName = 'csrf_token'
    axios.defaults.xsrfHeaderName = 'X-CSRF-Token'
    axios.interceptors.request.use(cfg => {
      const val = `; ${document.cookie}`.split('; csrf_token=').pop()!.split(';').shift()
      if (val) cfg.headers['X-CSRF-Token'] = val
      return cfg
    })
    try {
      const [meRes, versionRes] = await Promise.allSettled([
        axios.get(`${API_BASE}/auth/me`),
        axios.get(`${API_BASE}/version`),
      ])
      if (versionRes.status === 'fulfilled') {
        appVersion.value = versionRes.value.data.version || ''
      }
      if (meRes.status === 'fulfilled') {
        currentUser.value.email = meRes.value.data.username || currentUser.value.email
        if (meRes.value.data.datetime_format) datetimeFormat.value = meRes.value.data.datetime_format
        isAuthenticated.value = true
        isApiOnline.value = true
        fetchQuota()
      } else {
        isApiOnline.value = (meRes as PromiseRejectedResult).reason?.response ? true : false
      }
    } catch {
      isApiOnline.value = false
    }
  }

  async function handleLogin(): Promise<void> {
    loginErr.value = null; loginUserBad.value = false; loginPwdBad.value = false
    const emailRe = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRe.test(loginUser.value.trim())) {
      loginUserBad.value = true
      loginErr.value = 'Please enter a valid email address.'
      return
    }
    if (loginPwd.value.length < 4) {
      loginPwdBad.value = true
      loginErr.value = 'Please enter your password.'
      return
    }
    loginBusy.value = true
    try {
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
      isApiOnline.value = true
      fetchQuota()
      axios.get(`${API_BASE}/auth/me`).then(r => {
        if (r.data.datetime_format) datetimeFormat.value = r.data.datetime_format
      }).catch(() => {})
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        loginPwdBad.value = true
        loginErr.value = 'The username or password you entered is incorrect.'
      } else {
        loginErr.value = 'Server unavailable. Please try again later.'
      }
    } finally {
      loginBusy.value = false
    }
  }

  async function handleLogout(): Promise<void> {
    if (isApiOnline.value) { try { await axios.post(`${API_BASE}/auth/logout`) } catch { } }
    isAuthenticated.value = false
  }

  return {
    isAuthenticated, isApiOnline, currentUser, appVersion,
    loginUser, loginPwd, loginBusy, loginErr, loginUserBad, loginPwdBad,
    checkSession, handleLogin, handleLogout, fetchQuota, datetimeFormat,
  }
})
