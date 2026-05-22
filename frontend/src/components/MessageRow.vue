<template>
  <div
    class="message-row"
    :class="{ unread: !message.seen, selected: selected }"
    @click="open"
    :id="`msg-${message.uid}`"
  >
    <!-- Unread indicator vertical bar (Elastic style left edge) -->
    <div class="unread-bar" v-if="!message.seen"></div>

    <div class="row-content">
      <!-- Upper Line: From & Date -->
      <div class="upper-line">
        <span class="msg-from" :class="{ bold: !message.seen }">{{ fromName }}</span>
        <span class="msg-date" :class="{ bold: !message.seen }">{{ formattedDate }}</span>
      </div>

      <!-- Lower Line: Subject & Flags/Actions -->
      <div class="lower-line">
        <span class="msg-subject" :class="{ bold: !message.seen }">
          {{ message.subject || '(sem assunto)' }}
        </span>
        <div class="msg-icons-actions" @click.stop>
          <span class="flag-icon" @click="toggleFlag">
            <i :class="message.flagged ? 'pi pi-star-fill flagged' : 'pi pi-star'"></i>
          </span>
          <button class="action-btn" title="Excluir" @click="confirmDelete" :id="`delete-${message.uid}`">
            <i class="pi pi-trash"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { flagMessage, deleteMessage } from '@/api/mail.js'

const props = defineProps({
  message: { type: Object, required: true },
  mailbox: { type: String, required: true },
})
const emit = defineEmits(['deleted', 'moved'])

const router = useRouter()
const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const selected = computed(() => String(route.params.uid) === String(props.message.uid))

const fromName = computed(() => {
  const f = props.message.from
  if (!f) return 'Desconhecido'
  if (typeof f === 'string') return f
  return f.name || f.address || String(f)
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
  const d = props.message.date
  if (!d) return ''
  const date = parseDMYDate(d)
  if (!date) return d
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()
  if (isToday) {
    return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
})

function open() {
  router.push(`/mail/${encodeURIComponent(props.mailbox)}/${props.message.uid}`)
}

async function toggleFlag() {
  try {
    await flagMessage(props.mailbox, props.message.uid, 'flagged', !props.message.flagged)
    props.message.flagged = !props.message.flagged
  } catch {
    toast.add({ severity: 'error', summary: 'Erro ao marcar mensagem', life: 2000 })
  }
}

function confirmDelete() {
  confirm.require({
    message: props.mailbox.toLowerCase() === 'trash'
      ? 'Apagar permanentemente esta mensagem?'
      : 'Mover para a Lixeira?',
    header: 'Excluir mensagem',
    icon: 'pi pi-trash',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await deleteMessage(props.mailbox, props.message.uid)
        toast.add({ severity: 'success', summary: 'Mensagem excluída', life: 2000 })
        emit('deleted')
      } catch {
        toast.add({ severity: 'error', summary: 'Erro ao excluir', life: 2000 })
      }
    },
  })
}
</script>

<style scoped>
.message-row {
  display: flex;
  position: relative;
  padding: 0.75rem 1.25rem 0.75rem 1.25rem;
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  background-color: var(--color-surface);
  transition: background-color var(--transition);
  min-height: 68px;
  box-sizing: border-box;
}

.message-row:hover {
  background-color: var(--color-surface-2) !important;
}

.message-row.selected {
  background-color: var(--color-surface-2) !important;
  border-left: 4px solid var(--color-accent);
}

/* Elastic vertical unread bar on left edge */
.unread-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background-color: var(--color-unread-dot);
}

.row-content {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  width: 100%;
  min-width: 0;
}

.upper-line, .lower-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  gap: 1rem;
}

.msg-from {
  font-size: 0.9375rem;
  color: var(--color-text);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.msg-from.bold {
  font-weight: 700;
}

.msg-date {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  white-space: nowrap;
  flex-shrink: 0;
}

.msg-date.bold {
  color: var(--color-accent);
  font-weight: 600;
}

.msg-subject {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.msg-subject.bold {
  color: var(--color-text);
  font-weight: 600;
}

.msg-icons-actions {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  flex-shrink: 0;
}

.flag-icon {
  color: #cbd5e1;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  transition: color var(--transition);
}

.flag-icon:hover {
  color: var(--color-warning);
}

.flagged {
  color: var(--color-warning) !important;
}

.action-btn {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity var(--transition), color var(--transition), background-color var(--transition);
}

.message-row:hover .action-btn {
  opacity: 1;
}

.action-btn:hover {
  color: var(--color-danger);
  background-color: rgba(239, 68, 68, 0.08);
}
</style>
