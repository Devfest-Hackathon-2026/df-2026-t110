const STAGE_SYMBOLS = new Set(['S', 'G', '.', '#', '_', 'O', 'X'])
const TEMPLATE_SYMBOLS = new Set([...STAGE_SYMBOLS, '?'])
const MAX_GENERATED_VARIANTS = 512

export interface StageCell {
  x: number
  y: number
  symbol: string
  terrain: 'land' | 'rock' | 'sea'
  territoryOwner?: number
  objects: StageObject[]
  start: boolean
}
export interface StageObject {
  type: 'goal' | 'checkpoint' | 'luggage' | 'dock'
  id?: string
  order?: number
}
export interface StageLayout {
  id: string
  title: string
  rows: string[]
  width: number
  height: number
  rockCount: number
  drillLimit?: number
  cells: StageCell[]
  label: string
  robot: { position: { x: number; y: number }; direction: 'north' | 'south' | 'west' | 'east' }
}
export interface FieldData {
  id: string
  title: string
  width: number
  height: number
  rockCounts: number[]
  layouts: StageLayout[]
}

function rowsOf(raw: unknown, symbols: Set<string>, label: string): string[] {
  const rows = typeof raw === 'string' ? raw.trim().split(/\r?\n/) : raw
  if (!Array.isArray(rows) || !rows.length || rows.some((row) => typeof row !== 'string')) {
    throw new Error(`${label} は文字列の配列または改行区切りの文字列にしてください`)
  }
  const checked = rows as string[]
  const width = checked[0].length
  if (!width || checked.some((row) => row.length !== width))
    throw new Error('すべての行を同じ長さにしてください')
  if (checked.some((row) => [...row].some((symbol) => !symbols.has(symbol))))
    throw new Error(`${label} に使えない記号があります`)
  if (checked.join('').split('S').length !== 2) throw new Error('スタート S を1つ指定してください')
  return checked
}

export function normalizeStage(source: Record<string, unknown>, index = 0): StageLayout {
  if (!source || typeof source !== 'object') throw new Error('配置はJSONオブジェクトで指定してください')
  const rows = rowsOf(source.map ?? source.grid, STAGE_SYMBOLS, 'map')
  const cells = rows.flatMap((row, y) =>
    [...row].map((symbol, x): StageCell => ({
      x,
      y,
      symbol,
      terrain: symbol === '#' ? 'rock' : symbol === '_' ? 'sea' : 'land',
      objects:
        symbol === 'G'
          ? [{ type: 'goal' }]
          : symbol === 'O'
            ? [{ type: 'luggage', id: `parcel-${x}-${y}` }]
            : symbol === 'X'
              ? [{ type: 'dock' }]
              : [],
      start: symbol === 'S',
    })),
  )
  const start = cells.find((cell) => cell.start)!
  return {
    id: String(source.id ?? `layout-${index + 1}`),
    title: String(source.title ?? source.name ?? `配置 ${index + 1}`),
    rows,
    width: rows[0].length,
    height: rows.length,
    rockCount: rows.join('').split('#').length - 1,
    drillLimit: typeof source.drillLimit === 'number' ? source.drillLimit : undefined,
    cells,
    label: '',
    robot: { position: { x: start.x, y: start.y }, direction: 'east' },
  }
}

