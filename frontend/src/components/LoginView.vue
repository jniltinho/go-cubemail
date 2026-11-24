<script setup lang="ts">
/**
 * @component LoginView
 * @description The user authentication view panel. Collects email credentials
 * and dispatches authentication operations. If successful, requests the mailbox
 * loader initializer from the API base.
 */

import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import BrandMark from './BrandMark.vue'
import Icon from './Icon.vue'

/** Authentication store instance */
const auth = useAuthStore()
/** Mail store instance */
const mail = useMailStore()

/**
 * Handles the login form submission. Validates fields via auth store triggers,
 * and if the session is authorized, loads the primary mailbox data structures.
 */
async function onSubmit() {
  await auth.handleLogin()
  if (auth.isAuthenticated) await mail.loadFromApi()
}
</script>

<template>
  <div class="login-page min-h-full flex flex-col items-center pt-[100px] px-6 pb-10">
    <form
      class="w-full max-w-[620px] bg-header-brand border border-header-line text-on-header login-shadow rounded-surface overflow-hidden"
      @submit.prevent="onSubmit"
      novalidate
    >
      <!-- Card header -->
      <div class="flex items-center gap-3 px-5 py-3.5 bg-header border-b border-header-line">
        <BrandMark :size="26" />
        <div class="text-[17px] font-bold tracking-tight">
          <b>Web</b>mail<span
            v-if="auth.appVersion"
            class="ml-1.5 text-[11px] font-normal text-on-header-muted font-mono"
            >{{ auth.appVersion }}</span
          >
        </div>
      </div>

      <!-- Card body -->
      <div class="px-7 pt-6 pb-6 flex flex-col gap-3.5">
        <!-- Error banner -->
        <div
          v-if="auth.loginErr"
          class="flex items-start gap-2 px-2.5 py-1.5 border border-on-header-err bg-header text-on-header rounded-control text-[12px] leading-snug"
        >
          <Icon name="alert-triangle" :size="14" class="text-on-header-err mt-0.5 flex-shrink-0" />
          <span>{{ auth.loginErr }}</span>
        </div>

        <!-- Username -->
        <div class="grid items-center gap-3.5" style="grid-template-columns: 110px 1fr">
          <label for="lgn-u" class="text-[12.5px] text-on-header-muted tracking-tight">Username</label>
          <input
            id="lgn-u"
            type="email"
            autocomplete="email"
            autofocus
            :class="['login-input', { bad: auth.loginUserBad }]"
            v-model="auth.loginUser"
            @input="
              auth.loginUserBad = false;
              auth.loginErr = null
            "
            :disabled="auth.loginBusy"
          />
        </div>

        <!-- Password -->
        <div class="grid items-center gap-3.5" style="grid-template-columns: 110px 1fr">
          <label for="lgn-p" class="text-[12.5px] text-on-header-muted tracking-tight">Password</label>
          <input
            id="lgn-p"
            type="password"
            autocomplete="current-password"
            :class="['login-input', { bad: auth.loginPwdBad }]"
            v-model="auth.loginPwd"
            @input="
              auth.loginPwdBad = false;
              auth.loginErr = null
            "
            :disabled="auth.loginBusy"
          />
        </div>

        <!-- Submit -->
        <div class="flex justify-center mt-4">
          <button
            type="submit"
            class="w-[140px] h-[28px] bg-surface-2 border border-header-line rounded-control text-fg text-[12.5px] font-semibold cursor-pointer inline-flex items-center justify-center gap-2 hover:bg-surface hover:border-action-2-hover-line disabled:text-fg-subtle disabled:cursor-wait"
            :disabled="auth.loginBusy"
          >
            <template v-if="auth.loginBusy">
              <svg
                class="w-3 h-3 animate-spin text-accent-strong"
                viewBox="0 0 40 40"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <circle cx="20" cy="20" r="16" stroke="currentColor" stroke-width="6" stroke-opacity="0.25" />
                <path d="M20 4 A16 16 0 0 1 36 20" stroke="currentColor" stroke-width="6" stroke-linecap="round" />
              </svg>
              Signing in…
            </template>
            <template v-else>Sign In</template>
          </button>
        </div>
      </div>
    </form>
  </div>
</template>
