<template>
  <div class="mail-layout">
    <AppSidebar />
    <main class="page-main">
      <header class="topbar">
        <h2 class="page-title"><i class="pi pi-cog"></i> Configurações</h2>
      </header>
      <div class="page-content">
        <div class="placeholder-icon"><i class="pi pi-cog"></i></div>
        <h3>Configurações</h3>
        <p>Funcionalidade em desenvolvimento</p>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import AppSidebar from '@/components/AppSidebar.vue'
import { useFoldersStore } from '@/stores/folders.js'
const foldersStore = useFoldersStore()
onMounted(() => foldersStore.load())
</script>

<style scoped>
.mail-layout { display: flex; height: 100vh; overflow: hidden; }
.page-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.topbar {
  height: var(--topbar-height);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 1.25rem;
}
.page-title { font-size: 1rem; font-weight: 600; display: flex; align-items: center; gap: 0.5rem; }
.page-content {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  gap: 0.75rem; color: var(--color-text-muted);
}
.placeholder-icon { font-size: 3rem; opacity: 0.3; }
</style>
