<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import swordsIcon from '@iconify-icons/lucide/swords'
import playIcon from '@iconify-icons/lucide/play'
import saveIcon from '@iconify-icons/lucide/save'
import rotateCcwIcon from '@iconify-icons/lucide/rotate-ccw'
import { useMatchStore } from './useMatchStore.ts'
import { loginUrl } from './matchApi.ts'
import type { PicoProgram, PicoTypedProgram } from '../picolang/language/picolang.ts'

const BlocklyEditor = defineAsyncComponent(() => import('../picolang/editor/BlocklyEditor.vue'))
const MatchReplayScene = defineAsyncComponent(() => import('./MatchReplayScene.vue'))
const TutorAvatar = defineAsyncComponent(() => import('../tutorial/TutorAvatar.vue'))
const ProgramGenerator = defineAsyncComponent(() => import('../program-generation/ProgramGenerator.vue'))
const props = defineProps<{ matchId?: string | null }>()
const store = useMatchStore()
const programName = ref('')
const draft = ref<PicoProgram | null>(null)
const generatedDraft = ref<PicoTypedProgram | null>(null)
const generatedName = ref('')
const editorOpen = ref(false)
const editSelection = ref('')
const editingProgramId = ref('')
const editingName = ref('')
const editingSource = ref<PicoProgram | null>(null)
const editDraft = ref<PicoProgram | null>(null)
const editMessage = ref('')
const replayOpen = ref(false)
const score = computed(() => store.result?.summary.score)
const outcome = computed(() => {
  if (!score.value) return ''
  if (score.value.player === score.value.opponent) return '引き分け'
  return score.value.player > score.value.opponent ? '勝利' : '敗北'
})

onMounted(async () => {
  await store.loadMe()
  if (props.matchId) store.watchMatch(props.matchId)
  else await store.loadPrograms()
})
onBeforeUnmount(() => store.dispose())

async function save(): Promise<void> {
  if (!draft.value) return
  await store.saveProgram(programName.value, draft.value)
}
async function saveGenerated(): Promise<void> {
  if (!generatedDraft.value) return
  await store.saveProgram(generatedName.value, generatedDraft.value)
}
async function openSavedProgram(): Promise<void> {
  if (!editSelection.value) return
  editMessage.value = ''
  const program = await store.loadEditableProgram(editSelection.value)
  if (!program) return
  editingProgramId.value = program.id
  editingName.value = program.name
  editingSource.value = program.source
  editDraft.value = program.source
}
async function saveEditedProgram(): Promise<void> {
  if (!editingProgramId.value || !editDraft.value) return
  editMessage.value = ''
  if (await store.updateExistingProgram(editingProgramId.value, editingName.value, editDraft.value))
    editMessage.value = '変更を保存しました。'
}
function changeEditSelection(): void {
  editingProgramId.value = ''
  editingSource.value = null
  editDraft.value = null
  editMessage.value = ''
}
function onEditorToggle(event: Event): void {
  editorOpen.value = (event.target as HTMLDetailsElement).open
}
</script>

