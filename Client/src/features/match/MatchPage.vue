<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import swordsIcon from '@iconify-icons/lucide/swords'
import playIcon from '@iconify-icons/lucide/play'
import saveIcon from '@iconify-icons/lucide/save'
import rotateCcwIcon from '@iconify-icons/lucide/rotate-ccw'
import { useMatchStore } from './useMatchStore.ts'
import type { PicoProgram } from '../picolang/picolang.ts'

const BlocklyEditor = defineAsyncComponent(() => import('../picolang/BlocklyEditor.vue'))
const props = defineProps<{ matchId?: string | null }>()
const store = useMatchStore()
const fieldInput = ref('')
const programName = ref('')
const draft = ref<PicoProgram | null>(null)
const editorOpen = ref(false)
const frameIndex = ref(0)
const currentFrame = computed(() => store.frames[Math.min(frameIndex.value, store.frames.length - 1)] ?? null)
const score = computed(() => store.result?.summary.score)
const outcome = computed(() => {
  if (!score.value) return ''
  if (score.value.player === score.value.opposite) return '引き分け'
  return score.value.player > score.value.opposite ? '勝利' : '敗北'
})

onMounted(() => {
  if (props.matchId) store.watchMatch(props.matchId)
})
onBeforeUnmount(() => store.dispose())

async function chooseField(): Promise<void> {
  store.setField(fieldInput.value)
  if (store.fieldId) await store.loadPrograms()
}

async function save(): Promise<void> {
  if (draft.value) await store.saveProgram(programName.value, draft.value)
}

function cellLabel(
  cell: { type: string; kind?: { type: string }; hasLuggage?: boolean },
  x: number,
  y: number,
): string {
  const robot = currentFrame.value?.robots.find((item) => item.position.x === x && item.position.y === y)
  return `${x + 1}列${y + 1}行、${cell.type === 'rock' ? '岩' : cell.type === 'sea' ? '海' : '陸'}${cell.hasLuggage ? '、荷物あり' : ''}${robot ? `、ロボット ${robot.id}` : ''}`
}

function robotAt(x: number, y: number): string | undefined {
  return currentFrame.value?.robots.find((item) => item.position.x === x && item.position.y === y)?.id
}

function onEditorToggle(event: Event): void {
  editorOpen.value = (event.target as HTMLDetailsElement).open
}
</script>

