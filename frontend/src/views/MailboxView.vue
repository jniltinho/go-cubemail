<template>
  <div class="mail-layout">
    <AppSidebar />

    <!-- Coluna 2: Lista de Mensagens -->
    <main class="mail-main">
      <!-- Topbar -->
      <header class="topbar">
        <div class="topbar-left">
          <h2 class="mailbox-title">
            <i :class="foldersStore.folderIcon(route.params.mailbox)" class="mr-2"></i>
            {{ route.params.mailbox }}
          </h2>
          <span class="msg-count" v-if="!mailStore.loading">
            {{ mailStore.totalMessages }} mensagens
          </span>
        </div>
        <div class="topbar-right">
          <span class="p-input-icon-left search-wrap">
            <i class="pi pi-search"></i>
            <InputText
              id="mailbox-search"
              v-model="searchQuery"
              placeholder="Pesquisar nesta caixa..."
              class="search-input"
              @keydown.enter="doSearch"
            />
          </span>
          <Button
            icon="pi pi-refresh"
            class="refresh-btn"
            v-tooltip="'Atualizar'"
            @click="refresh"
            id="refresh-btn"
          />
        </div>
      </header>

      <!-- Toolbar -->
      <div class="toolbar">
        <Button
          v-if="isTrash"
          label="Esvaziar Lixeira"
          icon="pi pi-trash"
          class="toolbar-btn delete-btn-empty"
          @click="confirmEmptyTrash"
          id="empty-trash-btn"
        />
        
        <Button
          label="Marcar como lida"
          icon="pi pi-envelope"
          class="toolbar-btn"
          @click="refresh"
        />

        <div class="toolbar-right-info" v-if="mailStore.totalMessages > 0 && !mailStore.loading">
          {{ mailStore.messages.length }} de {{ mailStore.totalMessages }}
        </div>
      </div>

      <!-- Message list -->
      <div class="message-list-wrap">
        <div v-if="mailStore.loading" class="list-loading">
          <ProgressSpinner style="width:36px;height:36px;" strokeWidth="4" />
          <span>Carregando mensagens...</span>
        </div>

        <div v-else-if="mailStore.error" class="list-empty error-state">
          <i class="pi pi-exclamation-circle" style="font-size:2rem; color: var(--color-danger);"></i>
          <p>{{ mailStore.error }}</p>
        </div>

        <div v-else-if="mailStore.messages.length === 0" class="list-empty">
          <i class="pi pi-inbox" style="font-size:3rem; color: var(--color-text-muted);"></i>
          <p>Nenhuma mensagem</p>
        </div>

        <div v-else class="message-list">
          <MessageRow
            v-for="(msgItem, idx) in mailStore.messages"
            :key="msgItem.uid"
            :message="msgItem"
            :mailbox="route.params.mailbox"
            :class="{ 'row-even': idx % 2 === 0, 'row-odd': idx % 2 !== 0 }"
            @deleted="refresh"
            @moved="refresh"
          />
        </div>
      </div>

      <!-- Pagination -->
      <div class="pagination" v-if="totalPages > 1">
        <Button icon="pi pi-angle-left" class="p-button-text p-button-sm pagination-btn" :disabled="mailStore.page <= 1" @click="changePage(mailStore.page - 1)" id="prev-page-btn" />
        <span class="page-info">{{ mailStore.page }} / {{ totalPages }}</span>
        <Button icon="pi pi-angle-right" class="p-button-text p-button-sm pagination-btn" :disabled="mailStore.page >= totalPages" @click="changePage(mailStore.page + 1)" id="next-page-btn" />
      </div>
    </main>

    <!-- Coluna 3: Visualizador / Detalhes da Mensagem -->
    <section class="mail-detail-pane">
      <!-- Toolbar do Detalhe -->
      <header class="detail-topbar">
        <div class="detail-topbar-actions" v-if="msg">
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
            @click="confirmDeleteDetail"
            id="delete-msg-btn"
          />
        </div>
        <div class="detail-topbar-right" v-if="route.params.uid">
          <Button
            icon="pi pi-times"
            class="close-btn"
            v-tooltip="'Fechar visualização'"
            @click="closeDetail"
            id="close-detail-btn"
          />
        </div>
      </header>

      <!-- Estado: Carregando -->
      <div v-if="mailStore.messageLoading" class="detail-loading">
        <ProgressSpinner style="width:40px;height:40px;" strokeWidth="4" />
        <span>Carregando mensagem...</span>
      </div>

      <!-- Estado: Erro -->
      <div v-else-if="mailStore.error && route.params.uid" class="detail-error">
        <i class="pi pi-exclamation-circle" style="font-size: 2rem; color: var(--color-danger);"></i>
        <p>{{ mailStore.error }}</p>
      </div>

      <!-- Estado: Selecionado (Exibir Mensagem) -->
      <div v-else-if="msg" class="detail-container">
        <!-- Cabeçalho do Email (Envelope) -->
        <div class="detail-header">
          <h1 class="detail-subject">{{ msg.envelope?.subject || '(sem assunto)' }}</h1>
          
          <div class="detail-meta-block">
            <div class="detail-avatar">
              {{ fromInitials }}
            </div>
            <div class="detail-meta-details">
              <div class="detail-from-row">
                <span class="detail-from">{{ fromDisplay }}</span>
                <span class="detail-date">{{ formattedDateDetail }}</span>
              </div>
              <div class="detail-to-row" v-if="msg.envelope?.to?.length">
                <span class="detail-label">Para:</span>
                <span class="detail-to">{{ toDisplay }}</span>
              </div>
            </div>
          </div>

          <!-- Anexos -->
          <div class="detail-attachments" v-if="msg.attachments?.length">
            <div class="detail-att-label">
              <i class="pi pi-paperclip"></i>
              {{ msg.attachments.length }} anexo(s)
            </div>
            <div class="detail-att-list">
              <a
                v-for="att in msg.attachments"
                :key="att.part"
                :href="attachmentUrl(att.part)"
                :download="att.filename"
                class="detail-att-chip"
                :id="`att-${att.part}`"
              >
                <i class="pi pi-file"></i>
                <span>{{ att.filename }}</span>
                <span class="detail-att-size">({{ att.size_label }})</span>
              </a>
            </div>
          </div>
        </div>

        <!-- Corpo da Mensagem -->
        <div class="detail-body">
          <iframe
            v-if="msg.html_body"
            :srcdoc="msg.html_body"
            sandbox="allow-same-origin"
            class="detail-iframe"
            ref="iframeRef"
            @load="resizeIframe"
            id="message-iframe"
          ></iframe>
          <div v-else class="detail-plain">
            <pre>{{ msg.plain_body || 'Mensagem vazia.' }}</pre>
          </div>
        </div>
      </div>

      <!-- Estado: Sem Seleção (Placeholder) -->
      <div v-else class="detail-placeholder">
        <i class="pi pi-envelope placeholder-icon"></i>
        <h3>Nenhuma mensagem selecionada</h3>
        <p>Selecione um e-mail na lista para visualizá-lo.</p>
      </div>
    </section>
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
import { computed, watch, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import ProgressSpinner from 'primevue/progressspinner'
import Dialog from 'primevue/dialog'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import AppSidebar from '@/components/AppSidebar.vue'
import MessageRow from '@/components/MessageRow.vue'
import { useMailStore } from '@/stores/mail.js'
import { useFoldersStore } from '@/stores/folders.js'
import { emptyTrash, deleteMessage, downloadMessage, getAttachmentUrl, rawMessage, getRawMessage } from '@/api/mail.js'

const route = useRoute()
const router = useRouter()
const mailStore = useMailStore()
const foldersStore = useFoldersStore()
const confirm = useConfirm()
const toast = useToast()

const searchQuery = ref('')
const ROWS_PER_PAGE = 50
const iframeRef = ref(null)
const showSourceModal = ref(false)
const rawSource = ref('')
const loadingSource = ref(false)

const isTrash = computed(() =>
  route.params.mailbox?.toLowerCase() === 'trash',
)

const totalPages = computed(() =>
  Math.max(1, Math.ceil(mailStore.totalMessages / ROWS_PER_PAGE)),
)

// Computed properties for active message details
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

const formattedDateDetail = computed(() => {
  const d = msg.value?.envelope?.date
  if (!d) return ''
  const parsed = parseDMYDate(d)
  if (!parsed) return String(d)
  return parsed.toLocaleString('pt-BR')
})

function doSearch() {
  if (searchQuery.value.trim()) {
    router.push({ query: { q: searchQuery.value } })
  }
}

function refresh() {
  mailStore.loadMessages(route.params.mailbox, mailStore.page)
  foldersStore.loadUnreadCounts()
  if (route.params.uid) {
    mailStore.loadMessage(route.params.mailbox, route.params.uid)
  }
}

function changePage(p) {
  router.push({ params: { mailbox: route.params.mailbox }, query: { page: p } })
}

function confirmEmptyTrash() {
  confirm.require({
    message: 'Apagar permanentemente todas as mensagens da Lixeira?',
    header: 'Esvaziar Lixeira',
    icon: 'pi pi-trash',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await emptyTrash(route.params.mailbox)
        toast.add({ severity: 'success', summary: 'Lixeira esvaziada', life: 3000 })
        refresh()
      } catch {
        toast.add({ severity: 'error', summary: 'Erro ao esvaziar lixeira', life: 3000 })
      }
    },
  })
}

