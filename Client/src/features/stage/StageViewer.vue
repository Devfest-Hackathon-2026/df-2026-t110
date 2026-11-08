<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import StageScene from './StageScene.vue'
import { normalizeField } from './stage.ts'
import { SAMPLE_FIELD } from './sampleStages.ts'
import { chooseLayoutIndex, rouletteSteps } from './roulette.ts'
import type { AnimationEvent } from './robotAnimator.ts'

const field = ref(normalizeField(SAMPLE_FIELD))
const layouts = computed(() => field.value.layouts)
const selected = ref(0)
const stage = computed(() => layouts.value[selected.value])
const mode = ref<'preview' | 'camera'>('preview')
const fieldId = ref('')
const json = ref(JSON.stringify(SAMPLE_FIELD, null, 2))
const error = ref('')
const source = ref('サンプルデータ')
const loadedFieldId = ref<string | null>(null)
const phase = ref<'preview' | 'rolling' | 'running'>('preview')
const animationEvent = ref<AnimationEvent | null>(null)
const mobileTab = ref('stage')
const demoActions: Array<{ type: AnimationEvent['type']; label: string; direction?: 'left' | 'right' }> = [
  { type: 'forward', label: '前進' },
  { type: 'back', label: '後退' },
  { type: 'rotate', direction: 'left', label: '左回転' },
  { type: 'rotate', direction: 'right', label: '右回転' },
  { type: 'wait', label: '待機' },
  { type: 'pick_up', label: '拾う' },
  { type: 'release', label: '置く' },
  { type: 'drill', label: '目のビーム' },
  { type: 'clear', label: 'クリア' },
]
let rouletteTimer: ReturnType<typeof setTimeout> | null = null
let rouletteGeneration = 0
let eventId = 0

function cancelRoulette() {
  rouletteGeneration++
  if (rouletteTimer) clearTimeout(rouletteTimer)
  rouletteTimer = null
  if (phase.value === 'rolling') phase.value = 'preview'
}

function previewStage(index: number): void {
  cancelRoulette()
  selected.value = index
  phase.value = 'preview'
}

function runRoulette() {
  if (phase.value === 'rolling') return
  const generation = ++rouletteGeneration
  animationEvent.value = null
  let winner = 0
  try {
    winner = chooseLayoutIndex(layouts.value.length)
  } catch (cause) {
    error.value = `抽選結果を取得できませんでした: ${cause instanceof Error ? cause.message : String(cause)}`
    phase.value = 'preview'
    return
  }
  if (generation !== rouletteGeneration) return
  const steps = rouletteSteps(selected.value, winner, layouts.value.length)
  phase.value = 'rolling'
  error.value = ''
  let cursor = 0
  const advance = () => {
    if (generation !== rouletteGeneration) return
    selected.value = steps[cursor]
    cursor++
    if (cursor < steps.length) {
      const progress = cursor / steps.length
      rouletteTimer = setTimeout(advance, 65 + progress * progress * 235)
      return
    }
    phase.value = 'running'
    mobileTab.value = 'actions'
    rouletteTimer = null
    window.dispatchEvent(
      new CustomEvent('robot-field:run', {
        detail: {
          fieldId: loadedFieldId.value ?? field.value.id,
          layout: layouts.value[winner],
          candidateIndex: winner,
        },
      }),
    )
  }
  rouletteTimer = setTimeout(advance, 80)
}

function playDemoEvent(action: (typeof demoActions)[number]): void {
  if (phase.value !== 'running') return
  animationEvent.value = { id: ++eventId, type: action.type, direction: action.direction }
}

function receiveStageData(payload: Record<string, unknown>, label: string, id: string | null = null): void {
  cancelRoulette()
  field.value = normalizeField(payload)
  selected.value = 0
  phase.value = 'preview'
  animationEvent.value = null
  source.value = label
  loadedFieldId.value = id
  error.value = ''
}
onBeforeUnmount(cancelRoulette)

