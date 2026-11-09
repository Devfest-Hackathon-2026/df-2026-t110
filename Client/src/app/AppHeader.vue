<script setup lang="ts">
import { computed, onMounted } from 'vue'
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

const accountLabel = computed(() => {
  if (matchStore.authStatus === 'checking') return 'アカウント（ログイン状態を確認中）'
  if (matchStore.authStatus === 'signed-in')
    return matchStore.user?.name
      ? `アカウント（${matchStore.user.name}でログイン中）`
      : 'アカウント（ログイン中）'
  return 'アカウント（未ログイン）'
})
</script>

<template>
  <header class="site-nav container">
    <a class="site-logo" href="/" aria-label="PicoBots ホーム">Pico<span>Bots</span></a>
    <nav aria-label="メインナビゲーション">
      <a href="/" :aria-current="activePage === 'home' ? 'page' : undefined"
        ><Icon :icon="houseIcon" aria-hidden="true" /><span>ホーム</span></a
      >
      <a
        href="/tutorials"
        :aria-current="activePage === 'tutorials' || activePage === 'tutorial-detail' ? 'page' : undefined"
        ><Icon :icon="bookOpenIcon" aria-hidden="true" /><span>チュートリアル</span></a
      >
      <a href="/rooms" :aria-current="activePage === 'rooms' ? 'page' : undefined"
        ><Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦</span></a
      >
      <a href="/ranking" :aria-current="activePage === 'ranking' ? 'page' : undefined"
        ><Icon :icon="trophyIcon" aria-hidden="true" /><span>ランキング</span></a
      >
      <a
        href="/account"
        :aria-current="activePage === 'account' ? 'page' : undefined"
        :aria-label="accountLabel"
        :title="accountLabel"
        ><Icon :icon="userRoundIcon" aria-hidden="true" /><span>アカウント</span></a
      >
    </nav>
  </header>
</template>
