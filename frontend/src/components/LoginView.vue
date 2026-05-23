<script setup>
import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

const auth = useAuthStore()
const mail = useMailStore()

async function onSubmit() {
  await auth.handleLogin()
  if (auth.isAuthenticated) await mail.loadFromApi()
}
</script>

<template>
  <div
    class="min-h-full flex flex-col items-center pt-[100px] px-6 pb-10"
    style="background-color:#C9CDD3;background-image:repeating-linear-gradient(0deg,rgba(255,255,255,.18) 0 1px,transparent 1px 3px),repeating-linear-gradient(90deg,rgba(0,0,0,.05) 0 1px,transparent 1px 3px)"
  >
    <form
      class="w-full max-w-[620px] bg-accent border border-[#0B1F40] text-white login-shadow"
      @submit.prevent="onSubmit"
      novalidate
    >
      <!-- Card header -->
      <div class="flex items-center gap-3 px-5 py-3.5 bg-accent-bar border-b border-[#2A4978]">
        <div class="w-7 h-7 bg-[#66A0FF] text-accent-bar grid place-items-center font-mono font-extrabold text-[13px]">W</div>
        <div class="text-[17px] font-bold tracking-tight">
          <b>Web</b>mail<span class="ml-1.5 text-[11px] font-normal text-[#BFD0EA] font-mono">v2.4.1</span>
        </div>
      </div>

      <!-- Card body -->
      <div class="px-7 pt-6 pb-6 flex flex-col gap-3.5">
        <!-- Error banner -->
        <div
          v-if="auth.loginErr"
          class="flex items-start gap-2 px-2.5 py-1.5 border border-[#ff8a8a] bg-[#2A1F2F] text-[#FFD9D9] text-[12px] leading-snug"
        >
          <Icon name="alert-triangle" :size="14" class="text-[#ff8a8a] mt-0.5 flex-shrink-0" />
          <span>{{ auth.loginErr }}</span>
        </div>

        <!-- Username -->
        <div class="grid items-center gap-3.5" style="grid-template-columns:110px 1fr">
          <label for="lgn-u" class="text-[12.5px] text-[#BFD0EA] tracking-tight">Username</label>
          <input
            id="lgn-u"
            type="text"
            autocomplete="username"
            autofocus
            :class="['login-input', { bad: auth.loginUserBad }]"
            v-model="auth.loginUser"
            @input="auth.loginUserBad = false; auth.loginErr = null"
            :disabled="auth.loginBusy"
          />
        </div>

        <!-- Password -->
        <div class="grid items-center gap-3.5" style="grid-template-columns:110px 1fr">
          <label for="lgn-p" class="text-[12.5px] text-[#BFD0EA] tracking-tight">Password</label>
          <input
            id="lgn-p"
            type="password"
            autocomplete="current-password"
            :class="['login-input', { bad: auth.loginPwdBad }]"
            v-model="auth.loginPwd"
            @input="auth.loginPwdBad = false; auth.loginErr = null"
            :disabled="auth.loginBusy"
          />
        </div>

        <!-- Submit -->
        <div class="grid gap-3.5 mt-2" style="grid-template-columns:110px 1fr">
          <div></div>
          <button
            type="submit"
            class="min-w-[120px] h-[30px] bg-[#F5F7FA] border border-[#0B1F40] text-ink text-[13px] font-semibold cursor-pointer px-5 inline-flex items-center justify-center gap-2 hover:bg-white hover:border-accent-2 disabled:text-ink-mute disabled:cursor-wait"
            :disabled="auth.loginBusy"
          >
            <template v-if="auth.loginBusy">
              <span class="inline-block w-3 h-3 border-2 border-[#14305A]/25 border-t-[#14305A] animate-spin"></span>
              Signing in…
            </template>
            <template v-else>Sign In</template>
          </button>
        </div>
      </div>
    </form>
  </div>
</template>
