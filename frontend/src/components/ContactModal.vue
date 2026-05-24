<script setup lang="ts">
/**
 * @component ContactModal
 * @description The modal viewport window to add or update contact registry data.
 * Displays field inputs for name, email, job title, company name, phone, and optional notes.
 * Shows delete buttons on editing contexts, binds reactive updates, and handles form validations.
 */

import { reactive, ref, watch } from 'vue'
import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

/** Mail store instance containing contact CRUD integrations */
const mail = useMailStore()

/**
 * Returns true if the composer modal is opened in "edit contact" mode.
 */
const isEdit = () => !!mail.editingContact

/** The reactive form fields payload object */
const form = reactive({
  name:    '',
  email:   '',
  title:   '',
  company: '',
  phone:   '',
  notes:   '',
})

/**
 * Syncs the reactive form state with mail store editing fields
 * when opening the modal or choosing a different editing contact.
 */
watch(
  () => mail.editingContact,
  (c) => {
    form.name    = c?.name    ?? ''
    form.email   = c?.email   ?? ''
    form.title   = c?.title   ?? ''
    form.company = c?.company ?? ''
    form.phone   = c?.phone   ?? ''
    form.notes   = c?.notes   ?? ''
  },
  { immediate: true }
)

/** True if the contact payload save request is in progress */
const saving  = ref(false)
/** True if the contact deletion request is in progress */
const deleting = ref(false)
/** Holds validation or API error messages */
const error   = ref('')

/** Closes the modal if user clicks on the backdrop grey mask */
function backdrop(e: MouseEvent) {
  if (e.target === e.currentTarget) mail.closeContactModal()
}

/**
 * Asserts name/email inputs are not empty, and dispatches the update or save
 * payload call to the mail store contact actions. Closes the modal on success.
 */
async function submit() {
  error.value = ''
  if (!form.name.trim() || !form.email.trim()) {
    error.value = 'Name and email are required.'
    return
  }
  saving.value = true
  if (isEdit() && mail.editingContact?.id) {
    await mail.updateContact(mail.editingContact.id, { ...form })
  } else {
    await mail.saveContact({ ...form })
  }
  saving.value = false
  mail.closeContactModal()
}

/**
 * Dispatches contact deletion request on the mail store contact actions API
 * and closes the modal viewport.
 */
async function remove() {
  if (!mail.editingContact?.id) return
  deleting.value = true
  await mail.deleteContact(mail.editingContact.id)
  deleting.value = false
  mail.closeContactModal()
}
</script>

<template>
  <div class="modal-wrap" @click="backdrop">
    <div class="composer-shell" style="width:540px" role="dialog" :aria-label="isEdit() ? 'Edit Contact' : 'New Contact'">
      <!-- Header -->
      <div class="bg-accent-bar text-white py-2 px-3 flex items-center gap-2 text-[13px] font-semibold">
        <Icon :name="isEdit() ? 'user-pen' : 'user-plus'" :size="14" />
        <span>{{ isEdit() ? 'Edit Contact' : 'New Contact' }}</span>
        <button
          type="button"
          class="ml-auto flex-shrink-0 cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978]"
          @click="mail.closeContactModal()"
        >
          <Icon name="x" :size="12" />
        </button>
      </div>

      <!-- Form -->
      <div class="bg-white">
        <!-- Name row -->
        <div class="flex items-center border-b border-line px-3">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0">Name</span>
          <div class="flex items-center gap-2 flex-1 py-2">
            <div class="w-[34px] h-[34px] bg-accent-bar text-white grid place-items-center font-bold text-[13px] flex-shrink-0">
              {{ form.name ? form.name.trim().split(' ').map(w => w[0]).slice(0, 2).join('').toUpperCase() : '?' }}
            </div>
            <input
              v-model="form.name"
              type="text"
              placeholder="Full name (e.g. Helena Vargas)"
              class="flex-1 text-[13px] outline-none placeholder:text-ink-mute"
              autofocus
            />
          </div>
        </div>
        <!-- Email -->
        <div class="flex items-center border-b border-line px-3 py-2.5">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0">Email</span>
          <input
            v-model="form.email"
            type="email"
            placeholder="name@example.com"
            class="flex-1 text-[13px] outline-none placeholder:text-ink-mute"
          />
        </div>
        <!-- Title -->
        <div class="flex items-center border-b border-line px-3 py-2.5">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0">Title</span>
          <input
            v-model="form.title"
            type="text"
            placeholder="Role (e.g. Strategic Partnerships)"
            class="flex-1 text-[13px] outline-none placeholder:text-ink-mute"
          />
        </div>
        <!-- Company -->
        <div class="flex items-center border-b border-line px-3 py-2.5">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0">Company</span>
          <input
            v-model="form.company"
            type="text"
            placeholder="Organisation (e.g. Strataworks)"
            class="flex-1 text-[13px] outline-none placeholder:text-ink-mute"
          />
        </div>
        <!-- Phone -->
        <div class="flex items-center border-b border-line px-3 py-2.5">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0">Phone</span>
          <input
            v-model="form.phone"
            type="tel"
            placeholder="+351 21 000 0000"
            class="flex-1 text-[13px] outline-none placeholder:text-ink-mute"
          />
        </div>
        <!-- Notes -->
        <div class="flex items-start border-b border-line px-3 py-2.5">
          <span class="w-[80px] text-[12px] font-semibold text-ink-sub flex-shrink-0 pt-0.5">Notes</span>
          <textarea
            v-model="form.notes"
            rows="4"
            placeholder="Optional notes…"
            class="flex-1 text-[13px] outline-none placeholder:text-ink-mute resize-y"
          />
        </div>
      </div>

      <!-- Footer -->
      <div class="py-2 px-2.5 bg-panel-2 border-t border-line flex items-center gap-1.5">
        <!-- Delete (edit mode only) -->
        <button
          v-if="isEdit()"
          class="tbtn text-red-500 border-red-200 hover:bg-red-50"
          type="button"
          :disabled="deleting"
          @click="remove"
        >
          <Icon name="trash-2" :size="13" /> {{ deleting ? 'Deleting…' : 'Delete' }}
        </button>

        <template v-else>
          <Icon name="info" :size="13" class="text-ink-mute" />
          <span v-if="error" class="text-[12px] text-red-500">{{ error }}</span>
          <span v-else class="text-[12px] text-ink-sub">Required: name and email</span>
        </template>

        <span v-if="isEdit() && error" class="text-[12px] text-red-500 ml-1">{{ error }}</span>

        <div class="ml-auto flex gap-1.5">
          <button class="tbtn" type="button" @click="mail.closeContactModal()">Cancel</button>
          <button class="tbtn tbtn-primary" type="button" :disabled="saving" @click="submit">
            <Icon name="check" :size="13" /> {{ saving ? 'Saving…' : (isEdit() ? 'Save changes' : 'Save contact') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