// Action methods for details pane
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
      iframe.style.height = '400px'
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

function closeDetail() {
  router.push(`/mail/${encodeURIComponent(route.params.mailbox)}`)
}

function confirmDeleteDetail() {
  confirm.require({
    message: 'Mover esta mensagem para a Lixeira?',
    header: 'Excluir mensagem',
    icon: 'pi pi-trash',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await deleteMessage(route.params.mailbox, route.params.uid)
        toast.add({ severity: 'success', summary: 'Mensagem excluída', life: 2000 })
        const currentMailbox = route.params.mailbox
        closeDetail()
        mailStore.loadMessages(currentMailbox, mailStore.page)
        foldersStore.loadUnreadCounts()
      } catch {
        toast.add({ severity: 'error', summary: 'Erro ao excluir', life: 2000 })
      }
    },
  })
}

// Load on mount and when route changes
onMounted(async () => {
  foldersStore.load()
  const page = parseInt(route.query.page) || 1
  mailStore.loadMessages(route.params.mailbox, page)
  if (route.params.uid) {
    await mailStore.loadMessage(route.params.mailbox, route.params.uid)
  } else {
    mailStore.currentMessage = null
  }
})

watch(
  () => `${route.params.mailbox}_${route.query.page || 1}`,
  async () => {
    const mailbox = route.params.mailbox
    const page = parseInt(route.query.page) || 1
    if (mailbox) {
      mailStore.loadMessages(mailbox, page)
    }
  },
)

