<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import StageScene from './StageScene.vue'
import { normalizeField } from './stage.js'
import { SAMPLE_FIELD } from './sampleStages.js'
import { chooseLayoutIndex, resolveServerSelection, rouletteSteps } from './roulette.js'

const field = ref(normalizeField(SAMPLE_FIELD))
const layouts = computed(() => field.value.layouts)
const selected = ref(0)
const stage = computed(() => layouts.value[selected.value])
const mode = ref('preview')
const fieldId = ref('')
const json = ref(JSON.stringify(SAMPLE_FIELD, null, 2))
const error = ref('')
const source = ref('サンプルデータ')
const sourceType = ref('sample')
const loadedFieldId = ref(null)
const phase = ref('preview')
const animationEvent = ref(null)
const sceneRef = ref(null)
let rouletteTimer = 0
let matchController = null
let rouletteGeneration = 0
let eventId = 0

function cancelRoulette() {
  rouletteGeneration++
  clearTimeout(rouletteTimer)
  rouletteTimer = 0
  matchController?.abort()
  matchController = null
  if (phase.value === 'rolling' || phase.value === 'requesting') phase.value = 'preview'
}

function previewStage(index) {
  cancelRoulette()
  selected.value = index
  phase.value = 'preview'
}

async function runRoulette() {
  if (phase.value === 'rolling' || phase.value === 'requesting') return
  const generation = ++rouletteGeneration
  phase.value = 'requesting'
  animationEvent.value = null
  let winner, match = null
  try {
    if (sourceType.value === 'api') {
      const base = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
      const controller = new AbortController()
      matchController = controller
      const response = await fetch(`${base}/field/${encodeURIComponent(loadedFieldId.value)}/match`, {
        method: 'POST', signal: controller.signal,
      })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      match = await response.json()
      winner = resolveServerSelection(layouts.value, match)
    } else {
      winner = chooseLayoutIndex(layouts.value.length)
    }
  } catch (cause) {
    if (generation !== rouletteGeneration) return
    error.value = `抽選結果を取得できませんでした: ${cause.message}`
    phase.value = 'preview'
    return
  } finally {
    if (generation === rouletteGeneration) matchController = null
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
    rouletteTimer = 0
    window.dispatchEvent(new CustomEvent('robot-field:run', {
      detail: { fieldId: loadedFieldId.value ?? field.value.id, layout: layouts.value[winner], candidateIndex: winner, match },
    }))
  }
  rouletteTimer = setTimeout(advance, 80)
}

function playDemoEvent(type) {
  if (phase.value !== 'running') return
  const current = sceneRef.value?.robotCell ?? stage.value.cells.find(cell => cell.symbol === 'S')
  const event = { id: ++eventId, type }
  if (type === 'move') {
    const next = stage.value.cells.find(cell => cell.x === current.x + 1 && cell.y === current.y)
    event.to = next && !['#', '_'].includes(next.symbol) ? { x: next.x, y: next.y } : { x: current.x, y: current.y }
  }
  animationEvent.value = event
}

function receiveStageData(payload, label, kind = 'json', id = null) {
  cancelRoulette()
  field.value = normalizeField(payload)
  selected.value = 0
  phase.value = 'preview'
  animationEvent.value = null
  source.value = label
  sourceType.value = kind
  loadedFieldId.value = id
  error.value = ''
}
onBeforeUnmount(cancelRoulette)

function loadJson() {
  try { receiveStageData(JSON.parse(json.value), '入力したJSON') }
  catch (cause) { error.value = cause.message }
}

async function loadField() {
  if (!fieldId.value.trim()) { error.value = 'フィールドIDを入力してください'; return }
  try {
    const id = fieldId.value.trim()
    const base = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
    const response = await fetch(`${base}/field/${encodeURIComponent(id)}`)
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    receiveStageData(await response.json(), `GET /field/${id}`, 'api', id)
  } catch (cause) { error.value = `ステージを取得できませんでした: ${cause.message}` }
}
</script>

