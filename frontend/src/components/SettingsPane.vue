<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import Icon from './Icon.vue'
import { useMailStore } from '../stores/mail'
import type { Identity, UserSettings } from '../types'

const mail = useMailStore()

type Tab = 'identities' | 'preferences'
const activeTab = ref<Tab>('identities')

// ── Identity editor state ──────────────────────────────────────────────────
const identityForm = reactive<Identity>({
  display_name: '', email: '', reply_to: '', signature: '', is_default: false,
})
const editingId   = ref<number | null>(null)
const identityOpen = ref(false)
const savingId    = ref(false)

function openNewIdentity() {
  Object.assign(identityForm, { display_name: '', email: '', reply_to: '', signature: '', is_default: false })
  editingId.value = null
  identityOpen.value = true
}

function openEditIdentity(id: Identity) {
  Object.assign(identityForm, {
    display_name: id.display_name,
    email:        id.email,
    reply_to:     id.reply_to ?? '',
    signature:    id.signature ?? '',
    is_default:   id.is_default ?? false,
  })
  editingId.value = id.id ?? null
  identityOpen.value = true
}

async function saveIdentity() {
  if (!identityForm.email.trim()) return
  savingId.value = true
  try {
    const payload = {
      display_name: identityForm.display_name,
      email:        identityForm.email,
      reply_to:     identityForm.reply_to,
      signature:    identityForm.signature,
      is_default:   identityForm.is_default,
    }
    if (editingId.value) {
      await mail.updateIdentity(editingId.value, payload)
    } else {
      await mail.createIdentity(payload)
    }
    identityOpen.value = false
  } catch { /* toast shown by action */ }
  finally { savingId.value = false }
}

// ── Preferences form state ─────────────────────────────────────────────────
const prefs = reactive<UserSettings>({
  rows_per_page: 50,
  timezone:      'UTC',
  compose_html:  true,
  date_format:   '02/01/2006 15:04',
  signature_pos: 'below',
  oof_enabled:   false,
  oof_message:   '',
})
const savingPrefs = ref(false)

watch(() => mail.settings, (s) => {
  if (s) Object.assign(prefs, s)
}, { immediate: true })

const TIMEZONES = [
  'UTC','America/Sao_Paulo','America/New_York','America/Chicago','America/Los_Angeles',
  'America/Toronto','America/Buenos_Aires','America/Lima','America/Bogota',
  'Europe/London','Europe/Paris','Europe/Berlin','Europe/Madrid','Europe/Rome',
  'Europe/Lisbon','Europe/Moscow','Asia/Tokyo','Asia/Shanghai','Asia/Kolkata',
  'Asia/Dubai','Asia/Singapore','Australia/Sydney','Pacific/Auckland',
]

const DATE_FORMATS = [
  { value: '02/01/2006 15:04', label: 'DD/MM/YYYY HH:MM (e.g. 30/05/2026 14:30)' },
  { value: '01/02/2006 15:04', label: 'MM/DD/YYYY HH:MM (e.g. 05/30/2026 14:30)' },
  { value: '2006-01-02 15:04', label: 'YYYY-MM-DD HH:MM (e.g. 2026-05-30 14:30)' },
  { value: '02 Jan 2006 15:04', label: 'DD Mon YYYY HH:MM (e.g. 30 May 2026 14:30)' },
]

async function savePrefs() {
  savingPrefs.value = true
  try { await mail.saveSettings({ ...prefs }) }
  finally { savingPrefs.value = false }
}
</script>

