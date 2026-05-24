/**
 * @file auth.ts
 * @description Pinia store for managing user authentication state,
 * active user session data, quota checking, and form credential details.
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

/**
 * Global authentication store (`useAuthStore`).
 */
export const useAuthStore = defineStore('auth', () => {
  /** True if the user is currently authenticated */
  const isAuthenticated = ref(false)
  /** True if the API backend server is active/reachable */
  const isApiOnline     = ref(false)
  /** Active user profile data (email address and disk quota metrics) */
  const currentUser     = ref({ email: 'user@webmail.test', quotaUsedBytes: 0, quotaTotalBytes: 0 })
  /** Preferred date and time layout format (Go reference syntax) */
  const datetimeFormat  = ref('02/01/2006 15:04')
  /** Active backend application version string */
  const appVersion      = ref('')

  /** Username field model in the login form view */
  const loginUser    = ref('')
  /** Password field model in the login form view */
  const loginPwd     = ref('')
  /** True if the login network transaction is active */
  const loginBusy    = ref(false)
  /** Stores the active validation or login transaction error string */
  const loginErr     = ref<string | null>(null)
  /** True if the username input has formatting errors */
  const loginUserBad = ref(false)
  /** True if the password input has formatting errors */
  const loginPwdBad  = ref(false)

  /**
   * Fetches active user disk quota metrics (used space and maximum limit)
   * and populates the reactive `currentUser` metrics.
   */
  async function fetchQuota(): Promise<void> {
    try {
      const res = await axios.get(`${API_BASE}/auth/quota`)
      currentUser.value.quotaUsedBytes  = res.data.used  || 0
      currentUser.value.quotaTotalBytes = res.data.limit || 0
    } catch (e) {
      console.error('[quota] fetch failed:', e)
    }
  }

  /**
   * Verifies the active user session, sets global Axios CSRF cookie interceptors,
   * and loads current app version values.
   */
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

  /**
   * Performs form validations, formatting, and submits login credentials.
   * Compiles url-encoded parameters containing credentials.
   * On success, initializes authorized states and retrieves profile parameters.
   */
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

  /**
   * Submits a logout request to the server API and clears local
   * authentication records and session credentials.
   */
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
