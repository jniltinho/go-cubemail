<script setup lang="ts">
import { useToastStore } from '../stores/toast'
const toast = useToastStore()
</script>

<template>
  <Teleport to="body">
    <div class="toast-stack" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast">
        <div
          v-for="t in toast.toasts"
          :key="t.id"
          class="toast"
          :class="`toast--${t.type}`"
          role="alert"
        >
          <svg class="toast-icon" width="16" height="16" viewBox="0 0 16 16" fill="none">
            <!-- success -->
            <template v-if="t.type === 'success'">
              <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
              <path d="M5 8l2 2 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </template>
            <!-- error -->
            <template v-else-if="t.type === 'error'">
              <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
              <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </template>
            <!-- warning -->
            <template v-else-if="t.type === 'warning'">
              <path d="M8 2L14.5 13H1.5L8 2z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
              <path d="M8 6v3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <circle cx="8" cy="11" r=".6" fill="currentColor"/>
            </template>
            <!-- info -->
            <template v-else>
              <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
              <path d="M8 7v4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <circle cx="8" cy="5" r=".6" fill="currentColor"/>
            </template>
          </svg>

          <span class="toast-msg">{{ t.message }}</span>

          <button class="toast-close" @click="toast.remove(t.id)" aria-label="Dismiss">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  bottom: 20px;
  right: 20px;
  z-index: 300;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 260px;
  max-width: 420px;
  padding: 10px 12px;
  background: var(--color-panel);
  border: 1px solid var(--color-line);
  border-left-width: 3px;
  box-shadow: 0 4px 16px rgba(11, 31, 64, .16);
  pointer-events: all;
  font-size: 13px;
  color: var(--color-ink);
}

.toast--success { border-left-color: var(--color-success); }
.toast--error   { border-left-color: var(--color-danger); }
.toast--warning { border-left-color: var(--color-star); }
.toast--info    { border-left-color: var(--accent-2); }

.toast-icon {
  flex-shrink: 0;
}
.toast--success .toast-icon { color: var(--color-success); }
.toast--error   .toast-icon { color: var(--color-danger); }
.toast--warning .toast-icon { color: var(--color-star); }
.toast--info    .toast-icon { color: var(--accent-2); }

.toast-msg {
  flex: 1;
  line-height: 1.4;
  word-break: break-word;
}

.toast-close {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-ink-mute);
  padding: 0;
}
.toast-close:hover { color: var(--color-ink); }

/* Transition */
.toast-enter-active,
.toast-leave-active {
  transition: opacity 220ms ease, transform 220ms ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>
