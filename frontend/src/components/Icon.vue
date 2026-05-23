<script setup lang="ts">
import * as Icons from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<{
  name: string
  size?: string | number
  stroke?: string | number
}>()

const IconComp = computed(() => {
  const pascal = props.name
    .split('-')
    .map(s => s.charAt(0).toUpperCase() + s.slice(1))
    .join('')
  return Icons[pascal] || Icons['HelpCircle']
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
