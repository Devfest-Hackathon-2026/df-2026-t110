<script setup lang="ts">
import { onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import houseIcon from '@iconify-icons/lucide/house'
import bookOpenIcon from '@iconify-icons/lucide/book-open'
import swordsIcon from '@iconify-icons/lucide/swords'
import trophyIcon from '@iconify-icons/lucide/trophy'
import userRoundIcon from '@iconify-icons/lucide/user-round'
import { useMatchStore } from '../features/match/useMatchStore.ts'

defineProps<{
  activePage: 'home' | 'tutorials' | 'tutorial-detail' | 'rooms' | 'ranking' | 'account' | 'missing'
}>()

const matchStore = useMatchStore()
onMounted(() => void matchStore.loadMe())
</script>

<template>
  <header class="site-nav container">
    <a class="site-logo" href="/" aria-label="Robot Field ホーム">Robot <span>Field</span></a>
    <nav aria-label="メインナビゲーション">
      <a href="/" :aria-current="activePage === 'home' ? 'page' : undefined"
        ><Icon :icon="houseIcon" aria-hidden="true" /><span>ホーム</span></a
      >
      <a
        href="/tutorials"
        :aria-current="activePage === 'tutorials' || activePage === 'tutorial-detail' ? 'page' : undefined"
        ><Icon :icon="bookOpenIcon" aria-hidden="true" /><span>教材</span></a
      >
      <a href="/rooms" :aria-current="activePage === 'rooms' ? 'page' : undefined"
        ><Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦</span></a
      >
      <a href="/ranking" :aria-current="activePage === 'ranking' ? 'page' : undefined"
        ><Icon :icon="trophyIcon" aria-hidden="true" /><span>ランキング</span></a
      >
      <a href="/account" :aria-current="activePage === 'account' ? 'page' : undefined"
        ><Icon :icon="userRoundIcon" aria-hidden="true" /><span
          v-if="matchStore.authStatus === 'signed-in'"
          class="site-account-name"
          >{{ matchStore.user?.name || '認証済み' }}</span
        ><span v-else-if="matchStore.authStatus === 'checking'">確認中</span><span v-else>未ログイン</span></a
      >
    </nav>
  </header>
</template>
