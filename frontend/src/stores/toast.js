import { defineStore } from 'pinia'
import { ref } from 'vue'

let _nextId = 0

export const useToastStore = defineStore('toast', () => {
  const toasts = ref([])

  function add(message, type = 'info', duration = 4000) {
    const id = ++_nextId
    toasts.value.push({ id, message, type })
    if (duration > 0) setTimeout(() => remove(id), duration)
    return id
  }

  function remove(id) {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  const success = (msg, duration)  => add(msg, 'success', duration)
  const error   = (msg, duration)  => add(msg, 'error',   duration)
  const warning = (msg, duration)  => add(msg, 'warning', duration)
  const info    = (msg, duration)  => add(msg, 'info',    duration)

  return { toasts, add, remove, success, error, warning, info }
})
