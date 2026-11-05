import test from 'node:test'
import assert from 'node:assert/strict'
import { normalizeStage, normalizeStages } from '../src/stage.js'

test('文字列マップを座標付きセルへ変換する', () => {
  const stage = normalizeStage({ id: 'test', map: ['S.#', '_OG'] })
  assert.equal(stage.width, 3)
  assert.equal(stage.height, 2)
  assert.deepEqual(stage.cells.find(cell => cell.symbol === '#'), { x: 2, y: 0, symbol: '#' })
  assert.deepEqual(stage.cells.find(cell => cell.symbol === 'G'), { x: 2, y: 1, symbol: 'G' })
})

test('単一ステージと候補リストを受け取る', () => {
  assert.equal(normalizeStages({ map: ['SG'] }).length, 1)
  assert.equal(normalizeStages({ candidates: [{ map: ['SG'] }, { map: ['S.G'] }] }).length, 2)
})

test('不正な行幅や記号を拒否する', () => {
  assert.throws(() => normalizeStage({ map: ['S.', 'G'] }), /同じ長さ/)
  assert.throws(() => normalizeStage({ map: ['S?G'] }), /記号/)
})
