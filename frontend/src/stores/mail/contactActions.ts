import axios from 'axios'
import type { Ref } from 'vue'
import type { Contact } from '../../types'
import type { useAuthStore } from '../auth'
import type { useToastStore } from '../toast'


type AuthStore  = ReturnType<typeof useAuthStore>
type ToastStore = ReturnType<typeof useToastStore>

interface ContactActionsContext {
  auth:           AuthStore
  toast:          ToastStore
  contacts:       Ref<Contact[]>
  contactModal:   Ref<boolean>
  editingContact: Ref<Contact | null>
}

export function useContactActions({
  auth, toast, contacts, contactModal, editingContact,
}: ContactActionsContext) {

  function sortContacts(list: Contact[]): Contact[] {
    return [...list].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  }

  function openContactModal():            void { editingContact.value = null; contactModal.value = true }
  function openEditContact(c: Contact):   void { editingContact.value = c;    contactModal.value = true }
  function closeContactModal():           void { contactModal.value = false; editingContact.value = null }

  async function fetchContacts(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/contacts`)
      contacts.value = sortContacts(res.data as Contact[])
    } catch {}
  }

  async function saveContact(data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/contacts`, data)
      contacts.value = sortContacts([...contacts.value, res.data as Contact])
    } catch {}
  }

  async function updateContact(id: number, data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res     = await axios.put(`${API_BASE}/contacts/${id}`, data)
      const updated = res.data as Contact
      contacts.value = sortContacts(contacts.value.map(c => c.id === id ? updated : c))
    } catch {}
  }

  async function deleteContact(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/contacts/${id}`)
      contacts.value = contacts.value.filter(c => c.id !== id)
    } catch {}
  }

  async function importContacts(file: File): Promise<void> {
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await axios.post(`${API_BASE}/contacts/import`, fd)
      const { imported, total } = res.data
      await fetchContacts()
      if (imported === 0) {
        toast.warning(`No contacts found in file (${total} rows checked).`)
      } else {
        toast.success(`Imported ${imported} contact${imported !== 1 ? 's' : ''} successfully.`)
      }
    } catch (e: any) {
      toast.error(e.response?.data?.error || 'Failed to import contacts.')
    }
  }

  return {
    openContactModal, openEditContact, closeContactModal,
    fetchContacts, saveContact, updateContact, deleteContact, importContacts,
  }
}