<template>
  <main class="match-main container">
    <header class="match-heading">
      <h1 class="heading-with-icon"><Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦</span></h1>
      <p class="furigana-text">3台のロボットにプログラムを割り当てて、ほかのプレイヤーと競います。</p>
    </header>
    <p v-if="store.error" class="match-error" role="alert">{{ store.error }}</p>

    <template v-if="store.phase === 'preparing'">
      <section class="match-account" aria-labelledby="match-account-title">
        <div class="match-section-heading"><h2 id="match-account-title">アカウント</h2></div>
        <p v-if="store.authStatus === 'checking'">ログイン状態を確認しています…</p>
        <p v-else-if="store.authStatus === 'signed-in' && store.user?.name">
          <strong>{{ store.user.name }}</strong
          >として対戦します。
        </p>
        <p v-else-if="store.authStatus === 'signed-in'">
          対戦するには表示名の設定が必要です。<a href="/account">表示名を設定する</a>
        </p>
        <template v-else>
          <p>対戦には Google アカウントでのログインが必要です。</p>
          <a class="match-login" :href="loginUrl">Googleでログイン</a>
        </template>
      </section>

      <section
        v-if="store.authStatus === 'signed-in'"
        class="match-programs"
        aria-labelledby="match-program-title"
      >
        <div class="match-section-heading"><h2 id="match-program-title">ロボットのプログラム</h2></div>
        <p class="furigana-text">
          3台それぞれに対戦用プログラムを選びます。同じプログラムを複数のロボットに使えます。
        </p>
        <p v-if="store.busy && !store.programs.length" role="status">プログラムを読み込んでいます…</p>
        <div class="match-assignments">
          <div v-for="slot in 3" :key="slot" class="match-assignment">
            <div class="match-robot-label">
              <span aria-hidden="true">●</span><strong>ロボット {{ slot }}</strong>
            </div>
            <div class="match-assignment-control">
              <label :for="`match-program-${slot}`">割り当てるプログラム</label>
              <select
                :id="`match-program-${slot}`"
                v-model="store.selectedProgramIds[slot - 1]"
                :disabled="store.busy"
              >
                <option value="">プログラムを選択</option>
                <option v-for="program in store.programs" :key="program.id" :value="program.id">
                  {{ program.name }}
                </option>
              </select>
            </div>
          </div>
        </div>
        <p v-if="!store.busy && !store.error && !store.programs.length">
          対戦用プログラムがありません。下のエディタで作成してください。
        </p>
        <div class="match-settings">
          <button
            type="button"
            :disabled="store.busy || !store.canStart || store.settingsSaved"
            @click="store.saveSettings"
          >
            <Icon :icon="saveIcon" aria-hidden="true" />対戦用の設定を保存
          </button>
          <p v-if="store.settingsSaved" role="status">3台分の設定を保存しました。</p>
          <p v-else>設定を保存すると、ほかのプレイヤーとの対戦候補になります。</p>
        </div>

        <details class="match-editor-details" @toggle="onEditorToggle">
          <summary>新しいプログラムを作る</summary>
          <div v-if="editorOpen" class="match-editor">
            <BlocklyEditor tutorial-id="match:editor" @update:program="draft = $event" />
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
                <Icon :icon="saveIcon" aria-hidden="true" />保存する
              </button>
            </div>
          </form>
        </details>
        <details v-if="store.programs.length" class="match-editor-details">
          <summary>保存済みプログラムを編集する</summary>
          <div class="match-edit-selection">
            <label for="match-edit-program">プログラム</label>
            <select
              id="match-edit-program"
              v-model="editSelection"
              :disabled="store.busy"
              @change="changeEditSelection"
            >
              <option value="">選択してください</option>
              <option v-for="program in store.programs" :key="program.id" :value="program.id">
                {{ program.name }}
              </option>
            </select>
            <button type="button" :disabled="!editSelection || store.busy" @click="openSavedProgram">
              編集する
            </button>
          </div>
          <template v-if="editingProgramId && editingSource">
            <div class="match-editor">
              <BlocklyEditor
                :key="editingProgramId"
                :tutorial-id="`match:edit:${editingProgramId}`"
                :initial-program="editingSource"
                @update:program="editDraft = $event"
              />
            </div>
            <form class="match-save-form" @submit.prevent="saveEditedProgram">
              <label for="match-edit-name">保存する名前</label>
              <div>
                <input id="match-edit-name" v-model="editingName" type="text" required maxlength="80" />
                <button type="submit" :disabled="store.busy || !editDraft">
                  <Icon :icon="saveIcon" aria-hidden="true" />変更を保存
                </button>
              </div>
            </form>
            <p v-if="editMessage" role="status">{{ editMessage }}</p>
          </template>
        </details>
        <details class="match-editor-details">
          <summary>文章からプログラムを生成して試す</summary>
          <p class="furigana-text">
            文章から対戦用のプログラムを生成します。生成結果を確認してから保存できます。
          </p>
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
                <Icon :icon="saveIcon" aria-hidden="true" />保存する
              </button>
            </div>
          </form>
        </details>
      </section>

      <section
        v-if="store.authStatus === 'signed-in'"
        class="match-start"
        aria-labelledby="match-start-title"
      >
        <div class="match-section-heading"><h2 id="match-start-title">対戦を始める</h2></div>
        <p class="furigana-text">3台分の設定で対戦を始めます。対戦可能な相手が選ばれ、結果が表示されます。</p>
        <button
          type="button"
          class="action-primary"
          :disabled="store.busy || !store.canStart"
          @click="store.startMatch"
        >
          <Icon :icon="playIcon" aria-hidden="true" />{{ store.busy ? '対戦を準備中…' : '対戦を始める' }}
        </button>
      </section>
    </template>

    <template v-else>
      <section
        class="match-status"
        :class="{ 'match-status--result': store.phase === 'finished' && score }"
        aria-live="polite"
      >
        <h2>{{ store.phase === 'finished' ? `対戦結果 · ${outcome}` : '結果を読み込んでいます' }}</h2>
        <p>対戦ID: {{ store.match?.id }}</p>
        <p v-if="store.match?.fieldId">選ばれた配置: {{ store.match.fieldId }}</p>
        <template v-if="store.phase === 'running'">
          <div v-if="!store.error" class="match-progress" aria-hidden="true"><span /></div>
          <button v-if="store.error && store.match?.runId" type="button" @click="store.reconnect">
            <Icon :icon="rotateCcwIcon" aria-hidden="true" />結果を再取得
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
            <span>{{ store.match?.opponentName || '相手' }}</span
            ><strong>{{ score.opponent }}</strong>
            <TutorAvatar variant="purple" :animated="false" />
          </div>
        </div>
        <p v-if="store.phase === 'finished'">
          ロボットの動きを3Dで再生できます。カメラの映像に重ねることもできます。
        </p>
      </section>
      <details
        v-if="store.phase === 'finished' && store.frames.length"
        class="match-replay-details"
        :open="replayOpen"
        @toggle="replayOpen = ($event.target as HTMLDetailsElement).open"
      >
        <summary>対戦を再生</summary>
        <MatchReplayScene v-if="replayOpen" :frames="store.frames" />
      </details>
      <a class="match-back" href="/rooms">新しい対戦を準備する</a>
    </template>
  </main>
</template>
