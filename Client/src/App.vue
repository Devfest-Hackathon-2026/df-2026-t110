<script setup>
import { computed, ref } from 'vue'
import StageScene from './StageScene.vue'
import { normalizeStages } from './stage.js'
import { SAMPLE_STAGES } from './sampleStages.js'

const stages = ref(normalizeStages(SAMPLE_STAGES))
const selected = ref(0)
const stage = computed(() => stages.value[selected.value])
const mode = ref('preview')
const fieldId = ref('')
const json = ref(JSON.stringify(SAMPLE_STAGES, null, 2))
const error = ref('')
const source = ref('サンプルデータ')

function receiveStageData(payload, label) {
  stages.value = normalizeStages(payload)
  selected.value = 0
  source.value = label
  error.value = ''
}

function loadJson() {
  try { receiveStageData(JSON.parse(json.value), '入力したJSON') }
  catch (cause) { error.value = cause.message }
}

async function loadField() {
  if (!fieldId.value.trim()) { error.value = 'フィールドIDを入力してください'; return }
  try {
    const base = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
    const response = await fetch(`${base}/field/${encodeURIComponent(fieldId.value.trim())}`)
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    receiveStageData(await response.json(), `GET /field/${fieldId.value.trim()}`)
  } catch (cause) { error.value = `ステージを取得できませんでした: ${cause.message}` }
}
</script>

<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="brand"><span class="brand-icon">✳</span><span>ROBOT FIELD<small>AR STAGE VIEWER / ISSUE #2</small></span></div>
      <span class="status"><i /> {{ mode === 'camera' ? 'CAMERA' : '3D PREVIEW' }}</span>
    </header>

    <div class="viewer">
      <StageScene :stage="stage" :mode="mode" @error="error = $event" />
      <div class="scene-caption"><span>STAGE / {{ String(selected + 1).padStart(2, '0') }}</span><strong>{{ stage.title }}</strong><small>{{ mode === 'camera' ? '画面をタップして盤面を動かす' : 'ドラッグで回転 · ホイールで拡大縮小' }}</small></div>
      <div class="scene-legend"><span><b class="land" />陸</span><span><b class="rock" />岩</span><span><b class="sea" />海</span><span><b class="goal" />ゴール</span></div>
    </div>

    <aside class="panel">
      <div class="panel-eyebrow">FIELD DATA / LOCAL VIEWER</div>
      <h1>ステージを表示</h1>
      <p class="intro">受信したマップを立体的なフィールドに変換します。ロボットやゲーム処理はまだ含みません。</p>

      <div class="mode-switch"><button :class="{ active: mode === 'preview' }" @click="mode = 'preview'">3Dプレビュー</button><button :class="{ active: mode === 'camera' }" @click="mode = 'camera'">カメラに重ねる</button></div>

      <section class="panel-section">
        <div class="section-heading"><span>候補ステージ</span><small>{{ source }}</small></div>
        <div class="stage-list"><button v-for="(item, index) in stages" :key="item.id + index" :class="{ active: selected === index }" @click="selected = index"><span>{{ String(index + 1).padStart(2, '0') }}</span>{{ item.title }}<small>{{ item.width }} × {{ item.height }}</small></button></div>
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
      <p class="symbol-help">S スタート · G ゴール · . 陸 · # 岩 · _ 海 · O 荷物 · X 置き場</p>
    </aside>
  </main>
</template>
