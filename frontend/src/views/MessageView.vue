<template>
  <div class="mail-layout">
    <AppSidebar />

    <main class="msg-main">
      <!-- Topbar -->
      <header class="topbar">
        <Button
          icon="pi pi-arrow-left"
          class="toolbar-btn primary-btn"
          @click="router.back()"
          id="back-btn"
          label="Voltar"
        />
        <div class="topbar-actions" v-if="msg">
          <Button
            icon="pi pi-reply"
            label="Responder"
            class="toolbar-btn primary-btn"
            @click="reply"
            id="reply-btn"
          />
          <Button
            icon="pi pi-share-alt"
            label="Encaminhar"
            class="toolbar-btn"
            @click="forward"
            id="forward-btn"
          />
          <Button
            icon="pi pi-download"
            class="toolbar-btn"
            v-tooltip="'Baixar .eml'"
            @click="downloadMsg"
            id="download-btn"
          />
          <Button
            icon="pi pi-code"
            class="toolbar-btn"
            v-tooltip="'View Source'"
            @click="viewSource"
            id="view-source-btn"
          />
          <Button
            icon="pi pi-trash"
            class="toolbar-btn delete-btn-empty"
            v-tooltip="'Excluir'"
            @click="confirmDelete"
            id="delete-msg-btn"
          />
        </div>
      </header>

      <!-- Loading -->
      <div v-if="mailStore.messageLoading" class="msg-loading">
        <ProgressSpinner style="width:40px;height:40px;" strokeWidth="4" />
        <span>Carregando mensagem...</span>
      </div>

      <!-- Error -->
      <div v-else-if="mailStore.error" class="msg-error">
        <i class="pi pi-exclamation-circle"></i>
        {{ mailStore.error }}
      </div>

      <!-- Message -->
      <div v-else-if="msg" class="msg-container">
        <!-- Envelope/Header -->
        <div class="msg-header">
          <h1 class="msg-subject">{{ msg.envelope?.subject || '(sem assunto)' }}</h1>
          
          <div class="msg-meta-block">
            <div class="msg-avatar">
              {{ fromInitials }}
            </div>
            <div class="msg-meta-details">
              <div class="meta-from-row">
                <span class="meta-from">{{ fromDisplay }}</span>
                <span class="msg-date">{{ formattedDate }}</span>
              </div>
              <div class="meta-to-row" v-if="msg.envelope?.to?.length">
                <span class="meta-label">Para:</span>
                <span class="meta-to">{{ toDisplay }}</span>
              </div>
            </div>
          </div>

          <!-- Attachments -->
          <div class="attachments" v-if="msg.attachments?.length">
            <div class="att-label">
              <i class="pi pi-paperclip"></i>
              {{ msg.attachments.length }} anexo(s)
            </div>
            <div class="att-list">
              <a
                v-for="att in msg.attachments"
                :key="att.part"
                :href="attachmentUrl(att.part)"
                :download="att.filename"
                class="att-chip"
                :id="`att-${att.part}`"
              >
                <i class="pi pi-file"></i>
                <span>{{ att.filename }}</span>
                <span class="att-size">({{ att.size_label }})</span>
              </a>
            </div>
          </div>
        </div>

        <!-- Body -->
        <div class="msg-body">
          <iframe
            v-if="msg.html_body"
            :srcdoc="msg.html_body"
            sandbox="allow-same-origin"
            class="msg-iframe"
            ref="iframeRef"
            @load="resizeIframe"
            id="message-iframe"
          ></iframe>
          <div v-else class="msg-plain">
            <pre>{{ msg.plain_body || 'Mensagem vazia.' }}</pre>
          </div>
        </div>
      </div>
    </main>
  </div>

  <Dialog
    v-model:visible="showSourceModal"
    :modal="true"
    :closable="false"
    :draggable="true"
    :dismissableMask="true"
    class="source-dialog"
    header="Código Fonte da Mensagem"
    style="width: 1530px; height: 630px; max-width: 95vw; max-height: 95vh; display: flex; flex-direction: column;"
  >
    <template #header>
      <div class="source-header-bar" style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
        <h3 class="source-title" style="margin: 0; font-size: 1.1rem; font-weight: 600; display: flex; align-items: center; gap: 0.5rem; color: var(--color-text);">
          <i class="pi pi-code"></i>
          Código Fonte da Mensagem
        </h3>
        <div class="source-header-actions" style="display: flex; gap: 0.5rem; align-items: center;">
          <Button
            icon="pi pi-copy"
            class="p-button-text p-button-sm"
            v-tooltip="'Copiar Código Fonte'"
            @click="copySource"
            label="Copiar"
          />
          <Button
            icon="pi pi-times"
            class="p-button-text p-button-sm close-dialog-btn"
            v-tooltip="'Fechar'"
            @click="showSourceModal = false"
          />
        </div>
      </div>
    </template>

    <div class="source-content" style="flex: 1; padding: 1rem; display: flex; flex-direction: column; overflow: hidden; height: calc(100% - 10px);">
      <div v-if="loadingSource" style="display: flex; flex-direction: column; align-items: center; justify-content: center; flex: 1; gap: 1rem; color: var(--color-text-muted);">
        <ProgressSpinner style="width:40px;height:40px;" strokeWidth="4" />
        <span>Carregando código fonte...</span>
      </div>
      <textarea
        v-else
        readonly
        ref="sourceTextarea"
        v-model="rawSource"
        style="flex: 1; width: 100%; height: 100%; font-family: monospace; font-size: 0.85rem; padding: 0.75rem; border: 1px solid var(--color-border); background-color: var(--color-surface-2); color: var(--color-text); resize: none; outline: none; line-height: 1.4; white-space: pre;"
      ></textarea>
    </div>
  </Dialog>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import ProgressSpinner from 'primevue/progressspinner'
