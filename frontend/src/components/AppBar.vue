<script setup lang="ts">
/**
 * @component AppBar
 * @description The application's top navigation bar. Displays the brand,
 * tabs for view navigation (Mail, Contacts, Calendar), global search bar,
 * active user email, and logout controls.
 */

import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

/** Authentication store instance */
const auth = useAuthStore()
/** Mail and UI navigation store instance */
const mail = useMailStore()

/** Navigation tabs metadata */
const TABS = [
  { id: 'mail',     label: 'Mail',     icon: 'mail' },
  { id: 'contacts', label: 'Contacts', icon: 'users' },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
]
</script>

<template>
  <div class="h-11 flex items-stretch bg-accent-bar text-white border-b border-[#0B1F40] select-none flex-shrink-0">
    <!-- Brand -->
    <div class="flex items-center gap-2.5 px-4 font-bold text-[14px] tracking-[0.3px] bg-accent border-r border-[#0B1F40] min-w-[220px]">
      <div class="w-[22px] h-[22px] bg-white text-accent font-mono font-extrabold grid place-items-center text-[13px]">W</div>
      <div><b>Web</b>mail</div>
    </div>

    <!-- Navigation tabs -->
    <nav class="flex items-stretch">
      <button
        v-for="tab in TABS"
        :key="tab.id"
        type="button"
        :class="[
          'flex items-center gap-1.5 px-4 text-[12.5px] cursor-pointer border-r border-[#0B1F40] bg-transparent',
          mail.view === tab.id
            ? 'bg-accent text-white shadow-[inset_0_-3px_0_#66A0FF]'
            : 'text-[#D5E0F2] hover:bg-[#102744] hover:text-white',
        ]"
        @click="mail.view = tab.id"
      >
        <Icon :name="tab.icon" :size="14" />
        <span>{{ tab.label }}</span>
      </button>
    </nav>

    <!-- Search -->
    <div class="flex-1 flex items-center px-3">
      <form class="search-box w-full" @submit.prevent>
        <select>
          <option>All Folders</option>
          <option>Inbox</option>
          <option>Sent</option>
          <option>From…</option>
          <option>Subject…</option>
        </select>
        <input
          type="text"
          placeholder="Search mail (from, subject, body, attachment name…)"
          :value="mail.query"
          @input="mail.query = ($event.target as HTMLInputElement).value"
        />
        <button type="submit">Search</button>
      </form>
    </div>

    <!-- User + Logout -->
    <div class="flex items-center gap-2.5 px-4 border-l border-[#0B1F40]">
      <span class="text-white text-[12.5px]">{{ auth.currentUser.email }}</span>
      <button
        type="button"
        class="inline-flex items-center justify-center w-7 h-[26px] bg-transparent border border-[#4A6FA0] text-[#D5E0F2] hover:bg-[#102744] hover:text-white hover:border-[#66A0FF] cursor-pointer"
        title="Logout"
        @click="auth.handleLogout()"
      >
        <Icon name="log-out" :size="14" />
      </button>
    </div>
  </div>
</template>
