export function chooseLayoutIndex(count: number, random: () => number = Math.random) {
  if (!Number.isInteger(count) || count < 1) throw new Error('候補配置がありません')
  return Math.min(count - 1, Math.max(0, Math.floor(random() * count)))
}

export function rouletteSteps(from: number, to: number, count: number): number[] {
  if (count < 2) return [0]
  const distance = (to - from + count) % count
  return Array.from(
    { length: count * Math.max(2, Math.ceil(12 / count)) + distance },
    (_, step) => (from + step + 1) % count,
  )
}

export type SelectionResponse = {
  candidateIndex?: number
  selectedIndex?: number
  fieldId?: string
} | null

export function resolveServerSelection(
  layouts: ReadonlyArray<{ id: string }>,
  response: SelectionResponse,
): number {
  const index = response?.candidateIndex ?? response?.selectedIndex
  if (typeof index === 'number' && Number.isInteger(index) && index >= 0 && index < layouts.length)
    return index
  const matched = layouts.findIndex((layout) => layout.id === response?.fieldId)
  if (matched >= 0) return matched
  throw new Error('サーバーの抽選結果に候補配置のIDまたは番号がありません')
}
