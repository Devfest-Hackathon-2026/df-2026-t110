<script setup lang="ts">
import { computed, defineAsyncComponent, h, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import maximizeIcon from '@iconify-icons/lucide/maximize'
import minimizeIcon from '@iconify-icons/lucide/minimize'
import arrowLeftIcon from '@iconify-icons/lucide/arrow-left'
import arrowRightIcon from '@iconify-icons/lucide/arrow-right'
import playIcon from '@iconify-icons/lucide/play'
import rotateCcwIcon from '@iconify-icons/lucide/rotate-ccw'
import StageScene from '../stage/StageScene.vue'
import TutorAvatar from './TutorAvatar.vue'
import { isPicoProgram, type PicoProgram } from '../picolang/picolang.ts'
import { executePicolang } from '../picolang/executePicolang.ts'
import { createTutorialWorld } from './tutorialWorld.ts'
import { normalizeField } from '../stage/stage.ts'
import type { FieldData, StageLayout } from '../stage/stage.ts'
import type { AnimationEvent } from '../stage/robotAnimator.ts'
import { LOCAL_LESSONS, lessonField, localLesson } from './tutorialLessons.ts'
import { getTutorial, getTutorialField } from './tutorialApi.ts'
import type { TutorialRecord } from './tutorialApi.ts'
import type { Lesson } from './tutorialLessons.ts'

const BlocklyEditor = defineAsyncComponent({
  loader: () => import('../picolang/BlocklyEditor.vue'),
  loadingComponent: {
    render: () => h('p', { class: 'tutorial-editor-loading', role: 'status' }, 'エディタを読み込んでいます…'),
  },
  errorComponent: {
    render: () =>
      h(
        'p',
        { class: 'tutorial-editor-loading', role: 'alert' },
        'エディタを読み込めませんでした。ページを再読み込みしてください。',
      ),
  },
})

const props = defineProps<{ tutorialId: string }>()
const tutorial = ref<TutorialRecord | Lesson | null>(null)
const field = ref<FieldData | null>(null)
const source = ref('')
const loadState = ref('loading')
const loadMessage = ref('')
const sceneError = ref('')
const mode = ref<'preview' | 'camera'>('preview')
const modeBeforeHide = ref<'preview' | 'camera' | null>(null)
const layoutIndex = ref(0)
const guideIndex = ref(0)
const sceneRef = ref<{ reset: () => void; resetRobot: () => void; refreshVisuals: () => void } | null>(null)
const activeStage = ref<StageLayout | null>(null)
const animationEvent = ref<AnimationEvent | null>(null)
const runState = ref<'idle' | 'running' | 'finished' | 'error'>('idle')
const runMessage = ref('')
let runController: AbortController | null = null
let runGeneration = 0
let animationId = 0
const stageRef = ref<HTMLElement | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)
const viewRef = ref<HTMLElement | null>(null)
const editorWidth = ref(56)
const stageHeight = ref(66)
const stageFullscreen = ref(false)
const stageHidden = ref(false)
const picolangProgram = ref<PicoProgram | null>(null)
let resizing = false
const resizingStage = ref(false)
let controller: AbortController | null = null

const stage = computed(() => activeStage.value ?? field.value?.layouts[layoutIndex.value] ?? null)
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
  stopRun()
  if (document.fullscreenElement === stageRef.value) void document.exitFullscreen().catch(() => {})
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
  modeBeforeHide.value = null
  layoutIndex.value = 0
  guideIndex.value = 0
  editorWidth.value = 56
  stageHeight.value = 66
  stageFullscreen.value = false
  stageHidden.value = false
  activeStage.value = null
  animationEvent.value = null
  runState.value = 'idle'
  runMessage.value = ''

  try {
    const remote = await getTutorial(id, signal)
    if (signal.aborted) return
    if (!remote) {
      if (!local) loadState.value = 'not-found'
      return
    }
    if (typeof remote.title !== 'string' || (!remote.stage && typeof remote.fieldId !== 'string'))
      throw new Error('教材データに title または stage / fieldId がありません')
    tutorial.value = remote.fieldId === local?.fieldId ? { ...local, ...remote } : remote
    source.value = 'APIから取得'
    const remoteField = remote.stage ?? (await getTutorialField(remote.fieldId!, signal))
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
  if (activeStage.value) resetProgram()
  layoutIndex.value = (layoutIndex.value + delta + field.value.layouts.length) % field.value.layouts.length
}