function normalizeSchemaField(raw: Record<string, unknown>, index: number): StageLayout {
  const width = raw.width
  const height = raw.height
  const terrain = raw.terrain
  const robot = raw.robot as Record<string, unknown> | undefined
  const position = robot?.position as Record<string, unknown> | undefined
  const direction = robot?.direction
  if (!Number.isInteger(width) || !Number.isInteger(height) || Number(width) < 1 || Number(height) < 1)
    throw new Error('schema field の width / height が不正です')
  if (
    !Array.isArray(terrain) ||
    terrain.length !== height ||
    terrain.some(
      (row) =>
        !Array.isArray(row) ||
        row.length !== width ||
        row.some((kind) => !['land', 'rock', 'sea'].includes(kind)),
    )
  )
    throw new Error('schema field の terrain が盤面サイズ・地形定義と一致しません')
  if (
    !position ||
    !Number.isInteger(position.x) ||
    !Number.isInteger(position.y) ||
    Number(position.x) < 0 ||
    Number(position.x) >= Number(width) ||
    Number(position.y) < 0 ||
    Number(position.y) >= Number(height) ||
    !['north', 'south', 'west', 'east'].includes(String(direction))
  )
    throw new Error('schema field の robot が不正です')
  const objects = raw.objects
  if (!Array.isArray(objects)) throw new Error('schema field の objects が不正です')
  const cells: StageCell[] = (terrain as string[][]).flatMap((row, y) =>
    row.map((kind, x) => ({
      x,
      y,
      terrain: kind as StageCell['terrain'],
      symbol: '.',
      objects: [],
      start: false,
    })),
  )
  for (const value of objects) {
    if (!value || typeof value !== 'object') throw new Error('schema field に不正な object があります')
    const item = value as Record<string, unknown>
    const at = item.position as Record<string, unknown> | undefined
    if (
      !at ||
      !Number.isInteger(at.x) ||
      !Number.isInteger(at.y) ||
      Number(at.x) < 0 ||
      Number(at.x) >= Number(width) ||
      Number(at.y) < 0 ||
      Number(at.y) >= Number(height) ||
      !['goal', 'checkpoint', 'luggage'].includes(String(item.type))
    )
      throw new Error('schema field に不正な object があります')
    if (item.type === 'checkpoint' && !Number.isInteger(item.order))
      throw new Error('checkpoint の order が不正です')
    if (item.type === 'luggage' && (typeof item.id !== 'string' || !item.id))
      throw new Error('luggage の id が不正です')
    const cell = cells[Number(at.y) * Number(width) + Number(at.x)]
    cell.objects.push({
      type: item.type as StageObject['type'],
      ...(item.type === 'checkpoint' ? { order: item.order as number } : {}),
      ...(item.type === 'luggage' ? { id: item.id as string } : {}),
    })
  }
  const start = cells[Number(position.y) * Number(width) + Number(position.x)]
  if (start.terrain !== 'land') throw new Error('ロボットの初期位置は陸にしてください')
  start.start = true
  for (const cell of cells) {
    cell.symbol = cell.start
      ? 'S'
      : cell.terrain === 'rock'
        ? '#'
        : cell.terrain === 'sea'
          ? '_'
          : cell.objects.some((object) => object.type === 'goal')
            ? 'G'
            : cell.objects.some((object) => object.type === 'checkpoint')
              ? 'C'
              : cell.objects.some((object) => object.type === 'luggage')
                ? 'O'
                : '.'
  }
  const rows = Array.from({ length: Number(height) }, (_, y) =>
    cells
      .slice(y * Number(width), (y + 1) * Number(width))
      .map((cell) => cell.symbol)
      .join(''),
  )
  return {
    id: String(raw.id ?? `layout-${index + 1}`),
    title: String(raw.title ?? `配置 ${index + 1}`),
    rows,
    width: Number(width),
    height: Number(height),
    rockCount: cells.filter((cell) => cell.terrain === 'rock').length,
    drillLimit: typeof raw.drillLimit === 'number' ? raw.drillLimit : undefined,
    cells,
    label: '',
    robot: {
      position: { x: Number(position.x), y: Number(position.y) },
      direction: direction as StageLayout['robot']['direction'],
    },
  }
}

function rockCountsOf(raw: unknown, fixed: number, slots: number): number[] {
  if (
    !Array.isArray(raw) ||
    !raw.length ||
    raw.some((count) => !Number.isInteger(count) || count < fixed || count > fixed + slots)
  ) {
    throw new Error(`rockCounts は ${fixed}〜${fixed + slots} の整数配列にしてください`)
  }
  return [...new Set(raw as number[])].sort((a, b) => a - b)
}

