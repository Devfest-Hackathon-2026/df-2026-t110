import { isPicoProgram, newProgram, type PicoProgram } from '../picolang/language/picolang.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface TutorialRecord {
  id: string
  title: string
  fieldId?: string
  stage?: Record<string, unknown>
  goal?: string
  description: string
  lesson?: string
  guide?: string[]
  initialProgram?: PicoProgram
}

export function initialTutorialProgram(tutorial: TutorialRecord): PicoProgram {
  if (tutorial.initialProgram === undefined) return newProgram()
  if (!isPicoProgram(tutorial.initialProgram)) throw new Error('教材の初期プログラムの形式が不正です')
  return tutorial.initialProgram
}

export interface TutorialSummary {
  id: string
  title: string
}

async function getJson(path: string, signal: AbortSignal): Promise<unknown | null> {
  const request = new AbortController()
  const abort = () => request.abort()
  let timer: number | undefined
  const timeout = new Promise<never>((_, reject) => {
    timer = window.setTimeout(() => {
      request.abort()
      reject(new Error('チュートリアルAPIの応答が20秒以内に返りませんでした。ページを開き直してください。'))
    }, 20_000)
  })
  signal.addEventListener('abort', abort, { once: true })
  if (signal.aborted) request.abort()
  try {
    return await Promise.race([
      (async () => {
        const response = await fetch(`${base}${path}`, { signal: request.signal, credentials: 'same-origin' })
        if (response.status === 404) return null
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        if (!response.headers.get('content-type')?.includes('json')) {
          if (response.redirected)
            throw new Error(
              'チュートリアルAPIがログイン画面へ転送されました。ページを開き直して認証してください。',
            )
          throw new Error('チュートリアルAPIからJSONを取得できませんでした')
        }
        return await response.json()
      })(),
      timeout,
    ])
  } finally {
    if (timer !== undefined) window.clearTimeout(timer)
    signal.removeEventListener('abort', abort)
  }
}

export async function getTutorial(id: string, signal: AbortSignal): Promise<TutorialRecord | null> {
  return (await getJson(`/tutorial/${encodeURIComponent(id)}`, signal)) as TutorialRecord | null
}

export async function getTutorials(signal: AbortSignal): Promise<TutorialSummary[]> {
  const result = await getJson('/tutorial', signal)
  if (
    !Array.isArray(result) ||
    result.some(
      (item) =>
        !item || typeof item !== 'object' || typeof item.id !== 'string' || typeof item.title !== 'string',
    )
  ) {
    throw new Error('チュートリアル一覧の形式が不正です')
  }
  return result as TutorialSummary[]
}

export function getTutorialField(fieldId: string, signal: AbortSignal): Promise<unknown | null> {
  return getJson(`/field/${encodeURIComponent(fieldId)}`, signal)
}
