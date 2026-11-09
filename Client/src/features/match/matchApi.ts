import type { PicoProgram } from '../picolang/language/picolang.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface ProgramRecord {
  id: string
  name: string
  source: PicoProgram
  isBattle: boolean
}

export interface MatchUser {
  id?: string
  name?: string
}

export const loginUrl = `${base}/login`

export interface MatchRecord {
  id: string
  fieldId: string
  runId: string
  opponentName: string
  status?: 'ready' | 'running' | 'finished'
}

export interface MatchResult {
  history: unknown[]
  summary: { score: { player: number; opponent: number } }
}

export class MatchApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request(path: string, init: RequestInit = {}): Promise<unknown> {
  const response = await fetch(`${base}${path}`, { credentials: 'include', ...init })
  if (!response.ok) {
    let message = `API エラー (${response.status})`
    try {
      const body: unknown = await response.json()
      if (isObject(body) && typeof body.message === 'string') message = body.message
    } catch {
      // A status code is sufficient when the response has no JSON body.
    }
    throw new MatchApiError(response.status, message)
  }
  if (!response.headers.get('content-type')?.includes('application/json'))
    throw new Error('対戦データを取得できませんでした。しばらくしてから再試行してください。')
  return response.json()
}

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function parseProgram(value: unknown): ProgramRecord {
  if (
    !isObject(value) ||
    typeof value.id !== 'string' ||
    typeof value.name !== 'string' ||
    value.isBattle !== true ||
    !isObject(value.source)
  )
    throw new Error('対戦用プログラムの形式が不正です')
  return value as unknown as ProgramRecord
}

export function parseBattleProgramIds(value: unknown): [string, string, string] {
  if (!Array.isArray(value) || value.length !== 3 || value.some((id) => typeof id !== 'string' || !id))
    throw new Error('対戦用プログラム設定の形式が不正です')
  return value as [string, string, string]
}

export function parseMatchUser(value: unknown): MatchUser {
  if (!isObject(value)) throw new Error('ユーザー情報の形式が不正です')
  if (value.id != null && typeof value.id !== 'string') throw new Error('ユーザーIDの形式が不正です')
  if (value.name != null && typeof value.name !== 'string') throw new Error('表示名の形式が不正です')
  return {
    id: typeof value.id === 'string' ? value.id : undefined,
    name: typeof value.name === 'string' ? value.name : undefined,
  }
}

export async function getMe(): Promise<MatchUser> {
  return parseMatchUser(await request('/me'))
}

export async function setMyUsername(username: string): Promise<void> {
  const response = await fetch(`${base}/me/username`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  })
  if (!response.ok)
    throw new MatchApiError(response.status, `ユーザー名を設定できませんでした (${response.status})`)
}

export async function logout(): Promise<void> {
  const response = await fetch(`${base}/logout`, { method: 'POST', credentials: 'include' })
  if (!response.ok)
    throw new MatchApiError(response.status, `ログアウトできませんでした (${response.status})`)
}

export function parseMatch(value: unknown): MatchRecord {
  if (
    !isObject(value) ||
    typeof value.id !== 'string' ||
    typeof value.fieldId !== 'string' ||
    typeof value.runId !== 'string' ||
    typeof value.opponentName !== 'string'
  )
    throw new Error('対戦作成レスポンスの形式が不正です')
  return value as unknown as MatchRecord
}

export function parseMatchResult(value: unknown): MatchResult {
  if (!isObject(value) || !Array.isArray(value.history) || !isObject(value.summary))
    throw new Error('対戦結果の形式が不正です')
  const score = value.summary.score
  if (!isObject(score) || typeof score.player !== 'number' || typeof score.opponent !== 'number')
    throw new Error('対戦得点の形式が不正です')
  return value as unknown as MatchResult
}

export async function getBattlePrograms(signal?: AbortSignal): Promise<[string, string, string] | null> {
  try {
    return parseBattleProgramIds(await request('/battle/program', { signal }))
  } catch (cause) {
    if (cause instanceof MatchApiError && cause.status === 404) return null
    throw cause
  }
}

export async function setBattlePrograms(ids: [string, string, string]): Promise<void> {
  const response = await fetch(`${base}/battle/program`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ids),
  })
  if (!response.ok)
    throw new MatchApiError(response.status, `対戦用プログラムを登録できませんでした (${response.status})`)
}

export async function createProgram(program: Pick<ProgramRecord, 'name' | 'source'>): Promise<ProgramRecord> {
  const value = await request('/program', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...program, isBattle: true }),
  })
  return parseProgram(value)
}

export async function createMatch(ids: [string, string, string]): Promise<MatchRecord> {
  return parseMatch(
    await request('/match', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(ids.map((programId) => ({ programId }))),
    }),
  )
}

export function subscribeMatchResult(
  runId: string,
  onResult: (result: MatchResult) => void,
  onError: (message: string) => void,
): () => void {
  const source = new EventSource(`${base}/run/${encodeURIComponent(runId)}/events`, {
    withCredentials: true,
  })
  source.addEventListener('result', (event) => {
    try {
      onResult(parseMatchResult(JSON.parse((event as MessageEvent).data)))
    } catch (error) {
      onError(error instanceof Error ? error.message : '対戦結果を読み取れませんでした')
    } finally {
      source.close()
    }
  })
  source.onerror = () => {
    source.close()
    onError('結果を受信できませんでした。再接続してください。')
  }
  return () => source.close()
}