watch(
  () => route.params.uid,
  async (newUid) => {
    if (newUid) {
      await mailStore.loadMessage(route.params.mailbox, newUid)
    } else {
      mailStore.currentMessage = null
    }
  }
)
</script>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: var(--color-bg);
}

/* Coluna 2: Lista de mensagens */
.mail-main {
  width: 450px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background-color: var(--color-surface);
  border-right: 1px solid var(--color-border-light);
}

.topbar {
  height: var(--topbar-height);
  background-color: var(--color-surface);
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.25rem;
  gap: 0.75rem;
  flex-shrink: 0;
  color: var(--color-text);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.mailbox-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
  letter-spacing: -0.015em;
}

.mr-2 { margin-right: 0.125rem; }

.msg-count {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  background-color: var(--color-surface-2);
  border: 1px solid var(--color-border-light);
  padding: 0.1875rem 0.5rem;
  border-radius: 0;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.search-wrap .pi-search {
  position: absolute;
  left: 0.75rem;
  color: var(--color-text-muted);
  z-index: 1;
  font-size: 0.8125rem;
}

.search-input {
  padding-left: 2rem !important;
  height: 32px !important;
  width: 180px !important;
  background-color: var(--color-surface-2) !important;
  color: var(--color-text) !important;
  border: 1px solid var(--color-border-light) !important;
  font-size: 0.8125rem !important;
  border-radius: 0 !important;
}

.search-input:focus {
  background-color: var(--color-surface) !important;
}

.refresh-btn {
  background: transparent !important;
  border: none !important;
  color: var(--color-text-muted) !important;
  width: 32px !important;
  height: 32px !important;
  font-size: 0.9375rem !important;
  padding: 0 !important;
  border-radius: 0 !important;
}

.refresh-btn:hover {
  background-color: var(--color-surface-2) !important;
  color: var(--color-text) !important;
}

.toolbar {
  height: 48px;
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  padding: 0 1.25rem;
  gap: 0.5rem;
  background-color: var(--color-surface);
  flex-shrink: 0;
}

.toolbar-btn {
  background: var(--color-surface-2) !important;
  border: 1px solid var(--color-border-light) !important;
  color: var(--color-text) !important;
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  padding: 0.375rem 0.75rem !important;
  height: 32px !important;
  border-radius: 0 !important; /* Pure square design */
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

.toolbar-right-info {
  margin-left: auto;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  font-weight: 500;
}

.message-list-wrap {
  flex: 1;
  overflow-y: auto;
  background-color: var(--color-surface);
}

.list-loading, .list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  height: 200px;
  color: var(--color-text-muted);
}

.error-state { color: var(--color-danger); }

.message-list {
  display: flex;
  flex-direction: column;
}

.row-even, .row-odd {
  background-color: var(--color-surface);
}

.pagination {
  height: 48px;
  border-top: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.5rem;
  background-color: var(--color-surface);
  flex-shrink: 0;
}

.pagination-btn {
  background: transparent !important;
  border: none !important;
  color: var(--color-text) !important;
  width: 32px !important;
  height: 32px !important;
  padding: 0 !important;
  border-radius: 0 !important;
}

.pagination-btn:hover:not(:disabled) {
  background-color: var(--color-surface-2) !important;
}

.page-info {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  min-width: 80px;
  text-align: center;
}

/* Coluna 3: Visualizador / Detalhes da Mensagem */
.mail-detail-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
  background-color: var(--color-bg);
}