function generateLayouts(
  fieldId: string,
  title: string,
  template: string[],
  rockCounts: number[],
): StageLayout[] {
  const slots: Array<[number, number]> = []
  template.forEach((row, y) =>
    [...row].forEach((symbol, x) => {
      if (symbol === '?') slots.push([x, y])
    }),
  )
  const fixed = template.join('').split('#').length - 1
  const layouts: StageLayout[] = []
  for (const rockCount of rockCounts) {
    const choose = (offset: number, remaining: number, selected: number[]): void => {
      if (remaining === 0) {
        if (layouts.length >= MAX_GENERATED_VARIANTS)
          throw new Error('配置候補が多すぎます。候補マップを明示してください')
        const rows = template.map((row) => [...row].map((symbol) => (symbol === '?' ? '.' : symbol)))
        for (const slot of selected) {
          const [x, y] = slots[slot]
          rows[y][x] = '#'
        }
        const index = layouts.length
        layouts.push(
          normalizeStage(
            { id: `${fieldId}-layout-${index + 1}`, title, map: rows.map((row) => row.join('')) },
            index,
          ),
        )
        return
      }
      for (let index = offset; index <= slots.length - remaining; index++)
        choose(index + 1, remaining - 1, [...selected, index])
    }
    choose(0, rockCount - fixed, [])
  }
  return layouts
}

function validateSharedShape(layouts: StageLayout[]): void {
  const first = layouts[0]
  for (const layout of layouts.slice(1)) {
    if (layout.width !== first.width || layout.height !== first.height)
      throw new Error('候補配置の盤面サイズが一致しません')
  }
}

export function normalizeField(payload: Record<string, unknown>): FieldData {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload))
    throw new Error('フィールドはJSONオブジェクトで指定してください')
  const schemaVariants =
    Array.isArray(payload.variants) &&
    payload.variants.length > 0 &&
    payload.variants.every((variant) => variant && typeof variant === 'object' && 'terrain' in variant)
  if ('terrain' in payload || schemaVariants) {
    const variants = schemaVariants ? (payload.variants as Record<string, unknown>[]) : [payload]
    const layouts = variants.map(normalizeSchemaField)
    const rockCounts = [...new Set(layouts.map((layout) => layout.rockCount))].sort((a, b) => a - b)
    const serials = new Map<number, number>()
    for (const layout of layouts) {
      const serial = (serials.get(layout.rockCount) ?? 0) + 1
      serials.set(layout.rockCount, serial)
      layout.label = `岩 ${layout.rockCount} 個 · 配置 ${serial}`
    }
    return {
      id: String(payload.id ?? 'field'),
      title: String(payload.title ?? 'フィールド'),
      width: layouts[0].width,
      height: layouts[0].height,
      rockCounts,
      layouts,
    }
  }
  if ('stages' in payload)
    throw new Error(
      '1つのフィールドに候補配置を持たせてください。stages ではなく template と rockCounts を使います',
    )
  const id = String(payload.id ?? 'field')
  const title = String(payload.title ?? payload.name ?? 'フィールド')
  const template = payload.template != null ? rowsOf(payload.template, TEMPLATE_SYMBOLS, 'template') : null
  const explicit = payload.variants ?? payload.candidates ?? payload.maps
  let layouts: StageLayout[]
  let rockCounts: number[]

  if (explicit != null) {
    if (!Array.isArray(explicit) || !explicit.length) throw new Error('候補配置がありません')
    layouts = explicit.map((candidate: unknown, index: number) =>
      normalizeStage(
        typeof candidate === 'string' || Array.isArray(candidate)
          ? { id: `${id}-layout-${index + 1}`, title, map: candidate }
          : { title, ...(candidate as Record<string, unknown>) },
        index,
      ),
    )
    validateSharedShape(layouts)
    const fixed = template ? template.join('').split('#').length - 1 : 0
    const slots = template ? template.join('').split('?').length - 1 : layouts[0].width * layouts[0].height
    rockCounts = payload.rockCounts
      ? rockCountsOf(payload.rockCounts, fixed, slots)
      : [...new Set(layouts.map((layout) => layout.rockCount))].sort((a, b) => a - b)
    if (layouts.some((layout) => !rockCounts.includes(layout.rockCount)))
      throw new Error('候補配置の岩の数が rockCounts と一致しません')
    if (template)
      for (const layout of layouts)
        for (let y = 0; y < layout.height; y++)
          for (let x = 0; x < layout.width; x++) {
            const symbol = template[y]?.[x]
            if (
              !symbol ||
              (symbol !== '?' && symbol !== layout.rows[y][x]) ||
              (symbol === '?' && !['.', '#', '_', 'O', 'X'].includes(layout.rows[y][x]))
            ) {
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