import Dialog from 'primevue/dialog'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import AppSidebar from '@/components/AppSidebar.vue'
import { useMailStore } from '@/stores/mail.js'
import { useFoldersStore } from '@/stores/folders.js'
import { deleteMessage, downloadMessage, getAttachmentUrl, rawMessage, getRawMessage } from '@/api/mail.js'

const route = useRoute()
const router = useRouter()
const mailStore = useMailStore()
const foldersStore = useFoldersStore()
const confirm = useConfirm()
const toast = useToast()
const iframeRef = ref(null)
const showSourceModal = ref(false)
const rawSource = ref('')
const loadingSource = ref(false)

const msg = computed(() => mailStore.currentMessage)

const fromDisplay = computed(() => {
  const from = msg.value?.envelope?.from
  const fromEmail = msg.value?.envelope?.from_email
  if (!from) return 'Desconhecido'
  if (fromEmail && from !== fromEmail) {
    return `${from} <${fromEmail}>`
  }
  return from
})

const fromInitials = computed(() => {
  const name = msg.value?.envelope?.from || ''
  if (!name) return '?'
  const cleanName = name.replace(/<.*>/, '').replace(/['"]/g, '').trim()
  if (!cleanName) return '?'
  const parts = cleanName.split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return cleanName[0]?.toUpperCase() || '?'
})

const toDisplay = computed(() => {
  return msg.value?.envelope?.to || ''
})

function parseDMYDate(d) {
  if (!d) return null
  const match = String(d).match(/^(\d{2})\/(\d{2})\/(\d{4})\s+(\d{2}):(\d{2})$/)
  if (match) {
    const day = parseInt(match[1], 10)
    const month = parseInt(match[2], 10) - 1
    const year = parseInt(match[3], 10)
    const hours = parseInt(match[4], 10)
    const minutes = parseInt(match[5], 10)
    return new Date(year, month, day, hours, minutes)
  }
  const fallback = new Date(d)
  if (!isNaN(fallback.getTime())) {
    return fallback
  }
  return null
}

const formattedDate = computed(() => {
  const d = msg.value?.envelope?.date
  if (!d) return ''
  const parsed = parseDMYDate(d)
  if (!parsed) return String(d)
  return parsed.toLocaleString('pt-BR')
})

function attachmentUrl(part) {
  return getAttachmentUrl(route.params.mailbox, route.params.uid, part)
}

function downloadMsg() {
  window.location.href = downloadMessage(route.params.mailbox, route.params.uid)
}

async function viewSource() {
  showSourceModal.value = true
  loadingSource.value = true
  rawSource.value = ''
  try {
    const res = await getRawMessage(route.params.mailbox, route.params.uid)
    rawSource.value = res.data
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: 'Erro',
      detail: 'Não foi possível carregar o código fonte.',
      life: 3000,
    })
    showSourceModal.value = false
  } finally {
    loadingSource.value = false
  }
}

