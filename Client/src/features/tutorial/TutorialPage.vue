<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import StageScene from '../stage/StageScene.vue'
import TutorAvatar from './TutorAvatar.vue'
import { normalizeField } from '../stage/stage.ts'
import type { FieldData } from '../stage/stage.ts'
import { LOCAL_LESSONS, lessonField, localLesson } from './tutorialLessons.ts'
import { getTutorial, getTutorialField } from './tutorialApi.ts'
import type { TutorialRecord } from './tutorialApi.ts'
import type { Lesson } from './tutorialLessons.ts'

const props = defineProps<{ tutorialId: string }>()
const tutorial = ref<TutorialRecord | Lesson | null>(null)
const field = ref<FieldData | null>(null)
const source = ref('')
const loadState = ref('loading')
const loadMessage = ref('')
const sceneError = ref('')
const mode = ref<'preview' | 'camera'>('preview')
const layoutIndex = ref(0)
const guideIndex = ref(0)
const sceneRef = ref<{ reset: () => void; resetRobot: () => void } | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)
const viewRef = ref<HTMLElement | null>(null)
const editorWidth = ref(56)
const editorWidthBeforeRun = ref(56)
const stageHeight = ref(66)
const stageHeightBeforeRun = ref(66)
const stageExpanded = ref(false)
const activeCategory = ref('motion')
let resizing = false
const resizingStage = ref(false)
let controller: AbortController | null = null

const editorCategories = [
  { id: 'motion', label: '動き', color: '#337cc1', blocks: ['前進する', '後退する', '左に回る', '右に回る'] },
  { id: 'action', label: '操作', color: '#ab6a22', blocks: ['拾う', '置く', '岩を壊す', '海を埋める'] },
  { id: 'control', label: '制御', color: '#ad7420', blocks: ['もし〈条件〉なら', 'くり返す'] },
  { id: 'sense', label: '調べる', color: '#287e8f', blocks: ['前が岩？', '前が海？', 'ここがゴール？'] },
] as const
const selectedCategory = computed(
  () => editorCategories.find((category) => category.id === activeCategory.value) ?? editorCategories[0],
)

const stage = computed(() => field.value?.layouts[layoutIndex.value] ?? null)
const lessonCopy = computed(() => {
  const local = localLesson(props.tutorialId)
  return local && (!tutorial.value || tutorial.value.fieldId === local.fieldId) ? local : null
})
const guideLines = computed(() => {
  const remoteGuide = tutorial.value?.guide
  if (
    Array.isArray(remoteGuide) &&
    remoteGuide.every((line) => typeof line === 'string') &&
    remoteGuide.length
  )
    return remoteGuide
  if (lessonCopy.value) return lessonCopy.value.guide
  return [
    'この課題のフィールドを見て、ロボットをどこへ動かすか考えてみよう。',
    'まずはスタートとゴールを探してみて。進めないマスがあるかも確認しよう。',
  ]
})
const lessonPosition = computed(() => LOCAL_LESSONS.findIndex((item) => item.id === props.tutorialId))
const previousLesson = computed(() => {
  if (!lessonCopy.value || lessonPosition.value <= 0) return null
  return LOCAL_LESSONS[lessonPosition.value - 1] ?? null
})
const nextLesson = computed(() => {
  if (!lessonCopy.value) return null
  return LOCAL_LESSONS[lessonPosition.value + 1] ?? null
})
const previewDistance = computed(() =>
  Math.min(3.2, 2.75 + Math.max(stage.value?.width ?? 2, stage.value?.height ?? 1) * 0.05),
)

