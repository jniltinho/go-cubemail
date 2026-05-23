import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useDialogStore = defineStore('dialog', () => {
  const state = ref(null) // { type, message, defaultValue, resolve }

  function _show(options) {
    return new Promise(resolve => {
      state.value = { ...options, resolve }
    })
  }

  function prompt(message, defaultValue = '') {
    return _show({ type: 'prompt', message, defaultValue })
  }

  function confirm(message) {
    return _show({ type: 'confirm', message })
  }

  function alert(message) {
    return _show({ type: 'alert', message })
  }

  function respond(value) {
    const resolve = state.value?.resolve
    state.value = null
    resolve?.(value)
  }

  return { state, prompt, confirm, alert, respond }
})
