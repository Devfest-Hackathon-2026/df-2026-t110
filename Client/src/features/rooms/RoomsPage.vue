<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import swordsIcon from '@iconify-icons/lucide/swords'
import usersIcon from '@iconify-icons/lucide/users'
import { useRoomsStore } from './useRoomsStore.ts'
import { mountRoomMarkerTracker, type RoomMarkerTracker } from './roomMarkerTracking.ts'

const props = defineProps<{ roomId?: string | null }>()
const roomsStore = useRoomsStore()
const currentRoom = computed(() => (props.roomId ? roomsStore.getRoom(props.roomId) : undefined))
const code = ref('')
const copyMessage = ref('')
const markerScannerOpen = ref(false)
const markerContainer = ref<HTMLElement | null>(null)
const markerState = ref({ a: false, b: false })
const cameraMessage = ref('カメラを準備しています…')
let markerTracker: RoomMarkerTracker | undefined
let scannerVersion = 0

const isMember = computed(() => currentRoom.value?.participants.includes(roomsStore.playerId) ?? false)

onMounted(() => roomsStore.start())
onUnmounted(() => {
  closeMarkerScanner()
  roomsStore.stop()
})

async function openMarkerScanner(): Promise<void> {
  if (markerScannerOpen.value) return
  const version = ++scannerVersion
  markerState.value = { a: false, b: false }
  cameraMessage.value = 'カメラを準備しています…'
  markerScannerOpen.value = true
  await nextTick()
  if (!markerContainer.value || version !== scannerVersion) return
  try {
    const tracker = await mountRoomMarkerTracker(markerContainer.value, {
      onChange: (state) => {
        markerState.value = state
      },
      onCameraReady: () => {
        cameraMessage.value = 'カメラ準備完了'
      },
      onCameraError: (error) => {
        cameraMessage.value = error instanceof Error ? error.message : 'カメラを開始できませんでした'
      },
    })
    if (version === scannerVersion) markerTracker = tracker
    else tracker.stop()
  } catch (error) {
    if (version === scannerVersion) {
      cameraMessage.value = error instanceof Error ? error.message : 'マーカー読み取りを開始できませんでした'
    }
  }
}

function closeMarkerScanner(): void {
  scannerVersion++
  markerTracker?.stop()
  markerTracker = undefined
  markerScannerOpen.value = false
}

function openRoom(roomId: string): void {
  window.location.assign(`/rooms/${encodeURIComponent(roomId)}`)
}

function createRoom(): void {
  const room = roomsStore.create()
  if (room) openRoom(room.id)
}

function joinRoom(roomId: string): void {
  const room = roomsStore.join(roomId)
  if (room) openRoom(room.id)
}

function leaveRoom(): void {
  if (currentRoom.value) roomsStore.leave(currentRoom.value.id)
  window.location.assign('/rooms')
}

async function copyLink(): Promise<void> {
  if (!currentRoom.value) return
  try {
    await navigator.clipboard.writeText(`${window.location.origin}/rooms/${currentRoom.value.id}`)
    copyMessage.value = '参加リンクをコピーしました'
  } catch {
    copyMessage.value = 'コピーできませんでした。アドレス欄のURLを共有してください。'
  }
}
</script>

