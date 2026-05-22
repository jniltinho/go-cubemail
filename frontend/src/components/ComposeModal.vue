<template>
  <Dialog
    v-model:visible="mailStore.composeVisible"
    :modal="true"
    :closable="false"
    :draggable="true"
    :dismissableMask="false"
    class="compose-dialog"
    header="Nova Mensagem"
    style="width: 1530px; height: 630px; max-width: 95vw; max-height: 95vh; display: flex; flex-direction: column;"
    id="compose-dialog-root"
  >
    <template #header>
      <div class="compose-header-bar">
        <h3 class="compose-title">
          <i class="pi pi-pencil"></i>
          {{ modeLabel }}
        </h3>
        <div class="compose-header-actions">
          <Button
            icon="pi pi-times"
            class="close-dialog-btn"
            v-tooltip="'Fechar'"
            @click="mailStore.composeVisible = false"
            id="close-compose-dialog-btn"
          />
        </div>
      </div>
    </template>

    <div class="compose-content">
      <!-- Alerts -->
      <div v-if="error" class="compose-alert error" id="compose-error-alert">
        <i class="pi pi-exclamation-triangle"></i> {{ error }}
      </div>
      <div v-if="success" class="compose-alert success" id="compose-success-alert">
        <i class="pi pi-check-circle"></i> Email enviado com sucesso!
      </div>

      <!-- Recipients -->
      <div class="compose-fields-container">
        <div class="compose-field">
          <label class="compose-label" for="dialog-compose-to">Para</label>
          <InputText
            id="dialog-compose-to"
            v-model="mailStore.composeData.to"
            placeholder="email@exemplo.com, ..."
            class="w-full compose-input"
          />
        </div>
        <div class="compose-field">
          <label class="compose-label" for="dialog-compose-cc">Cc</label>
          <InputText
            id="dialog-compose-cc"
            v-model="mailStore.composeData.cc"
            placeholder="email@exemplo.com"
            class="w-full compose-input"
          />
        </div>
        <div class="compose-field">
          <label class="compose-label" for="dialog-compose-subject">Assunto</label>
          <InputText
            id="dialog-compose-subject"
            v-model="mailStore.composeData.subject"
            placeholder="Assunto do email"
            class="w-full compose-input"
          />
        </div>
      </div>

      <!-- Body / Editor -->
      <div class="compose-body">
        <Editor
          id="dialog-compose-editor"
          v-model="mailStore.composeData.body_html"
          :modules="editorModules"
          editorStyle="height: 220px; min-height: 180px; max-height: 250px; background: var(--color-surface); color: var(--color-text); border: none; font-family: inherit; font-size: 14px;"
        >
          <template #toolbar>
            <span class="ql-formats">
              <button class="ql-bold"></button>
              <button class="ql-italic"></button>
              <button class="ql-underline"></button>
              <button class="ql-strike"></button>
            </span>
            <span class="ql-formats">
              <button class="ql-list" value="ordered"></button>
              <button class="ql-list" value="bullet"></button>
            </span>
            <span class="ql-formats">
              <button class="ql-link"></button>
            </span>
          </template>
        </Editor>
      </div>

      <!-- Attachments -->
      <div class="compose-attachments">
        <FileUpload
          id="dialog-compose-attachments"
          name="attachments"
          :multiple="true"
          :maxFileSize="26214400"
          chooseLabel="Anexar arquivos"
          :showUploadButton="false"
          :showCancelButton="false"
          @select="onFilesSelected"
          class="attach-upload"
        />
      </div>
    </div>

    <template #footer>
      <div class="compose-footer-bar">
        <Button
          label="Descartar"
          icon="pi pi-trash"
          class="toolbar-btn delete-btn-empty"
          @click="mailStore.composeVisible = false"
          id="dialog-discard-btn"
        />
        <Button
          icon="pi pi-send"
          label="Enviar"
          :loading="sending"
          class="toolbar-btn primary-btn send-btn"
          @click="send"
          id="dialog-send-btn"
        />
      </div>
    </template>
  </Dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Editor from 'primevue/editor'
