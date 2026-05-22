import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as apiLogin, logout as apiLogout, me as apiMe } from '@/api/auth.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const loading = ref(false)
  const error = ref(null)

  async function fetchMe() {
    try {
      const res = await apiMe()
      user.value = res.data
    } catch {
      user.value = null
    }
  }

  async function login(payload) {
    loading.value = true
    error.value = null
    try {
      const res = await apiLogin(payload)
      user.value = res.data
      return true
    } catch (err) {
      error.value = err.response?.data?.error || 'Falha no login'
      return false
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await apiLogout()
    } finally {
      user.value = null
    }
  }

  const isAuthenticated = () => user.value !== null

  return { user, loading, error, fetchMe, login, logout, isAuthenticated }
})
