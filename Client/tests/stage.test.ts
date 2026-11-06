import test from 'node:test'
import assert from 'node:assert/strict'
import { normalizeField, normalizeStage } from '../src/features/stage/stage.ts'
import { SAMPLE_FIELD } from '../src/features/stage/sampleStages.ts'

test('1つの盤面形状と岩の個数候補から全配置を生成する', () => {
  const field = normalizeField({ id: 'lesson', template: ['S??G', '..__'], rockCounts: [1, 2] })
  assert.equal(field.width, 4)
  assert.equal(field.height, 2)
  assert.equal(field.layouts.length, 3)
  assert.deepEqual(
    field.layouts.map((layout) => layout.rows[0]),
    ['S#.G', 'S.#G', 'S##G'],
  )
  assert.deepEqual(
    field.layouts.map((layout) => layout.rockCount),
    [1, 1, 2],
  )
  assert.ok(field.layouts.every((layout) => layout.rows[1] === '..__'))
})

test('候補マップでは海・荷物・置き場も変えられる', () => {
  const sample = normalizeField(SAMPLE_FIELD)
  assert.equal(sample.layouts.length, 4)
  assert.deepEqual(
    sample.layouts.map((layout) => layout.rockCount),
    [1, 1, 2, 2],
  )
  assert.notDeepEqual(sample.layouts[0].rows, sample.layouts[1].rows)
  assert.notEqual(
    sample.layouts[0].cells.find((cell) => cell.symbol === 'X')?.x,
    sample.layouts[3].cells.find((cell) => cell.symbol === 'X')?.x,
  )
  const field = normalizeField({
    id: 'lesson',
    rockCounts: [1],
    candidates: [
      { id: 'one', map: ['S#.G', '__..'] },
      { id: 'two', map: ['S.#G', '_...'] },
    ],
  })
  assert.deepEqual(
    field.layouts.map((layout) => layout.id),
    ['one', 'two'],
  )
  assert.throws(
    () => normalizeField({ template: ['S??G'], rockCounts: [1], candidates: [{ map: ['S__G'] }] }),
    /rockCounts/,
  )
})

test('固定の岩も個数へ含め、可変マス以外は変えない', () => {
  const field = normalizeField({ template: ['S#?G'], rockCounts: [1, 2] })
  assert.deepEqual(
    field.layouts.map((layout) => layout.rows[0]),
    ['S#.G', 'S##G'],
  )
  assert.throws(() => normalizeField({ template: ['S#?G'], rockCounts: [0] }), /rockCounts/)
})

test('不正な行幅、記号、別ステージの配列を拒否する', () => {
  assert.throws(() => normalizeStage({ map: ['S.', 'G'] }), /同じ長さ/)
  assert.throws(() => normalizeStage({ map: ['S?G'] }), /記号/)
  assert.throws(() => normalizeField({ stages: [{ map: ['SG'] }] }), /1つのフィールド/)
})

test('現行Stage/Field schemaの地形・配置物・初期方向を表示データに変換する', () => {
  const field = normalizeField({
    variants: [
      {
        width: 3,
        height: 2,
        terrain: [
          ['land', 'rock', 'land'],
          ['sea', 'land', 'land'],
        ],
        objects: [
          { type: 'checkpoint', order: 2, position: { x: 2, y: 0 } },
          { type: 'luggage', id: 'parcel-a', position: { x: 1, y: 1 } },
          { type: 'goal', position: { x: 2, y: 1 } },
        ],
        robot: { position: { x: 0, y: 0 }, direction: 'south' },
        drillLimit: 1,
      },
    ],
    clearCondition: { type: 'visit_checkpoints' },
  })
  const layout = field.layouts[0]
  assert.equal(layout.robot.direction, 'south')
  assert.deepEqual(layout.rows, ['S#C', '_OG'])
  assert.equal(layout.cells.find((cell) => cell.x === 2 && cell.y === 0)?.objects[0].order, 2)
  assert.equal(layout.cells.find((cell) => cell.x === 1 && cell.y === 1)?.objects[0].id, 'parcel-a')
  assert.equal(layout.rockCount, 1)
})

test('schema fieldの寸法不一致と不正な配置を拒否する', () => {
  const base = {
    width: 2,
    height: 1,
    terrain: [['land', 'land']],
    objects: [],
    robot: { position: { x: 0, y: 0 }, direction: 'east' },
    drillLimit: 0,
  }
  assert.throws(() => normalizeField({ ...base, terrain: [['land']] }), /terrain/)
  assert.throws(
    () => normalizeField({ ...base, objects: [{ type: 'goal', position: { x: 2, y: 0 } }] }),
    /object/,
  )
})
