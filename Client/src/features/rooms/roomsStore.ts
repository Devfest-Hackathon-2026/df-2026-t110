export type RoomStatus = 'waiting' | 'matched'

export interface LocalRoom {
  id: string
  participants: string[]
  status: RoomStatus
  matchId?: string
  createdAt: number
}

const roomsKey = 'robot-field:local-rooms:v1'
const playerKey = 'robot-field:local-player:v1'
const roomLifetime = 24 * 60 * 60 * 1000
const listeners = new Set<() => void>()

function randomId(): string {
  return crypto.randomUUID()
}

function readRooms(): LocalRoom[] {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(roomsKey) ?? '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (room): room is LocalRoom =>
        typeof room === 'object' &&
        room !== null &&
        typeof room.id === 'string' &&
        Array.isArray(room.participants) &&
        typeof room.createdAt === 'number' &&
        Date.now() - room.createdAt < roomLifetime,
    )
  } catch {
    return []
  }
}

function saveRooms(rooms: LocalRoom[]): void {
  localStorage.setItem(roomsKey, JSON.stringify(rooms))
  listeners.forEach((listener) => listener())
}

export function getLocalPlayerId(): string {
  let id = sessionStorage.getItem(playerKey)
  if (!id) {
    id = randomId()
    sessionStorage.setItem(playerKey, id)
  }
  return id
}

export function listLocalRooms(): LocalRoom[] {
  return readRooms().sort((a, b) => b.createdAt - a.createdAt)
}

export function getLocalRoom(roomId: string): LocalRoom | undefined {
  return readRooms().find((room) => room.id === roomId.toUpperCase())
}

export function createLocalRoom(playerId: string): LocalRoom {
  const rooms = readRooms()
  let id: string
  do {
    id = randomId().replace(/-/g, '').slice(0, 6).toUpperCase()
  } while (rooms.some((room) => room.id === id))
  const room: LocalRoom = {
    id,
    participants: [playerId],
    status: 'waiting',
    createdAt: Date.now(),
  }
  saveRooms([...rooms, room])
  return room
}

export function joinLocalRoom(roomId: string, playerId: string): LocalRoom {
  const rooms = readRooms()
  const room = rooms.find((item) => item.id === roomId.toUpperCase())
  if (!room) throw new Error('部屋が見つかりません。コードを確認してください。')
  if (room.participants.includes(playerId)) return room
  if (room.participants.length >= 2) throw new Error('この部屋は2人そろっています。')
  room.participants.push(playerId)
  room.status = 'matched'
  room.matchId = randomId()
  saveRooms(rooms)
  return room
}

export function leaveLocalRoom(roomId: string, playerId: string): void {
  const rooms = readRooms()
  const room = rooms.find((item) => item.id === roomId.toUpperCase())
  if (!room || !room.participants.includes(playerId)) return
  room.participants = room.participants.filter((id) => id !== playerId)
  room.status = 'waiting'
  delete room.matchId
  saveRooms(rooms.filter((item) => item.participants.length > 0))
}

export function subscribeLocalRooms(listener: () => void): () => void {
  listeners.add(listener)
  const onStorage = (event: StorageEvent) => {
    if (event.key === roomsKey) listener()
  }
  window.addEventListener('storage', onStorage)
  return () => {
    listeners.delete(listener)
    window.removeEventListener('storage', onStorage)
  }
}
