/**
 * @file toast.ts
 * @description Pinia store to manage a queue of floating temporary status toast notifications
 * in the UI for action feedbaks (success, error, warning, info).
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Toast } from '../types'

/**
 * Internal sequential counter for generating unique toast notification IDs.
 */
let _nextId = 0

/**
 * Global toast notifications queue store (`useToastStore`).
 */
export const useToastStore = defineStore('toast', () => {
  /** Active queue list of toast notifications shown on screen */
  const toasts = ref<Toast[]>([])

  /**
   * Appends a new notification to the active queue list and programs its
   * automatic dismissal timer.
   * 
   * @param message - The notification message text.
   * @param type - The feedback type status of the message ('info', 'success', 'error', 'warning').
   * @param duration - Time to display in milliseconds. If <= 0, the toast stays until dismissed.
   * @returns The generated unique numeric ID for the notification toast.
   */
  function add(message: string, type: Toast['type'] = 'info', duration = 4000): number {
    const id = ++_nextId
    toasts.value.push({ id, message, type })
    if (duration > 0) setTimeout(() => remove(id), duration)
    return id
  }

  /**
   * Dismisses and removes a notification toast from the queue by its unique ID.
   * 
   * @param id - The unique ID of the notification to remove.
   */
  function remove(id: number): void {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  /** Shortcut helper to dispatch success notification toasts */
  const success = (msg: string, duration?: number) => add(msg, 'success', duration)
  /** Shortcut helper to dispatch error notification toasts */
  const error   = (msg: string, duration?: number) => add(msg, 'error',   duration)
  /** Shortcut helper to dispatch warning notification toasts */
  const warning = (msg: string, duration?: number) => add(msg, 'warning', duration)
  /** Shortcut helper to dispatch info notification toasts */
  const info    = (msg: string, duration?: number) => add(msg, 'info',    duration)

  return { toasts, add, remove, success, error, warning, info }
})
