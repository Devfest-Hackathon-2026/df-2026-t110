const base = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface TutorialRecord {
  id: string
  title: string
  fieldId: string
  goal?: string
  description?: string
  lesson?: string
  guide?: string[]
}

async function getJson(path: string, signal: AbortSignal): Promise<unknown | null> {
  const response = await fetch(`${base}${path}`, { signal })
  if (response.status === 404) return null
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  if (!response.headers.get('content-type')?.includes('json'))
    throw new Error('教材APIからJSONを取得できませんでした')
  return response.json()
}

export async function getTutorial(id: string, signal: AbortSignal): Promise<TutorialRecord | null> {
  return (await getJson(`/tutorial/${encodeURIComponent(id)}`, signal)) as TutorialRecord | null
}

export function getTutorialField(fieldId: string, signal: AbortSignal): Promise<unknown | null> {
  return getJson(`/field/${encodeURIComponent(fieldId)}`, signal)
}
