import { PICO_BUILTINS, type PicoBuiltin, type PicoBuiltinName, type PicoType } from '../language/picolang.ts'

export interface PicoFunctionOption extends PicoBuiltin {
  label: string
  category: string
}

export interface PicoVariableOption {
  name: string
  type: PicoType
}

const builtinLabels = {
  forward: { label: '前進する', category: '動き' },
  back: { label: '後退する', category: '動き' },
  rotate: { label: '回転する', category: '動き' },
  wait: { label: '待つ', category: '動き' },
  pick_up: { label: '拾う', category: '操作' },
  release: { label: '置く', category: '操作' },
  drill: { label: '岩を壊す', category: '操作' },
  drill_durability: { label: 'ドリルの残りを返す', category: '調べる' },
  distance: { label: '距離を返す', category: '調べる' },
  search_front: { label: '前のマスを調べて返す', category: '調べる' },
  search_now: { label: '今いるマスを調べて返す', category: '調べる' },
  is_land: { label: '陸なら「はい」を返す', category: '調べる' },
  is_goal: { label: 'ゴールなら「はい」を返す', category: '調べる' },
  is_checkpoint: { label: 'チェックポイントなら「はい」を返す', category: '調べる' },
  is_laggage: { label: '荷物なら「はい」を返す', category: '調べる' },
  is_sea: { label: '海なら「はい」を返す', category: '調べる' },
  is_rock: { label: '岩なら「はい」を返す', category: '調べる' },
  not: { label: '答えを反対にして返す', category: '論理' },
  and: { label: '両方「はい」か返す', category: '論理' },
  or: { label: 'どちらかが「はい」か返す', category: '論理' },
  bool_eq: { label: '2つの答えが同じか返す', category: '論理' },
  bool_neq: { label: '2つの答えが違うか返す', category: '論理' },
  add: { label: '足した数を返す', category: '計算' },
  minus: { label: '符号を反対にした数を返す', category: '計算' },
  sub: { label: '引いた数を返す', category: '計算' },
  mul: { label: '掛けた数を返す', category: '計算' },
  div: { label: '割った数を返す', category: '計算' },
  integer_eq: { label: '2つの数が同じか返す', category: '計算' },
  integer_neq: { label: '2つの数が違うか返す', category: '計算' },
  gt: { label: '左の数が大きいか返す', category: '計算' },
  lt: { label: '左の数が小さいか返す', category: '計算' },
  geq: { label: '左の数が右以上か返す', category: '計算' },
  leq: { label: '左の数が右以下か返す', category: '計算' },
  random_bool: { label: 'ランダムなはい・いいえを返す', category: '乱数' },
  random_integer: { label: 'ランダムな数を返す', category: '乱数' },
  new: { label: '空のスタックを作って返す', category: 'スタック' },
  push_back: { label: '値を足したスタックを返す', category: 'スタック' },
  pop_back: { label: '最後を除いたスタックを返す', category: 'スタック' },
  get_back: { label: '最後の値を返す', category: 'スタック' },
  size: { label: 'スタックの個数を返す', category: 'スタック' },
  print_integer: { label: '整数を表示', category: '表示' },
  print_bool: { label: '真理値を表示', category: '表示' },
} satisfies Record<PicoBuiltinName, { label: string; category: string }>

export const PICO_BUILTIN_OPTIONS: PicoFunctionOption[] = PICO_BUILTINS.map((builtin) => ({
  ...builtin,
  ...builtinLabels[builtin.name],
}))