function copySource() {
  navigator.clipboard.writeText(rawSource.value)
    .then(() => {
      toast.add({
        severity: 'success',
        summary: 'Sucesso',
        detail: 'Código fonte copiado para a área de transferência!',
        life: 3000,
      })
    })
    .catch(() => {
      toast.add({
        severity: 'error',
        summary: 'Erro',
        detail: 'Não foi possível copiar o código fonte.',
        life: 3000,
      })
    })
}

function resizeIframe() {
  if (iframeRef.value) {
    const iframe = iframeRef.value
    try {
      const h = iframe.contentDocument.body.scrollHeight
      iframe.style.height = (h + 32) + 'px'
    } catch { /* cross-origin */ }
  }
}

function reply() {
  mailStore.openCompose('reply', msg.value)
}

function forward() {
  mailStore.openCompose('forward', msg.value)
}

function confirmDelete() {
  confirm.require({
    message: 'Mover esta mensagem para a Lixeira?',
    header: 'Excluir mensagem',
    icon: 'pi pi-trash',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await deleteMessage(route.params.mailbox, route.params.uid)
        toast.add({ severity: 'success', summary: 'Mensagem excluída', life: 2000 })
        router.back()
      } catch {
        toast.add({ severity: 'error', summary: 'Erro ao excluir', life: 2000 })
      }
    },
  })
}

onMounted(async () => {
  foldersStore.load()
  await mailStore.loadMessage(route.params.mailbox, route.params.uid)
})
</script>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: var(--color-bg);
}

.msg-main {
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
  flex-shrink: 0;
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

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

.msg-loading, .msg-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  height: 200px;
  color: var(--color-text-muted);
}

.msg-error { color: var(--color-danger); font-size: 0.875rem; font-weight: bold; }

.msg-container {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  background-color: var(--color-surface);
}

.msg-header {
  padding: 1.5rem;
  border-bottom: 1px solid var(--color-border-light);
  background-color: var(--color-surface);
}

.msg-subject {
  font-size: 1.375rem;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.3;
  margin-bottom: 1.25rem;
  letter-spacing: -0.02em;
}

.msg-meta-block {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.msg-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background-color: rgba(8, 117, 225, 0.08);
  color: var(--color-accent);
  font-weight: 700;
  font-size: 0.9375rem;
  flex-shrink: 0;
  border: 1px solid rgba(8, 117, 225, 0.15);
}

.msg-meta-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.meta-from-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 1rem;
}

.meta-from {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--color-text);
}

.msg-date {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
}

.meta-to-row {
  display: flex;
  align-items: baseline;
  gap: 0.375rem;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}

.meta-label {
  font-weight: 500;
}

.meta-to {
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.attachments {
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border-light);
}

.att-label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 0.5rem;
}

.att-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.att-chip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--color-surface-2);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 0.375rem 0.75rem;
  color: var(--color-text);
  text-decoration: none;
  font-size: 0.8125rem;
  font-weight: 500;
  transition: background-color var(--transition), border-color var(--transition);
}

.att-chip:hover {
  background-color: rgba(8, 117, 225, 0.04);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.att-size {
  color: var(--color-text-muted);
  font-weight: normal;
  font-size: 0.75rem;
}

.msg-body {
  flex: 1;
  padding: 0;
  overflow: auto;
  background-color: var(--color-surface);
}

.msg-iframe {
  width: 100%;
  min-height: 400px;
  border: none;
  background: #ffffff;
  display: block;
}

.msg-plain {
  padding: 1.5rem;
  background-color: var(--color-surface);
}

.msg-plain pre {
  font-family: inherit;
  font-size: 0.875rem;
  color: var(--color-text);
  white-space: pre-wrap;
  word-wrap: break-word;
  line-height: 1.6;
}
</style>
