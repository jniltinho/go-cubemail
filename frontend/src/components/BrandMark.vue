<script setup lang="ts">
withDefaults(defineProps<{ size?: number | string }>(), { size: 22 })
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    :width="size"
    :height="size"
    viewBox="0 0 22 22"
    fill="currentColor"
    class="block shrink-0"
    aria-hidden="true"
    focusable="false"
  >
    <path d="M2 7 6 3h14l-4 4Z" opacity="0.6" />
    <path d="m16 7 4-4v12l-4 4Z" opacity="0.8" />
    <path d="M2 7h14l-7 5Z" />
    <path d="M2 9 9 14l7-5v10H2Z" />
  </svg>
</template>
