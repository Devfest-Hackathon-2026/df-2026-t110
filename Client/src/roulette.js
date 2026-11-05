export function chooseLayoutIndex(count, random = Math.random) {
  if (!Number.isInteger(count) || count < 1) throw new Error('候補配置がありません')
  return Math.min(count - 1, Math.max(0, Math.floor(random() * count)))
}

export function rouletteSteps(from, to, count) {
  if (count < 2) return [0]
  const distance = (to - from + count) % count
  return Array.from({ length: count * Math.max(2, Math.ceil(12 / count)) + distance }, (_, step) => (from + step + 1) % count)
}

export function resolveServerSelection(layouts, response) {
  const index = response?.candidateIndex ?? response?.selectedIndex
  if (Number.isInteger(index) && index >= 0 && index < layouts.length) return index
  const id = response?.selectedLayoutId ?? response?.layoutId ?? response?.layout?.id ?? response?.selectedStageId ?? response?.stageId ?? response?.stage?.id
  const matched = layouts.findIndex(layout => layout.id === String(id))
  if (matched >= 0) return matched
  throw new Error('サーバーの抽選結果に候補配置のIDまたは番号がありません')
}
