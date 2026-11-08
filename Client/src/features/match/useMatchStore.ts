import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  createMatch,
  createProgram,
  getMe,
  listPrograms,
  MatchApiError,
  subscribeMatchResult,
  type MatchRecord,
  type MatchResult,
  type MatchUser,
  type ProgramRecord,
} from './matchApi.ts'
import type { PicoProgram } from '../picolang/language/picolang.ts'
import { parseBoardEventHistory, replayFrames, type ReplayFrame } from './matchReplay.ts'

export const useMatchStore = defineStore('match', () => {
  const fieldId = ref('')
  const programs = ref<ProgramRecord[]>([])
  const selectedProgramId = ref('')
  const match = ref<MatchRecord | null>(null)
  const result = ref<MatchResult | null>(null)
  const frames = ref<ReplayFrame[]>([])
  const importedBoardEvents = ref(false)
  const user = ref<MatchUser | null>(null)
  const authStatus = ref<'checking' | 'signed-in' | 'guest'>('checking')
  const busy = ref(false)
  const error = ref('')
  const phase = computed(() => (result.value ? 'finished' : match.value ? 'running' : 'preparing'))
  let stopEvents: (() => void) | undefined
  let loadController: AbortController | undefined
  let authRequest: Promise<void> | undefined

  function dispose(): void {
    stopEvents?.()
    stopEvents = undefined
    loadController?.abort()
  }

  function setField(id: string): void {
    dispose()
    fieldId.value = id.trim()
    programs.value = []
    selectedProgramId.value = ''
    match.value = null
    result.value = null
    frames.value = []
    importedBoardEvents.value = false
    error.value = ''
  }

  function loadMe(force = false): Promise<void> {
    if (authRequest) return authRequest
    if (!force && authStatus.value !== 'checking') return Promise.resolve()
    authRequest = (async () => {
      try {
        user.value = await getMe()
        authStatus.value = 'signed-in'
      } catch {
        user.value = null
        authStatus.value = 'guest'
      }
    })().finally(() => {
      authRequest = undefined
    })
    return authRequest
  }

  async function loadPrograms(): Promise<void> {
    if (!fieldId.value) return
    if (authStatus.value !== 'signed-in') return
    loadController?.abort()
    loadController = new AbortController()
    busy.value = true
    error.value = ''
    try {
      programs.value = await listPrograms(fieldId.value, loadController.signal)
      if (!programs.value.some((item) => item.id === selectedProgramId.value)) selectedProgramId.value = ''
    } catch (cause) {
      if (!loadController.signal.aborted) error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  async function saveProgram(name: string, source: PicoProgram): Promise<void> {
    if (authStatus.value !== 'signed-in' || !fieldId.value || !name.trim()) return
    busy.value = true
    error.value = ''
    try {
      const saved = await createProgram(fieldId.value, { name: name.trim(), source })
      programs.value = [...programs.value, saved]
      selectedProgramId.value = saved.id ?? ''
    } catch (cause) {
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  async function startMatch(): Promise<void> {
    if (authStatus.value !== 'signed-in' || !fieldId.value || !selectedProgramId.value) return
    busy.value = true
    error.value = ''
    try {
      const created = await createMatch(fieldId.value, selectedProgramId.value)
      watchMatch(created.id, created)
      window.history.pushState(null, '', `/matches/${encodeURIComponent(created.id)}`)
    } catch (cause) {
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  function watchMatch(id: string, record?: MatchRecord): void {
    stopEvents?.()
    importedBoardEvents.value = false
    if (import.meta.env.DEV && id === 'demo') {
      void import('./replayDemo.ts').then(({ replayDemoHistory }) => {
        if (importedBoardEvents.value) return
        match.value = { id: 'demo', fieldId: 'demo', selectedLayoutId: 'sample' }
        result.value = { history: replayDemoHistory, summary: { score: { player: 1, opponent: 0 } } }
        frames.value = replayFrames(replayDemoHistory)
        error.value = ''
      })
      return
    }
    match.value = record ?? { id, fieldId: '', selectedLayoutId: '' }
    result.value = null
    frames.value = []
    error.value = ''
    stopEvents = subscribeMatchResult(
      id,
      (received) => {
        result.value = received
        try {
          frames.value = replayFrames(received.history)
        } catch (cause) {
          error.value = cause instanceof Error ? cause.message : '再生用の履歴を読み取れませんでした。'
        }
        stopEvents?.()
        stopEvents = undefined
      },
      (message) => {
        error.value = message
      },
    )
  }

  function importBoardEvents(json: string): void {
    const history = parseBoardEventHistory(json)
    const nextFrames = replayFrames(history)
    if (nextFrames.length === 0) throw new Error('再生できるfieldイベントがありません。')
    stopEvents?.()
    stopEvents = undefined
    match.value = { id: 'demo', fieldId: 'demo', selectedLayoutId: 'Boardイベント' }
    result.value = { history, summary: { score: { player: 0, opponent: 0 } } }
    frames.value = nextFrames
    importedBoardEvents.value = true
    error.value = ''
  }

  function reconnect(): void {
    if (match.value) watchMatch(match.value.id, match.value)
  }

  return {
    fieldId,
    programs,
    selectedProgramId,
    match,
    result,
    frames,
    importedBoardEvents,
    user,
    authStatus,
    busy,
    error,
    phase,
    loadMe,
    setField,
    loadPrograms,
    saveProgram,
    startMatch,
    watchMatch,
    importBoardEvents,
    reconnect,
    dispose,
  }
})

function errorText(cause: unknown): string {
  if (cause instanceof MatchApiError) {
    if (cause.status === 401) return 'ログインが必要です。'
    if (cause.status === 403)
      return 'サーバーが表示名の登録を要求しています。Googleアカウント名を使うにはサーバー側の対応が必要です。'
    if (cause.status === 409) return 'このフィールドには対戦相手の候補がいません。'
    if (cause.status === 501) return 'このAPIはサーバー側でまだ実装されていません。'
  }
  return cause instanceof Error ? cause.message : '通信に失敗しました。'
}