async function load(id: string): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  const signal = controller.signal
  const local = localLesson(id)
  tutorial.value = local ? { ...local } : null
  if (local) document.title = `${local.title} | Robot Field`
  field.value = local ? normalizeField(lessonField(local)) : null
  source.value = local ? 'サンプル教材' : ''
  loadState.value = local ? 'ready' : 'loading'
  loadMessage.value = ''
  sceneError.value = ''
  mode.value = 'preview'
  layoutIndex.value = 0
  guideIndex.value = 0
  editorWidth.value = 56
  editorWidthBeforeRun.value = 56
  stageHeight.value = 66
  stageHeightBeforeRun.value = 66
  stageExpanded.value = false

  try {
    const remote = await getTutorial(id, signal)
    if (signal.aborted) return
    if (!remote) {
      if (!local) loadState.value = 'not-found'
      return
    }
    if (typeof remote.title !== 'string' || typeof remote.fieldId !== 'string')
      throw new Error('教材データに title または fieldId がありません')
    tutorial.value = remote.fieldId === local?.fieldId ? { ...local, ...remote } : remote
    source.value = 'APIから取得'
    const remoteField = await getTutorialField(remote.fieldId, signal)
    if (signal.aborted) return
    if (!remoteField || typeof remoteField !== 'object' || Array.isArray(remoteField))
      throw new Error('教材のフィールドが見つかりません')
    field.value = normalizeField(remoteField as Record<string, unknown>)
    layoutIndex.value = 0
    loadState.value = 'ready'
    document.title = `${tutorial.value.title} | Robot Field`
  } catch (error) {
    if (signal.aborted) return
    if (local && source.value === 'サンプル教材') {
      loadState.value = 'ready'
      return
    }
    field.value = null
    loadState.value = 'error'
    loadMessage.value = error instanceof Error ? error.message : String(error)
  }
}

function changeLayout(delta: number): void {
  if (!field.value) return
  layoutIndex.value = (layoutIndex.value + delta + field.value.layouts.length) % field.value.layouts.length
}

async function runProgram(): Promise<void> {
  if (!field.value || !stage.value) return
  if (!stageExpanded.value) {
    editorWidthBeforeRun.value = editorWidth.value
    stageHeightBeforeRun.value = stageHeight.value
  }
  stageExpanded.value = true
  editorWidth.value = 32
  stageHeight.value = 80
  layoutIndex.value = Math.floor(Math.random() * field.value.layouts.length)
  await nextTick()
  sceneRef.value?.resetRobot()
  window.dispatchEvent(
    new CustomEvent('robot-field:tutorial-run', {
      detail: {
        tutorialId: props.tutorialId,
        layout: stage.value,
        candidateIndex: layoutIndex.value,
      },
    }),
  )
}

function resetProgram(): void {
  restoreWorkspaceSize()
  sceneRef.value?.reset()
  window.dispatchEvent(
    new CustomEvent('robot-field:tutorial-reset', {
      detail: { tutorialId: props.tutorialId },
    }),
  )
}

function restoreWorkspaceSize(): void {
  if (!stageExpanded.value) return
  stageExpanded.value = false
  editorWidth.value = editorWidthBeforeRun.value
  stageHeight.value = stageHeightBeforeRun.value
}

function setEditorWidth(clientX: number): void {
  const bounds = workspaceRef.value?.getBoundingClientRect()
  if (!bounds?.width) return
  editorWidth.value = Math.min(72, Math.max(28, ((clientX - bounds.left) / bounds.width) * 100))
}

function beginResize(event: PointerEvent): void {
  resizing = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  setEditorWidth(event.clientX)
}

function moveResize(event: PointerEvent): void {
  if (resizing) setEditorWidth(event.clientX)
}

function endResize(event: PointerEvent): void {
  resizing = false
  const handle = event.currentTarget as HTMLElement
  if (handle.hasPointerCapture(event.pointerId)) handle.releasePointerCapture(event.pointerId)
}

function resizeWithKeyboard(event: KeyboardEvent): void {
  const delta = event.key === 'ArrowLeft' ? -2 : event.key === 'ArrowRight' ? 2 : 0
  if (!delta) return
  event.preventDefault()
  editorWidth.value = Math.min(72, Math.max(28, editorWidth.value + delta))
}

function setStageHeight(clientY: number): void {
  const bounds = viewRef.value?.getBoundingClientRect()
  if (!bounds?.height) return
  stageHeight.value = Math.min(85, Math.max(30, ((clientY - bounds.top) / bounds.height) * 100))
}

function beginStageResize(event: PointerEvent): void {
  resizingStage.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  setStageHeight(event.clientY)
}

function moveStageResize(event: PointerEvent): void {
  if (resizingStage.value) setStageHeight(event.clientY)
}

function endStageResize(event: PointerEvent): void {
  resizingStage.value = false
  const handle = event.currentTarget as HTMLElement
  if (handle.hasPointerCapture(event.pointerId)) handle.releasePointerCapture(event.pointerId)
}

