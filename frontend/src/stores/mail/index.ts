/**
 * @file index.ts
 * @description Main entrypoint and assembly of the Pinia `useMailStore` store structure.
 * Orchestrates reactives of emails, folders, contacts, calendar agendas, query strings,
 * and composes actions exported by distributed submodules.
 */

import { defineStore } from 'pinia'
import { ref, computed, watch, watchEffect } from 'vue'
import { useAuthStore } from '../auth'
import { useDialogStore } from '../dialog'
import { useToastStore } from '../toast'
import { applyAccent, buildCalCells, parseMailDate } from '../../utils/helpers'
import { CAL_EVENTS } from './mockData'
import { useMailApi } from './api'
import { useFolderActions } from './folderActions'
import { useMailActions } from './mailActions'
import { useComposerActions } from './composerActions'
import { useContactActions } from './contactActions'
import type { MailMessage, Folder, Contact, CalCell } from '../../types'

/**
 * Global unified mail, folders, and contacts controller store (`useMailStore`).
 */
export const useMailStore = defineStore('mail', () => {
  const auth   = useAuthStore()
  const dialog = useDialogStore()
  const toast  = useToastStore()

  // ── State ──────────────────────────────────────────────────────────────────
  
  /** Primary hex layout theme color key (defaults to deep navy blue) */
  const accent         = ref('#1B3A6B')
  /** True if background loading API queries are active */
  const loading        = ref(false)
  /** Roster list of all email messages downloaded in local memory */
  const mails          = ref<MailMessage[]>([])
  /** Roster list of system or custom folders synced from backend */
  const folders        = ref<Folder[]>([])
  /** Roster list of address book contacts synced from backend */
  const contacts       = ref<Contact[]>([])
  /** True if contacts addition/modification form modal is open */
  const contactModal   = ref(false)
  /** Active contact being modified, or null if drafting new profile */
  const editingContact = ref<Contact | null>(null)
  /** Computed cell days representing monthly calendar grids */
  const calCells       = ref<CalCell[]>(buildCalCells(CAL_EVENTS))
  /** Abbreviated weekdays array used to draw column headers */
  const calDow         = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  /** Active primary layout panel view (e.g. 'mail', 'contacts', 'calendar') */
  const view           = ref('mail')
  /** Frontend ID key of the active folder viewed (e.g. 'inbox') */
  const folder         = ref('inbox')
  /** IMAP UID key of the active open email read on the viewport */
  const selectedId     = ref<string | null>(null)
  /** Set of email UIDs checked for batch/lote operations */
  const selectedIds    = ref<Set<string>>(new Set())
  /** Active query text entered in global search inputs */
  const query          = ref('')
  /** State of active email draft under composition, or null if closed */
  const composer       = ref<Record<string, unknown> | null>(null)
  /** Message whose raw source MIME text is viewed */
  const sourceMail     = ref<MailMessage | null>(null)
  /** Raw MIME RFC 822 source text loaded from backend */
  const sourceRaw      = ref('')
  /** Active sort field for the mail list */
  const sortBy         = ref<'date' | 'from' | 'subject' | 'size'>('date')
  /** Active sort direction for the mail list */
  const sortDir        = ref<'asc' | 'desc'>('desc')

  /**
   * Watches base layout theme accent hex modifications to automatically
   * recalculate and inject dynamic CSS variables to the document root element.
   */
  watchEffect(() => applyAccent(accent.value))

  // ── Computed ───────────────────────────────────────────────────────────────
  
  /**
   * Filters and returns emails eligible to show under the active folder,
   * applying search queries dynamically if the user entered search keys.
   */
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
    const dir = sortDir.value === 'asc' ? 1 : -1
    const byDate = (a: MailMessage, b: MailMessage) =>
      parseMailDate(b.rawDate) - parseMailDate(a.rawDate)
    return [...xs].sort((a, b) => {
      switch (sortBy.value) {
        case 'from': {
          const fa = (a.from?.name || a.from?.addr || '').trim().toLowerCase()
          const fb = (b.from?.name || b.from?.addr || '').trim().toLowerCase()
          return dir * fa.localeCompare(fb) || byDate(a, b)
        }
        case 'subject': {
          const cmp = dir * (a.subject || '').trim().toLowerCase()
            .localeCompare((b.subject || '').trim().toLowerCase())
          return cmp || byDate(a, b)
        }
        case 'size':
          return dir * ((a.size ?? 0) - (b.size ?? 0)) || byDate(a, b)
        case 'date':
        default:
          return dir * (parseMailDate(a.rawDate) - parseMailDate(b.rawDate))
      }
    })
  })

  /**
   * Computes simple total and unread email counters quickly.
   */
  const counts = computed(() => {
    const inbox = mails.value.filter(m => m.folder === 'inbox')
    return {
      inboxTotal:  inbox.length,
      inboxUnread: inbox.filter(m => m.unread).length,
      starred:     mails.value.filter(m => m.starred).length,
    }
  })

  /** Resolves the complete MailMessage model for the selected open email */
  const selected = computed(() => mails.value.find(m => m.id === selectedId.value) ?? null)

  /** Resolves text display label for the viewed folder (e.g. "Inbox") */
  const currentFolderLabel = computed(() =>
    folders.value.find(f => f.id === folder.value)?.label || 'Inbox'
  )

  // ── Watchers ───────────────────────────────────────────────────────────────
  
  /**
   * Assures that an active email focuses automatically when switching folders,
   * selecting the first visible row if the previously viewed message is no longer present.
   */
  watch([folder, visibleMails], () => {
    if (!visibleMails.value.find(m => m.id === selectedId.value))
      selectedId.value = visibleMails.value[0]?.id ?? null
  })

  /** Resets sort direction to a sensible default when the sort field changes. */
  watch(sortBy, (field) => {
    sortDir.value = field === 'from' || field === 'subject' ? 'asc' : 'desc'
  })

  /**
   * Monitors selected email swaps to automatically schedule seen/read flag toggles
   * after 400ms if the email opened was unread.
   */
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

  /**
   * Dispatches initial load routines to reload folder trees, sync directories,
   * and load contacts registers from backend endpoints.
   */
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
    sortBy, sortDir,
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
