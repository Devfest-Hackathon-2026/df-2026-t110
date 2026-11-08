import type { PicoFunction, PicoType } from '../language/picolang.ts'

const builtinArguments: Record<string, string[]> = {
  rotate: ['回す向き'],
  wait: ['待つ時間'],
  is_land: ['調べるマス'],
  is_goal: ['調べるマス'],
  is_checkpoint: ['調べるマス'],
  is_laggage: ['調べるマス'],
  is_sea: ['調べるマス'],
  is_rock: ['調べるマス'],
  not: ['反対にする答え'],
  and: ['1つ目の答え', '2つ目の答え'],
  or: ['1つ目の答え', '2つ目の答え'],
  bool_eq: ['1つ目の答え', '2つ目の答え'],
  bool_neq: ['1つ目の答え', '2つ目の答え'],
  add: ['左の数', '右の数'],
  minus: ['符号を変える数'],
  sub: ['引かれる数', '引く数'],
  mul: ['左の数', '右の数'],
  div: ['割られる数', '割る数'],
  integer_eq: ['左の数', '右の数'],
  integer_neq: ['左の数', '右の数'],
  gt: ['左の数', '右の数'],
  lt: ['左の数', '右の数'],
  geq: ['左の数', '右の数'],
  leq: ['左の数', '右の数'],
  bool_to_integer: ['数にする答え'],
  integer_to_bool: ['変換する数'],
  random_integer: ['最小値', '最大値'],
  push_back: ['値を足す配列', '足す値'],
  pop_back: ['最後を除く配列'],
  get_back: ['最後を見る配列'],
  size: ['個数を数える配列'],
  print_integer: ['表示する数'],
  print_bool: ['表示する答え'],
}

export function argumentLabel(name: string, index: number, functions: Record<string, PicoFunction>): string {
  return builtinArguments[name]?.[index] ?? functions[name]?.params[index]?.ident.name ?? `引数 ${index + 1}`
}

export function typeLabel(type: PicoType): string {
  if (typeof type !== 'string') return `${typeLabel(type.elementType)}の配列`
  return { bool: '真理値', integer: '整数', unit: '値なし', terrain: 'マスの種類', rotation: '向き' }[type]
}

export const resultGuidance: Record<string, string> = {
  new: '作った配列を使うには、変数へ入れてください。',
  push_back: '元の配列は変わりません。新しい配列を変数へ入れてください。',
  pop_back: '元の配列は変わりません。空の配列には使えません。',
  get_back: '配列は変わりません。空の配列には使えません。',
  size: '元の配列は変わりません。',
}
