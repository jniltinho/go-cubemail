/**
 * @file composerActions.ts
 * @description State controllers and methods for composing, replying,
 * forwarding, and viewing RFC 822 MIME sources of email messages.
 */

import axios from 'axios'
import type { Ref, ComputedRef } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'

type AuthStore = ReturnType<typeof useAuthStore>

/**
 * Context containing reactive states and dependencies needed
 * for running composer operations.
 */
interface ComposerActionsContext {
  /** Authentication store instance */
  auth:       AuthStore
  /** Reactive folder list reference */
  folders:    Ref<Folder[]>
  /** Selected/open email message (computed) */
  selected:   ComputedRef<MailMessage | null>
  /** Reactive state holding draft details under composition, or null if closed */
  composer:   Ref<Record<string, unknown> | null>
  /** Active message whose source code is being viewed */
  sourceMail: Ref<MailMessage | null>
  /** Raw MIME RFC 822 source text loaded from backend */
  sourceRaw:  Ref<string>
}

/**
 * Composable defining layout actions in the writing/composer panel.
 * 
 * @param context - The context containing reactive stores and states.
 * @returns Composer layout actions controller methods.
 */
export function useComposerActions({
  auth, folders, selected, composer, sourceMail, sourceRaw,
}: ComposerActionsContext) {

  /**
   * Prepares the editor layout to draft a reply message.
   * Resolves recipient address, prefixes "Re: " to subject, and quotes original text/HTML.
   */
  function reply(): void {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   s.from?.addr,
      subj: 'Re: ' + s.subject,
      quoted: {
        header: `On ${s.fullDate}, ${s.from?.name} &lt;${s.from?.addr}&gt; wrote:`,
        html:   s.htmlBody || null,
        text:   s.body     || [],
      },
    }
  }

  /**
   * Prepares the editor layout to draft a reply-all message.
   * Resolves recipient addresses including original sender and all other receivers,
   * excluding the current logged-in user email, prefixes "Re: " to subject, and quotes original.
   */
  function replyAll(): void {
    const s = selected.value; if (!s) return
    
    const myEmail = auth.currentUser.email?.toLowerCase().trim()
    const toEmails = (s.to || '').split(',')
      .map(x => x.trim())
      .filter(x => x && (!myEmail || x.toLowerCase().indexOf(myEmail) === -1))
    
    const recipients = [s.from?.addr]
    toEmails.forEach(email => {
      if (email && email.toLowerCase() !== s.from?.addr?.toLowerCase()) {
        recipients.push(email)
      }
    })

    composer.value = {
      to:   recipients.filter(Boolean).join(', '),
      subj: 'Re: ' + s.subject,
      quoted: {
        header: `On ${s.fullDate}, ${s.from?.name} &lt;${s.from?.addr}&gt; wrote:`,
        html:   s.htmlBody || null,
        text:   s.body     || [],
      },
    }
  }

  /**
   * Prepares the editor layout to forward the selected email message.
   * Prefixes "Fwd: " to subject and quotes original headers and body.
   */
  function forward(): void {
    const s = selected.value; if (!s) return
    composer.value = {
      to:   '',
      subj: 'Fwd: ' + s.subject,
      quoted: {
        header: `---------- Forwarded message ----------<br>From: ${s.from?.name} &lt;${s.from?.addr}&gt;<br>Date: ${s.fullDate}<br>Subject: ${s.subject}`,
        html:   s.htmlBody || null,
        text:   s.body     || [],
      },
    }
  }

  /**
   * Opens the composer panel modal in blank mode for drafting a new message.
   */
  function compose():       void { composer.value = {} }

  /**
   * Closes the composer modal and clears its active draft buffer.
   */
  function closeComposer(): void { composer.value = null }

  /**
   * Opens the source code viewer modal for the active email message.
   * Dispatches async request to query the raw MIME RFC 822 source code.
   */
  async function showSource(): Promise<void> {
    const m = selected.value
    if (!m) return
    sourceMail.value = m
    sourceRaw.value  = ''
    if (auth.isApiOnline) {
      try {
        const fd    = folders.value.find(f => f.id === m.folder)
        const label = fd?.name || fd?.label || 'INBOX'
        const res   = await axios.get(`${API_BASE}/mail/${encodeURIComponent(label)}/${m.id}/raw`, { responseType: 'text' })
        sourceRaw.value = res.data
      } catch {}
    }
  }

  /**
   * Closes the source viewer panel and clears raw source buffers.
   */
  function closeSource(): void { sourceMail.value = null; sourceRaw.value = '' }

  /**
   * Copies the raw source text segment into the user clipboard.
   * 
   * @param rawText - Raw MIME text content to copy.
   */
  function copySource(rawText: string): void {
    try { navigator.clipboard.writeText(rawText) } catch {}
  }

  return { reply, replyAll, forward, compose, closeComposer, showSource, closeSource, copySource }
}
