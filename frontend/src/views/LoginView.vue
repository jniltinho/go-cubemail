<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <div class="logo-box">
          <span class="logo-letter">W</span>
        </div>
        <h1 class="logo-text">Webmail</h1>
      </div>

      <form @submit.prevent="handleLogin" class="login-form" id="login-form">
        <div v-if="error" class="login-error">
          <i class="pi pi-exclamation-triangle"></i>
          <span>{{ error }}</span>
        </div>

        <div class="form-fields">
          <div class="field" v-if="showHostInput">
            <label for="imap-host" class="field-label">Server</label>
            <input
              id="imap-host"
              v-model="form.imap_host"
              type="text"
              placeholder="mail.example.com"
              autocomplete="off"
              class="field-input"
            />
          </div>

          <div class="field">
            <label for="username" class="field-label">Username</label>
            <input
              id="username"
              v-model="form.username"
              type="text"
              autocomplete="username"
              required
              class="field-input"
            />
          </div>

          <div class="field">
            <label for="password" class="field-label">Password</label>
            <input
              id="password"
              v-model="form.password"
              type="password"
              autocomplete="current-password"
              required
              class="field-input"
            />
          </div>
        </div>

        <div class="button-row">
          <button
            id="login-btn"
            type="submit"
            :disabled="authStore.loading"
            class="signin-btn"
          >
            {{ authStore.loading ? 'Signing in...' : 'Sign In' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const showHostInput = ref(false) // Set via config if needed
const error = ref(null)

const form = ref({
  imap_host: '',
  username: '',
  password: '',
})

onMounted(() => {
  // If already logged in, redirect
  if (authStore.user) {
    router.replace(route.query.redirect || '/mail/INBOX')
  }
})

async function handleLogin() {
  error.value = null
  const ok = await authStore.login(form.value)
  if (ok) {
    router.replace(route.query.redirect || '/mail/INBOX')
  } else {
    error.value = authStore.error
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: flex-start; /* items-start */
  justify-content: center;
  padding-top: 10vh; /* padding-top: 10vh */
  background: url('/static/img/linen.jpg') center repeat #dfdfdf; /* exact background from reference */
  position: relative;
  overflow: hidden;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}

.login-card {
  z-index: 1;
  background: linear-gradient(to bottom, #37517e, #2a3d5e); /* exact gradient from reference */
  border: 1px solid #2a3d5e;
  border-radius: 6px; /* rounded-md */
  width: 576px; /* max-w-xl is 576px */
  max-width: 90%;
  box-shadow: 0 0 15px rgba(0, 0, 0, 0.5); /* shadow-[0_0_15px_rgba(0,0,0,0.5)] */
  overflow: hidden;
  animation: cardIn 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes cardIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.login-header {
  padding: 1.5rem; /* px-6 py-6 */
  border-bottom: 1px solid rgba(255, 255, 255, 0.1); /* border-b border-white/10 */
  display: flex;
  align-items: center;
  gap: 0.75rem; /* gap-3 */
}

.logo-box {
  width: 32px; /* w-8 */
  height: 32px; /* h-8 */
  border-radius: 50%; /* rounded-full */
  background-color: rgba(59, 130, 246, 0.2); /* bg-blue-500/20 */
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-letter {
  color: #60a5fa; /* text-blue-400 */
  font-size: 1.25rem; /* text-xl */
  font-weight: bold;
}

.logo-text {
  font-size: 1.25rem; /* text-xl */
  font-weight: 600; /* font-semibold */
  color: #ffffff;
  letter-spacing: -0.025em; /* tracking-tight */
  margin: 0;
}

.login-form {
  padding: 2rem; /* px-8 py-8 */
  display: flex;
  flex-direction: column;
  gap: 1.25rem; /* gap-5 is 1.25rem */
}

.login-error {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  padding: 0.625rem;
  color: #fca5a5;
  font-size: 0.8125rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 500;
}

.form-fields {
  display: flex;
  flex-direction: column;
  gap: 1.25rem; /* gap-5 */
}

.field {
  display: flex; /* flex items-center */
  align-items: center;
  gap: 1rem; /* gap-4 is 1rem */
}

.field-label {
  width: 80px; /* w-20 is 80px */
  text-align: right;
  font-size: 0.875rem; /* text-sm */
  color: #d1d5db; /* text-gray-300 */
  font-weight: 500; /* font-medium */
}

.field-input {
  flex: 1; /* flex-1 */
  height: 36px;
  background-color: #ffffff;
  color: #1f2937; /* text-gray-800 */
  border: 1px solid #d1d5db; /* border-gray-300 */
  border-radius: 4px; /* rounded */
  padding: 0.5rem 0.75rem; /* px-3 py-2 */
  font-size: 0.875rem; /* text-sm */
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.06); /* shadow-inner */
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field-input:focus {
  border-color: #60a5fa;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.06), 0 0 0 2px rgba(96, 165, 250, 0.4);
}

.button-row {
  display: flex;
  justify-content: center;
  padding-top: 1.25rem; /* pt-5 is 1.25rem */
}

.signin-btn {
  padding: 0.375rem 1.5rem; /* px-6 py-1.5 */
  background-color: #f3f4f6; /* bg-gray-100 */
  color: #1f2937; /* text-gray-800 */
  font-weight: 500; /* font-medium */
  font-size: 0.875rem; /* text-sm */
  border: 1px solid #d1d5db; /* border-gray-300 */
  border-radius: 4px; /* rounded */
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05); /* shadow-sm */
  cursor: pointer;
  outline: none;
  transition: background-color 0.15s, border-color 0.15s;
}

.signin-btn:hover:not(:disabled) {
  background-color: #ffffff; /* hover:bg-white */
}

.signin-btn:active:not(:disabled) {
  background-color: #e5e7eb;
}

.signin-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>
