import test from 'node:test'
import assert from 'node:assert/strict'
import { executePicolang } from '../src/features/picolang/language/executePicolang.ts'
import {
  defaultExpr,
  id,
  makeStatement,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'
import { normalizeStage } from '../src/features/stage/stage.ts'
import { createTutorialWorld } from '../src/features/tutorial/tutorialWorld.ts'

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