function loadJson() {
  try {
    receiveStageData(JSON.parse(json.value), '入力したJSON')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function loadField() {
  if (!fieldId.value.trim()) {
    error.value = 'フィールドIDを入力してください'
    return
  }
  try {
    const id = fieldId.value.trim()
    const base = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
    const response = await fetch(`${base}/field/${encodeURIComponent(id)}`)
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    receiveStageData(await response.json(), `GET /field/${id}`, id)
  } catch (cause) {
    error.value = `ステージを取得できませんでした: ${cause instanceof Error ? cause.message : String(cause)}`
  }
}
</script>

<template>
  <main class="app-shell">
    <div class="viewer">
      <StageScene :stage="stage" :mode="mode" :animation-event="animationEvent" @error="error = $event" />
      <div class="scene-caption">
        <span>{{ field.title }}　配置 {{ selected + 1 }} / {{ layouts.length }}</span
        ><strong>{{ stage.label }}</strong
        ><small>{{
          mode === 'camera'
            ? 'Hiro または Kanji マーカーを映して配置を確定します'
            : 'ドラッグで回転、ホイールで拡大縮小'
        }}</small>
      </div>
      <div class="scene-legend">
        <span><b class="land" />陸</span><span><b class="rock" />岩</span><span><b class="sea" />海</span
        ><span><b class="goal" />ゴール</span>
      </div>
    </div>

    <aside class="panel">
      <h1>ステージを表示</h1>
      <p class="intro">
        1つのフィールドの盤面形状と岩の個数候補から、あり得る配置をすべて表示します。実行する配置を抽選し、GLBロボットの動作を確認できます。
      </p>

      <div class="mobile-panel-tabs" role="tablist" aria-label="ステージ操作">
        <button
          type="button"
          role="tab"
          aria-controls="stage-controls"
          :aria-selected="mobileTab === 'stage'"
          @click="mobileTab = 'stage'"
        >
          配置
        </button>
        <button
          type="button"
          role="tab"
          aria-controls="action-controls"
          :aria-selected="mobileTab === 'actions'"
          @click="mobileTab = 'actions'"
        >
          動作
        </button>
        <button
          type="button"
          role="tab"
          aria-controls="api-controls json-controls"
          :aria-selected="mobileTab === 'data'"
          @click="mobileTab = 'data'"
        >
          データ
        </button>
      </div>
      <div class="mode-switch">
        <button :class="{ active: mode === 'preview' }" @click="mode = 'preview'">3Dプレビュー</button
        ><button :class="{ active: mode === 'camera' }" @click="mode = 'camera'">カメラに重ねる</button>
      </div>
      <p v-if="error" class="error" role="alert">{{ error }}</p>

      <section
        id="stage-controls"
        class="panel-section"
        :class="{ 'mobile-active': mobileTab === 'stage' }"
        role="tabpanel"
      >
        <div class="section-heading">
          <span>{{ field.title }}、配置候補 {{ layouts.length }} 通り</span><small>{{ source }}</small>
        </div>
        <p class="stage-note">
          盤面 {{ field.width }} × {{ field.height }}。岩の個数候補 {{ field.rockCounts.join(' / ') }}。
        </p>
        <div class="stage-list">
          <button
            v-for="(item, index) in layouts"
            :key="item.id + index"
            :class="{ active: selected === index, rolling: phase === 'rolling' && selected === index }"
            :disabled="phase === 'rolling'"
            @click="previewStage(index)"
          >
            <span>{{ String(index + 1).padStart(2, '0') }}</span
            >{{ item.label }}<small>{{ item.width }} × {{ item.height }}</small>
          </button>
        </div>
        <p class="stage-note">
          このプレビューでは、取得元にかかわらず配置をローカルで抽選します。対戦は作成しません。
        </p>
        <button class="roulette-button" :disabled="phase === 'rolling'" @click="runRoulette">
          {{ phase === 'rolling' ? '抽選中…' : '配置を抽選して実行' }}
        </button>
        <p class="run-status" role="status">
          {{
            phase === 'rolling'
              ? '配置を選んでいます'
              : phase === 'running'
                ? `実行対象: ${stage.label}`
                : '実行する配置は未決定です'
          }}
        </p>
      </section>

      <section
        id="action-controls"
        class="panel-section"
        :class="{ 'mobile-active': mobileTab === 'actions' }"
        role="tabpanel"
      >
        <div class="section-heading"><span>アニメーション確認</span><small>Issue 1、3</small></div>
        <p class="stage-note">
          抽選後にイベントを送って動作を確認できます。ゲームエンジンとの接続はイベント形式の確定後に行います。
        </p>
        <div class="event-actions">
          <button
            v-for="item in demoActions"
            :key="`${item.type}-${item.direction ?? ''}`"
            :disabled="phase !== 'running'"
            @click="playDemoEvent(item)"
          >
            {{ item.label }}
          </button>
        </div>
      </section>

      <section
        id="api-controls"
        class="panel-section"
        :class="{ 'mobile-active': mobileTab === 'data' }"
        role="tabpanel"
      >
        <label for="field-id" class="section-heading">APIから受け取る <small>GET /field/{id}</small></label>
        <div class="field-form">
          <input id="field-id" v-model="fieldId" placeholder="フィールドID" @keyup.enter="loadField" /><button
            @click="loadField"
          >
            取得
          </button>
        </div>
      </section>

      <section
        id="json-controls"
        class="panel-section"
        :class="{ 'mobile-active': mobileTab === 'data' }"
        role="tabpanel"
      >
        <label for="stage-json" class="section-heading"
          >JSONを貼り付ける <small>API未接続時の確認用</small></label
        >
        <textarea id="stage-json" v-model="json" spellcheck="false" /><button
          class="apply-button"
          @click="loadJson"
        >
          このデータを表示
        </button>
      </section>
      <p class="symbol-help">S スタート · G ゴール · . 陸 · # 岩 · _ 海 · O 荷物 · X 置き場 · ? 可変マス</p>
    </aside>
  </main>
</template>
