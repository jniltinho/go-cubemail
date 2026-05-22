import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listMessages, getMessage as apiGetMessage } from '@/api/mail.js'

export const useMailStore = defineStore('mail', () => {
  const messages = ref([])
  const currentMessage = ref(null)
  const currentMailbox = ref('INBOX')
  const page = ref(1)
  const totalMessages = ref(0)
  const loading = ref(false)
  const messageLoading = ref(false)
  const error = ref(null)

  async function loadMessages(mailbox, p = 1) {
    loading.value = true
    error.value = null
    currentMailbox.value = mailbox
    page.value = p
    try {
      const res = await listMessages(mailbox, p)
      messages.value = res.data.messages || []
      totalMessages.value = res.data.total || 0
    } catch (err) {
      error.value = err.response?.data?.error || 'Erro ao carregar mensagens'
      messages.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadMessage(mailbox, uid) {
    messageLoading.value = true
    currentMessage.value = null
    error.value = null
    try {
      const res = await apiGetMessage(mailbox, uid)
      currentMessage.value = res.data
    } catch (err) {
      error.value = err.response?.data?.error || 'Erro ao carregar mensagem'
    } finally {
      messageLoading.value = false
    }
  }

  function markReadLocally(uid) {
    const msg = messages.value.find((m) => m.uid === uid)
    if (msg) msg.seen = true
  }

  const composeVisible = ref(false)
  const composeMode = ref('new')
  const composeData = ref({
    to: '',
    cc: '',
    bcc: '',
    subject: '',
    body_html: ''
  })

  function openCompose(mode = 'new', msgToReply = null) {
    composeMode.value = mode
    if (mode === 'reply' && msgToReply) {
      const from = msgToReply.envelope?.from || ''
      const fromEmail = msgToReply.envelope?.from_email || ''
      const toValue = fromEmail || from

      const subject = msgToReply.envelope?.subject || ''
      const cleanSubject = subject.toLowerCase().startsWith('re:') ? subject : `Re: ${subject}`

      const dateStr = msgToReply.envelope?.date || ''
      const bodyHtml = msgToReply.html_body || (msgToReply.plain_body ? `<pre>${msgToReply.plain_body}</pre>` : '')

      composeData.value = {
        to: toValue,
        cc: '',
        bcc: '',
        subject: cleanSubject,
        body_html: `<br><br><div class="gmail_quote">Em ${dateStr}, ${from} escreveu:<br><blockquote class="gmail_quote" style="margin:0px 0px 0px 0.8ex;border-left:1px solid rgb(204,204,204);padding-left:1ex">${bodyHtml}</blockquote></div>`
      }
    } else if (mode === 'forward' && msgToReply) {
      const subject = msgToReply.envelope?.subject || ''
      const cleanSubject = subject.toLowerCase().startsWith('fwd:') ? subject : `Fwd: ${subject}`

      const from = msgToReply.envelope?.from || ''
      const dateStr = msgToReply.envelope?.date || ''
      const toList = msgToReply.envelope?.to || ''
      const bodyHtml = msgToReply.html_body || (msgToReply.plain_body ? `<pre>${msgToReply.plain_body}</pre>` : '')

      composeData.value = {
        to: '',
        cc: '',
        bcc: '',
        subject: cleanSubject,
        body_html: `<br><br><div class="gmail_quote">---------- Mensagem encaminhada ----------<br>De: ${from}<br>Data: ${dateStr}<br>Assunto: ${subject}<br>Para: ${toList}<br><br><blockquote class="gmail_quote" style="margin:0px 0px 0px 0.8ex;border-left:1px solid rgb(204,204,204);padding-left:1ex">${bodyHtml}</blockquote></div>`
      }
    } else {
      composeData.value = {
        to: '',
        cc: '',
        bcc: '',
        subject: '',
        body_html: ''
      }
    }
    composeVisible.value = true
  }

  return {
    messages,
    currentMessage,
    currentMailbox,
    page,
    totalMessages,
    loading,
    messageLoading,
    error,
    composeVisible,
    composeMode,
    composeData,
    loadMessages,
    loadMessage,
    markReadLocally,
    openCompose,
  }
})
