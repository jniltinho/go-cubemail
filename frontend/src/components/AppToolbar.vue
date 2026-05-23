<script setup>
import { useMailStore } from '../stores/mail'
import Icon from './Icon.vue'

const mail = useMailStore()
</script>

<template>
  <div class="h-9 bg-panel-2 border-b border-line flex items-center px-2 gap-1 flex-shrink-0">
    <button class="tbtn tbtn-primary" type="button" @click="mail.compose()">
      <Icon name="pencil-line" :size="14" /> New Message
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>
    <button class="tbtn" type="button" @click="mail.loadFromApi()">
      <Icon name="refresh-cw" :size="14" /> Refresh
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.reply()">
      <Icon name="reply" :size="14" /> Reply
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.reply()">
      <Icon name="reply-all" :size="14" /> Reply All
    </button>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.forward()">
      <Icon name="forward" :size="14" /> Forward
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.archiveMail()">
      <Icon name="archive" :size="14" /> Archive
    </button>
    <button class="tbtn tbtn-danger" type="button" :disabled="!mail.selected" @click="mail.deleteMail()">
      <Icon name="trash-2" :size="14" /> Delete
    </button>
    <div class="w-px h-[18px] bg-line mx-1"></div>
    <button class="tbtn" type="button" :disabled="!mail.selected" @click="mail.toggleRead()">
      <Icon name="circle-dot" :size="14" /> Mark read/unread
    </button>
    <div class="flex-1"></div>
    <span class="text-[12px] text-ink-sub px-1.5">
      {{ mail.visibleMails.length }} items · {{ mail.counts.inboxUnread }} unread
    </span>
  </div>
</template>
