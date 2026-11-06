import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  createLocalRoom,
  getLocalPlayerId,
  joinLocalRoom,
  leaveLocalRoom,
  listLocalRooms,
  subscribeLocalRooms,
  type LocalRoom,
} from './roomsStore.ts'

export const useRoomsStore = defineStore('rooms', () => {
  const playerId = ref('')
  const rooms = ref<LocalRoom[]>([])
  const errorMessage = ref('')
  const availableRooms = computed(() => rooms.value.filter((room) => room.status === 'waiting'))
  let unsubscribe: (() => void) | undefined

  function refresh(): void {
    rooms.value = listLocalRooms()
  }

  function start(): void {
    if (unsubscribe) return
    playerId.value = getLocalPlayerId()
    refresh()
    unsubscribe = subscribeLocalRooms(refresh)
  }

  function stop(): void {
    unsubscribe?.()
    unsubscribe = undefined
  }

  function getRoom(roomId: string): LocalRoom | undefined {
    return rooms.value.find((room) => room.id === roomId.toUpperCase())
  }

  function create(): LocalRoom | undefined {
    errorMessage.value = ''
    try {
      return createLocalRoom(playerId.value)
    } catch {
      errorMessage.value = '部屋を保存できませんでした。ブラウザの保存設定を確認してください。'
    }
  }

  function join(roomId: string): LocalRoom | undefined {
    errorMessage.value = ''
    try {
      return joinLocalRoom(roomId.trim().toUpperCase(), playerId.value)
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '部屋に参加できませんでした。'
    }
  }

  function leave(roomId: string): void {
    leaveLocalRoom(roomId, playerId.value)
  }

  return { playerId, rooms, availableRooms, errorMessage, start, stop, refresh, getRoom, create, join, leave }
})
