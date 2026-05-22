<template>
  <div class="sidebar-wrapper">
    <!-- Left Navigation Bar (Elastic style) -->
    <div class="nav-bar">
      <div class="nav-logo">
        <span class="logo-icon"><i class="pi pi-envelope"></i></span>
      </div>
      <div class="nav-compose">
        <button class="nav-compose-btn" @click="mailStore.openCompose('new')" v-tooltip.right="'Criar E-mail'">
          <i class="pi pi-pencil"></i>
        </button>
      </div>
      <nav class="nav-menu">
        <router-link to="/mail/INBOX" class="nav-menu-item" :class="{ active: route.path.startsWith('/mail/') }" v-tooltip.right="'E-mail'">
          <i class="pi pi-inbox"></i>
        </router-link>
        <router-link to="/contacts" class="nav-menu-item" v-tooltip.right="'Contatos'">
          <i class="pi pi-users"></i>
        </router-link>
        <router-link to="/settings" class="nav-menu-item" v-tooltip.right="'Configurações'">
          <i class="pi pi-cog"></i>
        </router-link>
      </nav>
      <div class="nav-bar-bottom">
        <div class="nav-user-avatar" v-if="authStore.user" v-tooltip.right="authStore.user.username">
          {{ authStore.user.username[0].toUpperCase() }}
        </div>
        <button class="nav-menu-item logout-btn" @click="handleLogout" v-tooltip.right="'Sair'">
          <i class="pi pi-sign-out"></i>
        </button>
      </div>
    </div>

    <!-- Right Folders Bar -->
    <div class="folders-bar">
      <div class="folders-header">
        <span class="user-email" v-if="authStore.user">{{ authStore.user.username }}</span>
        <i class="pi pi-ellipsis-v options-icon"></i>
      </div>
      <nav class="folders-list" v-if="!foldersStore.loading">
        <div class="folders-title">Pastas</div>
        <router-link
          v-for="folder in foldersStore.folders"
          :key="folder.Name"
          :to="`/mail/${encodeURIComponent(folder.Name)}`"
          class="folder-item"
          :class="{ active: route.params.mailbox === folder.Name }"
        >
          <i :class="foldersStore.folderIcon(folder.Name)" class="folder-icon"></i>
          <span class="folder-label">{{ folder.Name }}</span>
          <span v-if="foldersStore.getCount(folder.Name) > 0" class="folder-badge">
            {{ foldersStore.getCount(folder.Name) }}
          </span>
        </router-link>
      </nav>
      <div class="folders-spinner" v-else>
        <ProgressSpinner style="width: 24px; height: 24px;" strokeWidth="4" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import ProgressSpinner from 'primevue/progressspinner'
import { useFoldersStore } from '@/stores/folders.js'
import { useAuthStore } from '@/stores/auth.js'
import { useMailStore } from '@/stores/mail.js'

const router = useRouter()
const route = useRoute()
const foldersStore = useFoldersStore()
const authStore = useAuthStore()
const mailStore = useMailStore()

async function handleLogout() {
  await authStore.logout()
  router.replace('/login')
}
</script>

<style scoped>
.sidebar-wrapper {
  display: flex;
  height: 100vh;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border);
}

/* ── Far-left Navigation Bar (Elastic) ── */
.nav-bar {
  width: 68px;
  background-color: #212931;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1rem 0;
  flex-shrink: 0;
}

.nav-logo {
  margin-bottom: 1.5rem;
}

.nav-logo .logo-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  background-color: var(--color-accent);
  color: #ffffff;
  border-radius: 0;
  font-size: 1.25rem;
  box-shadow: 0 4px 6px -1px rgba(8, 117, 225, 0.3);
}

.nav-compose {
  margin-bottom: 2rem;
}

.nav-compose-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background-color: var(--color-accent);
  color: #ffffff;
  border: none;
  border-radius: 0;
  cursor: pointer;
  box-shadow: 0 4px 10px rgba(8, 117, 225, 0.4);
  transition: background-color var(--transition), transform var(--transition);
}

.nav-compose-btn:hover {
  background-color: var(--color-accent-hover);
  transform: translateY(-1px);
}

.nav-compose-btn i {
  font-size: 1.1rem;
}

.nav-menu {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  width: 100%;
  align-items: center;
  flex: 1;
}

.nav-menu-item {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  color: #94a3b8;
  border-radius: 0;
  transition: background-color var(--transition), color var(--transition);
  cursor: pointer;
  text-decoration: none;
}

.nav-menu-item:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}

.nav-menu-item.active, .nav-menu-item.router-link-active {
  background-color: rgba(255, 255, 255, 0.12);
  color: var(--color-accent);
  font-weight: bold;
}

.nav-menu-item i {
  font-size: 1.25rem;
}

.nav-bar-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  width: 100%;
}

.nav-user-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 0;
  background-color: #475569;
  color: #ffffff;
  font-weight: 700;
  font-size: 0.875rem;
  border: 2px solid #64748b;
}

.logout-btn {
  color: #ef4444;
}

.logout-btn:hover {
  background-color: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
}

/* ── Folders Bar ── */
.folders-bar {
  width: 212px;
  background-color: var(--color-surface);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.folders-header {
  height: var(--topbar-height);
  padding: 0 1rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--color-border-light);
  flex-shrink: 0;
  background-color: var(--color-surface);
}

.user-email {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.options-icon {
  color: var(--color-text-muted);
  font-size: 0.875rem;
  cursor: pointer;
}

.folders-list {
  flex: 1;
  overflow-y: auto;
  padding: 1rem 0.5rem 0.5rem;
}

.folders-title {
  font-size: 0.6875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: 0 0.75rem 0.5rem;
}

.folder-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0;
  color: var(--color-text);
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
  transition: background-color var(--transition), color var(--transition);
  margin-bottom: 0.125rem;
}

.folder-item:hover {
  background-color: var(--color-surface-2);
  color: var(--color-accent);
}

.folder-item.active {
  background-color: rgba(8, 117, 225, 0.08);
  color: var(--color-accent);
  font-weight: 600;
}

.folder-icon {
  font-size: 0.9375rem;
  color: var(--color-text-muted);
}

.folder-item.active .folder-icon {
  color: var(--color-accent);
}

.folder-badge {
  background-color: var(--color-accent);
  color: #ffffff;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.125rem 0.5rem;
  border-radius: 0;
  margin-left: auto;
}

.folders-spinner {
  display: flex;
  justify-content: center;
  padding: 2rem;
}
</style>
