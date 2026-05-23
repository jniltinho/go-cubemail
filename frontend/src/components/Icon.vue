<script setup>
import * as Icons from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps({
  name:   { type: String, required: true },
  size:   { type: [String, Number], default: 16 },
  stroke: { type: [String, Number], default: 2 },
})

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