import FileUpload from 'primevue/fileupload'
import { useToast } from 'primevue/usetoast'
import { sendEmail } from '@/api/compose.js'
import { useFoldersStore } from '@/stores/folders.js'
import { useMailStore } from '@/stores/mail.js'

const toast = useToast()
const foldersStore = useFoldersStore()
const mailStore = useMailStore()

const sending = ref(false)
const error = ref(null)
const success = ref(false)
const attachedFiles = ref([])
const editorModules = {}

const modeLabel = computed(() => {
  switch (mailStore.composeMode) {
    case 'reply': return 'Responder'
    case 'forward': return 'Encaminhar'
    default: return 'Nova Mensagem'
  }
})

function onFilesSelected(event) {
  attachedFiles.value = event.files || []
}

async function send() {
  if (!mailStore.composeData.to.trim()) {
    error.value = 'Informe ao menos um destinatário.'
    return
  }
  sending.value = true
  error.value = null
  success.value = false

  const fd = new FormData()
  fd.append('to', mailStore.composeData.to)
  fd.append('cc', mailStore.composeData.cc || '')
  fd.append('bcc', mailStore.composeData.bcc || '')
  fd.append('subject', mailStore.composeData.subject || '')
  fd.append('body_html', mailStore.composeData.body_html || '')
  fd.append('body_plain', mailStore.composeData.body_html.replace(/<[^>]+>/g, '') || '')
  for (const file of attachedFiles.value) {
    fd.append('attachments', file)
  }

  try {
    await sendEmail(fd)
    success.value = true
    toast.add({ severity: 'success', summary: 'Email enviado!', life: 3000 })
    setTimeout(() => {
      mailStore.composeVisible = false
      foldersStore.loadUnreadCounts()
      // Refresh mailbox if we are currently in "Sent" folder
      if (mailStore.currentMailbox?.toLowerCase() === 'sent') {
        mailStore.loadMessages('Sent', mailStore.page)
      }
      // Reset form variables
      error.value = null
      success.value = false
      attachedFiles.value = []
    }, 1200)
  } catch (err) {
    error.value = err.response?.data?.error || 'Erro ao enviar email.'
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
/* ── Sleek Flat Modern Dialog Styling ── */
:deep(.p-dialog),
:deep(.p-dialog-header),
:deep(.p-dialog-content),
:deep(.p-dialog-footer),
:deep(.p-inputtext),
:deep(.p-editor-container),
:deep(.p-editor-content),
:deep(.p-fileupload),
:deep(.p-button),
:deep(.p-fileupload-choose) {
  border-radius: 0 !important;
}

:deep(.p-dialog) {
  border: 1px solid var(--color-border) !important;
  box-shadow: var(--shadow-lg) !important;
  background-color: var(--color-surface) !important;
}

:deep(.p-dialog-header) {
  background-color: var(--color-surface-2) !important;
  border-bottom: 1px solid var(--color-border-light) !important;
  padding: 0.75rem 1.25rem !important;
}

:deep(.p-dialog-content) {
  padding: 0 !important;
  background-color: var(--color-surface) !important;
}

:deep(.p-dialog-footer) {
  background-color: var(--color-surface-2) !important;
  border-top: 1px solid var(--color-border-light) !important;
  padding: 0.75rem 1.25rem !important;
}

.compose-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.compose-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  letter-spacing: -0.015em;
}

.compose-header-actions {
  display: flex;
  align-items: center;
}

.close-dialog-btn {
  background: transparent !important;
  border: none !important;
  color: var(--color-text-muted) !important;
  width: 32px !important;
  height: 32px !important;
  font-size: 0.9375rem !important;
  padding: 0 !important;
  border-radius: 0 !important;
  box-shadow: none !important;
}

.close-dialog-btn:hover {
  background-color: rgba(0, 0, 0, 0.05) !important;
  color: var(--color-text) !important;
}

.compose-content {
  display: flex;
  flex-direction: column;
  background-color: var(--color-surface);
}

.compose-alert {
  margin: 0.75rem 1.25rem;
  padding: 0.625rem 0.875rem;
  border-radius: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: bold;
}

.compose-alert.error {
  background: #fdf2f2;
  border: 1px solid #f5b8b8;
  color: #9b1c1c;
}

.compose-alert.success {
  background: #f3faf5;
  border: 1px solid #b7e4c7;
  color: #1b4332;
}

.compose-fields-container {
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid var(--color-border-light);
}

.compose-field {
  display: flex;
  align-items: center;
  padding: 0.5rem 1.25rem;
  gap: 0.75rem;
  border-bottom: 1px solid rgba(226, 232, 240, 0.5);
}

.compose-field:last-child {
  border-bottom: none;
}

.compose-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-muted);
  min-width: 50px;
  text-align: right;
}

