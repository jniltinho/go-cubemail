<script lang="ts">
/**
 * @component SpinnerIcon
 * @description A styled circular loader SVG spinner component.
 * Rotates dynamically (via animate-spin) to indicate background loading or saving actions.
 */
export default {
  name: 'SpinnerIcon'
}
</script>

<template>
  <svg class="w-16 h-16 animate-spin" viewBox="0 0 40 40" fill="none" xmlns="http://www.w3.org/2000/svg">
    <circle cx="20" cy="20" r="16" stroke="var(--accent)" stroke-width="4" />
    <path d="M20 4 A16 16 0 0 1 36 20" stroke="white" stroke-width="4" stroke-linecap="round" />
  </svg>
</template>
