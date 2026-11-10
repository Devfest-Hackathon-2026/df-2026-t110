<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { getTutorials, type TutorialSummary } from './tutorialApi.ts'

const tutorials = ref<TutorialSummary[]>([])
const loading = ref(true)
const error = ref('')
const pageSize = 10
const pageFromUrl = (): number => {
  const value = new URLSearchParams(window.location.search).get('page')
  if (!value || !/^[1-9]\d*$/.test(value)) return 1
  const page = Number(value)
  return Number.isSafeInteger(page) ? page : 1
}
const pageUrl = (page: number): string => (page === 1 ? '/tutorials' : `/tutorials?page=${page}`)
const currentPage = ref(pageFromUrl())
const totalPages = computed(() => Math.ceil(tutorials.value.length / pageSize))
const firstTutorialIndex = computed(() => (currentPage.value - 1) * pageSize)
const visibleTutorials = computed(() =>
  tutorials.value.slice(firstTutorialIndex.value, firstTutorialIndex.value + pageSize),
)
let request: AbortController | null = null

function syncPageFromUrl(): void {
  currentPage.value = Math.min(pageFromUrl(), totalPages.value || 1)
}

function changePage(page: number): void {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  if (window.location.pathname + window.location.search !== pageUrl(page)) {
    window.history.pushState(null, '', pageUrl(page))
  }
  document.querySelector('.tutorial-catalog-list-header')?.scrollIntoView({ block: 'start' })
}

async function load(): Promise<void> {
  request?.abort()
  const current = new AbortController()
  request = current
  loading.value = true
  error.value = ''
  try {
    tutorials.value = await getTutorials(current.signal)
    currentPage.value = Math.min(pageFromUrl(), totalPages.value || 1)
    if (window.location.pathname + window.location.search !== pageUrl(currentPage.value)) {
      window.history.replaceState(null, '', pageUrl(currentPage.value))
    }
  } catch (cause) {
    if (current.signal.aborted) return
    console.error('チュートリアル一覧の読み込みに失敗しました', cause)
    error.value = 'チュートリアル一覧を読み込めませんでした。再読み込みしてください。'
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
  <p v-if="loading" role="status">チュートリアルを読み込んでいます。</p>
  <div v-else-if="error" role="alert">
    <p>{{ error }}</p>
    <button type="button" @click="load">再読み込み</button>
  </div>
  <p v-else-if="tutorials.length === 0">チュートリアルはまだありません。</p>
  <section v-else aria-label="チュートリアル一覧">
    <div class="tutorial-catalog-list-header">
      <p>{{ firstTutorialIndex + 1 }}–{{ firstTutorialIndex + visibleTutorials.length }} 件目を表示</p>
    </div>
    <a
      v-for="(tutorial, index) in visibleTutorials"
      :key="tutorial.id"
      class="lesson-entry"
      :href="`/tutorials/${encodeURIComponent(tutorial.id)}`"
    >
      <span>{{ String(firstTutorialIndex + index + 1).padStart(2, '0') }}</span>
      <span
        ><strong class="furigana-heading">{{ tutorial.title }}</strong></span
      >
      <span aria-hidden="true">↗</span>
    </a>
    <nav v-if="totalPages > 1" class="tutorial-pagination" aria-label="チュートリアルのページ切り替え">
      <button type="button" :disabled="currentPage === 1" @click="changePage(currentPage - 1)">← 前へ</button>
      <span aria-live="polite">{{ currentPage }} / {{ totalPages }} ページ</span>
      <button type="button" :disabled="currentPage === totalPages" @click="changePage(currentPage + 1)">
        次へ →
      </button>
    </nav>
  </section>
</template>