.detail-topbar {
  height: var(--topbar-height);
  background-color: var(--color-surface);
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  flex-shrink: 0;
}

.detail-topbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.detail-topbar-right {
  display: flex;
  align-items: center;
}

.close-btn {
  background: transparent !important;
  border: none !important;
  color: var(--color-text-muted) !important;
  width: 32px !important;
  height: 32px !important;
  font-size: 0.9375rem !important;
  padding: 0 !important;
  border-radius: 0 !important;
}

.close-btn:hover {
  background-color: var(--color-surface-2) !important;
  color: var(--color-text) !important;
}

.detail-loading, .detail-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  flex: 1;
  color: var(--color-text-muted);
}

.detail-error p {
  color: var(--color-danger);
  font-size: 0.875rem;
  font-weight: bold;
}

.detail-container {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  background-color: var(--color-surface);
}

.detail-header {
  padding: 1.5rem;
  border-bottom: 1px solid var(--color-border-light);
  background-color: var(--color-surface);
}

.detail-subject {
  font-size: 1.375rem;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.3;
  margin-top: 0;
  margin-bottom: 1.25rem;
  letter-spacing: -0.02em;
}

.detail-meta-block {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.detail-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 0; /* Square per instructions */
  background-color: rgba(8, 117, 225, 0.08);
  color: var(--color-accent);
  font-weight: 700;
  font-size: 0.9375rem;
  flex-shrink: 0;
  border: 1px solid rgba(8, 117, 225, 0.15);
}

.detail-meta-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.detail-from-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 1rem;
}

.detail-from {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--color-text);
}

.detail-date {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
}

.detail-to-row {
  display: flex;
  align-items: baseline;
  gap: 0.375rem;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}

.detail-label {
  font-weight: 500;
}

.detail-to {
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-attachments {
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border-light);
}

.detail-att-label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 0.5rem;
}

.detail-att-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.detail-att-chip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--color-surface-2);
  border: 1px solid var(--color-border-light);
  border-radius: 0; /* Square per instructions */
  padding: 0.375rem 0.75rem;
  color: var(--color-text);
  text-decoration: none;
  font-size: 0.8125rem;
  font-weight: 500;
  transition: background-color var(--transition), border-color var(--transition);
}

.detail-att-chip:hover {
  background-color: rgba(8, 117, 225, 0.04);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.detail-att-size {
  color: var(--color-text-muted);
  font-weight: normal;
  font-size: 0.75rem;
}

.detail-body {
  flex: 1;
  padding: 0;
  overflow: auto;
  background-color: var(--color-surface);
}

.detail-iframe {
  width: 100%;
  min-height: 400px;
  border: none;
  background: #ffffff;
  display: block;
}

.detail-plain {
  padding: 1.5rem;
  background-color: var(--color-surface);
}

.detail-plain pre {
  font-family: inherit;
  font-size: 0.875rem;
  color: var(--color-text);
  white-space: pre-wrap;
  word-wrap: break-word;
  line-height: 1.6;
}

.detail-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  gap: 1rem;
  background-color: var(--color-bg);
  text-align: center;
  padding: 2rem;
}

.placeholder-icon {
  font-size: 4rem;
  color: var(--color-border-light);
}

.detail-placeholder h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.detail-placeholder p {
  font-size: 0.875rem;
  margin: 0;
  max-width: 280px;
  line-height: 1.5;
}
</style>
