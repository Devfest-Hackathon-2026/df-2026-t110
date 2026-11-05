import test from 'node:test'
import assert from 'node:assert/strict'
import { chooseLayoutIndex, resolveServerSelection, rouletteSteps } from '../src/roulette.js'

test('抽選結果は閲覧中の配置に依存せず乱数で決まる', () => {
  assert.equal(chooseLayoutIndex(3, () => 0), 0)
  assert.equal(chooseLayoutIndex(3, () => 0.5), 1)
  assert.equal(chooseLayoutIndex(3, () => 0.9999), 2)
})

test('ルーレットは候補を複数周して選ばれた配置で止まる', () => {
  const steps = rouletteSteps(2, 1, 3)
  assert.ok(steps.length >= 12)
  assert.equal(steps.at(-1), 1)
  assert.deepEqual(rouletteSteps(0, 0, 1), [0])
})

test('サーバーが指定した候補で止まり、不明な結果を拒否する', () => {
  const layouts = [{ id: 'a' }, { id: 'b' }]
  assert.equal(resolveServerSelection(layouts, { selectedLayoutId: 'b' }), 1)
  assert.equal(resolveServerSelection(layouts, { candidateIndex: 0 }), 0)
  assert.throws(() => resolveServerSelection(layouts, { selectedLayoutId: 'missing' }), /抽選結果/)
})
