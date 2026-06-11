<script setup lang="ts">
defineProps<{
  size?: number
}>()
</script>

<template>
  <svg
    :width="size ?? 132"
    :height="size ? Math.round(size * 115 / 132) : 115"
    viewBox="0 0 132 115"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    class="es-envelope"
  >
    <defs>
      <filter id="es-ls" x="-30%" y="-30%" width="160%" height="160%">
        <feDropShadow dx="0" dy="2" stdDeviation="4" flood-color="rgba(0,0,0,0.10)"/>
      </filter>
      <filter id="es-es" x="-15%" y="-5%" width="130%" height="140%">
        <feDropShadow dx="0" dy="5" stdDeviation="8" flood-color="rgba(0,0,0,0.10)"/>
      </filter>
      <linearGradient id="es-bl" x1="9" y1="47" x2="66" y2="110" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stop-color="#4a7cbf"/>
        <stop offset="100%" stop-color="#1B3A6B"/>
      </linearGradient>
      <linearGradient id="es-br" x1="123" y1="47" x2="66" y2="110" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stop-color="#4a7cbf"/>
        <stop offset="100%" stop-color="#1B3A6B"/>
      </linearGradient>
      <linearGradient id="es-fp" x1="9" y1="16" x2="123" y2="47" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stop-color="#2e5fa3"/>
        <stop offset="100%" stop-color="#1B3A6B"/>
      </linearGradient>
    </defs>
    <polygon points="9,47 66,16 123,47" fill="url(#es-fp)"/>
    <rect x="9" y="47" width="114" height="63" fill="white" filter="url(#es-es)"/>
    <polygon points="9,47 66,83 9,110" fill="url(#es-bl)"/>
    <polygon points="123,47 66,83 123,110" fill="url(#es-br)"/>
    <polygon points="9,110 66,83 123,110" fill="white"/>
    <line x1="9" y1="110" x2="66" y2="83" stroke="rgba(0,0,0,0.06)" stroke-width="0.75"/>
    <line x1="123" y1="110" x2="66" y2="83" stroke="rgba(0,0,0,0.06)" stroke-width="0.75"/>
    <rect x="41" y="9" width="50" height="53" fill="white" filter="url(#es-ls)"/>
    <rect x="51" y="21" width="30" height="2.5" fill="#c8d5e8"/>
    <rect x="51" y="29" width="22" height="2.5" fill="#c8d5e8"/>
    <rect x="51" y="37" width="26" height="2.5" fill="#c8d5e8"/>
    <rect x="51" y="45" width="18" height="2.5" fill="#c8d5e8"/>
  </svg>
</template>