<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="brand"><span class="brand-icon">✳</span><span>ROBOT FIELD<small>AR STAGE VIEWER / ISSUES #1–3</small></span></div>
      <span class="status"><i /> {{ mode === 'camera' ? 'CAMERA' : '3D PREVIEW' }}</span>
    </header>

    <div class="viewer">
      <StageScene ref="sceneRef" :stage="stage" :mode="mode" :animation-event="animationEvent" @error="error = $event" />
      <div class="scene-caption"><span>FIELD / {{ field.title }} · 配置 {{ selected + 1 }} / {{ layouts.length }}</span><strong>{{ stage.label }}</strong><small>{{ mode === 'camera' ? '画面をタップして盤面を動かす' : 'ドラッグで回転 · ホイールで拡大縮小' }}</small></div>
      <div class="scene-legend"><span><b class="land" />陸</span><span><b class="rock" />岩</span><span><b class="sea" />海</span><span><b class="goal" />ゴール</span></div>
    </div>

    <aside class="panel">
      <div class="panel-eyebrow">FIELD DATA / LOCAL VIEWER</div>
      <h1>ステージを表示</h1>
      <p class="intro">1つのフィールドの盤面形状と岩の個数候補から、あり得る配置をすべて表示します。実行する配置を抽選し、GLBロボットの動作を確認できます。</p>

      <div class="mode-switch"><button :class="{ active: mode === 'preview' }" @click="mode = 'preview'">3Dプレビュー</button><button :class="{ active: mode === 'camera' }" @click="mode = 'camera'">カメラに重ねる</button></div>

      <section class="panel-section">
        <div class="section-heading"><span>{{ field.title }} · 配置候補 {{ layouts.length }} 通り</span><small>{{ source }}</small></div>
        <p class="stage-note">盤面 {{ field.width }} × {{ field.height }} · 岩の個数候補 {{ field.rockCounts.join(' / ') }}。海や荷物も異なる配置を確認できます。</p>
        <div class="stage-list"><button v-for="(item, index) in layouts" :key="item.id + index" :class="{ active: selected === index, rolling: phase === 'rolling' && selected === index }" :disabled="phase === 'rolling'" @click="previewStage(index)"><span>{{ String(index + 1).padStart(2, '0') }}</span>{{ item.label }}<small>{{ item.width }} × {{ item.height }}</small></button></div>
        <p class="stage-note">APIで取得したフィールドは、サーバーが選んだ配置でルーレットが止まります。サンプル・貼り付けJSONではローカル抽選です。</p>
        <button class="roulette-button" :disabled="phase === 'rolling' || phase === 'requesting'" @click="runRoulette">{{ phase === 'requesting' ? 'サーバーの結果を待機中…' : phase === 'rolling' ? '抽選中…' : '配置を抽選して実行 →' }}</button>
        <p class="run-status" role="status">{{ phase === 'requesting' ? '実行する配置を問い合わせています' : phase === 'rolling' ? '配置を選んでいます' : phase === 'running' ? `実行対象: ${stage.label}` : '実行する配置は未決定です' }}</p>
      </section>

      <section class="panel-section">
        <div class="section-heading"><span>アニメーション確認</span><small>issue #1・#3</small></div>
        <p class="stage-note">抽選後にイベントを送って動作を確認できます。ゲームエンジンとの接続はイベント形式の確定後に行います。</p>
        <div class="event-actions"><button v-for="item in [{ type: 'move', label: '移動' }, { type: 'drill', label: '目のビーム' }, { type: 'get', label: '荷物回収' }, { type: 'clear', label: 'クリア' }]" :key="item.type" :disabled="phase !== 'running'" @click="playDemoEvent(item.type)">{{ item.label }}</button></div>
      </section>

      <section class="panel-section">
        <label for="field-id" class="section-heading">APIから受け取る <small>GET /field/{id}</small></label>
        <div class="field-form"><input id="field-id" v-model="fieldId" placeholder="フィールドID" @keyup.enter="loadField" /><button @click="loadField">取得</button></div>
      </section>

      <section class="panel-section">
        <label for="stage-json" class="section-heading">JSONを貼り付ける <small>API未接続時の確認用</small></label>
        <textarea id="stage-json" v-model="json" spellcheck="false" /><button class="apply-button" @click="loadJson">このデータを表示 →</button>
      </section>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p class="symbol-help">S スタート · G ゴール · . 陸 · # 岩 · _ 海 · O 荷物 · X 置き場 · ? 可変マス</p>
    </aside>
  </main>
</template>
