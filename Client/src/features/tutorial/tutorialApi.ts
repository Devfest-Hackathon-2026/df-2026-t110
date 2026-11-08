const base = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface TutorialRecord {
  id: string
  title: string
  fieldId?: string
  stage?: Record<string, unknown>
  goal?: string
  description: string
  lesson?: string
  guide?: string[]
}

export interface TutorialSummary {
  id: string
  title: string
}

async function getJson(path: string, signal: AbortSignal): Promise<unknown | null> {
  const request = new AbortController()
  const abort = () => request.abort()
  let timedOut = false
  const timer = window.setTimeout(() => {
    timedOut = true
    request.abort()
  }, 20_000)
  signal.addEventListener('abort', abort, { once: true })
  if (signal.aborted) request.abort()
  try {
    const response = await fetch(`${base}${path}`, { signal: request.signal })
    if (response.status === 404) return null
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    if (!response.headers.get('content-type')?.includes('json'))
      throw new Error('教材APIからJSONを取得できませんでした')
    return response.json()
  } catch (error) {
    if (timedOut && !signal.aborted) throw new Error('教材APIの応答が20秒以内に返りませんでした')
    throw error
  } finally {
    window.clearTimeout(timer)
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
    throw new Error('教材一覧の形式が不正です')
  }
  return result as TutorialSummary[]
}

export function getTutorialField(fieldId: string, signal: AbortSignal): Promise<unknown | null> {
  return getJson(`/field/${encodeURIComponent(fieldId)}`, signal)
}
