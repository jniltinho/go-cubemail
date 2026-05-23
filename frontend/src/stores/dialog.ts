import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DialogState } from '../types'

export const useDialogStore = defineStore('dialog', () => {
  const state = ref<DialogState | null>(null)

  function _show(options: Omit<DialogState, 'resolve'>): Promise<string | boolean | null | undefined> {
    return new Promise(resolve => {
      state.value = { ...options, resolve }
    })
  }

  function prompt(message: string, defaultValue = ''): Promise<string | boolean | null | undefined> {
    return _show({ type: 'prompt', message, defaultValue })
  }

  function confirm(message: string): Promise<string | boolean | null | undefined> {
    return _show({ type: 'confirm', message })
  }

  function alert(message: string): Promise<string | boolean | null | undefined> {
    return _show({ type: 'alert', message })
  }

  function respond(value?: string | boolean | null): void {
    const resolve = state.value?.resolve
    state.value = null
    resolve?.(value)
  }

  return { state, prompt, confirm, alert, respond }
})
