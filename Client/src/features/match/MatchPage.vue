<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import swordsIcon from '@iconify-icons/lucide/swords'
import playIcon from '@iconify-icons/lucide/play'
import saveIcon from '@iconify-icons/lucide/save'
import rotateCcwIcon from '@iconify-icons/lucide/rotate-ccw'
import { useMatchStore } from './useMatchStore.ts'
import { loginUrl } from './matchApi.ts'
import {
  isPicoProgram,
  makeStatement,
  newProgram,
  programFunctions,
  type PicoProgram,
  type PicoTypedProgram,
} from '../picolang/language/picolang.ts'
import type { generateProgram } from '../program-generation/programGenerationApi.ts'

const BlocklyEditor = defineAsyncComponent(() => import('../picolang/editor/BlocklyEditor.vue'))
const MatchReplayScene = defineAsyncComponent(() => import('./MatchReplayScene.vue'))
const TutorAvatar = defineAsyncComponent(() => import('../tutorial/TutorAvatar.vue'))
const ProgramGenerator = defineAsyncComponent(() => import('../program-generation/ProgramGenerator.vue'))
const props = defineProps<{ matchId?: string | null }>()
const store = useMatchStore()
const fieldInput = ref('')
const programName = ref('')
const draft = ref<PicoProgram | null>(null)
const generatedDraft = ref<PicoTypedProgram | null>(null)
const generatedName = ref('')
const editorOpen = ref(false)
const isDev = import.meta.env.DEV
const boardJson = ref('')
const boardImportError = ref('')
const replayOpen = ref(props.matchId === 'demo')
const score = computed(() => (store.importedBoardEvents ? null : store.result?.summary.score))
const selectedProgram = computed(() => store.programs.find((item) => item.id === store.selectedProgramId))
const outcome = computed(() => {
  if (!score.value) return ''
  if (score.value.player === score.value.opponent) return '引き分け'
  return score.value.player > score.value.opponent ? '勝利' : '敗北'
})

onMounted(async () => {
  await store.loadMe()
  if (props.matchId) {
    store.watchMatch(props.matchId)
  } else {
    fieldInput.value = sessionStorage.getItem('match:fieldId') ?? ''
    if (fieldInput.value && store.authStatus === 'signed-in') await chooseField()
  }
})
onBeforeUnmount(() => store.dispose())

function importBoardJson(): void {
  boardImportError.value = ''
  try {
    store.importBoardEvents(boardJson.value)
    replayOpen.value = true
  } catch (cause) {
    boardImportError.value = cause instanceof Error ? cause.message : 'イベントを読み込めませんでした。'
  }
}

async function importBoardFile(event: Event): Promise<void> {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  boardJson.value = await file.text()
  importBoardJson()
}

async function chooseField(): Promise<void> {
  store.setField(fieldInput.value)
  if (!store.fieldId) return
  sessionStorage.setItem('match:fieldId', store.fieldId)
  await store.loadPrograms()
}
async function save(): Promise<void> {
  if (!draft.value) return
  await store.saveProgram(programName.value, draft.value)
}
async function saveGenerated(): Promise<void> {
  if (!generatedDraft.value) return
  await store.saveProgram(generatedName.value, generatedDraft.value)
}
const previewGenerate: typeof generateProgram = async () => {
  const sample = newProgram()
  const main = programFunctions(sample).main
  main.body.stmts.unshift(makeStatement('forward', programFunctions(sample)))
  if (!isPicoProgram(sample)) throw new Error('表示確認用のプログラムを作成できませんでした。')
  return sample
}
function onEditorToggle(event: Event): void {
  editorOpen.value = (event.target as HTMLDetailsElement).open
}
</script>

