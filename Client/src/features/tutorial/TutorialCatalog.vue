<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { getTutorials, type TutorialSummary } from './tutorialApi.ts'

const tutorials = ref<TutorialSummary[]>([])
const loading = ref(true)
const error = ref('')
let request: AbortController | null = null

async function load(): Promise<void> {
  request?.abort()
  const current = new AbortController()
  request = current
  loading.value = true
  error.value = ''
  try {
    tutorials.value = await getTutorials(current.signal)
  } catch (cause) {
    if (current.signal.aborted) return
    console.error('チュートリアル一覧の読み込みに失敗しました', cause)
    error.value = 'チュートリアル一覧を読み込めませんでした。再読み込みしてください。'
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => request?.abort())
</script>

<template>
  <p v-if="loading" role="status">チュートリアルを読み込んでいます。</p>
  <div v-else-if="error" role="alert">
    <p>{{ error }}</p>
    <button type="button" @click="load">再読み込み</button>
  </div>
  <p v-else-if="tutorials.length === 0">チュートリアルはまだありません。</p>
  <a
    v-for="(tutorial, index) in tutorials"
    :key="tutorial.id"
    class="lesson-entry"
    :href="`/tutorials/${encodeURIComponent(tutorial.id)}`"
  >
    <span>{{ String(index + 1).padStart(2, '0') }}</span>
    <span
      ><strong class="furigana-heading">{{ tutorial.title }}</strong></span
    >
    <span aria-hidden="true">↗</span>
  </a>
</template>
