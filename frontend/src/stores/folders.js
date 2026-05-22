import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getFolders, getUnreadCount } from '@/api/folders.js'

export const useFoldersStore = defineStore('folders', () => {
  const folders = ref([])
  const unreadCounts = ref({})
  const loading = ref(false)

  // Common folder name → PrimeIcon mapping
  const folderIcon = (name) => {
    const lower = name.toLowerCase()
    if (lower === 'inbox') return 'pi pi-inbox'
    if (lower === 'sent' || lower === 'sent items' || lower === 'enviados') return 'pi pi-send'
    if (lower === 'drafts' || lower === 'rascunhos') return 'pi pi-file-edit'
    if (lower === 'trash' || lower === 'lixeira' || lower === 'deleted') return 'pi pi-trash'
    if (lower === 'spam' || lower === 'junk') return 'pi pi-ban'
    if (lower === 'archive' || lower === 'arquivo') return 'pi pi-archive'
    return 'pi pi-folder'
  }

  async function load() {
    loading.value = true
    try {
      const res = await getFolders()
      folders.value = res.data || []
      // Load unread counts in background
      loadUnreadCounts()
    } finally {
      loading.value = false
    }
  }

  async function loadUnreadCounts() {
    const counts = {}
    await Promise.allSettled(
      folders.value.map(async (f) => {
        try {
          const res = await getUnreadCount(f.Name)
          counts[f.Name] = res.data.unseen || 0
        } catch {
          counts[f.Name] = 0
        }
      }),
    )
    unreadCounts.value = counts
  }

  function getCount(name) {
    return unreadCounts.value[name] || 0
  }

  return { folders, unreadCounts, loading, load, loadUnreadCounts, getCount, folderIcon }
})
