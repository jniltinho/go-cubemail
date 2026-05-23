<script setup>
import { useMailStore } from '../stores/mail'
import { initials } from '../utils/helpers'
import Icon from './Icon.vue'

const mail = useMailStore()
</script>

<template>
  <div class="bg-white overflow-auto flex flex-col scroll-y" style="grid-column:2/4">
    <!-- Header -->
    <div class="py-3 px-4 bg-panel-2 border-b border-line flex items-center gap-3 flex-shrink-0">
      <h2 class="m-0 text-[15px] text-accent-bar font-bold">Contacts</h2>
      <span class="text-[12px] text-ink-sub">{{ mail.contacts.length }} entries · sorted by name</span>
      <div class="ml-auto flex gap-1.5">
        <button class="tbtn tbtn-primary" type="button">
          <Icon name="user-plus" :size="13" /> New contact
        </button>
        <button class="tbtn" type="button">
          <Icon name="upload" :size="13" /> Import
        </button>
      </div>
    </div>

    <!-- Contact grid -->
    <div class="grid gap-px bg-line flex-1" style="grid-template-columns:repeat(auto-fill,minmax(260px,1fr));align-content:start">
      <div
        v-for="c in mail.contacts"
        :key="c.email"
        class="bg-white py-3 px-3.5 flex gap-3 items-start"
      >
        <!-- Avatar -->
        <div class="w-[38px] h-[38px] bg-accent text-white grid place-items-center font-bold text-[14px] flex-shrink-0">
          {{ initials(c.name) }}
        </div>
        <!-- Info -->
        <div class="min-w-0 flex-1">
          <div class="text-[13px] font-semibold text-ink truncate">{{ c.name }}</div>
          <a
            href="#"
            class="text-[11.5px] mt-0.5 block text-accent-2 no-underline hover:underline truncate"
            @click.prevent="mail.composer = { to: c.email }"
          >{{ c.email }}</a>
          <div class="text-[11px] text-ink-mute mt-1 truncate">{{ c.title }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
