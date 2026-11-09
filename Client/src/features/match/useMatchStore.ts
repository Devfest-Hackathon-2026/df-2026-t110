import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  createMatch,
  createProgram,
  getBattlePrograms,
  getMe,
  MatchApiError,
  setBattlePrograms,
  subscribeMatchResult,
  type MatchRecord,
  type MatchResult,
  type MatchUser,
  type ProgramRecord,
} from './matchApi.ts'
import type { PicoProgram } from '../picolang/language/picolang.ts'
import { replayFrames, type ReplayFrame } from './matchReplay.ts'

type ProgramIds = [string, string, string]
type ProgramSummary = Pick<ProgramRecord, 'id' | 'name'>
const emptyIds = (): ProgramIds => ['', '', '']
const matchKey = (id: string) => `match:record:${id}`
const programKey = 'battle:created-programs'

function cachedPrograms(): ProgramSummary[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(programKey) ?? '[]')
    if (!Array.isArray(value)) return []
    return value.filter(
      (item): item is ProgramSummary =>
        item !== null &&
        typeof item === 'object' &&
        typeof item.id === 'string' &&
        typeof item.name === 'string',
    )
  } catch {
    return []
  }
}

function rememberProgram(program: ProgramSummary): void {
  try {
    const previous = cachedPrograms().filter((item) => item.id !== program.id)
    localStorage.setItem(programKey, JSON.stringify([...previous, program].slice(-100)))
  } catch {
    // The program remains available for this page even when storage is disabled.
  }
}

export const useMatchStore = defineStore('match', () => {
  const programs = ref<ProgramSummary[]>([])
  const selectedProgramIds = ref<ProgramIds>(emptyIds())
  const registeredProgramIds = ref<ProgramIds | null>(null)
  const match = ref<MatchRecord | null>(null)
  const result = ref<MatchResult | null>(null)
  const frames = ref<ReplayFrame[]>([])
  const user = ref<MatchUser | null>(null)
  const authStatus = ref<'checking' | 'signed-in' | 'guest'>('checking')
  const busy = ref(false)
  const error = ref('')
  const phase = computed(() => (result.value ? 'finished' : match.value ? 'running' : 'preparing'))
  const canStart = computed(() => selectedProgramIds.value.every(Boolean))
  const settingsSaved = computed(
    () => registeredProgramIds.value?.every((id, index) => id === selectedProgramIds.value[index]) ?? false,
  )
  let stopEvents: (() => void) | undefined
  let loadController: AbortController | undefined
  let authRequest: Promise<void> | undefined

  function dispose(): void {
    stopEvents?.()
    stopEvents = undefined
    loadController?.abort()
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
    if (authStatus.value !== 'signed-in') return
    loadController?.abort()
    const controller = new AbortController()
    loadController = controller
    busy.value = true
    error.value = ''
    programs.value = cachedPrograms()
    try {
      const ids = await getBattlePrograms(controller.signal)
      if (controller.signal.aborted) return
      if (!ids) {
        registeredProgramIds.value = null
        selectedProgramIds.value = emptyIds()
        return
      }
      const names = new Map(programs.value.map((program) => [program.id, program.name]))
      programs.value = [...new Set([...ids, ...names.keys()])].map((id) => ({
        id,
        name: names.get(id) ?? id,
      }))
      selectedProgramIds.value = [...ids]
      registeredProgramIds.value = [...ids]
    } catch (cause) {
      if (!controller.signal.aborted) error.value = errorText(cause)
    } finally {
      if (loadController === controller) {
        loadController = undefined
        busy.value = false
      }
    }
  }

  async function saveProgram(name: string, source: PicoProgram): Promise<void> {
    if (authStatus.value !== 'signed-in' || !name.trim()) return
    busy.value = true
    error.value = ''
    try {
      const saved = await createProgram({ name: name.trim(), source })
      programs.value = [...programs.value, { id: saved.id, name: saved.name }]
      rememberProgram(saved)
      const slot = selectedProgramIds.value.findIndex((id) => !id)
      selectedProgramIds.value[slot < 0 ? 0 : slot] = saved.id
    } catch (cause) {
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  async function saveSettings(): Promise<void> {
    if (authStatus.value !== 'signed-in' || !canStart.value) return
    busy.value = true
    error.value = ''
    try {
      const ids: ProgramIds = [...selectedProgramIds.value]
      await setBattlePrograms(ids)
      registeredProgramIds.value = ids
    } catch (cause) {
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  async function startMatch(): Promise<void> {
    if (authStatus.value !== 'signed-in' || !canStart.value || busy.value) return
    busy.value = true
    error.value = ''
    try {
      const ids: ProgramIds = [...selectedProgramIds.value]
      await setBattlePrograms(ids)
      registeredProgramIds.value = ids
      const created = await createMatch(ids)
      try {
        sessionStorage.setItem(matchKey(created.id), JSON.stringify(created))
      } catch {
        // The runId is also kept in the URL for reloads when storage is disabled.
      }
      window.history.pushState(
        null,
        '',
        `/matches/${encodeURIComponent(created.id)}?runId=${encodeURIComponent(created.runId)}`,
      )
      watchMatch(created.id, created)
    } catch (cause) {
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  function watchMatch(id: string, record?: MatchRecord): void {
    stopEvents?.()
    stopEvents = undefined
    let saved: MatchRecord | null = null
    try {
      const raw = sessionStorage.getItem(matchKey(id))
      if (raw) {
        const candidate = JSON.parse(raw) as MatchRecord
        if (candidate.id === id && typeof candidate.runId === 'string') saved = candidate
      }
    } catch {
      // The runId in the URL is sufficient to retrieve the result.
    }
    const runId =
      record?.runId ?? saved?.runId ?? new URLSearchParams(window.location.search).get('runId') ?? ''
    match.value = record ?? saved ?? { id, runId, fieldId: '', opponentName: '' }
    result.value = null
    frames.value = []
    error.value = ''
    if (!runId) {
      error.value = 'この対戦の結果を開けません。対戦を開始した画面から開き直してください。'
      return
    }
    stopEvents = subscribeMatchResult(
      runId,
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

  function reconnect(): void {
    if (match.value?.runId) watchMatch(match.value.id, match.value)
  }

  return {
    programs,
    selectedProgramIds,
    registeredProgramIds,
    match,
    result,
    frames,
    user,
    authStatus,
    busy,
    error,
    phase,
    canStart,
    settingsSaved,
    loadMe,
    loadPrograms,
    saveProgram,
    saveSettings,
    startMatch,
    watchMatch,
    reconnect,
    dispose,
  }
})

function errorText(cause: unknown): string {
  if (cause instanceof MatchApiError) {
    if (cause.status === 401) return 'ログインが必要です。'
    if (cause.status === 403) return '表示名と対戦用プログラムの設定を確認してください。'
    if (cause.status === 404) return '指定したプログラムが見つかりません。'
    if (cause.status === 409) return '対戦できる相手がまだいません。'
    if (cause.status === 501) return 'この機能は現在利用できません。'
  }
  return cause instanceof Error ? cause.message : '通信に失敗しました。'
}
