<script setup lang="ts">
import { ref } from 'vue'
import { useMailStore } from '../stores/mail'
import { initials } from '../utils/helpers'
import Icon from './Icon.vue'

const mail = useMailStore()
const fileInputRef = ref<HTMLInputElement | null>(null)
const importing = ref(false)

function openImport() { fileInputRef.value?.click() }

async function onFileSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  importing.value = true
  await mail.importContacts(file)
  importing.value = false
  ;(e.target as HTMLInputElement).value = ''
}

function exportContacts() {
  window.location.href = '/api/v1/contacts/export'
}
</script>

<template>
  <div class="bg-white overflow-auto flex flex-col scroll-y">
    <!-- Header -->
    <div class="h-10 px-3 bg-panel-2 border-b border-line flex items-center gap-3 flex-shrink-0">
      <h2 class="m-0 text-[15px] text-accent-bar font-bold">Contacts</h2>
      <span class="text-[12px] text-ink-sub">{{ mail.contacts.length }} entries · sorted by name</span>
      <div class="ml-auto flex gap-1.5">
        <button class="tbtn tbtn-primary" type="button" @click="mail.openContactModal()">
          <Icon name="user-plus" :size="13" /> New contact
        </button>
        <button class="tbtn" type="button" :disabled="importing" @click="openImport">
          <Icon :name="importing ? 'loader-2' : 'upload'" :size="13" :class="{ 'animate-spin': importing }" />
          {{ importing ? 'Importing…' : 'Import' }}
        </button>
        <button class="tbtn" type="button" :disabled="!mail.contacts.length" @click="exportContacts">
          <Icon name="download" :size="13" /> Export
        </button>
        <input ref="fileInputRef" type="file" accept=".csv,.vcf,.vcard" class="hidden" @change="onFileSelected" />
      </div>
    </div>

    <!-- Contact grid -->
    <div class="grid bg-white border-t border-l border-line" style="grid-template-columns:repeat(auto-fill,minmax(260px,1fr));align-content:start">
      <div
        v-for="c in mail.contacts"
        :key="c.email"
        class="group relative bg-white py-3 px-3.5 flex gap-3 items-start hover:bg-[#F5F7FA] border-b border-r border-line"
      >
        <!-- Edit button (appears on hover) -->
        <button
          class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 w-[24px] h-[24px] grid place-items-center text-ink-sub hover:text-accent hover:bg-[#E8EEF7] transition-opacity"
          type="button"
          title="Edit contact"
          @click="mail.openEditContact(c)"
        >
          <Icon name="pencil" :size="13" />
        </button>

        <!-- Avatar -->
        <div class="w-[38px] h-[38px] bg-accent text-white grid place-items-center font-bold text-[14px] flex-shrink-0">
          {{ initials(c.name) }}
        </div>
        <!-- Info -->
        <div class="min-w-0 flex-1 pr-6">
          <div class="text-[13px] font-semibold text-ink truncate">{{ c.name }}</div>
          <a
            href="#"
            class="text-[11.5px] mt-0.5 block text-accent-2 no-underline hover:underline truncate"
            @click.prevent="mail.composer = { to: c.email }"
          >{{ c.email }}</a>
          <div class="text-[11px] text-ink-mute mt-1 truncate">
            {{ [c.title, c.company].filter(Boolean).join(' · ') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
