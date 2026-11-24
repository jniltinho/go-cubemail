<script setup lang="ts">
/**
 * @component AppBar
 * @description The application's top navigation bar. Displays the brand,
 * tabs for view navigation (Mail, Contacts, Calendar), global search bar,
 * active user email, and logout controls.
 */

import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'
import BrandMark from './BrandMark.vue'
import Icon from './Icon.vue'

/** Authentication store instance */
const auth = useAuthStore()
/** Mail and UI navigation store instance */
const mail = useMailStore()

/** Navigation tabs metadata */
const TABS = [
  { id: 'mail', label: 'Mail', icon: 'mail' },
  { id: 'contacts', label: 'Contacts', icon: 'users' },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'settings', label: 'Settings', icon: 'settings' },
]
</script>

<template>
  <div
    class="on-header h-11 flex items-stretch bg-header text-on-header border-b border-header-line select-none flex-shrink-0"
  >
    <!-- Brand -->
    <div
      class="flex items-center gap-2.5 px-4 font-bold text-[14px] tracking-[0.3px] bg-header-brand border-r border-header-line min-w-[220px]"
    >
      <div class="w-7 h-7 bg-on-header text-header-brand grid place-items-center">
        <BrandMark :size="22" />
      </div>
      <div><b>Web</b>mail</div>
    </div>

    <!-- Navigation tabs -->
    <nav class="flex items-stretch">
      <button
        v-for="tab in TABS"
        :key="tab.id"
        type="button"
        :class="[
          'flex items-center gap-1.5 px-4 text-[12.5px] cursor-pointer border-r border-header-line',
          mail.view === tab.id
            ? 'tab-on'
            : 'bg-transparent text-on-header-muted hover:bg-header-hover hover:text-on-header',
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
    <div class="flex items-center gap-2.5 px-4 border-l border-header-line">
      <span class="text-on-header text-[12.5px]">{{ auth.currentUser.email }}</span>
      <button
        type="button"
        class="inline-flex items-center justify-center w-7 h-[26px] bg-transparent border border-header-control-line text-on-header-muted hover:bg-header-hover hover:text-on-header hover:border-tab-mark rounded-control cursor-pointer"
        title="Logout"
        @click="auth.handleLogout()"
      >
        <Icon name="log-out" :size="14" />
      </button>
    </div>
  </div>
</template>
