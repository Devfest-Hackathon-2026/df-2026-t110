const base = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

async function getJson(path, signal) {
  const response = await fetch(`${base}${path}`, { signal })
  if (response.status === 404) return null
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  if (!response.headers.get('content-type')?.includes('json')) throw new Error('教材APIからJSONを取得できませんでした')
  return response.json()
}

export function getTutorial(id, signal) {
  return getJson(`/tutorial/${encodeURIComponent(id)}`, signal)
}

export function getTutorialField(fieldId, signal) {
  return getJson(`/field/${encodeURIComponent(fieldId)}`, signal)
}
