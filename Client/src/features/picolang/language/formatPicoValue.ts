import type { PicoTerrain } from './picolang.ts'

export function formatPicoValue(value: boolean | number | null | string | PicoTerrain): string {
  if (value === null) return '()'
  return typeof value === 'object' ? JSON.stringify(value) : String(value)
}
