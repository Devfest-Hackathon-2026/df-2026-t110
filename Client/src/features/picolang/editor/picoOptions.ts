import { PICO_BUILTINS, type PicoBuiltin, type PicoBuiltinName, type PicoType } from '../language/picolang.ts'

export interface PicoFunctionOption extends PicoBuiltin {
  label: string
  category: string
  blockMessage?: string
}

export interface PicoVariableOption {
  name: string
  type?: PicoType
}

const builtinLabels = {
  forward: { label: '前に進む', category: '動き' },
  back: { label: '後ろに進む', category: '動き' },
  rotate: { label: '回転する', category: '動き', blockMessage: '%1 向きに回転する' },
  wait: { label: '待つ', category: '動き', blockMessage: '%1 待つ' },
  pick_up: { label: '拾う', category: '操作' },
  release: { label: '置く', category: '操作' },
  drill: { label: '岩を壊す', category: '操作' },
  drill_durability: { label: 'ドリルの残りを返す', category: '調べる' },
  distance: { label: '距離を返す', category: '調べる' },
  search_front: { label: '前のマスを調べて返す', category: '調べる' },
  search_now: { label: '今いるマスを調べて返す', category: '調べる' },
  is_land: { label: '陸なら「はい」を返す', category: '調べる', blockMessage: '%1 は陸' },
  is_goal: { label: 'ゴールなら「はい」を返す', category: '調べる', blockMessage: '%1 はゴール' },
  is_checkpoint: {
    label: 'チェックポイントなら「はい」を返す',
    category: '調べる',
    blockMessage: '%1 はチェックポイント',
  },
  is_laggage: { label: '荷物なら「はい」を返す', category: '調べる', blockMessage: '%1 は荷物' },
  is_sea: { label: '海なら「はい」を返す', category: '調べる', blockMessage: '%1 は海' },
  is_rock: { label: '岩なら「はい」を返す', category: '調べる', blockMessage: '%1 は岩' },
  not: { label: '答えを反対にして返す', category: '論理', blockMessage: '%1 ではない' },
  and: { label: '両方「はい」か返す', category: '論理', blockMessage: '%1 かつ %2' },
  or: { label: 'どちらかが「はい」か返す', category: '論理', blockMessage: '%1 または %2' },
  bool_eq: { label: '2つの答えが同じか返す', category: '論理', blockMessage: '%1 と %2 は等しい' },
  bool_neq: { label: '2つの答えが違うか返す', category: '論理', blockMessage: '%1 と %2 は等しくない' },
  add: { label: '足した数を返す', category: '計算', blockMessage: '%1 + %2' },
  minus: { label: '符号を反対にした数を返す', category: '計算', blockMessage: '- %1' },
  sub: { label: '引いた数を返す', category: '計算', blockMessage: '%1 - %2' },
  mul: { label: '掛けた数を返す', category: '計算', blockMessage: '%1 x %2' },
  div: { label: '割った数を返す', category: '計算', blockMessage: '%1 / %2' },
  integer_eq: { label: '2つの数が同じか返す', category: '計算', blockMessage: '%1 は %2 と等しい' },
  integer_neq: { label: '2つの数が違うか返す', category: '計算', blockMessage: '%1 は %2 と等しくない' },
  gt: { label: '左の数が大きいか返す', category: '計算', blockMessage: '%1 は %2 より大きい' },
  lt: { label: '左の数が小さいか返す', category: '計算', blockMessage: '%1 は %2 より小さい' },
  geq: { label: '左の数が右以上か返す', category: '計算', blockMessage: '%1 は %2 と等しいか大きい' },
  leq: { label: '左の数が右以下か返す', category: '計算', blockMessage: '%1 は %2 と等しいか小さい' },
  random_bool: { label: '「はい・いいえ」からランダムに選ぶ', category: '乱数' },
  random_integer: {
    label: 'ランダムな数を返す',
    category: '乱数',
    blockMessage: '%1 から %2 からランダムに選ぶ',
  },
  new: { label: '空のスタック', category: 'スタック' },
  push_back: {
    label: '値を足したスタックを返す',
    category: 'スタック',
    blockMessage: '%1 の一番うしろに %2 を追加したスタック',
  },
  pop_back: {
    label: '最後を除いたスタックを返す',
    category: 'スタック',
    blockMessage: '%1 の一番うしろを取り除いたスタック',
  },
  get_back: { label: '最後の値を返す', category: 'スタック', blockMessage: '%1 の一番うしろ' },
  size: { label: 'スタックの個数を返す', category: 'スタック', blockMessage: '%1 の長さ' },
  unwrap: { label: '値を取り出す', category: 'オプション', blockMessage: '%1 の値を取り出す' },
  is_some: { label: '値があるか調べる', category: 'オプション', blockMessage: '%1 に値がある' },
  print_integer: { label: '整数を表示', category: '表示', blockMessage: '%1 と言う' },
  print_bool: { label: '真理値を表示', category: '表示', blockMessage: '%1 と言う' },
  print_unit: { label: '値なしを表示', category: '表示', blockMessage: '%1 と言う' },
  print_terrain: { label: 'マスの種類を表示', category: '表示', blockMessage: '%1 と言う' },
  print_rotation: { label: '向きを表示', category: '表示', blockMessage: '%1 と言う' },
  print_distance_target: { label: '探索対象を表示', category: '表示', blockMessage: '%1 と言う' },
} satisfies Record<PicoBuiltinName, { label: string; category: string; blockMessage?: string }>

export const PICO_BUILTIN_OPTIONS: PicoFunctionOption[] = PICO_BUILTINS.map((builtin) => ({
  ...builtin,
  ...builtinLabels[builtin.name],
}))