<template>
  <div class="bg-white flex flex-col overflow-hidden" style="grid-column:2/4">
    <!-- Header -->
    <div class="h-10 px-4 bg-panel-2 border-b border-line flex items-center gap-4 flex-shrink-0">
      <span class="font-semibold text-ink text-[13px]">Settings</span>
      <div class="flex gap-1 ml-4">
        <button
          :class="['tbtn text-[12px]', activeTab==='identities' ? 'tbtn-primary' : '']"
          @click="activeTab='identities'"
        >Identities</button>
        <button
          :class="['tbtn text-[12px]', activeTab==='preferences' ? 'tbtn-primary' : '']"
          @click="activeTab='preferences'"
        >Preferences</button>
      </div>
    </div>

    <!-- ── IDENTITIES TAB ───────────────────────────────────────────────── -->
    <div v-if="activeTab==='identities'" class="flex-1 overflow-y-auto p-5">
      <div class="max-w-2xl">
        <div class="flex items-center justify-between mb-4">
          <p class="text-[13px] text-ink-sub">
            Identities define your "From:" name and email address when composing messages.
          </p>
          <button class="tbtn tbtn-primary text-[12px]" @click="openNewIdentity()">
            <Icon name="plus" :size="12" class="mr-1" /> Add Identity
          </button>
        </div>

        <!-- Identity list -->
        <div class="border border-line rounded overflow-hidden">
          <div v-if="mail.identities.length === 0" class="px-4 py-6 text-center text-[13px] text-ink-sub italic">
            No identities yet. Add one to customise your "From:" name.
          </div>
          <div
            v-for="id in mail.identities"
            :key="id.id"
            class="flex items-center gap-3 px-4 py-3 border-b border-line last:border-b-0 hover:bg-accent-soft/40"
          >
            <!-- Default star -->
            <button
              class="flex-shrink-0"
              :title="id.is_default ? 'Default identity' : 'Set as default'"
              @click="id.id && mail.setDefaultIdentity(id.id)"
            >
              <Icon
                :name="id.is_default ? 'star' : 'star'"
                :size="14"
                :class="id.is_default ? 'text-yellow-400 fill-yellow-400' : 'text-ink-sub'"
              />
            </button>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <div class="text-[13px] font-medium text-ink truncate">
                {{ id.display_name || id.email }}
                <span v-if="id.display_name" class="text-ink-sub font-normal ml-1">&lt;{{ id.email }}&gt;</span>
              </div>
              <div v-if="id.reply_to" class="text-[11px] text-ink-sub truncate">Reply-To: {{ id.reply_to }}</div>
              <div v-if="id.is_default" class="text-[11px] text-accent font-semibold mt-0.5">Default</div>
            </div>

            <!-- Actions -->
            <div class="flex gap-1 flex-shrink-0">
              <button class="tbtn text-[11px] py-0.5" @click="openEditIdentity(id)">
                <Icon name="pencil" :size="12" />
              </button>
              <button
                class="tbtn text-[11px] py-0.5 text-red-500 hover:bg-red-50"
                :disabled="id.is_default && mail.identities.length === 1"
                @click="id.id && mail.deleteIdentity(id.id)"
              >
                <Icon name="trash-2" :size="12" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── PREFERENCES TAB ─────────────────────────────────────────────── -->
    <div v-else class="flex-1 overflow-y-auto p-5">
      <div class="max-w-lg space-y-5">

        <!-- Rows per page -->
        <div>
          <label class="block text-[12px] font-semibold text-ink mb-1">Messages per page</label>
          <select v-model.number="prefs.rows_per_page"
            class="border border-line rounded px-2.5 py-1.5 text-[13px] w-full focus:outline-none focus:ring-1 focus:ring-accent">
            <option :value="25">25</option>
            <option :value="50">50</option>
            <option :value="100">100</option>
            <option :value="200">200</option>
          </select>
        </div>

        <!-- Timezone -->
        <div>
          <label class="block text-[12px] font-semibold text-ink mb-1">Timezone</label>
          <select v-model="prefs.timezone"
            class="border border-line rounded px-2.5 py-1.5 text-[13px] w-full focus:outline-none focus:ring-1 focus:ring-accent">
            <option v-for="tz in TIMEZONES" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>

        <!-- Date format -->
        <div>
          <label class="block text-[12px] font-semibold text-ink mb-1">Date format</label>
          <select v-model="prefs.date_format"
            class="border border-line rounded px-2.5 py-1.5 text-[13px] w-full focus:outline-none focus:ring-1 focus:ring-accent">
            <option v-for="df in DATE_FORMATS" :key="df.value" :value="df.value">{{ df.label }}</option>
          </select>
        </div>

        <!-- Compose HTML -->
        <label class="flex items-center gap-2 cursor-pointer text-[13px]">
          <input type="checkbox" v-model="prefs.compose_html" class="checkbox" />
          Compose messages in HTML (rich text)
        </label>

        <!-- Signature position -->
        <div>
          <label class="block text-[12px] font-semibold text-ink mb-1">Signature position</label>
          <div class="flex gap-3">
            <label class="flex items-center gap-1.5 text-[13px] cursor-pointer">
              <input type="radio" v-model="prefs.signature_pos" value="below" /> Below reply text
            </label>
            <label class="flex items-center gap-1.5 text-[13px] cursor-pointer">
              <input type="radio" v-model="prefs.signature_pos" value="above" /> Above reply text
            </label>
          </div>
        </div>

        <!-- OOF -->
        <div class="border border-line rounded p-4 space-y-3">
          <label class="flex items-center gap-2 cursor-pointer text-[13px] font-semibold">
            <input type="checkbox" v-model="prefs.oof_enabled" class="checkbox" />
            Out of Office auto-reply
          </label>
          <div v-if="prefs.oof_enabled">
            <label class="block text-[12px] text-ink-sub mb-1">Auto-reply message</label>
            <textarea
              v-model="prefs.oof_message"
              rows="3"
              class="w-full border border-line rounded px-2.5 py-1.5 text-[13px] resize-none focus:outline-none focus:ring-1 focus:ring-accent"
              placeholder="I'm currently out of office…"
            />
          </div>
        </div>

        <!-- Save -->
        <div class="pt-2">
          <button
            class="tbtn tbtn-primary"
            :disabled="savingPrefs"
            @click="savePrefs"
          >
            <Icon v-if="savingPrefs" name="loader-circle" :size="13" class="animate-spin mr-1" />
            Save Preferences
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Identity editor modal -->
  <div
    v-if="identityOpen"
    class="fixed inset-0 z-50 flex items-center justify-center"
    @click.self="identityOpen = false"
  >
    <div class="absolute inset-0 bg-black/40" />
    <div class="relative bg-white rounded-lg shadow-xl w-full max-w-md mx-4 z-10">
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-line">
        <span class="font-semibold text-ink text-[13px]">
          {{ editingId ? 'Edit Identity' : 'New Identity' }}
        </span>
        <button class="tbtn p-1" @click="identityOpen = false">
          <Icon name="x" :size="14" />
        </button>
      </div>
      <div class="px-5 py-4 space-y-3">

        <div>
          <label class="block text-[12px] text-ink-sub mb-1">Display name</label>
          <input v-model="identityForm.display_name"
            class="w-full border border-line rounded px-2.5 py-1.5 text-[13px] focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="Your Name" autofocus />
        </div>

        <div>
          <label class="block text-[12px] text-ink-sub mb-1">Email address *</label>
          <input v-model="identityForm.email" type="email"
            class="w-full border border-line rounded px-2.5 py-1.5 text-[13px] focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="you@example.com" />
        </div>

        <div>
          <label class="block text-[12px] text-ink-sub mb-1">Reply-To (optional)</label>
          <input v-model="identityForm.reply_to" type="email"
            class="w-full border border-line rounded px-2.5 py-1.5 text-[13px] focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="replies@example.com" />
        </div>

        <div>
          <label class="block text-[12px] text-ink-sub mb-1">Signature</label>
          <textarea v-model="identityForm.signature" rows="4"
            class="w-full border border-line rounded px-2.5 py-1.5 text-[13px] resize-none focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="-- &#10;Your signature" />
        </div>

        <label class="flex items-center gap-2 text-[13px] cursor-pointer">
          <input type="checkbox" v-model="identityForm.is_default" class="checkbox" />
          Set as default identity
        </label>
      </div>
      <div class="flex justify-end gap-2 px-5 py-3 border-t border-line">
        <button class="tbtn" @click="identityOpen = false">Cancel</button>
        <button
          class="tbtn tbtn-primary"
          :disabled="savingId || !identityForm.email.trim()"
          @click="saveIdentity"
        >
          <Icon v-if="savingId" name="loader-circle" :size="13" class="animate-spin mr-1" />
          {{ editingId ? 'Save' : 'Create' }}
        </button>
      </div>
    </div>
  </div>
</template>
