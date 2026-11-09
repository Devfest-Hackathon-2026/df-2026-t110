import test from 'node:test'
import assert from 'node:assert/strict'
import { speechForValue } from '../src/features/tutorial/tutorialSpeech.ts'

test('発話は配列を横並びに分け、観測した地形をモデル用の値として保持する', () => {
  const values = speechForValue([2, 5, 8], 1)
  assert.deepEqual(values.items, ['2', '5', '8'])
  assert.equal(values.label, '[2, 5, 8]')
  const terrain = { type: 'land' as const, kind: { type: 'goal' as const }, hasLuggage: true }
  const observed = speechForValue(terrain, 2)
  assert.equal(observed.terrain, terrain)
  assert.deepEqual(observed.items, ['ゴール', '荷物あり'])
})
