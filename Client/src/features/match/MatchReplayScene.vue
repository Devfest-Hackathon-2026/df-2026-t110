<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { createArmPivots } from '../stage/armController.ts'
import {
  CARRIED_LUGGAGE_HEIGHT,
  createCarriedLuggage,
  disposeCarriedLuggage,
} from '../stage/carriedLuggage.ts'
import { createRobotAnimator, type AnimationEvent } from '../stage/robotAnimator.ts'
import { createRockBreakEffect } from '../stage/rockBreakEffect.ts'
import { loadStageAssets, type StageAssets } from '../stage/stageAssets.ts'
import {
  advanceWater,
  createStageVisuals,
  disposeStageVisuals,
  type StageAnimationTarget,
} from '../stage/stageVisuals.ts'
import {
  replayFrameToStage,
  replayPrintMessage,
  type ReplayAction,
  type ReplayFrame,
  type ReplayRobot,
} from './matchReplay.ts'

const props = defineProps<{ frames: ReplayFrame[] }>()
const host = ref<HTMLElement | null>(null)
const video = ref<HTMLVideoElement | null>(null)
const index = ref(0)
const playing = ref(false)
const transitioning = ref(false)
const mode = ref<'preview' | 'camera'>('preview')
const error = ref('')
const bubbles = ref<Array<{ id: string; text: string; x: number; y: number }>>([])
const current = computed(() => props.frames[index.value])
const actions = computed(() => current.value?.actions ?? [])
const printActions = computed(() =>
  actions.value
    .map((action) => ({ ...action, message: replayPrintMessage(action) }))
    .filter((action) => action.message !== undefined),
)
const actionLabel: Record<ReplayAction['kind'], string> = {
  forward: '前進',
  back: '後退',
  rotate: '回転',
  wait: '待機',
  drill: '岩を破壊',
  pick_up: '荷物を拾う',
  release: '荷物を置く',
  print_integer: '整数を表示',
  print_bool: '真理値を表示',
  print_unit: '値なしを表示',
  print_terrain: 'マスの種類を表示',
  print_rotation: '向きを表示',
  print_distance_target: '探索対象を表示',
}

type Rig = {
  group: THREE.Group
  model: THREE.Object3D
  animator: ReturnType<typeof createRobotAnimator>
  cargo: THREE.Group | null
  cargoFromAssets: boolean
}
function disposeRig(rig: Rig): void {
  rig.animator.stop()
  if (rig.cargo) disposeCarriedLuggage(rig.cargo)
  rig.model.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) return
    const materials = Array.isArray(object.material) ? object.material : [object.material]
    for (const material of materials) material.dispose()
  })
  rig.group.removeFromParent()
}
let renderer: THREE.WebGLRenderer | null = null
let scene: THREE.Scene | null = null
let camera: THREE.PerspectiveCamera | null = null
let board: THREE.Group | null = null
let template: THREE.Object3D | null = null
let assets: StageAssets | undefined
let animated: StageAnimationTarget[] = []
const rigs = new Map<string, Rig>()
let effects: Array<{ effect: ReturnType<typeof createRockBreakEffect>; start: number | null }> = []
let cargoTransitions: Array<{
  cargo: THREE.Group
  from: THREE.Vector3
  to: THREE.Vector3
  start: number | null
}> = []
let raf = 0
let timer: ReturnType<typeof setTimeout> | null = null
let observer: ResizeObserver | null = null
let stream: MediaStream | null = null
let alive = false
let yaw = 0.55
let pitch = 0.65
let distance = 3.25
let dragging = false
let lastPointer = [0, 0]
let eventId = 0
let cameraPlacement = { x: 0, y: -0.05 }
const heading = (frame: ReplayFrame): string =>
  frame.actions.length
    ? frame.actions
        .map(
          (action) =>
            `${action.robotId}: ${actionLabel[action.kind]}${action.success === false ? '（失敗）' : ''}${action.kind === 'wait' ? ` ${action.ticks}ターン` : ''}`,
        )
        .join(' / ')
    : frame.removedRobotIds.length
      ? 'ロボットが盤面から離れました'
      : '盤面の状態を更新'
