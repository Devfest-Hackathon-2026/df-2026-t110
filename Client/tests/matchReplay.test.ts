import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import {
  parseBoardEventHistory,
  replayFrameToStage,
  replayFrames,
} from '../src/features/match/matchReplay.ts'
import {
  getMe,
  parseIapIdentity,
  parseMatch,
  parseMatchResult,
  parseMatchUser,
} from '../src/features/match/matchApi.ts'

test('追加したprimitiveのprintを観測時の値でリプレイする', () => {
  const history = parseBoardEventHistory(
    JSON.stringify([
      {
        type: 'field',
        tick: 0,
        state: {
          kind: 'initial',
          terrain: [[{ type: 'land', kind: { type: 'plain' } }]],
          robots: [{ id: 'a', position: { x: 0, y: 0 }, direction: 'east' }],
        },
      },
      ...[
        { kind: 'print_unit' },
        {
          kind: 'print_terrain',
          terrain: { type: 'land', kind: { type: 'checkpoint', order: 2 }, hasLuggage: true },
        },
        { kind: 'print_rotation', rotation: 'right' },
        { kind: 'print_distance_target', distanceTarget: 'enemy' },
      ].map((action, index) => ({
        type: 'field',
        tick: index + 1,
        state: { kind: 'update', actions: [{ ...action, robotId: 'a' }] },
      })),
    ]),
  )
  const frames = replayFrames(history)
  assert.deepEqual(
    frames.slice(1).map((frame) => frame.robots[0].speech),
    ['()', '{"type":"land","kind":{"type":"checkpoint","order":2},"hasLuggage":true}', 'right', 'enemy'],
  )
  assert.equal(frames[0].robots[0].speech, undefined)
  assert.deepEqual(frames[4].robots[0].position, { x: 0, y: 0 })
  assert.equal(frames[4].robots[0].direction, 'east')
})

test('Go BoardOutput events can be imported for replay', () => {
  const initial = {
    type: 'field',
    tick: 0,
    state: {
      kind: 'initial',
      terrain: [[{ type: 'land', kind: { type: 'plain' } }, { type: 'rock' }]],
      robots: [{ id: 'a', position: { x: 0, y: 0 }, direction: 'east' }],
    },
  }
  const update = {
    type: 'field',
    tick: 1,
    state: {
      kind: 'update',
      actions: [{ kind: 'drill', robotId: 'a', success: true, target: { x: 1, y: 0 }, drillDurability: 2 }],
    },
  }
  const history = parseBoardEventHistory(
    JSON.stringify([{ Events: [initial] }, { Decision: { RobotID: 'a' }, Events: [update] }]),
  )
  const frames = replayFrames(history)
  assert.equal(frames.length, 2)
  assert.equal(frames[1].terrain[0][1].type, 'land')
  assert.throws(() => parseBoardEventHistory(JSON.stringify([{ Events: [update] }])), /初期盤面/)
})

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

test('IAP identity maps the Google account email and subject to the account display', () => {
  assert.deepEqual(
    parseIapIdentity({
      email: 'accounts.google.com:ada@example.com',
      sub: 'accounts.google.com:google-subject',
    }),
    { id: 'google-subject', name: 'ada@example.com' },
  )
  assert.throws(() => parseIapIdentity({ email: 'ada@example.com', sub: 'google-subject' }))
})

test('deployed account falls back to IAP identity while /api/me is unavailable', async () => {
  const originalFetch = globalThis.fetch
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const calls: string[] = []
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { location: { origin: 'https://server-639320660675.asia-northeast1.run.app' } },
  })
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    calls.push(String(input))
    if (calls.length === 1) return new Response('{}', { status: 501 })
    return new Response(
      JSON.stringify({
        email: 'accounts.google.com:ada@example.com',
        sub: 'accounts.google.com:google-subject',
      }),
      { headers: { 'content-type': 'application/json' } },
    )
  }) as typeof fetch
  try {
    assert.deepEqual(await getMe(), { id: 'google-subject', name: 'ada@example.com' })
    assert.deepEqual(calls, [
      '/api/me',
      'https://server-639320660675.asia-northeast1.run.app/account?gcp-iap-mode=IDENTITY',
    ])
  } finally {
    globalThis.fetch = originalFetch
    if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow)
    else Reflect.deleteProperty(globalThis, 'window')
  }
})

test('schema history replays drill, messages and wait without losing board state', () => {
  const history = JSON.parse(
    readFileSync(new URL('../../Schema/game_event/examples/history.json', import.meta.url), 'utf8'),
  ) as unknown[]
  const frames = replayFrames(history)
  const at = (tick: number) => frames.find((frame) => frame.tick === tick)!
  assert.equal(at(2).terrain[0][2].hasLuggage, false)
  assert.equal(at(3).robots.find((robot) => robot.id === 'robot-2')?.speech, '-42')
  assert.equal(at(4).robots.find((robot) => robot.id === 'robot-2')?.speech, 'false')
  assert.deepEqual(at(5).terrain[1][3], { type: 'land', kind: { type: 'plain' } })
  assert.equal(at(5).robots.find((robot) => robot.id === 'robot-2')?.drillDurability, 1)
  assert.equal(at(7).actions[0].ticks, 3)
  assert.equal(frames.at(-1)?.tick, 9)
  assert.deepEqual(frames.at(-1)?.robots, [])
  const stage = replayFrameToStage(at(5))
  assert.equal(stage.width, 6)
  assert.equal(stage.cells.find((cell) => cell.x === 3 && cell.y === 1)?.terrain, 'land')
  assert.equal(stage.cells.find((cell) => cell.x === 3 && cell.y === 0)?.objects[0]?.type, 'goal')
})
