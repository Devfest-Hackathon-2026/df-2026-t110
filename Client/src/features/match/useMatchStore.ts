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
import type { PicoProgram } from '../picolang/picolang.ts'
import { guestFieldIds, loadGuestPrograms, runGuestProgram, saveGuestProgram } from './guestMatch.ts'

export const useMatchStore = defineStore('match', () => {
  const fieldId = ref('')
  const programs = ref<ProgramRecord[]>([])
  const selectedProgramId = ref('')
  const match = ref<MatchRecord | null>(null)
  const result = ref<MatchResult | null>(null)
  const user = ref<MatchUser | null>(null)
  const authStatus = ref<'checking' | 'signed-in' | 'guest'>('checking')
  const guestMode = computed(() => authStatus.value === 'guest')
  const isGuestMatch = ref(false)
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
    isGuestMatch.value = false
    error.value = ''
  }

  async function loadMe(): Promise<void> {
    authStatus.value = 'checking'
    try {
      user.value = await getMe()
      authStatus.value = 'signed-in'
    } catch {
      user.value = null
      authStatus.value = 'guest'
    }
  }

  async function loadPrograms(): Promise<void> {
    if (!fieldId.value) return
    if (guestMode.value) {
      programs.value = loadGuestPrograms(fieldId.value)
      if (!programs.value.some((item) => item.id === selectedProgramId.value)) selectedProgramId.value = ''
      error.value = ''
      return
    }
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
    if (!fieldId.value || !name.trim()) return
    busy.value = true
    error.value = ''
    try {
      const saved = guestMode.value
        ? saveGuestProgram(fieldId.value, name, source)
        : await createProgram(fieldId.value, { name: name.trim(), source })
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
      if (guestMode.value) {
        if (!guestFieldIds().includes(fieldId.value)) throw new Error('サンプルフィールドを選んでください。')
        const program = programs.value.find((item) => item.id === selectedProgramId.value)
        if (!program) throw new Error('プログラムを選び直してください。')
        isGuestMatch.value = true
        match.value = {
          id: `guest-${crypto.randomUUID()}`,
          fieldId: fieldId.value,
          selectedLayoutId: fieldId.value,
          status: 'running',
        }
        const started = Date.now()
        const reached = await runGuestProgram(fieldId.value, program.source)
        await new Promise<void>((resolve) => setTimeout(resolve, Math.max(0, 850 - (Date.now() - started))))
        result.value = { history: [], summary: { score: { player: reached ? 1 : 0, opponent: 0 } } }
        return
      }
      const created = await createMatch(fieldId.value, selectedProgramId.value)
      watchMatch(created.id, created)
      window.history.pushState(null, '', `/matches/${encodeURIComponent(created.id)}`)
    } catch (cause) {
      if (isGuestMatch.value) {
        match.value = null
        isGuestMatch.value = false
      }
      error.value = errorText(cause)
    } finally {
      busy.value = false
    }
  }

  function watchMatch(id: string, record?: MatchRecord): void {
    stopEvents?.()
    match.value = record ?? { id, fieldId: '', selectedLayoutId: '' }
    result.value = null
    error.value = ''
    stopEvents = subscribeMatchResult(
      id,
      (received) => {
        result.value = received
        stopEvents?.()
        stopEvents = undefined
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
    user,
    authStatus,
    guestMode,
    isGuestMatch,
    busy,
    error,
    phase,
    loadMe,
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
    if (cause.status === 403)
      return 'サーバーが表示名の登録を要求しています。Googleアカウント名を使うにはサーバー側の対応が必要です。'
    if (cause.status === 409) return 'このフィールドには対戦相手の候補がいません。'
    if (cause.status === 501) return 'このAPIはサーバー側でまだ実装されていません。'
  }
  return cause instanceof Error ? cause.message : '通信に失敗しました。'
}