function resizeStageWithKeyboard(event: KeyboardEvent): void {
  const delta = event.key === 'ArrowUp' ? -2 : event.key === 'ArrowDown' ? 2 : 0
  if (!delta) return
  event.preventDefault()
  stageHeight.value = Math.min(85, Math.max(30, stageHeight.value + delta))
}

watch(() => props.tutorialId, load, { immediate: true })
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div
    ref="workspaceRef"
    class="tutorial-page"
    :class="{ 'tutorial-page--executing': stageExpanded }"
    :style="{ '--editor-width': `${editorWidth}%` }"
  >
    <section id="tutorial-editor" class="tutorial-editor" aria-labelledby="tutorial-editor-title">
      <div class="tutorial-editor-heading">
        <h2 id="tutorial-editor-title">コード</h2>
        <span>エディタ準備中</span>
      </div>
      <div class="tutorial-editor-body">
        <slot name="program-editor" :tutorial-id="tutorialId">
          <div class="tutorial-editor-mock">
            <nav class="tutorial-editor-categories" aria-label="命令の種類">
              <button
                v-for="category in editorCategories"
                :key="category.id"
                type="button"
                :aria-pressed="activeCategory === category.id"
                :style="{ '--category-color': category.color }"
                @click="activeCategory = category.id"
              >
                <span aria-hidden="true"></span>{{ category.label }}
              </button>
            </nav>
            <div class="tutorial-editor-palette" :style="{ '--category-color': selectedCategory.color }">
              <h3>{{ selectedCategory.label }}</h3>
              <div v-for="block in selectedCategory.blocks" :key="block" class="tutorial-example-block">
                {{ block }}
              </div>
              <p>命令の見本です。エディタ接続後に編集できます。</p>
            </div>
            <div class="tutorial-editor-workspace">
              <div class="tutorial-editor-placeholder">
                <strong>ここに命令を並べます</strong>
                <p>プログラミングエディタをこの領域に接続します。</p>
              </div>
            </div>
          </div>
        </slot>
      </div>
    </section>

    <div
      class="tutorial-resizer"
      role="separator"
      tabindex="0"
      aria-label="エディタとARの幅を調整"
      aria-orientation="vertical"
      aria-controls="tutorial-editor tutorial-view"
      aria-valuemin="28"
      aria-valuemax="72"
      :aria-valuenow="Math.round(editorWidth)"
      @pointerdown="beginResize"
      @pointermove="moveResize"
      @pointerup="endResize"
      @pointercancel="endResize"
      @keydown="resizeWithKeyboard"
    >
      <span aria-hidden="true"></span>
    </div>

    <div
      id="tutorial-view"
      ref="viewRef"
      class="tutorial-view"
      :class="{ 'tutorial-view--resizing': resizingStage }"
      :style="{ '--stage-height': `${stageHeight}%` }"
    >
      <section id="tutorial-stage" class="tutorial-stage" aria-label="教材のフィールド">
        <StageScene
          v-if="stage"
          ref="sceneRef"
          :stage="stage"
          :mode="mode"
          :preview-distance="previewDistance"
          @error="sceneError = $event"
        />
        <div v-else class="tutorial-stage-empty">
          {{ loadState === 'loading' ? 'フィールドを読み込んでいます' : 'フィールドを表示できません' }}
        </div>
        <div v-if="tutorial && stage" class="tutorial-program-actions" aria-label="プログラムの操作">
          <button type="button" class="tutorial-run-button" @click="runProgram">
            <span aria-hidden="true">▶</span> プログラムを実行
          </button>
          <button type="button" class="tutorial-reset-button" @click="resetProgram">
            <span aria-hidden="true">↺</span> リセット
          </button>
          <button
            v-if="stageExpanded"
            type="button"
            class="tutorial-restore-button"
            @click="restoreWorkspaceSize"
          >
            編集へ戻る
          </button>
        </div>
        <div v-if="stage" class="tutorial-stage-toolbar">
          <span v-if="field && field.layouts.length > 1"
            >候補 {{ layoutIndex + 1 }} / {{ field.layouts.length }}</span
          >
          <div class="tutorial-view-switch" aria-label="表示モード">
            <button type="button" :aria-pressed="mode === 'preview'" @click="mode = 'preview'">3D</button>
            <button type="button" :aria-pressed="mode === 'camera'" @click="mode = 'camera'">カメラ</button>
          </div>
        </div>
      </section>

      <div
        class="tutorial-stage-resizer"
        role="separator"
        tabindex="0"
        aria-label="ARと課題説明の高さを調整"
        aria-orientation="horizontal"
        aria-controls="tutorial-stage tutorial-explanation"
        aria-valuemin="30"
        aria-valuemax="85"
        :aria-valuenow="Math.round(stageHeight)"
        @pointerdown="beginStageResize"
        @pointermove="moveStageResize"
        @pointerup="endStageResize"
        @pointercancel="endStageResize"
        @keydown="resizeStageWithKeyboard"
      >
        <span aria-hidden="true"></span>
      </div>

      <aside id="tutorial-explanation" class="tutorial-panel">
        <div class="tutorial-panel-scroll">
          <div class="tutorial-back-row">
            <a href="/tutorials">教材一覧へ戻る</a><span v-if="source">{{ source }}</span>
          </div>
          <div v-if="loadState === 'loading'" class="tutorial-load-message" role="status">
            教材を読み込んでいます。
          </div>
          <div v-else-if="loadState === 'not-found'" class="tutorial-load-message">
            <h1>教材が見つかりません。</h1>
            <a href="/tutorials">教材一覧を見る</a>
          </div>
          <template v-else-if="tutorial">
            <header class="tutorial-heading">
              <p v-if="lessonCopy" class="tutorial-count">
                {{ String(LOCAL_LESSONS.findIndex((item) => item.id === tutorialId) + 1).padStart(2, '0') }} /
                {{ String(LOCAL_LESSONS.length).padStart(2, '0') }}
              </p>
              <h1>{{ tutorial.title }}</h1>
              <p>
                この章の目標:
                {{ tutorial.goal ?? tutorial.description ?? 'フィールドを見て、ロボットをゴールへ導こう。' }}
              </p>
            </header>
            <div class="tutor-conversation">
              <TutorAvatar />
              <div class="tutor-speech">
                <span>案内役のロボット</span>
                <p aria-live="polite">{{ guideLines[guideIndex] }}</p>
              </div>
            </div>
            <div class="tutor-stepper" aria-label="説明を切り替える">
              <button type="button" :disabled="guideIndex === 0" @click="guideIndex--">前の説明</button>
              <span>{{ guideIndex + 1 }} / {{ guideLines.length }}</span>
              <button type="button" :disabled="guideIndex === guideLines.length - 1" @click="guideIndex++">
                次の説明
              </button>
            </div>

            <section class="tutorial-note" aria-labelledby="tutorial-note-title">
              <h2 id="tutorial-note-title">この課題で考えること</h2>
              <p>{{ tutorial.lesson ?? 'スタートとゴールの位置を見て、必要な命令を考えよう。' }}</p>
            </section>
            <div v-if="(field?.layouts.length ?? 0) > 1" class="tutorial-layout-nav">
              <span>配置候補 {{ layoutIndex + 1 }} / {{ field?.layouts.length }}</span>
              <div>
                <button type="button" @click="changeLayout(-1)">前の配置</button
                ><button type="button" @click="changeLayout(1)">次の配置</button>
              </div>
              <small>個人の教材では、実行直前に候補から1つ選びます。</small>
            </div>
            <div v-if="loadState === 'error'" class="tutorial-error" role="alert">
              <span>{{ loadMessage }}</span
              ><button type="button" @click="load(tutorialId)">再読み込み</button>
            </div>
            <p v-if="sceneError" class="tutorial-error" role="alert">{{ sceneError }}</p>
            <div class="tutorial-bottom-link" aria-label="課題の移動">
              <a v-if="previousLesson" :href="`/tutorials/${previousLesson.id}`"
                ><span aria-hidden="true">←</span> 前の課題を見る</a
              ><span v-else class="tutorial-nav-unavailable" aria-disabled="true"
                ><span aria-hidden="true">←</span> 前の課題を見る</span
              >
              <a v-if="nextLesson" :href="`/tutorials/${nextLesson.id}`"
                >次の課題を見る <span aria-hidden="true">→</span></a
              ><a v-else href="/tutorials">教材一覧へ <span aria-hidden="true">→</span></a>
            </div>
          </template>
          <div v-else class="tutorial-load-message" role="alert">
            {{ loadMessage }} <button type="button" @click="load(tutorialId)">再読み込み</button>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>
