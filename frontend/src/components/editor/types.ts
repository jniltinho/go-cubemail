/**
 * @file types.ts
 * @description TypeScript interfaces and types for the Rich Text Editor components
 * (TipTap implementation). All identifiers and documentation are in English per project policy.
 */

import type { Editor } from '@tiptap/core'
import type { Ref, ShallowRef } from 'vue'

/**
 * Props accepted by the public RichTextEditor component.
 * Mirrors the surface of the legacy TinyEditor for easy migration.
 */
export interface RichTextEditorProps {
  /** Two-way bound HTML content */
  modelValue?: string
  /** Placeholder text shown when editor is empty */
  placeholder?: string
  /** Minimum height of the editable content area (px or CSS value) */
  minHeight?: number | string
  /** Maximum height before scrolling (px or CSS value) */
  maxHeight?: number | string
  /** Disables all editing and toolbar actions */
  disabled?: boolean
  /** Makes the editor read-only while still allowing selection/scroll */
  readonly?: boolean
}

/**
 * Events emitted by RichTextEditor.
 */
export interface RichTextEditorEmits {
  /** v-model update */
  (e: 'update:modelValue', value: string): void
  /** Editor gained focus */
  (e: 'focus'): void
  /** Editor lost focus */
  (e: 'blur'): void
  /** Content changed (after any transaction) */
  (e: 'update', value: string): void
}

/**
 * Public methods exposed via defineExpose on RichTextEditor.
 * Provides compatibility with legacy getContent() usage.
 */
export interface RichTextEditorExpose {
  /** Returns the current editor HTML content */
  getContent: () => string
  /** Sets HTML content into the editor (optionally emitting update) */
  setContent: (html: string, emitUpdate?: boolean) => void
  /** Moves focus into the editable area */
  focus: () => void
  /** Returns approximate word count for the current document */
  getWordCount: () => number
}

/**
 * Shape returned by the useTipTapEditor composable.
 * Consumers receive reactive refs + helpers. The editor ref can be passed
 * directly to <EditorContent :editor="..." /> from @tiptap/vue-3.
 */
export interface UseTipTapEditorReturn {
  /** Reactive reference to the TipTap Editor instance (undefined until mounted) */
  editor: ShallowRef<Editor | undefined>
  /** True while the editor is initializing */
  isLoading: Ref<boolean>
  /** Programmatically set content (bypasses some history) */
  setContent: (html: string, emit?: boolean) => void
  /** Retrieve current HTML */
  getHTML: () => string
  /** Approximate word count (paragraphs + words) */
  getWordCount: () => number
  /** Destroy the editor instance (cleanup) */
  destroy: () => void
}

/**
 * Configuration options passed to the TipTap editor factory.
 */
export interface TipTapEditorOptions {
  /** Initial HTML content */
  content?: string
  /** Placeholder text */
  placeholder?: string
  /** Called whenever the document changes (after update) */
  onUpdate?: (html: string) => void
  /** Called when editor receives focus */
  onFocus?: () => void
  /** Called when editor loses focus */
  onBlur?: () => void
  /** Whether the editor starts in read-only mode */
  editable?: boolean
}