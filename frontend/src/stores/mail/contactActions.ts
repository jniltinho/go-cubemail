/**
 * @file contactActions.ts
 * @description Action modules for contacts agenda CRUD and persistence operations.
 * Governs saving, modifying, deleting, importing CSV/VCF files, and modal behaviors.
 */

import axios from 'axios'
import type { Ref } from 'vue'
import type { Contact } from '../../types'
import type { useAuthStore } from '../auth'
import type { useToastStore } from '../toast'

type AuthStore  = ReturnType<typeof useAuthStore>
type ToastStore = ReturnType<typeof useToastStore>

/**
 * Context containing reactive states and dependencies needed
 * for contacts database operations.
 */
interface ContactActionsContext {
  /** Authentication store instance */
  auth:           AuthStore
  /** Toast notifications store instance */
  toast:          ToastStore
  /** Reactive list reference of contact profiles */
  contacts:       Ref<Contact[]>
  /** Reactive visibility status of contacts modal */
  contactModal:   Ref<boolean>
  /** Active contact being modified, or null if drafting a new contact */
  editingContact: Ref<Contact | null>
}

/**
 * Composable defining actions on the address book panels.
 * 
 * @param context - The context containing reactive stores and references.
 * @returns Contact actions controller methods.
 */
export function useContactActions({
  auth, toast, contacts, contactModal, editingContact,
}: ContactActionsContext) {

  /**
   * Sorts a list of contacts alphabetically by their `name` property.
   * 
   * @param list - The unsorted contacts list.
   * @returns A new alphabetically sorted list.
   */
  function sortContacts(list: Contact[]): Contact[] {
    return [...list].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  }

  /**
   * Opens the contact form modal to add a brand new entry.
   */
  function openContactModal():            void { editingContact.value = null; contactModal.value = true }

  /**
   * Opens the contact form modal prefilled with details from an existing entry.
   * 
   * @param c - The contact registry entry to edit.
   */
  function openEditContact(c: Contact):   void { editingContact.value = c;    contactModal.value = true }

  /**
   * Closes the active contact form modal and resets target states.
   */
  function closeContactModal():           void { contactModal.value = false; editingContact.value = null }

  /**
   * Fetches the entire directory of registered contacts from the backend,
   * sorting the response alphabetically before committing to the local store.
   */
  async function fetchContacts(): Promise<void> {
    if (!auth.isApiOnline) return
    try {
      const res = await axios.get(`${API_BASE}/contacts`)
      contacts.value = sortContacts(res.data as Contact[])
    } catch {}
  }

  /**
   * Posts new contact profiles payload to the backend server.
   * 
   * @param data - Contact information excluding primary ID key.
   */
  async function saveContact(data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res = await axios.post(`${API_BASE}/contacts`, data)
      contacts.value = sortContacts([...contacts.value, res.data as Contact])
    } catch {}
  }

  /**
   * Submits a put request to update an existing contact profile's metrics on the server.
   * 
   * @param id - Unique database record identifier.
   * @param data - Updated profile details.
   */
  async function updateContact(id: number, data: Omit<Contact, 'id'>): Promise<void> {
    try {
      const res     = await axios.put(`${API_BASE}/contacts/${id}`, data)
      const updated = res.data as Contact
      contacts.value = sortContacts(contacts.value.map(c => c.id === id ? updated : c))
    } catch {}
  }

  /**
   * Dispatches a delete request to remove a contact entry from the database.
   * 
   * @param id - The unique ID key of the target profile.
   */
  async function deleteContact(id: number): Promise<void> {
    try {
      await axios.delete(`${API_BASE}/contacts/${id}`)
      contacts.value = contacts.value.filter(c => c.id !== id)
    } catch {}
  }

  /**
   * Submits a file upload package to import contacts from CSV or VCard attachments.
   * Displays status feedback metrics via toast alerts upon import finalization.
   * 
   * @param file - The CSV/VCF file chosen by the user.
   */
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