.compose-input,
.compose-input :deep(.p-inputtext) {
  background-color: var(--color-surface) !important;
  color: var(--color-text) !important;
  border: 1px solid var(--color-border-light) !important;
  font-size: 0.875rem !important;
  padding: 0.375rem 0.75rem !important;
  border-radius: 0 !important;
  box-shadow: none !important;
  transition: border-color var(--transition) !important;
  height: 32px !important;
}

.compose-input:focus,
.compose-input:focus-within,
.compose-input :deep(.p-inputtext:focus) {
  border-color: var(--color-accent) !important;
}

.w-full {
  width: 100%;
}

.compose-body {
  display: flex;
  flex-direction: column;
  background-color: var(--color-surface);
}

.compose-body :deep(.p-editor-toolbar) {
  background-color: var(--color-surface-2) !important;
  border: none !important;
  border-bottom: 1px solid var(--color-border-light) !important;
}

.compose-body :deep(.p-editor-content) {
  border: none !important;
}

.compose-attachments {
  border-top: 1px solid var(--color-border-light);
  padding: 0.75rem 1.25rem;
  background-color: var(--color-surface-2);
}

.compose-attachments :deep(.p-fileupload-choose) {
  background: var(--color-surface) !important;
  border: 1px solid var(--color-border-light) !important;
  color: var(--color-text) !important;
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  box-shadow: none !important;
  height: 32px !important;
  padding: 0.375rem 0.875rem !important;
  border-radius: 0 !important;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.compose-attachments :deep(.p-fileupload-choose:hover) {
  background: var(--color-border-light) !important;
  border-color: var(--color-border) !important;
}

.compose-attachments :deep(.p-fileupload-choose .p-button-icon) {
  font-size: 0.8125rem !important;
  color: var(--color-text-muted);
}

.compose-footer-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.toolbar-btn {
  background: var(--color-surface-2) !important;
  border: 1px solid var(--color-border-light) !important;
  color: var(--color-text) !important;
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  padding: 0.375rem 0.875rem !important;
  height: 32px !important;
  border-radius: 0 !important;
  box-shadow: none !important;
}

.toolbar-btn :deep(.p-button-icon) {
  font-size: 0.8125rem !important;
  color: var(--color-text-muted);
}

.toolbar-btn:hover {
  background: var(--color-border-light) !important;
  border-color: var(--color-border) !important;
}

.primary-btn {
  background: rgba(8, 117, 225, 0.08) !important;
  border-color: rgba(8, 117, 225, 0.2) !important;
  color: var(--color-accent) !important;
}

.primary-btn:hover {
  background: rgba(8, 117, 225, 0.15) !important;
  border-color: var(--color-accent) !important;
}

.delete-btn-empty {
  background: rgba(239, 68, 68, 0.08) !important;
  border-color: rgba(239, 68, 68, 0.2) !important;
  color: var(--color-danger) !important;
}

.delete-btn-empty:hover {
  background: rgba(239, 68, 68, 0.15) !important;
  border-color: var(--color-danger) !important;
}

.send-btn {
  height: 32px !important;
}
</style>

<style>
/* Global resets for the teleported compose dialog to guarantee 100% square corners */
.compose-dialog,
.compose-dialog *,
.p-dialog.compose-dialog,
.p-dialog.compose-dialog *,
[id^="compose-dialog-root"],
[id^="compose-dialog-root"] * {
  border-radius: 0px !important;
}
</style>
