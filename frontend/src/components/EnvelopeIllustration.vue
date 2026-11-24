<script setup lang="ts">
defineProps<{
  size?: number
}>()
</script>

<template>
  <svg
    :width="size ?? 132"
    :height="size ? Math.round((size * 115) / 132) : 115"
    viewBox="0 0 132 115"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    class="es-envelope"
  >
    <defs>
      <filter id="es-ls" x="-30%" y="-30%" width="160%" height="160%">
        <feDropShadow dx="0" dy="2" stdDeviation="4" style="flood-color: var(--color-overlay)" flood-opacity="0.3" />
      </filter>
      <filter id="es-es" x="-15%" y="-5%" width="130%" height="140%">
        <feDropShadow dx="0" dy="5" stdDeviation="8" style="flood-color: var(--color-overlay)" flood-opacity="0.3" />
      </filter>
      <linearGradient id="es-bl" x1="9" y1="47" x2="66" y2="110" gradientUnits="userSpaceOnUse">
        <stop offset="0%" style="stop-color: var(--color-action-hover)" />
        <stop offset="100%" style="stop-color: var(--color-accent)" />
      </linearGradient>
      <linearGradient id="es-br" x1="123" y1="47" x2="66" y2="110" gradientUnits="userSpaceOnUse">
        <stop offset="0%" style="stop-color: var(--color-action-hover)" />
        <stop offset="100%" style="stop-color: var(--color-accent)" />
      </linearGradient>
      <linearGradient id="es-fp" x1="9" y1="16" x2="123" y2="47" gradientUnits="userSpaceOnUse">
        <stop offset="0%" style="stop-color: var(--color-action-hover)" />
        <stop offset="100%" style="stop-color: var(--color-accent)" />
      </linearGradient>
    </defs>
    <polygon points="9,47 66,16 123,47" fill="url(#es-fp)" />
    <rect x="9" y="47" width="114" height="63" class="fill-surface" filter="url(#es-es)" />
    <polygon points="9,47 66,83 9,110" fill="url(#es-bl)" />
    <polygon points="123,47 66,83 123,110" fill="url(#es-br)" />
    <polygon points="9,110 66,83 123,110" class="fill-surface" />
    <line x1="9" y1="110" x2="66" y2="83" class="stroke-line-soft" stroke-width="0.75" />
    <line x1="123" y1="110" x2="66" y2="83" class="stroke-line-soft" stroke-width="0.75" />
    <rect x="41" y="9" width="50" height="53" class="fill-surface" filter="url(#es-ls)" />
    <rect x="51" y="21" width="30" height="2.5" class="fill-line" />
    <rect x="51" y="29" width="22" height="2.5" class="fill-line" />
    <rect x="51" y="37" width="26" height="2.5" class="fill-line" />
    <rect x="51" y="45" width="18" height="2.5" class="fill-line" />
  </svg>
</template>
