import type { PicoTerrain, PicoValue } from '../picolang/language/picolang.ts'
import { formatPicoValue } from '../picolang/language/formatPicoValue.ts'

export interface TutorialSpeech {
  id: number
  items: string[]
  terrain?: PicoTerrain
  label: string
}

function isTerrain(value: PicoValue): value is PicoTerrain {
  return value !== null && typeof value === 'object' && !Array.isArray(value) && 'type' in value
}

export function speechForValue(value: PicoValue, id: number): TutorialSpeech {
  if (isTerrain(value)) {
    const kind = value.type === 'land' ? value.kind?.type : value.type
    const label = {
      plain: '陸',
      goal: 'ゴール',
      checkpoint: 'チェックポイント',
      rock: '岩',
      sea: '海',
    }[kind ?? 'plain']
    const details = [label]
    if (value.kind?.type === 'checkpoint') details.push(String(value.kind.order ?? '?'))
    if (value.hasLuggage) details.push('荷物あり')
    if (value.territory) details.push(value.territory === 'own' ? '自分の領地' : '相手の領地')
    return { id, items: details, terrain: value, label: details.join('、') }
  }
  const items = Array.isArray(value)
    ? value.map((item) => speechForValue(item, id).label)
    : [formatPicoValue(value)]
  return { id, items, label: Array.isArray(value) ? `[${items.join(', ')}]` : items[0] }
}
