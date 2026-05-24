/**
 * @file dialog.ts
 * @description Pinia store to manage interactive UI dialog prompts (alert, confirm, prompt)
 * in the frontend, substituting native browser synchronous modal methods.
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DialogState } from '../types'

/**
 * Global interactive dialog controller store (`useDialogStore`).
 */
export const useDialogStore = defineStore('dialog', () => {
  /** Reactive state of the active dialog box, or null if none is open */
  const state = ref<DialogState | null>(null)

  /**
   * Internal method to display a modal popup dialog, returning a Promise
   * resolved when the user selects an option.
   * 
   * @param options - The dialog parameters excluding callback actions.
   * @returns Promise containing the user response selection.
   */
  function _show(options: Omit<DialogState, 'resolve'>): Promise<string | boolean | null | undefined> {
    return new Promise(resolve => {
      state.value = { ...options, resolve }
    })
  }

  /**
   * Opens a text input prompt dialog modal.
   * 
   * @param message - The question or instruction text.
   * @param defaultValue - Pre-filled value for the text field.
   * @returns Promise containing input value string, or null/false if dismissed.
   */
  function prompt(message: string, defaultValue = ''): Promise<string | boolean | null | undefined> {
    return _show({ type: 'prompt', message, defaultValue })
  }

  /**
   * Opens a binary confirmation dialog modal (with OK/Cancel choices).
   * 
   * @param message - The prompt question.
   * @returns Promise containing true for confirmation and false/null for dismissal.
   */
  function confirm(message: string): Promise<string | boolean | null | undefined> {
    return _show({ type: 'confirm', message })
  }

  /**
   * Opens a simple informational alert dialog box.
   * 
   * @param message - The information text to show.
   * @returns Promise resolved when the user dismisses the alert.
   */
  function alert(message: string): Promise<string | boolean | null | undefined> {
    return _show({ type: 'alert', message })
  }

  /**
   * Resolves the pending promise callback of the active dialog,
   * cleaning up active states to close the modal popup.
   * 
   * @param value - The response value payload returned to the promise.
   */
  function respond(value?: string | boolean | null): void {
    const resolve = state.value?.resolve
    state.value = null
    resolve?.(value)
  }

  return { state, prompt, confirm, alert, respond }
})
