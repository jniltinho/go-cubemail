/**
 * @file useTipTapEditor.ts
 * @description Composable that encapsulates TipTap Editor creation, lifecycle,
 * content synchronization, and common command helpers. Designed for reuse
 * inside TipTapEditor.vue and to keep RichTextEditor.vue thin.
 * All code, comments, and identifiers are in English (project policy).
 */

import { ref, onBeforeUnmount, type Ref } from 'vue'
import type { Editor } from '@tiptap/core'
import { Editor as TipTapEditor } from '@tiptap/core'

// Core
import StarterKit from '@tiptap/starter-kit'

// Marks & Nodes for parity with legacy TinyMCE
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import TextAlign from '@tiptap/extension-text-align'
import Color from '@tiptap/extension-color'
import Highlight from '@tiptap/extension-highlight'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import Typography from '@tiptap/extension-typography'

import type { TipTapEditorOptions, UseTipTapEditorReturn } from '../types'

/**
 * Factory that creates and configures a TipTap Editor instance.
 * Returns reactive state + imperative helpers.
 */
export function useTipTapEditor(options: TipTapEditorOptions = {}): UseTipTapEditorReturn {
  const editor: Ref<Editor | null> = ref(null)
  const isLoading = ref(true)

  /** Default extensions providing near feature parity with the previous TinyMCE setup */
  const extensions = [
    StarterKit.configure({
      // We manage history explicitly if needed; StarterKit includes it.
      history: { depth: 100 },
    }),
    Underline,
    Link.configure({
      openOnClick: false, // composer: user controls link editing via toolbar
      autolink: true,
      defaultProtocol: 'https',
    }),
    TextAlign.configure({
      types: ['heading', 'paragraph'],
      alignments: ['left', 'center', 'right'],
    }),
    Color.configure({ types: ['textStyle'] }),
    Highlight.configure({ multicolor: true }),
    Placeholder.configure({
      placeholder: options.placeholder || 'Write your message…',
    }),
    Image.configure({
      inline: false,
      allowBase64: true, // support simple pasted or small images; prod may limit
    }),
    Table.configure({
      resizable: false,
    }),
    TableRow,
    TableHeader,
    TableCell,
    Typography,
  ]

  /** Create the editor */
  const instance = new TipTapEditor({
    extensions,
    content: options.content || '',
    editable: options.editable ?? true,
    onCreate() {
      isLoading.value = false
    },
    onUpdate: ({ editor: ed }) => {
      const html = ed.getHTML()
      options.onUpdate?.(html)
    },
    onFocus: () => {
      options.onFocus?.()
    },
    onBlur: () => {
      options.onBlur?.()
    },
    editorProps: {
      // Preserve reasonable paste behavior from email clients / Word
      handlePaste: (view, event, slice) => {
        // Let TipTap handle by default; can be extended for custom sanitization
        return false
      },
    },
  })

  editor.value = instance

  /**
   * Sets HTML content into the editor.
   * When emit=false the update callback is not triggered (prevents loops).
   */
  function setContent(html: string, emit = true) {
    if (!editor.value) return
    // transaction=false keeps the operation lightweight for init/quoted content
    editor.value.commands.setContent(html || '', emit)
  }

  /** Returns the current document as HTML string */
  function getHTML(): string {
    return editor.value?.getHTML() ?? ''
  }

  /**
   * Very rough word count (good enough for status display).
   * Counts words in text content; ignores markup.
   */
  function getWordCount(): number {
    if (!editor.value) return 0
    const text = editor.value.getText()
    const words = text.trim().split(/\s+/).filter(Boolean)
    return words.length
  }

  /** Cleanup on unmount */
  function destroy() {
    editor.value?.destroy()
    editor.value = null
  }

  onBeforeUnmount(() => {
    destroy()
  })

  // Return reactive refs so that Vue components can bind directly (editor ref for EditorContent)
  return {
    editor, // Ref<Editor | null>
    isLoading,
    setContent,
    getHTML,
    getWordCount,
    destroy,
  }
}