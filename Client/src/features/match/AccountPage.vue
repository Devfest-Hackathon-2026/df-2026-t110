<script setup lang="ts">
import { onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import logInIcon from '@iconify-icons/lucide/log-in'
import { useMatchStore } from './useMatchStore.ts'

const store = useMatchStore()
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
      <p class="inner-lead">
        ゲストとしてローカル練習を利用できます。プログラムはこのブラウザに保存されます。
      </p>
      <a class="action-primary" href="/rooms">ローカル練習へ</a>
    </template>
  </main>
</template>
