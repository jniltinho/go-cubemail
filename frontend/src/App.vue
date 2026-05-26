<script setup lang="ts">
/**
 * @component App
 * @description The root layout and orchestrator of the frontend application.
 * Manages global keyboard shortcuts, monitors the authentication state,
 * triggers background new-mail polling (10 min), and conditionally renders sub-views or modals.
 */

import { onMounted, onBeforeUnmount, watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { useMailStore } from './stores/mail'
import { startNewMailPolling, stopNewMailPolling } from './utils/sse'
import LoginView     from './components/LoginView.vue'
import AppBar        from './components/AppBar.vue'
import AppToolbar    from './components/AppToolbar.vue'
import AppSidebar    from './components/AppSidebar.vue'
import MailList      from './components/MailList.vue'
import ReadingPane   from './components/ReadingPane.vue'
import ContactsPane  from './components/ContactsPane.vue'
import CalendarPane  from './components/CalendarPane.vue'
import ComposerModal from './components/ComposerModal.vue'
import SourceViewer  from './components/SourceViewer.vue'
import ContactModal  from './components/ContactModal.vue'
import DialogModal   from './components/DialogModal.vue'
import ToastContainer from './components/ToastContainer.vue'
import SpinnerIcon   from './components/SpinnerIcon.vue'

/** Authentication state store instance */
const auth = useAuthStore()
/** Mail and application state store instance */
const mail = useMailStore()

/**
 * Handles global keydown shortcuts when input fields are not focused and
 * the mail composer is closed. Supports Vim-like J/K keys for navigation,
 * R for reply, E for archiving, #/Delete for deleting, and C for composing.
 * 
 * @param e - The keyboard event payload.
 */
function onKey(e) {
  if (['INPUT', 'TEXTAREA'].includes(e.target.tagName) || mail.composer !== null) return
  const list = mail.visibleMails
  const i = list.findIndex(m => m.id === mail.selectedId)
  if      (e.key === 'j')                      mail.selectedId = list[Math.min(i + 1, list.length - 1)]?.id ?? mail.selectedId
  else if (e.key === 'k')                      mail.selectedId = list[Math.max(i - 1, 0)]?.id ?? mail.selectedId
  else if (e.key === 'r')                      mail.reply()
  else if (e.key === 'e')                      mail.archiveMail()
  else if (e.key === '#' || e.key === 'Delete') mail.deleteMail()
  else if (e.key === 'c')                      mail.compose()
}

/**
 * Watches the user authentication state to establish or tear down
 * the background new-mail polling connection and load inbox data from the API.
 */
watch(() => auth.isAuthenticated, async (authed) => {
  if (authed) {
    startNewMailPolling(mail, auth)
    await mail.loadFromApi()
  } else {
    stopNewMailPolling()
  }
})

/**
 * Registers global keyboard listeners and initiates the session checks
 * when the component is mounted.
 */
onMounted(async () => {
  window.addEventListener('keydown', onKey)
  await auth.checkSession()
})

/**
 * Removes the global keyboard event listener and cleans up polling
 * timers before the component gets unmounted.
 */
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  stopNewMailPolling()
})
</script>

<template>
  <!-- Login page -->
  <LoginView v-if="!auth.isAuthenticated" />

  <!-- Loading screen -->
  <div v-else-if="mail.loading" class="flex flex-col h-full items-center justify-center bg-gray-50 gap-3">
    <SpinnerIcon />
    <span class="text-sm text-gray-500">Loading mailbox…</span>
  </div>

  <!-- Main app -->
  <div v-else class="flex flex-col h-full">
    <AppBar />
    <AppToolbar />

    <!-- 3-column layout -->
    <div
      class="grid flex-1 bg-app-bg min-h-0"
      style="grid-template-columns: 220px 380px 1fr"
    >
      <AppSidebar v-if="mail.view === 'mail' || mail.view === 'contacts' || mail.view === 'calendar'" />

      <template v-if="mail.view === 'mail'">
        <MailList />
        <ReadingPane />
      </template>
      <ContactsPane  v-else-if="mail.view === 'contacts'" style="grid-column:2/4" />
      <CalendarPane  v-else-if="mail.view === 'calendar'" style="grid-column:2/4" />
    </div>
  </div>

  <!-- Modals (outside layout flow) -->
  <ComposerModal v-if="mail.composer !== null" :prefill="mail.composer" @close="mail.closeComposer()" />
  <SourceViewer  v-if="mail.sourceMail" />
  <ContactModal  v-if="mail.contactModal" />
  <DialogModal />
  <ToastContainer />
</template>
