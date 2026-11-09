import { parsePicolangExecutionJson } from '../picolang/language/executionJson.ts'
import type { PicoProgram } from '../picolang/language/picolang.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface TutorialResult {
  history: unknown[]
  summary: { cleared: boolean; tick: number; reason: 'all_stopped' | 'turn_limit' | 'condition_met' }
}

export class TutorialRunApiError extends Error {
  readonly status: number
  constructor(status: number, action: string) {
    super(
      status === 401
        ? '実行にはログインが必要です。'
        : status === 501
          ? 'プログラムの実行は現在利用できません。'
          : `${action}ができませんでした。もう一度お試しください。`,
    )
    this.status = status
  }
}

async function postJson(path: string, body: unknown, signal: AbortSignal, action: string): Promise<unknown> {
  const response = await fetch(`${base}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  })
  if (!response.ok) throw new TutorialRunApiError(response.status, action)
  if (!response.headers.get('content-type')?.includes('json'))
    throw new Error('チュートリアルAPIからJSONが返されませんでした')
  return response.json()
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function parseTutorialResult(json: string): TutorialResult {
  const result = parsePicolangExecutionJson(json)

  if (
    !isRecord(result) ||
    !Array.isArray(result.history) ||
    !isRecord(result.summary) ||
    typeof result.summary.cleared !== 'boolean' ||
    !Number.isInteger(result.summary.tick) ||
    !['all_stopped', 'turn_limit', 'condition_met'].includes(String(result.summary.reason))
  )
    throw new Error('チュートリアルの実行結果の形式が不正です')
  return result as unknown as TutorialResult
}

export async function saveTutorialProgram(
  tutorialId: string,
  title: string,
  source: PicoProgram,
  signal: AbortSignal,
): Promise<string> {
  const saved = await postJson(
    '/program',
    {
      name: title,
      source,
      isBattle: false,
      tutorialID: tutorialId,
    },
    signal,
    'プログラムの保存',
  )
  if (!isRecord(saved) || typeof saved.id !== 'string' || !saved.id)
    throw new Error('保存したプログラムのIDが返されませんでした')
  return saved.id
}

export async function startTutorialRun(
  tutorialId: string,
  programId: string,
  signal: AbortSignal,
): Promise<{ runId: string; fieldId: string }> {
  const started = await postJson(
    `/tutorial/${encodeURIComponent(tutorialId)}/run`,
    { programId },
    signal,
    'プログラムの実行',
  )
  if (
    !isRecord(started) ||
    typeof started.runId !== 'string' ||
    !started.runId ||
    typeof started.fieldId !== 'string' ||
    !started.fieldId
  )
    throw new Error('実行結果のIDとフィールドIDが返されませんでした')
  return { runId: started.runId, fieldId: started.fieldId }
}

export function receiveTutorialResult(runId: string, signal: AbortSignal): Promise<TutorialResult> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException('実行を中断しました', 'AbortError'))
      return
    }
    const source = new EventSource(`${base}/run/${encodeURIComponent(runId)}/events`, {
      withCredentials: true,
    })
    const cleanup = () => {
      source.close()
      signal.removeEventListener('abort', abort)
    }
    const abort = () => {
      cleanup()
      reject(new DOMException('実行を中断しました', 'AbortError'))
    }
    signal.addEventListener('abort', abort, { once: true })
    source.addEventListener('result', (event) => {
      try {
        const result = parseTutorialResult((event as MessageEvent).data)
        cleanup()
        resolve(result)
      } catch (error) {
        cleanup()
        reject(error)
      }
    })
    source.onerror = () => {
      cleanup()
      reject(
        new Error('チュートリアルの実行結果を受信できませんでした。接続とログイン状態を確認してください。'),
      )
    }
  })
}
