import test from 'node:test'
import assert from 'node:assert/strict'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import { usePicolangPlayback } from '../src/features/picolang/editor/playback/usePicolangPlayback.ts'
import { normalizeStage } from '../src/features/stage/stage.ts'

async function flush() {
  for (let n = 0; n < 20; n++) await Promise.resolve()
}
function fixture() {
  const program = newProgram()
  const fn = program.picolang.v0.program.main
  const history = [
    { type: 'field', tick: 0, state: { kind: 'initial', terrain: [[{ type: 'land' }]], robots: [] } },
    ...[fn.hintId, fn.body.hintId, fn.body.stmts[0].hintId].map((hintId) => ({
      type: 'picolang',
      interpreterId: 'robot-1',
      event: { kind: 'current_node', hintId },
    })),
  ]
  return {
    program,
    result: { history, summary: { cleared: false } },
    target: { stage: normalizeStage({ map: ['S'] }), refreshVisuals() {}, setCarrying() {} },
  }
}

test('pause holds the current node, stepping advances once, and reset cancels a paused replay', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const { program, result, target } = fixture()
  const player = usePicolangPlayback()
  const pending = player.play(result, program, target, new AbortController().signal)
  const cancelled = assert.rejects(pending, { name: 'AbortError' })
  await flush()
  const first = player.executionTrace.value
  assert.ok(first?.current)
  player.togglePause()
  t.mock.timers.tick(300)
  await flush()
  assert.equal(player.executionTrace.value, first)
  player.stepPlayback()
  t.mock.timers.tick(50)
  await flush()
  assert.notEqual(player.executionTrace.value?.current?.hintId, first.current.hintId)
  const second = player.executionTrace.value
  t.mock.timers.tick(300)
  await flush()
  assert.equal(player.executionTrace.value, second)
  player.reset()
  await cancelled
  t.mock.timers.tick(1000)
  await flush()
  assert.equal(player.executionTrace.value, null)
  assert.equal(player.playbackReady.value, false)
  assert.equal(player.playbackPaused.value, false)
})

test('restarting a replay prevents the previous replay cleanup from changing the new state', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const { program, result, target } = fixture()
  const player = usePicolangPlayback()
  const previous = player.play(result, program, target, new AbortController().signal)
  const cancelled = assert.rejects(previous, { name: 'AbortError' })
  await flush()
  const signal = new AbortController()
  const current = player.play(result, program, target, signal.signal)
  const interrupted = assert.rejects(current, { name: 'AbortError' })
  await cancelled
  await flush()
  assert.equal(player.playbackReady.value, true)
  assert.ok(player.executionTrace.value?.current)
  signal.abort()
  await interrupted
  assert.equal(player.playbackReady.value, false)
})
