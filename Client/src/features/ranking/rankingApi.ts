import { api, type components } from '../../api/client.ts'

export type RankingEntry = components['schemas']['RankingEntry']

export async function getRanking(signal: AbortSignal): Promise<RankingEntry[]> {
  const { data, response } = await api.GET('/ranking', { signal })
  if (!response.ok) throw new Error('ランキングを表示できませんでした。時間をおいて再読み込みしてください。')
  if (!Array.isArray(data)) throw new Error('ランキングのデータ形式が不正です')
  return data
}
