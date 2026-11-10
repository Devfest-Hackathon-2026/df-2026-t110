<script setup lang="ts">
import { computed } from 'vue'
import './roomMarkerTracking.css'
import type { RoomMarkerState } from './roomMarkerTracking.ts'

const props = defineProps<{
  markerState: RoomMarkerState
  markerActive: boolean
  markerLoading: boolean
  cameraReady: boolean
  placementConfirmed: boolean
  trackingPoseAvailable: boolean
  calibrated: boolean
}>()
const emit = defineEmits<{ confirm: [] }>()
const markerSeen = computed(() => props.markerState.a || props.markerState.b)
const markerCount = computed(() => Number(props.markerState.a) + Number(props.markerState.b))
</script>

<template>
  <div v-if="markerLoading || markerActive" class="marker-overlay">
    <div class="marker-status" role="status">
      <strong>{{ cameraReady ? `マーカー検出 ${markerCount}/2` : 'カメラ準備中' }}</strong>
      <span :class="{ detected: markerState.a }">Hiro {{ markerState.a ? '検出中' : '未検出' }}</span>
      <span :class="{ detected: markerState.b }">Kanji {{ markerState.b ? '検出中' : '未検出' }}</span>
    </div>
    <p v-if="markerLoading" class="marker-guidance" role="status">マーカー追跡を準備しています。</p>
    <div
      v-if="markerActive && (!markerSeen || !placementConfirmed || !trackingPoseAvailable || !calibrated)"
      class="marker-guidance"
    >
      <p v-if="!cameraReady" role="status">カメラを起動しています。アクセスを許可してください。</p>
      <p v-else-if="!markerSeen" role="status">
        {{
          placementConfirmed
            ? 'マーカーを見失いました。Hiro または Kanji をもう一度映してください。'
            : 'Hiro または Kanji マーカーをカメラに映してください。'
        }}
        <template v-if="!placementConfirmed">
          <a href="/markers/marker-hiro.png" target="_blank" rel="noopener">Hiro</a>・<a
            href="/markers/marker-kanji.png"
            target="_blank"
            rel="noopener"
            >Kanji</a
          >
        </template>
      </p>
      <button v-else-if="!placementConfirmed" type="button" @click.stop="emit('confirm')">配置を確定</button>
      <p v-else-if="!trackingPoseAvailable" role="status">
        配置を決めたときのマーカーを映してください。そのまま、もう1枚も一緒に映してください。
      </p>
      <p v-else role="status">もう1枚のマーカーも一度同時に映すと、見失ったときに切り替えられます。</p>
    </div>
  </div>
</template>

<style scoped>
.marker-overlay {
  position: absolute;
  z-index: 4;
  bottom: max(76px, calc(env(safe-area-inset-bottom) + 76px));
  right: max(8px, env(safe-area-inset-right));
  display: flex;
  width: min(280px, calc(100% - 16px));
  max-height: min(45%, 240px);
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
.marker-status {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  flex: none;
  padding: 8px;
  border-radius: 8px;
  background: #172c24;
  color: white;
  font-size: 13px;
}
.marker-status strong,
.marker-status span {
  padding: 3px 6px;
}
.marker-status strong {
  grid-column: 1 / -1;
}
.marker-status span {
  border-radius: 4px;
  background: #405047;
}
.marker-status span.detected {
  background: #b5f7d0;
  color: #172c24;
}
.marker-guidance {
  flex: none;
  margin: 0;
  padding: 10px 14px;
  border-radius: 8px;
  background: #172c24;
  color: white;
}
.marker-guidance a {
  color: #b5f7d0;
}
.marker-guidance p {
  margin: 0;
}
.marker-guidance button {
  border: 0;
  border-radius: 6px;
  padding: 8px 12px;
  background: #b5f7d0;
  color: #172c24;
  font: inherit;
  cursor: pointer;
}
@media (max-width: 600px) {
  .marker-overlay {
    width: min(280px, calc(100% - 104px));
  }
}
</style>
