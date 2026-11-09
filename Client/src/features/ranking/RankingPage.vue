<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import trophyIcon from '@iconify-icons/lucide/trophy'
import RankingCard from './RankingCard.vue'
import { getRanking, type RankingEntry } from './rankingApi.ts'
import './ranking.css'

const entries = ref<RankingEntry[]>([])
const loading = ref(true)
const error = ref('')
const pageSize = 20
const rankingPath = /^\/ranking(?:\/([1-9]\d*))?\/?$/
const pageFromUrl = (): number => Number(rankingPath.exec(window.location.pathname)?.[1] ?? 1)
const pageUrl = (page: number): string => (page === 1 ? '/ranking' : `/ranking/${page}`)
const currentPage = ref(pageFromUrl())
const totalPages = computed(() => Math.ceil(entries.value.length / pageSize))
const firstEntryIndex = computed(() => (currentPage.value - 1) * pageSize)
const visibleEntries = computed(() =>
  entries.value.slice(firstEntryIndex.value, firstEntryIndex.value + pageSize),
)
let request: AbortController | null = null

function syncPageFromUrl(): void {
  if (!rankingPath.test(window.location.pathname)) {
    window.location.reload()
    return
  }
  currentPage.value = pageFromUrl()
}

function changePage(page: number): void {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  window.history.pushState(null, '', pageUrl(page))
  document.querySelector('.ranking-list-header')?.scrollIntoView({ block: 'start' })
}

async function load(): Promise<void> {
  request?.abort()
  const current = new AbortController()
  request = current
  loading.value = true
  error.value = ''
  try {
    entries.value = await getRanking(current.signal)
    const page = currentPage.value
    currentPage.value = Number.isSafeInteger(page) ? Math.min(Math.max(page, 1), totalPages.value || 1) : 1
    if (window.location.pathname !== pageUrl(currentPage.value)) {
      window.history.replaceState(null, '', pageUrl(currentPage.value))
    }
  } catch {
    if (current.signal.aborted) return
    error.value = 'ランキングを表示できませんでした。時間をおいて再読み込みしてください。'
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}

onMounted(() => {
  window.addEventListener('popstate', syncPageFromUrl)
  void load()
})
onBeforeUnmount(() => {
  request?.abort()
  window.removeEventListener('popstate', syncPageFromUrl)
})
</script>

<template>
  <main class="ranking-page container">
    <header class="ranking-heading">
      <h1><Icon :icon="trophyIcon" aria-hidden="true" />ランキング</h1>
      <p>対戦におけるユーザーの順位とレートの確認ができます。</p>
    </header>

    <p v-if="loading" class="ranking-message" role="status">ランキングを読み込んでいます…</p>
    <div v-else-if="error" class="ranking-message" role="alert">
      <p>{{ error }}</p>
      <button type="button" @click="load">再読み込み</button>
    </div>
    <div v-else-if="entries.length === 0" class="ranking-message">
      <p>まだランキングに登録されたユーザーはいません。</p>
      <p>表示名を設定して対戦すると、ここに順位が表示されます。</p>
      <a href="/rooms">対戦を始める</a>
    </div>
    <section v-else aria-label="ユーザーの順位一覧">
      <div class="ranking-list-header">
        <p>参加者 {{ entries.length }} 人</p>
        <p>{{ firstEntryIndex + 1 }}–{{ firstEntryIndex + visibleEntries.length }} 人目を表示</p>
      </div>
      <ol class="ranking-list">
        <li v-for="(entry, index) in visibleEntries" :key="`${entry.rank}-${entry.name}-${index}`">
          <RankingCard :entry="entry" />
        </li>
      </ol>
      <nav v-if="totalPages > 1" class="ranking-pagination" aria-label="ランキングのページ切り替え">
        <button type="button" :disabled="currentPage === 1" @click="changePage(currentPage - 1)">
          ← 前へ
        </button>
        <span aria-live="polite">{{ currentPage }} / {{ totalPages }} ページ</span>
        <button type="button" :disabled="currentPage === totalPages" @click="changePage(currentPage + 1)">
          次へ →
        </button>
      </nav>
    </section>
  </main>
</template>
