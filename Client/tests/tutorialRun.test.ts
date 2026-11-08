import test from 'node:test'
import assert from 'node:assert/strict'
import { executePicolang } from '../src/features/picolang/language/executePicolang.ts'
import {
  defaultExpr,
  id,
  makeStatement,
  makeCall,
  PICO_DISTANCE_TARGETS,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'
import { normalizeStage } from '../src/features/stage/stage.ts'
import { createTutorialWorld } from '../src/features/tutorial/tutorialWorld.ts'
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

test('mainの命令を順番に実行し、forの半開区間を守る', async () => {
  const source = newProgram()
  const loop = makeStatement('for', programFunctions(source))
  if (loop.kind !== 'for') throw new Error('forを作れませんでした')
  if (loop.end.kind === 'literal') loop.end.value = 3
  loop.body.stmts.push(makeStatement('forward', programFunctions(source)))
  programFunctions(source).main.body.stmts.unshift(loop)
  const calls: string[] = []
  await executePicolang(
    source,
    {
      async call(name) {
        calls.push(name)
        return true
      },
    },
    new AbortController().signal,
  )
  assert.deepEqual(calls, ['forward', 'forward', 'forward'])
})

test('実行を中断すると後続の命令を呼ばない', async () => {
  const source = newProgram()
  programFunctions(source).main.body.stmts.unshift(
    makeStatement('forward', programFunctions(source)),
    makeStatement('back', programFunctions(source)),
  )
  const controller = new AbortController()
  const calls: string[] = []
  await assert.rejects(
    executePicolang(
      source,
      {
        async call(name) {
          calls.push(name)
          controller.abort()
          return true
        },
      },
      controller.signal,
    ),
    { name: 'AbortError' },
  )
  assert.deepEqual(calls, ['forward'])
})

test('削除したboolとintegerの変換関数を組み込み関数として実行できない', async () => {
  for (const [name, type] of [
    ['bool_to_integer', 'bool'],
    ['integer_to_bool', 'integer'],
  ] as const) {
    const source = newProgram()
    programFunctions(source).main.body.stmts.unshift({
      kind: 'call',
      hintId: id(),
      func: name,
      args: [defaultExpr(type)],
    })
    const world = createTutorialWorld(normalizeStage({ map: ['SG'] }))
    await assert.rejects(
      executePicolang(
        source,
        {
          async call(func, args) {
            return world.call(func, args).value
          },
        },
        new AbortController().signal,
      ),
      { message: `未対応の関数: ${name}` },
    )
  }
})

test('岩で停止し、ビーム後に前進してゴールへ到達できる', () => {
  const stage = normalizeStage({ map: ['S#G'] })
  const world = createTutorialWorld(stage)
  assert.deepEqual(world.call('forward', []).animation, { type: 'forward', success: false })
  assert.equal(world.call('drill', []).value, true)
  assert.equal(stage.cells[1].terrain, 'land')
  assert.equal(stage.rows[0], 'S.G')
  assert.equal(world.call('forward', []).value, true)
  assert.equal(world.call('forward', []).value, true)
  assert.equal(world.reachedGoal, true)
})

test('追加したprimitiveのprintがunitを返し、観測値とリテラルを表示できる', async () => {
  const source = newProgram()
  const prints = ['print_unit', 'print_terrain', 'print_rotation', 'print_distance_target'].map((name) =>
    makeCall(name),
  )
  prints[1].args[0] = makeCall('search_front')
  programFunctions(source).main.body.stmts.unshift(...prints)
  const stage = normalizeStage({ map: ['SG'] })
  const before = structuredClone(stage)
  const world = createTutorialWorld(stage)
  const messages: string[] = []
  await executePicolang(
    source,
    {
      async call(name, args) {
        const result = world.call(name, args)
        if (result.speech) {
          assert.equal(result.value, null)
          messages.push(result.speech.label)
        }
        return result.value
      },
    },
    new AbortController().signal,
  )
  assert.deepEqual(messages, ['()', 'ゴール', 'left', 'goal'])
  assert.deepEqual(stage, before)
  for (const target of PICO_DISTANCE_TARGETS)
    assert.equal(world.call('print_distance_target', [target]).speech?.label, target)
  assert.equal(world.call('print_rotation', ['right']).speech?.label, 'right')
})
