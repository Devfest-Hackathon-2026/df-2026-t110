import assert from 'node:assert/strict'
import test from 'node:test'
import { replayFrames } from '../src/features/match/matchReplay.ts'
import { parseMatch, parseMatchResult } from '../src/features/match/matchApi.ts'

test('field history replays movement, rotation and removal in order', () => {
  const frames = replayFrames([
    { type: 'picolang', interpreterId: 'a', event: {} },
    {
      type: 'field',
      tick: 0,
      state: {
        kind: 'initial',
        terrain: [[{ type: 'land', kind: { type: 'plain' } }, { type: 'rock' }]],
        robots: [{ id: 'r1', position: { x: 0, y: 0 }, direction: 'east' }],
      },
    },
    {
      type: 'field',
      tick: 1,
      state: {
        kind: 'update',
        actions: [{ kind: 'forward', robotId: 'r1', success: true, to: { x: 1, y: 0 } }],
      },
    },
    {
      type: 'field',
      tick: 2,
      state: {
        kind: 'update',
        actions: [{ kind: 'rotate', robotId: 'r1', rotation: 'left', direction: 'north' }],
        removedRobotIds: ['r1'],
      },
    },
  ])
  assert.equal(frames.length, 3)
  assert.deepEqual(frames[0].robots[0].position, { x: 0, y: 0 })
  assert.deepEqual(frames[1].robots[0].position, { x: 1, y: 0 })
  assert.equal(frames[1].robots[0].direction, 'east')
  assert.deepEqual(frames[2].robots, [])
})

test('match responses require identifiers and scores', () => {
  assert.equal(parseMatch({ id: 'm1', fieldId: 'f1', selectedLayoutId: 'l1' }).id, 'm1')
  assert.throws(() => parseMatch({ id: 'm1' }))
  assert.deepEqual(
    parseMatchResult({ history: [], summary: { score: { player: 2, opposite: 1 } } }).summary.score,
    { player: 2, opposite: 1 },
  )
  assert.throws(() => parseMatchResult({ history: [], summary: {} }))
})
