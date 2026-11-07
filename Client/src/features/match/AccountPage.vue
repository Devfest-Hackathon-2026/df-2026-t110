<script setup lang="ts">
import { onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import logInIcon from '@iconify-icons/lucide/log-in'
import { loginUrl } from './matchApi.ts'
import { useMatchStore } from './useMatchStore.ts'

const store = useMatchStore()
const onAuthApp = window.location.origin === new URL(loginUrl).origin
onMounted(() => {
  void store.loadMe()
})
</script>

<template>
  <main class="inner-page container">
    <h1 class="heading-with-icon"><Icon :icon="logInIcon" aria-hidden="true" /><span>アカウント</span></h1>
    <p v-if="store.authStatus === 'checking'" class="inner-lead">ログイン状態を確認しています…</p>
    <template v-else-if="store.authStatus === 'signed-in'">
      <p class="inner-lead">{{ store.user?.name || 'ログイン中' }}として利用しています。</p>
      <a class="action-primary" href="/rooms">対戦の準備へ</a>
    </template>
    <template v-else>
      <p class="inner-lead">対戦用のアカウント情報を取得できませんでした。</p>
      <a v-if="!onAuthApp" class="action-primary" :href="loginUrl">Google認証して公開版を開く</a>
    </template>
  </main>
</template>
