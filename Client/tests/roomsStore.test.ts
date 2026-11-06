import assert from 'node:assert/strict'
import { beforeEach, test } from 'node:test'
import {
  createLocalRoom,
  getLocalRoom,
  joinLocalRoom,
  leaveLocalRoom,
  listLocalRooms,
} from '../src/features/rooms/roomsStore.ts'

const entries = new Map<string, string>()
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: {
    getItem: (key: string) => entries.get(key) ?? null,
    setItem: (key: string, value: string) => entries.set(key, value),
  },
})

beforeEach(() => entries.clear())

test('second participant matches and a third cannot enter', () => {
  const room = createLocalRoom('host')
  assert.equal(room.status, 'waiting')
  assert.equal(room.participants.length, 1)

  const matched = joinLocalRoom(room.id, 'guest')
  assert.equal(matched.status, 'matched')
  assert.equal(matched.participants.length, 2)
  assert.ok(matched.matchId)
  assert.equal(joinLocalRoom(room.id, 'guest').matchId, matched.matchId)
  assert.throws(() => joinLocalRoom(room.id, 'third'), /2人そろっています/)
})

test('leaving returns the remaining participant to waiting and removes an empty room', () => {
  const room = createLocalRoom('host')
  joinLocalRoom(room.id, 'guest')
  leaveLocalRoom(room.id, 'guest')

  const waiting = getLocalRoom(room.id)
  assert.equal(waiting?.status, 'waiting')
  assert.deepEqual(waiting?.participants, ['host'])
  assert.equal(waiting?.matchId, undefined)

  leaveLocalRoom(room.id, 'host')
  assert.equal(getLocalRoom(room.id), undefined)
  assert.deepEqual(listLocalRooms(), [])
})
