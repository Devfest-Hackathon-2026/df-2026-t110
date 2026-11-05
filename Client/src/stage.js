const SYMBOLS = new Set(['S', 'G', '.', '#', '_', 'O', 'X'])

export function normalizeStage(source, index = 0) {
  if (!source || typeof source !== 'object') throw new Error('ステージはJSONオブジェクトで指定してください')
  const raw = source.map ?? source.grid ?? source.field
  const rows = typeof raw === 'string' ? raw.trim().split(/\r?\n/) : raw
  if (!Array.isArray(rows) || !rows.length || rows.some(row => typeof row !== 'string')) {
    throw new Error('map は文字列の配列または改行区切りの文字列にしてください')
  }
  const width = rows[0].length
  if (!width || rows.some(row => row.length !== width)) throw new Error('すべての行を同じ長さにしてください')
  if (rows.some(row => [...row].some(symbol => !SYMBOLS.has(symbol)))) {
    throw new Error('使用できる記号は S G . # _ O X です')
  }
  if (rows.join('').split('S').length !== 2) throw new Error('スタート S を1つ指定してください')
  return {
    id: String(source.id ?? `stage-${index + 1}`),
    title: String(source.title ?? source.name ?? `ステージ ${index + 1}`),
    rows,
    width,
    height: rows.length,
    cells: rows.flatMap((row, y) => [...row].map((symbol, x) => ({ x, y, symbol }))),
  }
}

export function normalizeStages(payload) {
  const candidates = Array.isArray(payload) ? payload : payload?.stages ?? payload?.candidates ?? [payload]
  if (!Array.isArray(candidates) || !candidates.length) throw new Error('ステージデータがありません')
  return candidates.map(normalizeStage)
}
