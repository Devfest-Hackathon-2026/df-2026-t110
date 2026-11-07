import type { PicoProgram } from '../picolang/picolang.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')
const authAppOrigin = (
  import.meta.env?.VITE_AUTH_APP_ORIGIN ?? 'https://server-639320660675.asia-northeast1.run.app'
).replace(/\/$/, '')

export interface ProgramRecord {
  id?: string
  name: string
  source: PicoProgram
}

export interface MatchUser {
  id?: string
  name?: string
}

// IAP handles Google sign-in before the deployed app is served.
export const loginUrl = `${authAppOrigin}/account`

export interface MatchRecord {
  id: string
  fieldId: string
  selectedLayoutId: string
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
    throw new Error(
      'APIからJSONが返されませんでした。開発サーバーを再起動し、Goサーバーの起動を確認してください。',
    )
  return response.json()
}

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function parsePrograms(value: unknown): ProgramRecord[] {
  if (!Array.isArray(value)) throw new Error('プログラム一覧の形式が不正です')
  return value.filter(
    (item): item is ProgramRecord =>
      isObject(item) && typeof item.id === 'string' && typeof item.name === 'string',
  )
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

export function parseIapIdentity(value: unknown): MatchUser {
  if (!isObject(value) || typeof value.email !== 'string' || typeof value.sub !== 'string')
    throw new Error('Googleアカウント情報の形式が不正です')
  const prefix = 'accounts.google.com:'
  if (!value.email.startsWith(prefix) || !value.sub.startsWith(prefix))
    throw new Error('Googleアカウント情報の形式が不正です')
  const email = value.email.slice(prefix.length)
  const id = value.sub.slice(prefix.length)
  if (!email || !id) throw new Error('Googleアカウント情報の形式が不正です')
  return { id, name: email }
}

export async function getMe(): Promise<MatchUser> {
  try {
    return parseMatchUser(await request('/me'))
  } catch (error) {
    // The deployed app is protected by IAP, while /api/me is not implemented yet.
    if (!(error instanceof MatchApiError) || error.status !== 501 || window.location.origin !== authAppOrigin)
      throw error
    const url = new URL('/account', window.location.origin)
    url.searchParams.set('gcp-iap-mode', 'IDENTITY')
    const response = await fetch(url, { credentials: 'same-origin' })
    if (!response.ok || !response.headers.get('content-type')?.includes('application/json'))
      throw new Error('Googleアカウント情報を取得できませんでした')
    return parseIapIdentity(await response.json())
  }
}

export function parseMatch(value: unknown): MatchRecord {
  if (
    !isObject(value) ||
    typeof value.id !== 'string' ||
    typeof value.fieldId !== 'string' ||
    typeof value.selectedLayoutId !== 'string'
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

export async function listPrograms(fieldId: string, signal?: AbortSignal): Promise<ProgramRecord[]> {
  return parsePrograms(await request(`/field/${encodeURIComponent(fieldId)}/program`, { signal }))
}

export async function createProgram(
  fieldId: string,
  program: Omit<ProgramRecord, 'id'>,
): Promise<ProgramRecord> {
  const value = await request(`/field/${encodeURIComponent(fieldId)}/program`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(program),
  })
  if (!isObject(value) || typeof value.id !== 'string' || typeof value.name !== 'string')
    throw new Error('保存したプログラムの形式が不正です')
  return value as unknown as ProgramRecord
}

export async function createMatch(fieldId: string, programId: string): Promise<MatchRecord> {
  return parseMatch(
    await request(`/field/${encodeURIComponent(fieldId)}/match`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ programId }),
    }),
  )
}

export function subscribeMatchResult(
  matchId: string,
  onResult: (result: MatchResult) => void,
  onError: (message: string) => void,
): () => void {
  const source = new EventSource(`${base}/match/${encodeURIComponent(matchId)}/events`, {
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
