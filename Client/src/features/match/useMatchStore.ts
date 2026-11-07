import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  createMatch,
  createProgram,
  listPrograms,
  MatchApiError,
  subscribeMatchResult,
  type MatchRecord,
  type MatchResult,
  type ProgramRecord,
} from './matchApi.ts'
import { replayFrames, type ReplayFrame } from './matchReplay.ts'
import type { PicoProgram } from '../picolang/picolang.ts'

export const useMatchStore = defineStore('match', () => {
  const fieldId = ref('')
  const programs = ref<ProgramRecord[]>([])
  const selectedProgramId = ref('')
  const match = ref<MatchRecord | null>(null)
  const result = ref<MatchResult | null>(null)
  const frames = ref<ReplayFrame[]>([])
  const busy = ref(false)
  const error = ref('')
  const phase = computed(() => (result.value ? 'finished' : match.value ? 'running' : 'preparing'))
  let stopEvents: (() => void) | undefined
  let loadController: AbortController | undefined

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
    error.value = ''
  }

  async function loadPrograms(): Promise<void> {
    if (!fieldId.value) return
    loadController?.abort()
    loadController = new AbortController()
    busy.value = true
    error.value = ''
    try {
      programs.value = await listPrograms(fieldId.value, loadController.signal)
      if (!programs.value.some((item) => item.id === selectedProgramId.value))
        selectedProgramId.value = programs.value.find((item) => item.id)?.id ?? ''
    } catch (cause) {
      if (!loadController.signal.aborted) error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  async function saveProgram(name: string, source: PicoProgram): Promise<void> {
    if (!fieldId.value || !name.trim()) return
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
    if (!fieldId.value || !selectedProgramId.value) return
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
    match.value = record ?? { id, fieldId: '', selectedLayoutId: '' }
    result.value = null
    frames.value = []
    error.value = ''
    stopEvents = subscribeMatchResult(
      id,
      (received) => {
        try {
          frames.value = replayFrames(received.history)
          result.value = received
          stopEvents?.()
          stopEvents = undefined
        } catch (cause) {
          error.value = errorText(cause)
        }
      },
      (message) => {
        error.value = message
      },
    )
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
    busy,
    error,
    phase,
    setField,
    loadPrograms,
    saveProgram,
    startMatch,
    watchMatch,
    reconnect,
    dispose,
  }
})

function errorText(cause: unknown): string {
  if (cause instanceof MatchApiError) {
    if (cause.status === 401) return 'ログインが必要です。'
    if (cause.status === 403) return 'ランク対戦には表示名の設定が必要です。'
    if (cause.status === 409) return 'このフィールドには対戦相手の候補がいません。'
    if (cause.status === 501) return 'サーバーの対戦機能はまだ実装されていません。'
  }
  return cause instanceof Error ? cause.message : '通信に失敗しました。'
}
