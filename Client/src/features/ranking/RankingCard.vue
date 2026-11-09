<script setup lang="ts">
import type { RankingEntry } from './rankingApi.ts'

defineProps<{ entry: RankingEntry }>()

const numberFormat = new Intl.NumberFormat('ja-JP')
</script>

<template>
  <article class="ranking-card" :class="{ 'ranking-card--first': entry.rank === 1 }">
    <div class="ranking-card-position">
      <span class="ranking-card-label">順位</span>
      <strong>{{ numberFormat.format(entry.rank) }}</strong>
    </div>
    <div class="ranking-card-player">
      <span class="ranking-card-label">ユーザー</span>
      <strong :title="entry.name">{{ entry.name }}</strong>
    </div>
    <div class="ranking-card-rating">
      <span class="ranking-card-label">レート</span>
      <strong>{{ numberFormat.format(entry.rating) }}</strong>
    </div>
  </article>
</template>
