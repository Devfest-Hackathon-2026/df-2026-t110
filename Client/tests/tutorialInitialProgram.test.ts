import test from 'node:test'
import assert from 'node:assert/strict'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import { initialTutorialProgram } from '../src/features/tutorial/tutorialApi.ts'

const tutorial = {
  id: 'turn',
  title: '向きを変える',
  description: '見本の向きを変える',
  stage: {},
}

test('教材のinitialProgramを最初のエディタ内容として選ぶ', () => {
  const sample = newProgram()
  sample.picolang.v0.program.main!.body.stmts.unshift(
    {
      kind: 'call',
      hintId: 'a0000002-0000-4000-8000-000000000002',
      func: 'rotate',
      args: [
        { kind: 'literal', hintId: 'a0000002-0000-4000-8000-000000000001', type: 'rotation', value: 'right' },
      ],
    },
    { kind: 'call', hintId: 'a0000002-0000-4000-8000-000000000003', func: 'forward', args: [] },
  )
  assert.equal(initialTutorialProgram({ ...tutorial, initialProgram: sample }), sample)
  assert.deepEqual(
    sample.picolang.v0.program.main?.body.stmts.map((stmt) => stmt.kind),
    ['call', 'call', 'return'],
  )
})

test('initialProgramのない教材は空のプログラムから始める', () => {
  const initial = initialTutorialProgram(tutorial)
  assert.deepEqual(
    initial.picolang.v0.program.main?.body.stmts.map((stmt) => stmt.kind),
    ['return'],
  )
})

test('不正な初期プログラムを下書きで隠さない', () => {
  assert.throws(
    () => initialTutorialProgram({ ...tutorial, initialProgram: {} as ReturnType<typeof newProgram> }),
    /初期プログラムの形式/,
  )
})
