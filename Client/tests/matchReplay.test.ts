import assert from 'node:assert/strict'
import test from 'node:test'
import { replayFrames } from '../src/features/match/matchReplay.ts'
import { parseMatch, parseMatchResult, parseMatchUser } from '../src/features/match/matchApi.ts'

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

test('pickup and release replay luggage transfers and discards', () => {
  const frames = replayFrames([
    {
      type: 'field',
      tick: 0,
      state: {
        kind: 'initial',
        terrain: [[{ type: 'land' }, { type: 'land', hasLuggage: true, territoryOwner: 2 }]],
        robots: [{ id: 'r1', position: { x: 0, y: 0 }, direction: 'east' }],
      },
    },
    {
      type: 'field',
      tick: 1,
      state: { kind: 'update', actions: [{ kind: 'pick_up', robotId: 'r1', success: false }] },
    },
    {
      type: 'field',
      tick: 2,
      state: {
        kind: 'update',
        actions: [{ kind: 'pick_up', robotId: 'r1', success: true, target: { x: 1, y: 0 } }],
      },
    },
    {
      type: 'field',
      tick: 3,
      state: {
        kind: 'update',
        actions: [{ kind: 'release', robotId: 'r1', success: true, target: { x: 1, y: 0 } }],
      },
    },
    {
      type: 'field',
      tick: 4,
      state: {
        kind: 'update',
        actions: [{ kind: 'pick_up', robotId: 'r1', success: true, target: { x: 1, y: 0 } }],
      },
    },
    {
      type: 'field',
      tick: 5,
      state: {
        kind: 'update',
        actions: [{ kind: 'rotate', robotId: 'r1', direction: 'north' }],
      },
    },
    {
      type: 'field',
      tick: 6,
      state: {
        kind: 'update',
        actions: [{ kind: 'release', robotId: 'r1', success: true, discarded: true }],
      },
    },
  ])
  assert.equal(frames[0].terrain[0][1].hasLuggage, true)
  assert.equal(frames[1].terrain[0][1].hasLuggage, true)
  assert.equal(frames[1].robots[0].carryingLuggage, undefined)
  assert.deepEqual(frames[2].terrain[0][1], { type: 'land', hasLuggage: false, territoryOwner: 2 })
  assert.equal(frames[2].robots[0].carryingLuggage, true)
  assert.deepEqual(frames[3].terrain[0][1], { type: 'land', hasLuggage: true, territoryOwner: 2 })
  assert.equal(frames[3].robots[0].carryingLuggage, false)
  assert.deepEqual(frames[6].terrain, frames[5].terrain)
  assert.equal(frames[6].robots[0].carryingLuggage, false)
})

test('match responses require identifiers and scores', () => {
  assert.equal(parseMatch({ id: 'm1', fieldId: 'f1', selectedLayoutId: 'l1' }).id, 'm1')
  assert.throws(() => parseMatch({ id: 'm1' }))
  assert.deepEqual(
    parseMatchResult({ history: [], summary: { score: { player: 2, opponent: 1 } } }).summary.score,
    { player: 2, opponent: 1 },
  )
  assert.throws(() => parseMatchResult({ history: [], summary: {} }))
})

test('account response accepts an authenticated user without a registered name', () => {
  assert.deepEqual(parseMatchUser({ id: 'google-subject', name: null }), {
    id: 'google-subject',
    name: undefined,
  })
  assert.deepEqual(parseMatchUser({ name: 'Ada' }), { id: undefined, name: 'Ada' })
  assert.throws(() => parseMatchUser({ name: 12 }))
})
