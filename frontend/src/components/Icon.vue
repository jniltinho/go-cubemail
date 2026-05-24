<script setup lang="ts">
/**
 * @component Icon
 * @description A generic wrapper component for Lucide icons.
 * Resolves Lucide icon component structures dynamically by converting kebab-case names
 * to PascalCase (with a generic HelpCircle fallback for safety).
 */

import * as Icons from '@lucide/vue'
import { computed } from 'vue'

/** Component properties */
const props = defineProps<{
  /** The kebab-case name of the icon to draw (e.g. 'folder-plus') */
  name: string
  /** Visual dimension size of the icon (width and height) */
  size?: string | number
  /** Line drawing weight/thickness (stroke-width) */
  stroke?: string | number
}>()

/** Dynamically resolves the Lucide Vue component reference by parsing kebab-case props */
const IconComp = computed(() => {
  const pascal = props.name
    .split('-')
    .map(s => s.charAt(0).toUpperCase() + s.slice(1))
    .join('')
  return (Icons as any)[pascal] || Icons['HelpCircle']
})
</script>

<template>
  <component
    :is="IconComp"
    :size="Number(size)"
    :stroke-width="Number(stroke)"
    class="inline-block align-[-2px]"
  />
</template>
