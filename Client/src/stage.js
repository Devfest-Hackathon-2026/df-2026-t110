const STAGE_SYMBOLS = new Set(['S', 'G', '.', '#', '_', 'O', 'X'])
const TEMPLATE_SYMBOLS = new Set([...STAGE_SYMBOLS, '?'])
const MAX_GENERATED_VARIANTS = 512

function rowsOf(raw, symbols, label) {
  const rows = typeof raw === 'string' ? raw.trim().split(/\r?\n/) : raw
  if (!Array.isArray(rows) || !rows.length || rows.some(row => typeof row !== 'string')) {
    throw new Error(`${label} は文字列の配列または改行区切りの文字列にしてください`)
  }
  const width = rows[0].length
  if (!width || rows.some(row => row.length !== width)) throw new Error('すべての行を同じ長さにしてください')
  if (rows.some(row => [...row].some(symbol => !symbols.has(symbol)))) throw new Error(`${label} に使えない記号があります`)
  if (rows.join('').split('S').length !== 2) throw new Error('スタート S を1つ指定してください')
  return rows
}

export function normalizeStage(source, index = 0) {
  if (!source || typeof source !== 'object') throw new Error('配置はJSONオブジェクトで指定してください')
  const rows = rowsOf(source.map ?? source.grid, STAGE_SYMBOLS, 'map')
  return {
    id: String(source.id ?? `layout-${index + 1}`),
    title: String(source.title ?? source.name ?? `配置 ${index + 1}`),
    rows,
    width: rows[0].length,
    height: rows.length,
    rockCount: rows.join('').split('#').length - 1,
    cells: rows.flatMap((row, y) => [...row].map((symbol, x) => ({ x, y, symbol }))),
  }
}

function rockCountsOf(raw, fixed, slots) {
  if (!Array.isArray(raw) || !raw.length || raw.some(count => !Number.isInteger(count) || count < fixed || count > fixed + slots)) {
    throw new Error(`rockCounts は ${fixed}〜${fixed + slots} の整数配列にしてください`)
  }
  return [...new Set(raw)].sort((a, b) => a - b)
}

function generateLayouts(fieldId, title, template, rockCounts) {
  const slots = []
  template.forEach((row, y) => [...row].forEach((symbol, x) => { if (symbol === '?') slots.push([x, y]) }))
  const fixed = template.join('').split('#').length - 1
  const layouts = []
  for (const rockCount of rockCounts) {
    const choose = (offset, remaining, selected) => {
      if (remaining === 0) {
        if (layouts.length >= MAX_GENERATED_VARIANTS) throw new Error('配置候補が多すぎます。候補マップを明示してください')
        const rows = template.map(row => [...row].map(symbol => symbol === '?' ? '.' : symbol))
        for (const slot of selected) {
          const [x, y] = slots[slot]
          rows[y][x] = '#'
        }
        const index = layouts.length
        layouts.push(normalizeStage({ id: `${fieldId}-layout-${index + 1}`, title, map: rows.map(row => row.join('')) }, index))
        return
      }
      for (let index = offset; index <= slots.length - remaining; index++) choose(index + 1, remaining - 1, [...selected, index])
    }
    choose(0, rockCount - fixed, [])
  }
  return layouts
}

function validateSharedShape(layouts) {
  const first = layouts[0]
  for (const layout of layouts.slice(1)) {
    if (layout.width !== first.width || layout.height !== first.height) throw new Error('候補配置の盤面サイズが一致しません')
  }
}

export function normalizeField(payload) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) throw new Error('フィールドはJSONオブジェクトで指定してください')
  if ('stages' in payload) throw new Error('1つのフィールドに候補配置を持たせてください。stages ではなく template と rockCounts を使います')
  const id = String(payload.id ?? 'field')
  const title = String(payload.title ?? payload.name ?? 'フィールド')
  const template = payload.template != null ? rowsOf(payload.template, TEMPLATE_SYMBOLS, 'template') : null
  const explicit = payload.variants ?? payload.candidates ?? payload.maps
  let layouts, rockCounts

  if (explicit != null) {
    if (!Array.isArray(explicit) || !explicit.length) throw new Error('候補配置がありません')
    layouts = explicit.map((candidate, index) => normalizeStage(
      typeof candidate === 'string' || Array.isArray(candidate)
        ? { id: `${id}-layout-${index + 1}`, title, map: candidate }
        : { title, ...candidate }, index,
    ))
    validateSharedShape(layouts)
    const fixed = template ? template.join('').split('#').length - 1 : 0
    const slots = template ? template.join('').split('?').length - 1 : layouts[0].width * layouts[0].height
    rockCounts = payload.rockCounts ? rockCountsOf(payload.rockCounts, fixed, slots) : [...new Set(layouts.map(layout => layout.rockCount))].sort((a, b) => a - b)
    if (layouts.some(layout => !rockCounts.includes(layout.rockCount))) throw new Error('候補配置の岩の数が rockCounts と一致しません')
    if (template) for (const layout of layouts) for (let y = 0; y < layout.height; y++) for (let x = 0; x < layout.width; x++) {
      const symbol = template[y]?.[x]
      if (!symbol || (symbol !== '?' && symbol !== layout.rows[y][x]) || (symbol === '?' && !['.', '#', '_', 'O', 'X'].includes(layout.rows[y][x]))) {
        throw new Error('候補配置が template の固定位置と一致しません')
      }
    }
  } else {
    const base = template ?? rowsOf(payload.map ?? payload.grid, STAGE_SYMBOLS, 'map')
    const fixed = base.join('').split('#').length - 1
    const slots = base.join('').split('?').length - 1
    rockCounts = payload.rockCounts ? rockCountsOf(payload.rockCounts, fixed, slots) : [fixed]
    layouts = generateLayouts(id, title, base, rockCounts)
  }
  if (!layouts.length) throw new Error('候補配置がありません')
  validateSharedShape(layouts)
  const serials = new Map()
  for (const layout of layouts) {
    const serial = (serials.get(layout.rockCount) ?? 0) + 1
    serials.set(layout.rockCount, serial)
    layout.label = `岩 ${layout.rockCount} 個 · 配置 ${serial}`
  }
  return { id, title, width: layouts[0].width, height: layouts[0].height, rockCounts, layouts }
}
