import axios from 'axios'
import type { Ref, ComputedRef } from 'vue'
import type { MailMessage, Folder } from '../../types'
import type { useAuthStore } from '../auth'


type AuthStore = ReturnType<typeof useAuthStore>

interface ComposerActionsContext {
  auth:       AuthStore
  folders:    Ref<Folder[]>
  selected:   ComputedRef<MailMessage | null>
  composer:   Ref<Record<string, unknown> | null>
  sourceMail: Ref<MailMessage | null>
  sourceRaw:  Ref<string>
}

export function useComposerActions({
  auth, folders, selected, composer, sourceMail, sourceRaw,
}: ComposerActionsContext) {

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

  function compose():       void { composer.value = {} }
  function closeComposer(): void { composer.value = null }

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

  function closeSource(): void { sourceMail.value = null; sourceRaw.value = '' }

  function copySource(rawText: string): void {
    try { navigator.clipboard.writeText(rawText) } catch {}
  }

  return { reply, forward, compose, closeComposer, showSource, closeSource, copySource }
}
