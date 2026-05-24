<script setup lang="ts">
/**
 * @component DialogModal
 * @description The HTML Teleport portal component for custom modal dialogs.
 * Provides custom visual styling and keyboard controls for Alert, Confirm, and Prompt
 * modal boxes. Intercepts ESC and Enter buttons for easy navigation, autofocuses inputs,
 * and resolves the prompt promise in the Dialog store.
 */

import { ref, watch, nextTick, computed } from 'vue'
import { useDialogStore } from '../stores/dialog'

/** Dialog store instance containing layout type and response callbacks */
const dialog = useDialogStore()

/** Input text value for prompts */
const inputValue = ref('')
/** Reference to prompt text input field */
const inputRef   = ref<HTMLInputElement | null>(null)
/** Reference to OK confirm button */
const okRef      = ref<HTMLButtonElement | null>(null)

/** Computes true if active dialog is a text prompt input box */
const isPrompt  = computed(() => dialog.state?.type === 'prompt')
/** Computes true if active dialog is a binary confirmation question */
const isConfirm = computed(() => dialog.state?.type === 'confirm')
/** Computes true if active dialog is a simple alert notification box */
const isAlert   = computed(() => dialog.state?.type === 'alert')

/**
 * Monitors the dialog store state, setting default values and autofocusing 
 * either the text input field or the default action button upon render.
 */
watch(() => dialog.state, async (s) => {
  if (!s) return
  inputValue.value = s.defaultValue ?? ''
  await nextTick()
  if (s.type === 'prompt') inputRef.value?.focus()
  else okRef.value?.focus()
})

/** Resolves the active dialog with positive value (true, input text, or void) */
function onOk() {
  if (isPrompt.value)  dialog.respond(inputValue.value.trim() || null)
  else if (isConfirm.value) dialog.respond(true)
  else                      dialog.respond()
}

/** Resolves the active dialog with a negative or default cancel value (false, null, or void) */
function onCancel() {
  if (isAlert.value) dialog.respond()
  else if (isPrompt.value)  dialog.respond(null)
  else                      dialog.respond(false)
}

/**
 * Handles global keydown listening for Escape to cancel or Enter to confirm.
 * 
 * @param e - The keyboard event.
 */
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { e.preventDefault(); onCancel() }
  if (e.key === 'Enter' && !isPrompt.value) { e.preventDefault(); onOk() }
}

/**
 * Handles keyboard events inside the text prompt input box.
 * 
 * @param e - The keyboard event.
 */
function onInputKey(e: KeyboardEvent) {
  if (e.key === 'Enter') { e.preventDefault(); onOk() }
  if (e.key === 'Escape') { e.preventDefault(); onCancel() }
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="dialog.state"
      class="dlg-backdrop"
      @keydown="onKey"
      @click.self="onCancel"
    >
      <div class="dlg-shell" role="dialog" aria-modal="true">
        <!-- Icon + message -->
        <div class="dlg-body">
          <div class="dlg-icon" :class="{ 'dlg-icon--danger': isConfirm }">
            <svg v-if="isAlert || isConfirm" width="20" height="20" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="10" r="9" stroke="currentColor" stroke-width="1.5"/>
              <path d="M10 6v5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <circle cx="10" cy="14" r=".75" fill="currentColor"/>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="10" r="9" stroke="currentColor" stroke-width="1.5"/>
              <path d="M10 6v5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <circle cx="10" cy="14" r=".75" fill="currentColor"/>
            </svg>
          </div>
          <p class="dlg-message">{{ dialog.state.message }}</p>
        </div>

        <!-- Input for prompt -->
        <div v-if="isPrompt" class="dlg-input-wrap">
          <input
            ref="inputRef"
            v-model="inputValue"
            class="dlg-input"
            type="text"
            @keydown="onInputKey"
          />
        </div>

        <!-- Buttons -->
        <div class="dlg-footer">
          <button v-if="!isAlert" class="dlg-btn dlg-btn--cancel" @click="onCancel">
            Cancel
          </button>
          <button ref="okRef" class="dlg-btn dlg-btn--ok" @click="onOk">
            OK
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.dlg-backdrop {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: grid;
  place-items: center;
  background: rgba(11, 31, 64, .40);
}

.dlg-shell {
  width: 380px;
  max-width: calc(100vw - 32px);
  background: var(--color-panel);
  border: 1px solid var(--accent-bar);
  box-shadow: 0 8px 32px rgba(11, 31, 64, .22);
  display: flex;
  flex-direction: column;
}

.dlg-body {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 20px 20px 16px;
}

.dlg-icon {
  flex-shrink: 0;
  color: var(--accent-2);
  margin-top: 1px;
}
.dlg-icon--danger {
  color: var(--color-danger);
}

.dlg-message {
  margin: 0;
  font-size: 13px;
  color: var(--color-ink);
  line-height: 1.5;
  word-break: break-word;
}

.dlg-input-wrap {
  padding: 0 20px 16px;
}

.dlg-input {
  width: 100%;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--color-line);
  background: white;
  font-size: 13px;
  color: var(--color-ink);
  outline: none;
}
.dlg-input:focus {
  border-color: var(--accent-2);
  box-shadow: 0 0 0 2px var(--accent-soft);
}

.dlg-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--color-line-soft);
  background: var(--color-panel-2);
}

.dlg-btn {
  height: 28px;
  padding: 0 16px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--color-line);
  background: white;
  color: var(--color-ink);
}
.dlg-btn:hover {
  background: var(--color-panel-2);
}
.dlg-btn--ok {
  background: var(--accent);
  border-color: var(--accent-bar);
  color: white;
}
.dlg-btn--ok:hover {
  background: var(--accent-2);
}
.dlg-btn--cancel {
  background: white;
}
</style>