<template>
  <main class="match-main container">
    <header class="match-heading">
      <p class="match-eyebrow">ROBOT FIELD / MATCH</p>
      <h1 class="heading-with-icon"><Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦</span></h1>
      <p>ロボットにプログラムを割り当てて、ほかのプレイヤーと競います。</p>
    </header>
    <p v-if="store.error" class="match-error" role="alert">{{ store.error }}</p>
    <p v-if="isDev && store.phase === 'preparing'" class="match-demo-link">
      <a href="/matches/demo">開発用の再生サンプルを見る</a>
    </p>

    <template v-if="store.phase === 'preparing'">
      <section class="match-account" aria-labelledby="match-account-title">
        <div class="match-section-heading">
          <h2 id="match-account-title">アカウント</h2>
        </div>
        <p v-if="store.authStatus === 'checking'">ログイン状態を確認しています…</p>
        <p v-else-if="store.authStatus === 'signed-in'">
          <strong>{{ store.user?.name || 'ログイン中' }}</strong
          >として対戦します。
        </p>
        <template v-else>
          <p>Google認証を以下のリンクから行ってください</p>
          <a class="match-login" :href="loginUrl">Googleでログイン</a>
        </template>
      </section>

      <section v-if="isDev && store.authStatus === 'guest'" class="match-programs">
        <h2>生成コンポーネントの表示確認</h2>
        <p>ローカル用の固定サンプルを表示します。入力した文章はAPIへ送信されず、保存もされません。</p>
        <ProgramGenerator :is-battle="true" :generate="previewGenerate" />
      </section>

      <section
        v-if="store.authStatus === 'signed-in'"
        class="match-programs"
        aria-labelledby="match-program-title"
      >
        <div class="match-section-heading">
          <h2 id="match-program-title">ロボットのプログラム</h2>
        </div>
        <p>ロボット1台にプログラムを1つ割り当てます。</p>
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
            <button type="submit" :disabled="store.busy">プログラムを表示</button>
          </div>
          <small>現行APIにはフィールド一覧がないため、IDを入力してください。</small>
        </form>
        <template v-if="store.fieldId">
          <div class="match-assignment">
            <div class="match-robot-label"><span aria-hidden="true">●</span><strong>ロボット 1</strong></div>
            <div class="match-assignment-control">
              <label for="match-program">割り当てるプログラム</label>
              <select id="match-program" v-model="store.selectedProgramId" :disabled="store.busy">
                <option value="">プログラムを選択</option>
                <option v-for="program in store.programs" :key="program.id" :value="program.id">
                  {{ program.name }}
                </option>
              </select>
              <p v-if="selectedProgram" class="match-selection">{{ selectedProgram.name }}を提出します。</p>
              <p v-else-if="!store.busy && !store.error && store.programs.length === 0">
                保存したプログラムはありません。下のエディタで作成できます。
              </p>
            </div>
          </div>
          <details class="match-editor-details" @toggle="onEditorToggle">
            <summary>新しいプログラムを作る</summary>
            <div v-if="editorOpen" class="match-editor">
              <BlocklyEditor :tutorial-id="`match:${store.fieldId}`" @update:program="draft = $event" />
            </div>
            <form v-if="editorOpen" class="match-save-form" @submit.prevent="save">
              <label for="match-program-name">保存する名前</label>
              <div>
                <input
                  id="match-program-name"
                  v-model="programName"
                  type="text"
                  required
                  maxlength="80"
                  placeholder="例：岩を避けるロボット"
                />
                <button type="submit" :disabled="store.busy || !draft">
                  <Icon :icon="saveIcon" aria-hidden="true" />保存して割り当てる
                </button>
              </div>
            </form>
          </details>
          <details class="match-editor-details">
            <summary>文章からプログラムを生成して試す</summary>
            <p>文章から対戦用のプログラムを生成します。生成結果を確認してから保存できます。</p>
            <ProgramGenerator
              :is-battle="true"
              @started="generatedDraft = null"
              @generated="generatedDraft = $event"
            />
            <form v-if="generatedDraft" class="match-save-form" @submit.prevent="saveGenerated">
              <label for="match-generated-name">生成したプログラムの名前</label>
              <div>
                <input
                  id="match-generated-name"
                  v-model="generatedName"
                  type="text"
                  required
                  maxlength="80"
                  placeholder="例：生成したロボット"
                />
                <button type="submit" :disabled="store.busy">
                  <Icon :icon="saveIcon" aria-hidden="true" />保存して割り当てる
                </button>
              </div>
            </form>
          </details>
        </template>
      </section>

      <section
        v-if="store.authStatus === 'signed-in'"
        class="match-start"
        aria-labelledby="match-start-title"
      >
        <div class="match-section-heading">
          <h2 id="match-start-title">マッチング</h2>
        </div>
        <p>相手と盤面の配置はサーバーが決定します。対戦の計算が終わるまで、この画面で待ちます。</p>
        <button
          type="button"
          class="action-primary"
          :disabled="store.busy || !store.selectedProgramId"
          @click="store.startMatch"
        >
          <Icon :icon="playIcon" aria-hidden="true" />対戦を始める
        </button>
      </section>
    </template>

    <template v-else>
      <details v-if="isDev && props.matchId === 'demo'" class="match-board-import">
        <summary>BoardイベントJSONを読み込む</summary>
        <p>
          Board.Start() の初期イベントと BoardOutput.Events
          の更新イベントを含むJSONを読み込みます。データはサーバーへ送信しません。
        </p>
        <label>
          JSONファイル
          <input type="file" accept=".json,application/json" @change="importBoardFile" />
        </label>
        <label>
          またはJSONを貼り付け
          <textarea v-model="boardJson" rows="5" spellcheck="false" />
        </label>
        <button type="button" @click="importBoardJson">このイベントを再生</button>
        <p v-if="boardImportError" role="alert">{{ boardImportError }}</p>
      </details>
      <section
        class="match-status"
        :class="{ 'match-status--result': store.phase === 'finished' && score }"
        aria-live="polite"
      >
        <p class="match-eyebrow">{{ store.phase === 'finished' ? 'MATCH COMPLETE' : 'MATCH IN PROGRESS' }}</p>
        <h2>
          {{
            store.phase === 'finished'
              ? store.importedBoardEvents
                ? 'Boardイベントの再生'
                : `対戦結果 · ${outcome}`
              : '結果を待っています'
          }}
        </h2>
        <p>対戦ID: {{ store.match?.id }}</p>
        <p v-if="store.match?.selectedLayoutId">選ばれた配置: {{ store.match.selectedLayoutId }}</p>
        <p v-if="store.importedBoardEvents">
          Go BoardのイベントJSONを再生中です。得点判定は含まれていません。
        </p>
        <template v-if="store.phase === 'running'">
          <div class="match-progress" aria-hidden="true"><span /></div>
          <p>サーバーで盤面と対戦結果を計算しています。結果が確定すると自動で表示されます。</p>
          <button v-if="store.error" type="button" @click="store.reconnect">
            <Icon :icon="rotateCcwIcon" aria-hidden="true" />結果に再接続
          </button>
        </template>
        <div v-if="score" class="match-score">
          <div class="match-score-side match-score-side--player">
            <TutorAvatar variant="green" :animated="false" />
            <span>{{ store.user?.name || 'あなた' }}</span
            ><strong>{{ score.player }}</strong>
          </div>
          <span class="match-score-separator">–</span>
          <div class="match-score-side match-score-side--opponent">
            <span>相手</span><strong>{{ score.opponent }}</strong>
            <TutorAvatar variant="purple" :animated="false" />
          </div>
        </div>
        <p v-if="store.phase === 'finished'">対戦の計算過程を、3Dまたはカメラに重ねて確認できます。</p>
      </section>
      <details
        v-if="store.phase === 'finished' && store.frames.length"
        class="match-replay-details"
        :open="replayOpen"
        @toggle="replayOpen = ($event.target as HTMLDetailsElement).open"
      >
        <summary>計算過程を見る</summary>
        <MatchReplayScene v-if="replayOpen" :frames="store.frames" />
      </details>
      <a class="match-back" href="/rooms">新しい対戦を準備する</a>
    </template>
  </main>
</template>
