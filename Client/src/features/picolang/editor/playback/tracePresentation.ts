import type { TraceNode } from '../../language/picolangTrace.ts'
import { formatPicoValue } from '../../language/formatPicoValue.ts'
import { PICO_BUILTIN_OPTIONS } from '../picoOptions.ts'

const labels: Record<string, string> = {
  function: '関数に入る',
  block: '処理のまとまり',
  declare: '変数を作る',
  assign: '代入',
  if: '条件分岐',
  while: 'くり返し',
  for: '回数を指定してくり返し',
  return: '戻る',
  break: 'くり返しを抜ける',
  continue: '次のくり返しへ',
  param: '受け取る値',
}
const calls = new Map(PICO_BUILTIN_OPTIONS.map((option) => [option.name, option.label]))
export function traceNodeLabel(node: TraceNode): string {
  if (node.kind === 'call') return calls.get(node.func ?? '') ?? node.func ?? node.kind
  if (node.kind === 'ident') return node.name ?? node.kind
  if (node.kind === 'literal') return formatPicoValue(node.value ?? null)
  return labels[node.kind] ?? node.kind
}
