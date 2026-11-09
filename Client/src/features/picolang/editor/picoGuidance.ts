import { PICO_BUILTIN_SIGNATURES } from '../language/builtins.gen.ts'
import type { PicoBuiltinName, PicoFunction, PicoSignatureType } from '../language/picolang.ts'

type BuiltinArgumentLabels = {
  [
    Name in PicoBuiltinName as (typeof PICO_BUILTIN_SIGNATURES)[Name]['params']['length'] extends 0
      ? never
      : Name
  ]: readonly string[] & { readonly length: (typeof PICO_BUILTIN_SIGNATURES)[Name]['params']['length'] }
}

export const builtinArguments: Partial<Record<PicoBuiltinName, readonly string[]>> = {
  distance: ['探索対象'],
  is_own_territory: ['調べるマス'],
  is_enemy_territory: ['調べるマス'],
  rotate: ['回す向き'],
  wait: ['待つターン数'],
  is_land: ['調べるマス'],
  is_goal: ['調べるマス'],
  is_checkpoint: ['調べるマス'],
  is_luggage: ['調べるマス'],
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
  random_integer: ['最小値', '最大値'],
  push_back: ['値を足すスタック', '足す値'],
  pop_back: ['最後を除くスタック'],
  get_back: ['最後を見るスタック'],
  size: ['個数を数えるスタック'],
  some: ['オプションに入れる値'],
  unwrap: ['値を取り出すオプション'],
  is_some: ['値の有無を調べるオプション'],
  print_integer: ['表示する数'],
  print_bool: ['表示する答え'],
  print_unit: ['表示する値なし'],
  print_terrain: ['表示するマス'],
  print_rotation: ['表示する向き'],
  print_distance_target: ['表示する探索対象'],
} satisfies BuiltinArgumentLabels

export function argumentLabel(name: string, index: number, functions: Record<string, PicoFunction>): string {
  return (
    builtinArguments[name as PicoBuiltinName]?.[index] ??
    functions[name]?.params[index]?.ident.name ??
    `引数 ${index + 1}`
  )
}

export function typeLabel(type: PicoSignatureType | undefined): string {
  if (!type || (typeof type !== 'string' && type.kind === 'type_parameter')) return '未確定'
  if (typeof type !== 'string')
    return `${typeLabel(type.elementType)}の${type.kind === 'option' ? 'オプション' : 'スタック'}`
  return {
    bool: '真理値',
    integer: '整数',
    unit: '値なし',
    terrain: 'マスの種類',
    rotation: '向き',
    distance_target: '探索対象',
  }[type]
}

export const resultGuidance: Record<string, string> = {
  new: '作ったスタックを使うには、変数へ入れてください。',
  push_back: '元のスタックは変わりません。新しいスタックを変数へ入れてください。',
  pop_back: '元のスタックは変わりません。空のスタックには使えません。',
  get_back: 'スタックは変わりません。空のスタックには使えません。',
  size: '元のスタックは変わりません。',
  some: '値を入れたオプションを返します。中の値は「値を取り出す」で使えます。',
  none: '値のないオプションを返します。値の種類は、代入する変数や呼び出す関数などから決まります。',
  unwrap: '値がない場合はエラーで実行が止まります。先に「値があるか調べる」で確認してください。',
  is_some: '値がある場合は「はい」、値がない場合は「いいえ」を返します。',
}
