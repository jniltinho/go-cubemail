<template>
  <div class="mail-layout">
    <AppSidebar />

    <main class="compose-main">
      <header class="topbar">
        <Button icon="pi pi-times" label="Descartar" class="toolbar-btn delete-btn-empty" @click="router.back()" id="discard-btn" />
        <h2 class="topbar-title">
          <i class="pi pi-pencil"></i>
          {{ modeLabel }}
        </h2>
        <div class="topbar-right">
          <Button
            icon="pi pi-send"
            label="Enviar"
            :loading="sending"
            class="toolbar-btn primary-btn send-btn"
            @click="send"
            id="send-btn"
          />
        </div>
      </header>

      <div class="compose-container">
        <!-- Alert -->
        <div v-if="error" class="compose-alert error">
          <i class="pi pi-exclamation-triangle"></i> {{ error }}
        </div>
        <div v-if="success" class="compose-alert success">
          <i class="pi pi-check-circle"></i> Email enviado com sucesso!
        </div>

        <!-- Recipients -->
        <div class="compose-field">
          <label class="compose-label" for="compose-to">Para</label>
          <InputText id="compose-to" v-model="form.to" placeholder="email@exemplo.com, ..." class="w-full compose-input" />
        </div>
        <Divider class="field-divider" />
        <div class="compose-field">
          <label class="compose-label" for="compose-cc">Cc</label>
          <InputText id="compose-cc" v-model="form.cc" placeholder="email@exemplo.com" class="w-full compose-input" />
        </div>
        <Divider class="field-divider" />
        <div class="compose-field">
          <label class="compose-label" for="compose-subject">Assunto</label>
          <InputText id="compose-subject" v-model="form.subject" placeholder="Assunto do email" class="w-full compose-input" />
        </div>
        <Divider class="field-divider" />

        <!-- Body -->
        <div class="compose-body">
          <Editor
            id="compose-editor"
            v-model="form.body_html"
            :modules="editorModules"
            editorStyle="min-height: 320px; background: var(--color-surface); color: var(--color-text); border: none; font-family: inherit; font-size: 14px;"
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
            id="compose-attachments"
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
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Editor from 'primevue/editor'
import Divider from 'primevue/divider'
import FileUpload from 'primevue/fileupload'
import { useToast } from 'primevue/usetoast'
import AppSidebar from '@/components/AppSidebar.vue'
import { sendEmail } from '@/api/compose.js'
import { useFoldersStore } from '@/stores/folders.js'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const foldersStore = useFoldersStore()

const sending = ref(false)
const error = ref(null)
const success = ref(false)
const attachedFiles = ref([])
const editorModules = {}

const form = ref({
  to: '',
  cc: '',
  bcc: '',
  subject: '',
  body_html: '',
})

const modeLabel = computed(() => {
  switch (route.query.mode) {
    case 'reply': return 'Responder'
    case 'forward': return 'Encaminhar'
    default: return 'Nova Mensagem'
  }
})

function onFilesSelected(event) {
  attachedFiles.value = event.files || []
}

async function send() {
  if (!form.value.to.trim()) {
    error.value = 'Informe ao menos um destinatário.'
    return
  }
  sending.value = true
  error.value = null
  success.value = false

  const fd = new FormData()
  fd.append('to', form.value.to)
  fd.append('cc', form.value.cc || '')
  fd.append('bcc', form.value.bcc || '')
  fd.append('subject', form.value.subject || '')
  fd.append('body_html', form.value.body_html || '')
  fd.append('body_plain', form.value.body_html.replace(/<[^>]+>/g, '') || '')
  for (const file of attachedFiles.value) {
    fd.append('attachments', file)
  }

  try {
    await sendEmail(fd)
    success.value = true
    toast.add({ severity: 'success', summary: 'Email enviado!', life: 3000 })
    setTimeout(() => router.back(), 1500)
  } catch (err) {
    error.value = err.response?.data?.error || 'Erro ao enviar email.'
  } finally {
    sending.value = false
  }
}

onMounted(() => {
  foldersStore.load()
  // Pre-fill To field if coming from reply/forward
  if (route.query.to) form.value.to = route.query.to
  if (route.query.subject) form.value.subject = route.query.subject
})
</script>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: var(--color-bg);
}

.compose-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
  background-color: var(--color-surface);
}

.topbar {
  height: var(--topbar-height);
  background-color: var(--color-surface);
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  gap: 1rem;
  flex-shrink: 0;
}

.topbar-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  letter-spacing: -0.015em;
}

.topbar-right { display: flex; gap: 0.5rem; }

.toolbar-btn {
  background: var(--color-surface-2) !important;
  border: 1px solid var(--color-border-light) !important;
  color: var(--color-text) !important;
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  padding: 0.375rem 0.875rem !important;
  height: 32px !important;
  border-radius: var(--radius-md) !important;
  box-shadow: var(--shadow-sm) !important;
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

.compose-container {
  flex: 1;
  overflow-y: auto;
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
}

.compose-alert {
  margin: 1rem 1.5rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
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

.compose-field {
  display: flex;
  align-items: center;
  padding: 0.75rem 1.5rem;
  gap: 1rem;
}

.compose-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-muted);
  min-width: 60px;
  text-align: right;
}

.compose-input,
.compose-input :deep(.p-inputtext) {
  background-color: var(--color-surface) !important;
  color: var(--color-text) !important;
  border: 1px solid var(--color-border-light) !important;
  font-size: 0.875rem !important;
  padding: 0.375rem 0.75rem !important;
  border-radius: var(--radius-md) !important;
  box-shadow: none !important;
  transition: border-color var(--transition), box-shadow var(--transition) !important;
}

.compose-input:focus,
.compose-input:focus-within,
.compose-input :deep(.p-inputtext:focus) {
  border-color: var(--color-accent) !important;
  box-shadow: 0 0 0 2px rgba(8, 117, 225, 0.1) !important;
}

.w-full { width: 100%; }

.field-divider {
  margin: 0 1.5rem !important;
  border-color: var(--color-border-light) !important;
}

.compose-body {
  flex: 1;
  border-top: 1px solid var(--color-border-light);
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
  padding: 1rem 1.5rem;
  background-color: var(--color-surface-2);
}

.compose-attachments :deep(.p-fileupload-choose) {
  background: var(--color-surface) !important;
  border: 1px solid var(--color-border-light) !important;
  color: var(--color-text) !important;
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  box-shadow: var(--shadow-sm) !important;
  height: 32px !important;
  padding: 0.375rem 0.875rem !important;
  border-radius: var(--radius-md) !important;
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
</style>
