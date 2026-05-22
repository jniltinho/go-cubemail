<template>
  <div class="app-root">
    <Toast position="top-right" />
    <ConfirmDialog />
    <ComposeModal />
    <router-view />
  </div>
</template>

<script setup>
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import ComposeModal from '@/components/ComposeModal.vue'
</script>

<style>
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

:root {
  --sidebar-width: 280px;
  --topbar-height: 56px;
  --color-bg: #f5f6f8;
  --color-surface: #ffffff;
  --color-surface-2: #f8fafc;
  --color-border: #cbd5e1;
  --color-border-light: #e2e8f0;
  --color-text: #0f172a;
  --color-text-muted: #64748b;
  --color-accent: #0875e1;
  --color-accent-hover: #065eb8;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-unread-dot: #0875e1;
  --radius-md: 0px;
  --radius-lg: 0px;
  --p-border-radius-xs: 0px !important;
  --p-border-radius-sm: 0px !important;
  --p-border-radius-md: 0px !important;
  --p-border-radius-lg: 0px !important;
  --p-border-radius-xl: 0px !important;
  --p-dialog-border-radius: 0px !important;
  --p-dialog-rounded: 0px !important;
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05);
  --transition: 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

html, body {
  height: 100%;
  background-color: var(--color-bg);
  color: var(--color-text);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}

#app {
  height: 100vh;
  overflow: hidden;
}

.app-root {
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* ── Scrollbars ── */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: #cbd5e1; border-radius: 0; }
::-webkit-scrollbar-thumb:hover { background: #94a3b8; }

/* ── PrimeVue overrides ── */
.p-button { 
  border-radius: var(--radius-md) !important; 
  font-family: inherit !important; 
  font-size: 0.875rem !important; 
  font-weight: 500 !important;
  box-shadow: var(--shadow-sm) !important;
}
.p-inputtext { 
  border-radius: var(--radius-md) !important; 
  font-family: inherit !important; 
  font-size: 0.875rem !important; 
  border-color: var(--color-border) !important;
  transition: border-color var(--transition), box-shadow var(--transition) !important;
}
.p-inputtext:focus {
  border-color: var(--color-accent) !important;
  box-shadow: 0 0 0 2px rgba(8, 117, 225, 0.2) !important;
}
.p-password input { border-radius: var(--radius-md) !important; font-family: inherit !important; }
</style>
