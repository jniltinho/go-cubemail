import { defineStore } from 'pinia'
import { ref, computed, watch, watchEffect } from 'vue'
import { useAuthStore } from '../auth'
import { useDialogStore } from '../dialog'
import { useToastStore } from '../toast'
import { applyAccent, buildCalCells } from '../../utils/helpers'
import { CAL_EVENTS } from './mockData'
import { useMailApi } from './api'
import { useFolderActions } from './folderActions'
import { useMailActions } from './mailActions'
import { useComposerActions } from './composerActions'
import { useContactActions } from './contactActions'
import type { MailMessage, Folder, Contact, CalCell } from '../../types'


export const useMailStore = defineStore('mail', () => {
  const auth   = useAuthStore()
  const dialog = useDialogStore()
  const toast  = useToastStore()

  // ── State ──────────────────────────────────────────────────────────────────
  const accent         = ref('#1B3A6B')
  const loading        = ref(false)
  const mails          = ref<MailMessage[]>([])
  const folders        = ref<Folder[]>([])
  const contacts       = ref<Contact[]>([])
  const contactModal   = ref(false)
  const editingContact = ref<Contact | null>(null)
  const calCells       = ref<CalCell[]>(buildCalCells(CAL_EVENTS))
  const calDow         = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const view           = ref('mail')
  const folder         = ref('inbox')
  const selectedId     = ref<string | null>(null)
  const selectedIds    = ref<Set<string>>(new Set())
  const query          = ref('')
  const composer       = ref<Record<string, unknown> | null>(null)
  const sourceMail     = ref<MailMessage | null>(null)
  const sourceRaw      = ref('')

  watchEffect(() => applyAccent(accent.value))

  // ── Computed ───────────────────────────────────────────────────────────────
  const visibleMails = computed(() => {
    let xs = folder.value === 'starred'
      ? mails.value.filter(m => m.starred)
      : mails.value.filter(m => m.folder === folder.value)
    const q = query.value.trim().toLowerCase()
    if (q) xs = xs.filter(m =>
      m.from?.name?.toLowerCase().includes(q) ||
      m.from?.addr?.toLowerCase().includes(q) ||
      m.subject?.toLowerCase().includes(q) ||
      m.snippet?.toLowerCase().includes(q) ||
      (m.attachments || []).some(a => a.name?.toLowerCase().includes(q))
    )
    return xs
  })

  const counts = computed(() => {
    const inbox = mails.value.filter(m => m.folder === 'inbox')
    return {
      inboxTotal:  inbox.length,
      inboxUnread: inbox.filter(m => m.unread).length,
      starred:     mails.value.filter(m => m.starred).length,
    }
  })

  const selected = computed(() => mails.value.find(m => m.id === selectedId.value) ?? null)

  const currentFolderLabel = computed(() =>
    folders.value.find(f => f.id === folder.value)?.label || 'Inbox'
  )

  // ── Watchers ───────────────────────────────────────────────────────────────
  watch([folder, visibleMails], () => {
    if (!visibleMails.value.find(m => m.id === selectedId.value))
      selectedId.value = visibleMails.value[0]?.id ?? null
  })

  watch(selectedId, () => {
    const s = selected.value
    if (s?.unread) setTimeout(() => {
      const m = mails.value.find(x => x.id === s.id)
      if (m) m.unread = false
    }, 400)
  })

  // ── Composables ────────────────────────────────────────────────────────────
  const { fetchFolderMessages, loadFromApi: _loadFromApi, fetchMessageBody } = useMailApi({
    auth, folders, mails, folder, selectedId,
  })

  async function loadFromApi(): Promise<void> {
    loading.value = true
    try {
      await _loadFromApi()
      await contactApi.fetchContacts()
    } finally {
      loading.value = false
    }
  }

  const folderApi   = useFolderActions({ auth, dialog, folders, mails, folder, view, selectedId, selectedIds, fetchFolderMessages })
  const mailApi     = useMailActions({ auth, mails, folders, folder, selectedId, selectedIds, visibleMails, fetchMessageBody })
  const composerApi = useComposerActions({ auth, folders, selected, composer, sourceMail, sourceRaw })
  const contactApi  = useContactActions({ auth, toast, contacts, contactModal, editingContact })

  // ── Return ─────────────────────────────────────────────────────────────────
  return {
    // state
    accent, loading, mails, folders, contacts, contactModal, editingContact, calCells, calDow,
    view, folder, selectedId, selectedIds, query, composer, sourceMail, sourceRaw,
    // computed
    visibleMails, counts, selected, currentFolderLabel,
    // api
    loadFromApi, fetchFolderMessages, fetchMessageBody,
    ...folderApi,
    ...mailApi,
    ...composerApi,
    ...contactApi,
  }
})