function stopRun(): void {
  runGeneration++
  runController?.abort()
  runController = null
}

function waitForAnimation(duration: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException('実行を中断しました', 'AbortError'))
      return
    }
    const timer = window.setTimeout(() => {
      signal.removeEventListener('abort', abort)
      resolve()
    }, duration)
    const abort = () => {
      clearTimeout(timer)
      reject(new DOMException('実行を中断しました', 'AbortError'))
    }
    signal.addEventListener('abort', abort, { once: true })
  })
}

async function runProgram(): Promise<void> {
  if (!field.value || runState.value === 'running') return
  if (!isPicoProgram(picolangProgram.value)) {
    sceneError.value = 'プログラムの構造を確認してください。'
    return
  }
  sceneError.value = ''
  const program = JSON.parse(JSON.stringify(picolangProgram.value)) as PicoProgram
  stopRun()
  const generation = runGeneration
  const controller = new AbortController()
  runController = controller
  layoutIndex.value = Math.floor(Math.random() * field.value.layouts.length)
  activeStage.value = JSON.parse(JSON.stringify(field.value.layouts[layoutIndex.value])) as StageLayout
  animationEvent.value = null
  runState.value = 'running'
  runMessage.value = 'プログラムを実行しています'
  await nextTick()
  if (generation !== runGeneration || !activeStage.value) return
  sceneRef.value?.resetRobot()
  window.dispatchEvent(
    new CustomEvent('robot-field:tutorial-run', {
      detail: {
        tutorialId: props.tutorialId,
        layout: activeStage.value,
        candidateIndex: layoutIndex.value,
        program,
      },
    }),
  )
  const world = createTutorialWorld(activeStage.value)
  try {
    await executePicolang(
      program,
      {
        call: async (name, args, signal) => {
          const result = world.call(name, args)
          if (result.message) runMessage.value = result.message
          if (result.animation) {
            animationEvent.value = { id: ++animationId, ...result.animation }
            const duration =
              result.animation.type === 'wait' ? Math.max(1, result.animation.ticks ?? 1) * 1000 : 1000
            await waitForAnimation(duration + 40, signal)
            if (result.changed) sceneRef.value?.refreshVisuals()
          }
          return result.value
        },
      },
      controller.signal,
    )
    if (generation !== runGeneration) return
    if (world.reachedGoal) {
      animationEvent.value = { id: ++animationId, type: 'clear' }
      runMessage.value = 'ゴールに到着しました！'
    } else runMessage.value = '実行が終了しました。ゴールへの道を考えてみましょう。'
    runState.value = 'finished'
  } catch (error) {
    if (generation !== runGeneration || controller.signal.aborted) return
    runState.value = 'error'
    runMessage.value = error instanceof Error ? error.message : 'プログラムを実行できませんでした'
  } finally {
    if (generation === runGeneration) runController = null
  }
}

function resetProgram(): void {
  stopRun()
  animationEvent.value = null
  activeStage.value = null
  runState.value = 'idle'
  runMessage.value = ''
  void nextTick(() => sceneRef.value?.reset())
  window.dispatchEvent(
    new CustomEvent('robot-field:tutorial-reset', {
      detail: { tutorialId: props.tutorialId },
    }),
  )
}

function toggleStageFullscreen(): void {
  if (stageFullscreen.value) {
    stageFullscreen.value = false
    if (document.fullscreenElement === stageRef.value) void document.exitFullscreen().catch(() => {})
    return
  }
  stageFullscreen.value = true
  const request = stageRef.value?.requestFullscreen?.()
  if (request) void request.catch(() => {})
}

function hideStage(): void {
  if (stageFullscreen.value) toggleStageFullscreen()
  modeBeforeHide.value = mode.value
  stageHidden.value = true
  if (mode.value === 'camera') mode.value = 'preview'
}

function showStage(): void {
  stageHidden.value = false
  if (modeBeforeHide.value === 'camera') mode.value = 'camera'
  modeBeforeHide.value = null
}

function syncFullscreen(): void {
  if (!document.fullscreenElement) stageFullscreen.value = false
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
  stageHeight.value = Math.min(85, Math.max(30, ((bounds.bottom - clientY) / bounds.height) * 100))
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
  const delta = event.key === 'ArrowUp' ? 2 : event.key === 'ArrowDown' ? -2 : 0
  if (!delta) return
  event.preventDefault()
  stageHeight.value = Math.min(85, Math.max(30, stageHeight.value + delta))
}