<template>
  <main class="match-main container">
    <header class="match-heading">
      <h1 class="heading-with-icon"><Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦</span></h1>
      <p>プログラムを提出し、サーバーで計算された対戦を再生します。</p>
    </header>

    <p v-if="store.error" class="match-error" role="alert">{{ store.error }}</p>

    <template v-if="store.phase === 'preparing'">
      <form class="match-field-form" @submit.prevent="chooseField">
        <label for="match-field">フィールドID</label>
        <div>
          <input
            id="match-field"
            v-model="fieldInput"
            type="text"
            required
            autocomplete="off"
            placeholder="フィールドIDを入力"
          />
          <button type="submit" :disabled="store.busy">プログラムを読み込む</button>
        </div>
      </form>

      <template v-if="store.fieldId">
        <section class="match-programs" aria-labelledby="match-program-title">
          <h2 id="match-program-title">1. プログラム</h2>
          <p>フィールド「{{ store.fieldId }}」で使うプログラムを選ぶか、新しく作成してください。</p>
          <label for="match-program">提出するプログラム</label>
          <select id="match-program" v-model="store.selectedProgramId" :disabled="store.busy">
            <option value="">選択してください</option>
            <option v-for="program in store.programs" :key="program.id" :value="program.id">
              {{ program.name }}
            </option>
          </select>
          <details class="match-editor-details" @toggle="onEditorToggle">
            <summary>新しいプログラムを作る</summary>
            <div v-if="editorOpen" class="match-editor">
              <BlocklyEditor :tutorial-id="`match:${store.fieldId}`" @update:program="draft = $event" />
            </div>
            <form v-if="editorOpen" class="match-save-form" @submit.prevent="save">
              <label for="match-program-name">プログラム名</label>
              <div>
                <input
                  id="match-program-name"
                  v-model="programName"
                  type="text"
                  required
                  maxlength="80"
                  placeholder="プログラム名"
                />
                <button type="submit" :disabled="store.busy || !draft">
                  <Icon :icon="saveIcon" aria-hidden="true" />保存
                </button>
              </div>
            </form>
          </details>
        </section>

        <section class="match-start" aria-labelledby="match-start-title">
          <h2 id="match-start-title">2. 対戦を始める</h2>
          <p>相手と配置はサーバーがランダムに選びます。岩や荷物の配置は対戦作成時に確定します。</p>
          <p class="match-api-note">
            ユーザーを指定する対戦は現在のAPIにないため、ランダム対戦のみ利用できます。
          </p>
          <button
            type="button"
            class="action-primary"
            :disabled="store.busy || !store.selectedProgramId"
            @click="store.startMatch"
          >
            <Icon :icon="playIcon" aria-hidden="true" />ランダム対戦を開始
          </button>
        </section>
      </template>
    </template>

    <template v-else>
      <section class="match-status" aria-live="polite">
        <h2>{{ store.phase === 'finished' ? `対戦結果 · ${outcome}` : 'シミュレーション中' }}</h2>
        <p>対戦ID: {{ store.match?.id }}</p>
        <p v-if="store.match?.selectedLayoutId">確定した配置: {{ store.match.selectedLayoutId }}</p>
        <p v-if="store.phase === 'running'">
          サーバーから結果を待っています。このページを再読み込みしても結果を取得できます。
        </p>
        <button v-if="store.phase === 'running' && store.error" type="button" @click="store.reconnect">
          <Icon :icon="rotateCcwIcon" aria-hidden="true" />結果に再接続
        </button>
        <div v-if="score" class="match-score">
          <span
            >あなた <strong>{{ score.player }}</strong></span
          ><span
            >相手 <strong>{{ score.opposite }}</strong></span
          >
        </div>
      </section>

      <section v-if="currentFrame" class="match-replay" aria-labelledby="match-replay-title">
        <div class="match-replay-heading">
          <h2 id="match-replay-title">盤面の再生</h2>
          <span>ターン {{ currentFrame.tick }} · {{ frameIndex + 1 }} / {{ store.frames.length }}</span>
        </div>
        <div class="match-board-scroll">
          <div
            class="match-board"
            :style="{ gridTemplateColumns: `repeat(${currentFrame.terrain[0]?.length ?? 1}, 1fr)` }"
          >
            <template v-for="(row, y) in currentFrame.terrain" :key="y">
              <div
                v-for="(cell, x) in row"
                :key="`${x}-${y}`"
                class="match-cell"
                :class="[`match-cell-${cell.type}`, { 'match-cell-luggage': cell.hasLuggage }]"
                :aria-label="cellLabel(cell, x, y)"
                :title="cellLabel(cell, x, y)"
              >
                <span v-if="robotAt(x, y)" class="match-robot">{{ robotAt(x, y)?.slice(0, 2) }}</span>
                <span v-else-if="cell.hasLuggage" aria-hidden="true">■</span>
                <span v-else-if="cell.kind?.type === 'goal'" aria-hidden="true">G</span>
                <span v-else-if="cell.territoryOwner" class="match-owner">{{ cell.territoryOwner }}</span>
              </div>
            </template>
          </div>
        </div>
        <input
          v-model.number="frameIndex"
          type="range"
          min="0"
          :max="Math.max(0, store.frames.length - 1)"
          aria-label="再生位置"
        />
        <div class="match-replay-controls">
          <button type="button" :disabled="frameIndex === 0" @click="frameIndex--">前へ</button>
          <button type="button" :disabled="frameIndex >= store.frames.length - 1" @click="frameIndex++">
            次へ
          </button>
        </div>
      </section>
      <p v-else-if="store.phase === 'finished'" class="match-error">再生できる盤面イベントがありません。</p>
      <a class="match-back" href="/rooms">新しい対戦を準備する</a>
    </template>
  </main>
</template>