const directionYaw: Record<ReplayRobot['direction'], number> = {
  north: 0,
  east: -Math.PI / 2,
  south: Math.PI,
  west: Math.PI / 2,
}

function resize(): void {
  if (!host.value || !renderer || !camera) return
  const width = Math.round(host.value.clientWidth)
  const height = Math.round(host.value.clientHeight)
  if (!width || !height) return
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
}
function setView(): void {
  if (!camera || !board) return
  if (mode.value === 'camera') {
    camera.position.set(0, 0, 0)
    camera.rotation.set(0, 0, 0)
    board.rotation.set(1, 0, 0)
    board.position.set(cameraPlacement.x, cameraPlacement.y, -3)
  } else {
    const horizontal = Math.cos(pitch) * distance
    camera.position.set(Math.sin(yaw) * horizontal, Math.sin(pitch) * distance, Math.cos(yaw) * horizontal)
    camera.lookAt(0, 0, 0)
    board.rotation.set(0, 0, 0)
    board.position.set(0, 0, 0)
  }
}
function makeRig(teamId?: number): Rig | null {
  if (!template) return null
  const model = template.clone(true)
  model.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) return
    const material = object.material
    if (!(material instanceof THREE.MeshStandardMaterial)) return
    const copy = material.clone()
    if (teamId === 2 && /^(Body|left_arm|right_arm)$/.test(object.name)) {
      copy.color.set('#9860d7')
      copy.emissive.set('#492471')
      copy.emissiveIntensity = 0.12
    }
    object.material = copy
  })
  const { pivots } = createArmPivots(model, THREE)
  const bounds = new THREE.Box3().setFromObject(model)
  const size = bounds.getSize(new THREE.Vector3())
  const center = bounds.getCenter(new THREE.Vector3())
  const scale = Math.min(0.72 / Math.max(size.x, size.z), 0.96 / size.y)
  model.scale.setScalar(scale)
  model.position.set(-center.x * scale, -bounds.min.y * scale, -center.z * scale)
  const group = new THREE.Group()
  group.position.y = 0.07
  group.add(model)
  return {
    group,
    model,
    animator: createRobotAnimator(group, model, pivots),
    cargo: null,
    cargoFromAssets: false,
  }
}
function syncCargo(rig: Rig, carrying: boolean): void {
  if (!carrying) {
    if (rig.cargo) disposeCarriedLuggage(rig.cargo)
    rig.cargo = null
    return
  }
  if (rig.cargo && rig.cargoFromAssets === Boolean(assets)) return
  if (rig.cargo) disposeCarriedLuggage(rig.cargo)
  rig.cargo = createCarriedLuggage(assets)
  rig.cargoFromAssets = Boolean(assets)
  rig.group.add(rig.cargo)
}
function luggagePoint(rig: Rig, target?: { x: number; y: number }): THREE.Vector3 {
  board?.updateMatrixWorld(true)
  const tile = board?.children.find(
    (child) => child.userData.cellX === target?.x && child.userData.cellY === target?.y,
  )
  const world = tile
    ? tile.localToWorld(new THREE.Vector3(0, 0.39, 0))
    : rig.group.localToWorld(new THREE.Vector3(0, 0.39, -0.9))
  return rig.group.worldToLocal(world)
}
function renderFrame(): void {
  if (!scene || !current.value) return
  cargoTransitions = []
  for (const item of effects) item.effect.dispose()
  effects = []
  if (board) {
    for (const rig of rigs.values()) rig.group.removeFromParent()
    scene.remove(board)
    disposeStageVisuals(board)
  }
  const stage = replayFrameToStage(current.value)
  const visuals = createStageVisuals(stage, assets)
  board = visuals.root
  animated = visuals.animated
  scene.add(board)
  const active = new Set(current.value.robots.map((robot) => robot.id))
  for (const [id, rig] of rigs) {
    rig.animator.stop()
    if (!active.has(id)) {
      disposeRig(rig)
      rigs.delete(id)
    }
  }
  for (const robot of current.value.robots) {
    let rig = rigs.get(robot.id)
    if (!rig) {
      rig = makeRig(robot.teamId) ?? undefined
      if (!rig) continue
      rigs.set(robot.id, rig)
    }
    board.add(rig.group)
    rig.group.position.set(
      robot.position.x - (stage.width - 1) / 2,
      0.07,
      robot.position.y - (stage.height - 1) / 2,
    )
    rig.group.rotation.y = directionYaw[robot.direction]
    rig.animator.setCarrying(Boolean(robot.carryingLuggage))
    syncCargo(rig, Boolean(robot.carryingLuggage))
  }
  setView()
}
function animate(time: number): void {
  if (!alive || !renderer || !scene || !camera) return
  raf = requestAnimationFrame(animate)
  advanceWater(time)
  for (const rig of rigs.values()) rig.animator.update(time)
  for (const entry of cargoTransitions) {
    entry.start ??= time
    const progress = Math.min(1, (time - entry.start) / 900)
    const eased = progress * progress * (3 - 2 * progress)
    entry.cargo.position.lerpVectors(entry.from, entry.to, eased)
  }
  for (const entry of effects) {
    entry.start ??= time
    entry.effect.update(Math.min(1, (time - entry.start) / 1000))
  }
  for (const target of animated) {
    target.object.rotation.y += target.kind === 'hover' ? 0.006 : 0.004
    if (target.kind === 'hover') target.object.position.y = target.baseY + Math.sin(time * 0.002) * 0.035
  }
  if (board && host.value && current.value?.robots.some((robot) => robot.speech)) {
    board.updateMatrixWorld(true)
    camera.updateMatrixWorld()
    const width = host.value.clientWidth
    const height = host.value.clientHeight
    bubbles.value =
      current.value?.robots.flatMap((robot) => {
        if (!robot.speech) return []
        const rig = rigs.get(robot.id)
        if (!rig) return []
        const point = new THREE.Vector3(0, 1.02, 0).applyMatrix4(rig.group.matrixWorld).project(camera!)
        return [
          {
            id: robot.id,
            text: robot.speech,
            x: (point.x * 0.5 + 0.5) * width,
            y: (-point.y * 0.5 + 0.5) * height,
          },
        ]
      }) ?? []
  } else if (bubbles.value.length) {
    bubbles.value = []
  }
  renderer.render(scene, camera)
}
function animateAction(action: ReplayAction): void {
  const rig = rigs.get(action.robotId)
  if (!rig || !board || !current.value) return
  if (
    action.kind === 'print_integer' ||
    action.kind === 'print_bool' ||
    action.kind === 'print_unit' ||
    action.kind === 'print_terrain' ||
    action.kind === 'print_rotation' ||
    action.kind === 'print_distance_target'
  )
    return
  const event: AnimationEvent = {
    id: ++eventId,
    type: action.kind,
    success: action.success,
    to: action.to,
    direction: action.rotation,
    ticks: action.ticks,
  }
  let destination: THREE.Vector3 | undefined
  let targetYaw: number | undefined
  if ((action.kind === 'forward' || action.kind === 'back') && action.success && action.to) {
    destination = new THREE.Vector3(
      action.to.x - (current.value.terrain[0].length - 1) / 2,
      0.07,
      action.to.y - (current.value.terrain.length - 1) / 2,
    )
  }
  if (action.kind === 'rotate' && action.rotation)
    targetYaw = rig.group.rotation.y + (action.rotation === 'left' ? Math.PI / 2 : -Math.PI / 2)
  if (action.kind === 'drill' && action.success && action.target) {
    board.traverse((object) => {
      if (
        object.name === 'stage-rock' &&
        object.userData.cellX === action.target?.x &&
        object.userData.cellY === action.target?.y
      )
        effects.push({ effect: createRockBreakEffect(object), start: null })
    })
  }
  if (action.kind === 'pick_up' && action.success && action.target) {
    syncCargo(rig, true)
    if (rig.cargo) {
      const tile = board.children.find(
        (child) => child.userData.cellX === action.target?.x && child.userData.cellY === action.target?.y,
      )
      const groundLuggage = tile?.getObjectByName('stage-luggage')
      if (groundLuggage) groundLuggage.visible = false
      const from = luggagePoint(rig, action.target)
      const to = new THREE.Vector3(0, CARRIED_LUGGAGE_HEIGHT, 0)
      rig.cargo.position.copy(from)
      cargoTransitions.push({ cargo: rig.cargo, from, to, start: null })
    }
  }
  if (action.kind === 'release' && action.success && rig.cargo) {
    const from = rig.cargo.position.clone()
    const to = action.target ? luggagePoint(rig, action.target) : new THREE.Vector3(0, -0.2, -0.6)
    cargoTransitions.push({ cargo: rig.cargo, from, to, start: null })
  }
  rig.animator.play(event, destination, targetYaw)
}
function stop(): void {
  playing.value = false
  transitioning.value = false
  if (timer) clearTimeout(timer)
  timer = null
  renderFrame()
}
function step(auto = false): void {
  if (index.value >= props.frames.length - 1) {
    playing.value = false
    return
  }
  if (timer) clearTimeout(timer)
  transitioning.value = true
  const next = props.frames[index.value + 1]
  for (const action of next.actions) animateAction(action)
  const wait = next.actions.find((action) => action.kind === 'wait')
  const duration = wait
    ? Math.min(3000, Math.max(1000, (wait.ticks ?? 1) * 1000))
    : next.actions.length
      ? 1050
      : 350
  timer = setTimeout(() => {
    timer = null
    transitioning.value = false
    index.value++
    renderFrame()
    if (auto && playing.value) step(true)
  }, duration)
}
function togglePlay(): void {
  if (playing.value) {
    stop()
    return
  }
  if (timer) stop()
  if (index.value >= props.frames.length - 1) index.value = 0
  renderFrame()
  playing.value = true
  step(true)
}
function seek(event: Event): void {
  stop()
  index.value = Number((event.target as HTMLInputElement).value)
  renderFrame()
}
function previous(): void {
  if (index.value === 0) return
  stop()
  index.value--
  renderFrame()
}
async function setMode(next: 'preview' | 'camera'): Promise<void> {
  mode.value = next
  stream?.getTracks().forEach((track) => track.stop())
  stream = null
  if (video.value) video.value.srcObject = null
  if (next === 'camera') {
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: 'environment' } },
        audio: false,
      })
      if (video.value) {
        video.value.srcObject = stream
        await video.value.play()
      }
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : 'カメラを開けませんでした'
      mode.value = 'preview'
    }
  }
  setView()
}
function pointerDown(event: PointerEvent): void {
  if (mode.value === 'camera') {
    if (!host.value || !board || !camera) return
    const rect = host.value.getBoundingClientRect()
    const halfHeight = Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * 3
    cameraPlacement = {
      x: (((event.clientX - rect.left) / rect.width) * 2 - 1) * halfHeight * camera.aspect,
      y: (1 - ((event.clientY - rect.top) / rect.height) * 2) * halfHeight,
    }
    board.position.set(cameraPlacement.x, cameraPlacement.y, -3)
    return
  }
  dragging = true
  lastPointer = [event.clientX, event.clientY]
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function pointerMove(event: PointerEvent): void {
  if (!dragging || mode.value !== 'preview') return
  yaw -= (event.clientX - lastPointer[0]) * 0.006
  pitch = THREE.MathUtils.clamp(pitch + (event.clientY - lastPointer[1]) * 0.006, 0.2, 1.35)
  lastPointer = [event.clientX, event.clientY]
  setView()
}
function wheel(event: WheelEvent): void {
  if (mode.value !== 'preview') return
  distance = THREE.MathUtils.clamp(distance + Math.sign(event.deltaY) * 0.35, 2.4, 9)
  setView()
}
onMounted(() => {
  if (!host.value) return
  alive = true
  try {
    renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    renderer.setClearColor(0x000000, 0)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    renderer.shadowMap.enabled = true
    host.value.append(renderer.domElement)
    scene = new THREE.Scene()
    camera = new THREE.PerspectiveCamera(48, 1, 0.05, 100)
    scene.add(new THREE.HemisphereLight(0xe5ffed, 0x304756, 2.2))
    const sun = new THREE.DirectionalLight(0xffeed1, 2.5)
    sun.position.set(-3, 7, 5)
    scene.add(sun)
    renderFrame()
    observer = new ResizeObserver(resize)
    observer.observe(host.value)
    resize()
    raf = requestAnimationFrame(animate)
    void loadStageAssets().then(
      (loaded) => {
        if (alive) {
          assets = loaded
          renderFrame()
        }
      },
      () => {
        error.value = 'ステージ素材を読み込めませんでした。簡易表示で再生します。'
      },
    )
    new GLTFLoader().load(
      '/robot.glb',
      (gltf) => {
        if (alive) {
          template = gltf.scene
          renderFrame()
        }
      },
      undefined,
      () => {
        error.value = 'ロボットモデルを読み込めませんでした'
      },
    )
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '3D再生を開始できませんでした'
  }
})
onBeforeUnmount(() => {
  alive = false
  if (timer) clearTimeout(timer)
  cancelAnimationFrame(raf)
  observer?.disconnect()
  stream?.getTracks().forEach((track) => track.stop())
  for (const rig of rigs.values()) disposeRig(rig)
  for (const entry of effects) entry.effect.dispose()
  if (board) disposeStageVisuals(board)
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <section class="match-replay" aria-labelledby="match-replay-title">
    <div class="match-replay-heading">
      <div>
        <p class="match-eyebrow">REPLAY</p>
        <h2 id="match-replay-title">対戦を再生</h2>
      </div>
      <span>ターン {{ current?.tick ?? 0 }} · {{ index + 1 }} / {{ frames.length }}</span>
    </div>
    <div class="match-replay-viewport">
      <div
        ref="host"
        class="match-replay-scene"
        @pointerdown="pointerDown"
        @pointermove="pointerMove"
        @pointerup="dragging = false"
        @pointercancel="dragging = false"
        @wheel.prevent="wheel"
      >
        <video v-show="mode === 'camera'" ref="video" autoplay muted playsinline />
        <div
          v-for="bubble in bubbles"
          :key="bubble.id"
          class="match-replay-bubble"
          :style="{ left: `${bubble.x}px`, top: `${bubble.y}px` }"
        >
          {{ bubble.text }}
        </div>
      </div>
      <div class="match-replay-mode">
        <button type="button" :aria-pressed="mode === 'preview'" @click="setMode('preview')">3D</button>
        <button type="button" :aria-pressed="mode === 'camera'" @click="setMode('camera')">カメラ</button>
      </div>
    </div>
    <p v-if="error" class="match-error" role="alert">{{ error }}</p>
    <p class="match-replay-action" aria-live="polite">{{ current ? heading(current) : '' }}</p>
    <div v-if="printActions.length" class="match-replay-speech">
      <span v-for="(action, actionIndex) in printActions" :key="actionIndex">
        {{ action.robotId }}:
        {{ action.message }}
      </span>
    </div>
    <input
      type="range"
      min="0"
      :max="Math.max(0, frames.length - 1)"
      :value="index"
      aria-label="再生位置"
      @input="seek"
    />
    <div class="match-replay-controls">
      <button type="button" :disabled="index === 0" @click="previous">前へ</button>
      <button type="button" @click="togglePlay">{{ playing ? '一時停止' : '再生' }}</button>
      <button
        type="button"
        :disabled="index >= frames.length - 1 || playing || transitioning"
        @click="step()"
      >
        次へ
      </button>
    </div>
  </section>
</template>
