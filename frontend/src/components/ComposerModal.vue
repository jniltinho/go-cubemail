<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  prefill: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['close'])

const to     = ref(props.prefill?.to   || '')
const cc     = ref('')
const subj   = ref(props.prefill?.subj || '')
const body   = ref(props.prefill?.body || '')
const showCc = ref(false)
const sent   = ref(false)
const taRef  = ref(null)
let destroyEditor = null

onMounted(() => {
  if (!window.tinymce || !taRef.value) return

  // Convert plain-text prefill to HTML paragraphs
  const html = body.value
    .split('\n')
    .map(l => l.trim() === '' ? '<p>&nbsp;</p>' : `<p>${l.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}</p>`)
    .join('')

  let editor = null
  let suppress = false

  tinymce.init({
    target: taRef.value,
    inline: false,
    menubar: false,
    branding: false,
    promotion: false,
    statusbar: false,
    license_key: 'gpl',
    height: 240,
    placeholder: 'Write your message…',
    plugins: 'autolink lists link image table code emoticons charmap searchreplace wordcount',
    toolbar:
      'bold italic underline strikethrough | forecolor backcolor | ' +
      'alignleft aligncenter alignright | bullist numlist outdent indent | ' +
      'link image table emoticons | removeformat | code',
    toolbar_mode: 'wrap',
    content_style: [
      'body { font-family: "Segoe UI","Helvetica Neue",Arial,sans-serif;',
      '       font-size: 13.5px; color: #1A1F2A; line-height: 1.6; margin: 10px 12px; }',
      'p { margin: 0 0 10px; }',
    ].join(' '),
    skin: 'oxide',
    content_css: 'default',
    setup(ed) {
      editor = ed
      ed.on('init', () => {
        suppress = true
        ed.setContent(html)
        suppress = false
      })
      ed.on('input change keyup undo redo', () => {
        if (!suppress) body.value = ed.getContent()
      })
    }
  })

  destroyEditor = () => { try { editor?.remove() } catch {} }
})

onBeforeUnmount(() => destroyEditor?.())

function send() {
  sent.value = true
  setTimeout(() => emit('close'), 900)
}

function backdrop(e) {
  if (e.target === e.currentTarget) emit('close')
}
</script>

<template>
  <div class="modal-wrap" @click="backdrop">
    <div class="composer-shell" role="dialog" aria-label="New message">
      <!-- Header -->
      <div class="bg-accent-bar text-white py-2 px-3 flex items-center gap-2 text-[13px] font-semibold">
        <Icon name="pencil-line" :size="14" />
        <span>{{ sent ? 'Message sent' : (prefill?.subj ? 'Re: ' + (subj || 'Message') : 'New Message') }}</span>
        <button class="ml-auto cursor-pointer w-[22px] h-[22px] grid place-items-center bg-transparent border border-[#4A6FA0] text-white hover:bg-[#2A4978] text-[11px]"
                type="button" @click="$emit('close')">
          <Icon name="x" :size="12" />
        </button>
      </div>

      <!-- Sent confirmation -->
      <div v-if="sent" class="py-10 px-7 text-center">
        <Icon name="check-circle-2" :size="38" class="text-success" />
        <div class="text-[14px] text-ink mt-2 font-semibold">Sent</div>
        <div class="text-[12px] text-ink-sub mt-1">A copy has been saved to Sent Items.</div>
      </div>

      <template v-else>
        <div class="composer-field">
          <label>To</label>
          <input v-model="to" placeholder="recipient@example.com" />
        </div>
        <div v-if="showCc" class="composer-field">
          <label>Cc</label>
          <input v-model="cc" />
        </div>
        <div v-else class="py-1 px-2.5 border-b border-line-soft bg-panel-2">
          <a href="#" @click.prevent="showCc = true"
             class="text-[11px] text-accent-2 no-underline hover:underline">+ Add Cc / Bcc</a>
        </div>
        <div class="composer-field">
          <label>Subject</label>
          <input v-model="subj" placeholder="Subject" />
        </div>
        <!-- TinyMCE rich editor -->
        <div class="composer-field composer-rich" style="grid-template-columns:1fr">
          <textarea ref="taRef" placeholder="Write your message…"></textarea>
        </div>
        <!-- Footer actions -->
        <div class="py-2 px-2.5 bg-panel-2 border-t border-line flex items-center gap-1.5">
          <button class="tbtn" type="button"><Icon name="paperclip" :size="13" /> Attach</button>
          <button class="tbtn" type="button"><Icon name="signature" :size="13" /> Signature</button>
          <button class="tbtn" type="button"><Icon name="clock" :size="13" /> Schedule send</button>
          <div class="ml-auto flex gap-1.5">
            <button class="tbtn" type="button" @click="$emit('close')">Discard</button>
            <button class="tbtn tbtn-primary" type="button" @click="send">
              <Icon name="send" :size="13" /> Send
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
