/**
 * @file useTipTapEditor.ts
 * @description Composable that encapsulates TipTap Editor creation, lifecycle,
 * content synchronization, and common command helpers. Uses useEditor() from
 * @tiptap/vue-3 so the editor initialises after mount — keeping the modal
 * responsive on first render instead of blocking the Vue setup cycle.
 * All code, comments, and identifiers are in English (project policy).
 */

import { computed } from 'vue'
import { useEditor } from '@tiptap/vue-3'

// Core
import StarterKit from '@tiptap/starter-kit'

// Marks & Nodes for parity with legacy TinyMCE
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import TextAlign from '@tiptap/extension-text-align'
import TextStyle from '@tiptap/extension-text-style'
import Color from '@tiptap/extension-color'
import Highlight from '@tiptap/extension-highlight'
import Placeholder from '@tiptap/extension-placeholder'
import { SelectableImage } from '../extensions/selectableImage'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import { FontFamily } from '../extensions/fontFamily'
import { FontSize } from '../extensions/fontSize'
import { Indent } from '../extensions/indent'
// Typography intentionally excluded — it silently rewrites characters
// (smart quotes, em-dashes, etc.) which corrupts email body content.

import type { TipTapEditorOptions, UseTipTapEditorReturn } from '../types'

/**
 * Factory that creates and configures a TipTap Editor instance.
 * Returns reactive state + imperative helpers.
 * Must be called from within a Vue component's setup context.
 */
export function useTipTapEditor(options: TipTapEditorOptions = {}): UseTipTapEditorReturn {
  const extensions = [
    StarterKit.configure({
      history: { depth: 100 },
    }),
    Underline,
    Link.configure({
      openOnClick: false,
      autolink: true,
      defaultProtocol: 'https',
    }),
    TextAlign.configure({
      types: ['heading', 'paragraph'],
      alignments: ['left', 'center', 'right'],
    }),
    // TextStyle must come before Color/FontFamily/FontSize — they all call setMark('textStyle', ...)
    TextStyle,
    Color.configure({ types: ['textStyle'] }),
    FontFamily,
    FontSize,
    Highlight.configure({ multicolor: true }),
    Placeholder.configure({
      placeholder: options.placeholder || 'Write your message…',
    }),
    // inline: true → image sits inside a <p> element so TextAlign on the paragraph
    // controls image alignment (left/center/right). Produces email-compatible HTML:
    // <p style="text-align: center"><img ...></p>
    // SelectableImage extends the base Image extension with a ProseMirror plugin
    // that converts a click on an image into a NodeSelection, enabling visual
    // selection feedback and toolbar alignment commands.
    SelectableImage.configure({
      inline: true,
      allowBase64: true,
    }),
    Indent,
    Table.configure({ resizable: false }),
    TableRow,
    TableHeader,
    TableCell,
  ]

  // useEditor defers new Editor() to onMounted, keeping the modal render fast.
  // It also registers onBeforeUnmount cleanup automatically.
  const editor = useEditor({
    extensions,
    content: options.content || '',
    editable: options.editable ?? true,
    onUpdate: ({ editor: ed }) => {
      options.onUpdate?.(ed.getHTML())
    },
    onFocus: () => options.onFocus?.(),
    onBlur: () => options.onBlur?.(),
    editorProps: {
      handlePaste: (_view, _event, _slice) => false,
    },
  })

  const isLoading = computed(() => !editor.value)

  function setContent(html: string, emit = true) {
    editor.value?.commands.setContent(html || '', emit)
  }

  function getHTML(): string {
    return editor.value?.getHTML() ?? ''
  }

  function getWordCount(): number {
    if (!editor.value) return 0
    const text = editor.value.getText()
    return text.trim().split(/\s+/).filter(Boolean).length
  }

  function destroy() {
    editor.value?.destroy()
  }

  return {
    editor,
    isLoading,
    setContent,
    getHTML,
    getWordCount,
    destroy,
  }
}