<template>
  <main class="rooms-main container">
    <header class="rooms-heading">
      <h1 class="heading-with-icon">
        <Icon :icon="swordsIcon" aria-hidden="true" />
        <span>{{ roomId ? '対戦の待機室' : '対戦相手を探す' }}</span>
      </h1>
      <p>
        {{
          roomId
            ? '同じ部屋に2人そろうと、マッチングが成立します。'
            : '部屋を作るか、共有されたコードで参加できます。'
        }}
      </p>
    </header>

    <p class="rooms-demo-note">
      クライアントだけの試作です。同じブラウザの別タブ間で同期します。別端末との対戦にはサーバー接続が必要です。
    </p>
    <p v-if="roomsStore.errorMessage" class="rooms-feedback" role="alert">{{ roomsStore.errorMessage }}</p>

    <template v-if="roomId">
      <a class="rooms-back" href="/rooms">部屋一覧へ戻る</a>
      <section v-if="currentRoom" class="room-waiting" aria-labelledby="room-code-title">
        <div class="room-code-row">
          <div>
            <span class="rooms-label">部屋コード</span>
            <h2 id="room-code-title">{{ currentRoom.id }}</h2>
          </div>
          <button type="button" class="room-copy-button" @click="copyLink">参加リンクをコピー</button>
        </div>
        <p v-if="copyMessage" class="rooms-feedback" role="status">{{ copyMessage }}</p>

        <div class="room-status-line" role="status" aria-live="polite">
          <Icon :icon="usersIcon" aria-hidden="true" />
          <strong>{{ currentRoom.participants.length }} / 2 人</strong>
          <span>{{ currentRoom.status === 'matched' ? 'マッチング成立' : '相手を待っています' }}</span>
        </div>
        <ol class="room-players" aria-label="参加者">
          <li v-for="(participant, index) in currentRoom.participants" :key="participant">
            <span>{{ index + 1 }}</span>
            <strong>{{ participant === roomsStore.playerId ? 'あなた' : '対戦相手' }}</strong>
            <span>参加中</span>
          </li>
          <li v-if="currentRoom.participants.length < 2" class="room-empty-player">
            <span>2</span><strong>空き</strong><span>待機中</span>
          </li>
        </ol>

        <p v-if="currentRoom.status === 'matched'" class="room-next-note">
          2人そろいました。対戦画面への接続は次の実装で追加します。
        </p>
        <div class="room-actions">
          <button
            v-if="!isMember && currentRoom.status === 'waiting'"
            type="button"
            class="action-primary"
            @click="joinRoom(currentRoom.id)"
          >
            この部屋に参加
          </button>
          <button v-if="isMember" type="button" class="room-text-button" @click="leaveRoom">
            部屋から退出
          </button>
        </div>
      </section>
      <p v-else class="rooms-feedback" role="alert">部屋が見つかりません。コードを確認してください。</p>
    </template>

    <template v-else>
      <div class="rooms-controls">
        <section aria-labelledby="create-room-title">
          <h2 id="create-room-title">部屋を作る</h2>
          <p>6文字の部屋コードを発行します。</p>
          <button type="button" class="action-primary" @click="createRoom">新しい部屋を作成</button>
          <button type="button" class="room-text-button room-marker-open" @click="openMarkerScanner">
            ARマーカーを読み取る
          </button>
        </section>
        <section aria-labelledby="join-room-title">
          <h2 id="join-room-title">コードで参加</h2>
          <form class="rooms-join-form" @submit.prevent="joinRoom(code)">
            <label for="room-code-input">部屋コード</label>
            <div>
              <input
                id="room-code-input"
                v-model="code"
                type="text"
                maxlength="6"
                autocomplete="off"
                autocapitalize="characters"
                required
                placeholder="例: A1B2C3"
              />
              <button type="submit">参加</button>
            </div>
          </form>
        </section>
      </div>
      <section class="rooms-list" aria-labelledby="rooms-list-title">
        <h2 id="rooms-list-title">待機中の部屋</h2>
        <p v-if="roomsStore.availableRooms.length === 0">まだ部屋はありません。新しく作成してください。</p>
        <button
          v-for="room in roomsStore.availableRooms"
          :key="room.id"
          type="button"
          class="rooms-list-row"
          @click="joinRoom(room.id)"
        >
          <strong>{{ room.id }}</strong
          ><span>{{ room.participants.length }} / 2 人</span><span>参加する</span>
        </button>
      </section>
    </template>
    <div
      v-if="markerScannerOpen"
      class="room-marker-scanner"
      role="dialog"
      aria-modal="true"
      aria-label="ARマーカー読み取り"
    >
      <div ref="markerContainer" class="room-marker-camera"></div>
      <div class="room-marker-top">
        <strong>ARマーカー読み取り</strong>
        <button type="button" @click="closeMarkerScanner">閉じる</button>
      </div>
      <div class="room-marker-bottom">
        <p>HiroとKanjiを1枚ずつ、次に2枚を同時に映してください。</p>
        <div class="room-marker-statuses" aria-live="polite">
          <span :class="{ 'is-found': markerState.a }"
            >A Hiro · {{ markerState.a ? '認識中' : '未検出' }}</span
          >
          <span :class="{ 'is-found': markerState.b }"
            >B Kanji · {{ markerState.b ? '認識中' : '未検出' }}</span
          >
        </div>
        <p class="room-marker-result">
          {{ markerState.a && markerState.b ? '2枚とも認識しています' : cameraMessage }}
        </p>
        <p class="room-marker-note">現在は読み取り確認のみです。部屋の作成には接続していません。</p>
      </div>
    </div>
  </main>
</template>
