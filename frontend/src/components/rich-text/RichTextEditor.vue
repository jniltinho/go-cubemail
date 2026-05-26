<script setup lang="ts">
/**
 * @component RichTextEditor
 * @description Public rich text editor component for the email composer.
 * This is the stable contract that replaces the legacy TinyEditor.
 * It currently delegates to the TipTap implementation (TipTapEditor) but is
 * designed so that future editor engines (or an A/B choice) can be swapped
 * with minimal or no changes to consumers such as ComposerModal.vue.
 *
 * Props, events, and the exposed API closely mirror the previous TinyEditor
 * for a near zero-diff migration.
 *
 * All code, documentation, and strings are written in English per project rules.
 */

import { ref } from 'vue'
import TipTapEditor from './TipTapEditor.vue'
import type { RichTextEditorProps, RichTextEditorEmits, RichTextEditorExpose } from './types'

/** Component props (v-model friendly + optional presentation controls) */
const props = withDefaults(defineProps<RichTextEditorProps>(), {
  placeholder: 'Write your message…',
  minHeight: 380,
  maxHeight: 520,
})

/** Emitted events */
const emit = defineEmits<RichTextEditorEmits>()

/** Reference to the inner implementation for method forwarding */
const innerRef = ref<any>(null)

/** v-model two-way binding */
const model = ref(props.modelValue || '')

function handleUpdate(val: string) {
  model.value = val
  emit('update:modelValue', val)
  emit('update', val)
}

/** Public API exposed to parent (exact parity with legacy TinyEditor) */
defineExpose<RichTextEditorExpose>({
  getContent: () => innerRef.value?.getContent?.() ?? '',
  setContent: (html: string, emitUpdate = true) => {
    innerRef.value?.setContent?.(html, emitUpdate)
  },
  focus: () => innerRef.value?.focus?.(),
  getWordCount: () => innerRef.value?.getWordCount?.() ?? 0,
})

/** Forward focus/blur if needed by parent */
function onFocus() { emit('focus') }
function onBlur() { emit('blur') }
</script>

<template>
  <TipTapEditor
    ref="innerRef"
    :model-value="model"
    :placeholder="placeholder"
    :min-height="minHeight"
    :max-height="maxHeight"
    :disabled="disabled"
    :readonly="readonly"
    @update:model-value="handleUpdate"
    @focus="onFocus"
    @blur="onBlur"
    @update="emit('update', $event)"
  />
</template>