watch(() => props.tutorialId, load, { immediate: true })
onMounted(() => document.addEventListener('fullscreenchange', syncFullscreen))
onBeforeUnmount(() => {
  stopRun()
  controller?.abort()
  document.removeEventListener('fullscreenchange', syncFullscreen)
  if (document.fullscreenElement === stageRef.value) void document.exitFullscreen().catch(() => {})
})
</script>

<template>
  <div ref="workspaceRef" class="tutorial-page" :style="{ '--editor-width': `${editorWidth}%` }">
    <section id="tutorial-editor" class="tutorial-editor" aria-labelledby="tutorial-editor-title">
      <div class="tutorial-editor-heading">
        <h2 id="tutorial-editor-title">コード</h2>
        <span>Picolang v0</span>
      </div>
      <div class="tutorial-editor-body">
        <BlocklyEditor :tutorial-id="tutorialId" @update:program="picolangProgram = $event" />
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
    ></div>

    <div
      id="tutorial-view"
      ref="viewRef"
      class="tutorial-view"
      :class="{ 'tutorial-view--resizing': resizingStage, 'tutorial-view--stage-hidden': stageHidden }"
      :style="{ '--stage-height': `${stageHeight}%` }"
    >
      <aside id="tutorial-explanation" class="tutorial-panel">
        <button v-if="stageHidden" type="button" class="tutorial-show-stage-button" @click="showStage">
          ARを表示
        </button>
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
            <section class="tutorial-note" aria-labelledby="tutorial-note-title">
              <h2 id="tutorial-note-title">この課題で考えること</h2>
              <p>{{ tutorial.lesson ?? 'スタートとゴールの位置を見て、必要な命令を考えよう。' }}</p>
            </section>
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
                ><Icon :icon="arrowLeftIcon" aria-hidden="true" /> 前の課題を見る</a
              ><span v-else class="tutorial-nav-unavailable" aria-disabled="true"
                ><Icon :icon="arrowLeftIcon" aria-hidden="true" /> 前の課題を見る</span
              >
              <a v-if="nextLesson" :href="`/tutorials/${nextLesson.id}`"
                >次の課題を見る <Icon :icon="arrowRightIcon" aria-hidden="true" /></a
              ><a v-else href="/tutorials">教材一覧へ <Icon :icon="arrowRightIcon" aria-hidden="true" /></a>
            </div>
          </template>
          <div v-else class="tutorial-load-message" role="alert">
            {{ loadMessage }} <button type="button" @click="load(tutorialId)">再読み込み</button>
          </div>
        </div>
      </aside>

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
      ></div>

      <section
        id="tutorial-stage"
        ref="stageRef"
        class="tutorial-stage"
        :class="{ 'tutorial-stage--fullscreen': stageFullscreen }"
        aria-label="教材のフィールド"
      >
        <StageScene
          v-if="stage"
          ref="sceneRef"
          :stage="stage"
          :mode="mode"
          :animation-event="animationEvent"
          :preview-distance="previewDistance"
          @error="sceneError = $event"
        />
        <div v-else class="tutorial-stage-empty">
          {{ loadState === 'loading' ? 'フィールドを読み込んでいます' : 'フィールドを表示できません' }}
        </div>
        <div v-if="tutorial && stage" class="tutorial-program-actions" aria-label="プログラムの操作">
          <button
            type="button"
            class="tutorial-run-button"
            :disabled="runState === 'running'"
            @click="runProgram"
          >
            <Icon :icon="playIcon" aria-hidden="true" /> {{ runState === 'running' ? '実行中' : '実行' }}
          </button>
          <button type="button" class="tutorial-reset-button" @click="resetProgram">
            <Icon :icon="rotateCcwIcon" aria-hidden="true" /> リセット
          </button>
          <button
            type="button"
            class="tutorial-fullscreen-button"
            :aria-pressed="stageFullscreen"
            :aria-label="stageFullscreen ? '全画面を閉じる' : 'ARを全画面表示'"
            :title="stageFullscreen ? '全画面を閉じる' : 'ARを全画面表示'"
            @click="toggleStageFullscreen"
          >
            <Icon :icon="stageFullscreen ? minimizeIcon : maximizeIcon" aria-hidden="true" />
          </button>
          <button type="button" class="tutorial-hide-stage-button" @click="hideStage">ARを隠す</button>
        </div>
        <p v-if="runMessage" class="tutorial-run-feedback" role="status">{{ runMessage }}</p>
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
    </div>
  </div>
</template>
