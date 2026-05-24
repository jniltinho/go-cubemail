<script setup lang="ts">
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { useMailStore } from './stores/mail'

let evtSource = null

function startSSE(mail, auth) {
  evtSource?.close()
  evtSource = new EventSource('/api/v1/events')
  evtSource.addEventListener('new-mail', () => {
    mail.fetchFolderMessages('inbox')
    if (mail.folder !== 'inbox') mail.fetchFolderMessages(mail.folder)
  })
  evtSource.onerror = () => {
    evtSource.close()
    evtSource = null
    setTimeout(() => {
      if (auth.isAuthenticated) startSSE(mail, auth)
    }, 30_000)
  }
}
import LoginView     from './components/LoginView.vue'
import AppBar        from './components/AppBar.vue'
import AppToolbar    from './components/AppToolbar.vue'
import AppSidebar    from './components/AppSidebar.vue'
import MailList      from './components/MailList.vue'
import ReadingPane   from './components/ReadingPane.vue'
import ContactsPane  from './components/ContactsPane.vue'
import CalendarPane  from './components/CalendarPane.vue'
import ComposerModal  from './components/ComposerModal.vue'
import SourceViewer   from './components/SourceViewer.vue'
import ContactModal   from './components/ContactModal.vue'
import DialogModal     from './components/DialogModal.vue'
import ToastContainer  from './components/ToastContainer.vue'

const auth = useAuthStore()
const mail = useMailStore()

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

watch(() => auth.isAuthenticated, (authed) => {
  if (authed) startSSE(mail, auth)
  else { evtSource?.close(); evtSource = null }
})

onMounted(async () => {
  window.addEventListener('keydown', onKey)
  await auth.checkSession()
  if (auth.isAuthenticated) await mail.loadFromApi()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  evtSource?.close()
})
</script>

<template>
  <!-- Login page -->
  <LoginView v-if="!auth.isAuthenticated" />

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
