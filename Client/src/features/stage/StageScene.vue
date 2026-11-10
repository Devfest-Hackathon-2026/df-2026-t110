<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { advanceWater, createStageVisuals, disposeStageVisuals } from './stageVisuals.ts'
import { loadStageAssets } from './stageAssets.ts'
import type { StageAssets } from './stageAssets.ts'
import type { StageAnimationTarget } from './stageVisuals.ts'
import { createArmPivots } from './armController.ts'
import { CARRIED_LUGGAGE_HEIGHT, createCarriedLuggage, disposeCarriedLuggage } from './carriedLuggage.ts'
import { createRobotAnimator } from './robotAnimator.ts'
import { createRockBreakEffect } from './rockBreakEffect.ts'
import RoomMarkerOverlay from '../rooms/RoomMarkerOverlay.vue'
import { useRoomMarkerPlacement } from '../rooms/useRoomMarkerPlacement.ts'
import type { StageCell, StageLayout } from './stage.ts'
import type { AnimationEvent } from './robotAnimator.ts'
import type { PicolangSpeech } from '../picolang/editor/playback/picolangSpeech.ts'
import TerrainSpeechModel from '../tutorial/TerrainSpeechModel.vue'

const props = withDefaults(
  defineProps<{
    stage: StageLayout
    mode: 'preview' | 'camera'
    animationEvent?: AnimationEvent | null
    speech?: PicolangSpeech | null
    previewDistance?: number
  }>(),
  { animationEvent: null, speech: null, previewDistance: 4.8 },
)
const emit = defineEmits<{ error: [message: string] }>()
const host = ref<HTMLElement | null>(null)
let renderer!: THREE.WebGLRenderer
let scene!: THREE.Scene
let camera!: THREE.PerspectiveCamera
let board!: THREE.Group
let animated: StageAnimationTarget[] = []
let stageAssets: StageAssets | null = null
let mounted = false
let frame = 0
let resizeObserver: ResizeObserver | null = null
let canvasWidth = 0
let canvasHeight = 0
const {
  markerState,
  markerActive,
  markerLoading,
  cameraReady,
  placementConfirmed,
  trackingPoseAvailable,
  calibrated,
  confirmPlacement,
  retainBoardPose: retainCameraBoardPose,
  resize: resizeTracking,
  start: startCamera,
  stop: stopCamera,
  update: updateMarkerPlacement,
} = useRoomMarkerPlacement({
  host,
  getBoard: () => board,
  getCamera: () => camera,
  isCamera: () => props.mode === 'camera',
  onError: (message) => emit('error', message),
})
let robotRig: THREE.Group | null = null
let robotModel: THREE.Object3D | null = null
let robotAnimator: ReturnType<typeof createRobotAnimator> | null = null
let carriedLuggage: THREE.Group | null = null
let cargoFromAssets = false
let cargoTransition: { from: THREE.Vector3; to: THREE.Vector3; start: number | null } | null = null
let rockBreak: ReturnType<typeof createRockBreakEffect> | null = null
let rockBreakStart: number | null = null
let pendingEvent: AnimationEvent | null = null
const robotCell = ref<{ x: number; y: number } | null>(null)
const speechPosition = ref<{ x: number; y: number } | null>(null)
const speechBubble = ref<HTMLElement | null>(null)
defineExpose({ robotCell, reset: replaceStage, resetRobot, refreshVisuals, setCarrying })
let yaw = 0.55
let pitch = 0.65
let distance = props.previewDistance
let dragging = false
let lastPointer = [0, 0]

function setView() {
  if (!camera || !board) return
  if (props.mode === 'camera') {
    camera.position.set(0, 0, 0)
    camera.rotation.set(0, 0, 0)
    board.visible = false
  } else {
    const horizontal = Math.cos(pitch) * distance
    camera.position.set(Math.sin(yaw) * horizontal, Math.sin(pitch) * distance, Math.cos(yaw) * horizontal)
    camera.lookAt(0, 0, 0)
    board.position.set(0, 0, 0)
    board.rotation.set(0, 0, 0)
    board.scale.set(1, 1, 1)
    board.visible = true
  }
}

function replaceStage() {
  if (!scene) return
  const previousBoard = board
  distance = props.previewDistance
  pendingEvent = null
  setCarrying(false)
  clearRockBreak()
  if (board) {
    robotAnimator?.stop()
    if (robotRig) board.remove(robotRig)
    scene.remove(board)
    disposeStageVisuals(board)
  }
  const visuals = createStageVisuals(props.stage, stageAssets ?? undefined)
  board = visuals.root
  animated = visuals.animated
  scene.add(board)
  const start = props.stage.cells.find((cell) => cell.start)
  robotCell.value = start ? { x: start.x, y: start.y } : null
  if (robotRig) {
    board.add(robotRig)
    positionRobot(start)
  }
  board.position.z = 0
  if (!retainCameraBoardPose(previousBoard)) setView()
}

function refreshVisuals(): void {
  if (!scene || !board) return
  const previousBoard = board
  const currentYaw = robotRig?.rotation.y
  clearRockBreak()
  robotAnimator?.stop()
  if (robotRig) board.remove(robotRig)
  scene.remove(board)
  disposeStageVisuals(board)
  const visuals = createStageVisuals(props.stage, stageAssets ?? undefined)
  board = visuals.root
  animated = visuals.animated
  scene.add(board)
  if (robotRig) {
    board.add(robotRig)
    positionRobot(robotCell.value)
    if (currentYaw !== undefined) robotRig.rotation.y = currentYaw
  }
  if (!retainCameraBoardPose(previousBoard)) setView()
}

function setCarrying(value: boolean): void {
  cargoTransition = null
  robotAnimator?.setCarrying(value)
  if (!value || !robotRig) {
    if (carriedLuggage) disposeCarriedLuggage(carriedLuggage)
    carriedLuggage = null
    return
  }
  if (carriedLuggage && cargoFromAssets === Boolean(stageAssets)) return
  if (carriedLuggage) disposeCarriedLuggage(carriedLuggage)
  carriedLuggage = createCarriedLuggage(stageAssets ?? undefined)
  cargoFromAssets = Boolean(stageAssets)
  robotRig.add(carriedLuggage)
}

function cargoPoint(target: { x: number; y: number }): THREE.Vector3 {
  board.updateMatrixWorld(true)
  const tile = board.children.find(
    (child) => child.userData.cellX === target.x && child.userData.cellY === target.y,
  )
  const world = tile
    ? tile.localToWorld(new THREE.Vector3(0, 0.39, 0))
    : robotRig!.localToWorld(new THREE.Vector3(0, 0.39, -0.9))
  return robotRig!.worldToLocal(world)
}

function positionRobot(cell: StageCell | { x: number; y: number } | null | undefined): void {
  if (!robotRig || !cell) return
  robotRig.position.set(cell.x - (props.stage.width - 1) / 2, 0.07, cell.y - (props.stage.height - 1) / 2)
  robotRig.rotation.y = {
    north: 0,
    east: -Math.PI / 2,
    south: Math.PI,
    west: Math.PI / 2,
  }[props.stage.robot.direction]
}

function resetRobot(): void {
  pendingEvent = null
  clearRockBreak()
  robotAnimator?.stop()
  setCarrying(false)
  const start = props.stage.cells.find((cell) => cell.start)
  robotCell.value = start ? { x: start.x, y: start.y } : null
  positionRobot(start)
}

function loadRobot() {
  new GLTFLoader().load(
    '/robot.glb',
    (gltf) => {
      if (!scene) return
      robotModel = gltf.scene
      robotModel.name = 'electronic-fairy-robot'
      const { pivots, found } = createArmPivots(robotModel, THREE)
      const bounds = new THREE.Box3().setFromObject(robotModel)
      const size = bounds.getSize(new THREE.Vector3())
      const center = bounds.getCenter(new THREE.Vector3())
      const scale = Math.min(0.72 / Math.max(size.x, size.z), 0.96 / size.y)
      robotModel.scale.setScalar(scale)
      robotModel.position.set(-center.x * scale, -bounds.min.y * scale, -center.z * scale)
      robotRig = new THREE.Group()
      robotRig.name = 'robot-rig'
      robotRig.add(robotModel)
      board.add(robotRig)
      positionRobot(robotCell.value)
      robotAnimator = createRobotAnimator(robotRig, robotModel, pivots)
      if (!found.every(Boolean)) emit('error', 'ロボットの動きを表示できませんでした。')
      if (pendingEvent) {
        playAnimationEvent(pendingEvent)
        pendingEvent = null
      }
    },
    undefined,
    (error) => {
      console.error('ロボットモデルの読み込みに失敗しました', error)
      emit('error', 'ロボットを表示できませんでした。ページを再読み込みしてください。')
    },
  )
}

function gridDirection(yaw: number): { x: number; y: number } {
  return { x: Math.round(-Math.sin(yaw)), y: Math.round(-Math.cos(yaw)) }
}

function facingForStep(dx: number, dy: number): number | null {
  if (dx === 1 && dy === 0) return -Math.PI / 2
  if (dx === -1 && dy === 0) return Math.PI / 2
  if (dx === 0 && dy === 1) return Math.PI
  if (dx === 0 && dy === -1) return 0
  return null
}

function clearRockBreak(): void {
  rockBreak?.dispose()
  rockBreak = null
  rockBreakStart = null
}

function rockAhead(): THREE.Object3D | null {
  if (!board || !robotRig || !robotCell.value) return null
  const direction = gridDirection(robotRig.rotation.y)
  const x = robotCell.value.x + direction.x
  const y = robotCell.value.y + direction.y
  let found: THREE.Object3D | null = null
  board.traverse((object) => {
    if (object.name === 'stage-rock' && object.userData.cellX === x && object.userData.cellY === y)
      found = object
  })
  return found
}

function playAnimationEvent(event: AnimationEvent | null): void {
  if (!event) return
  if (!robotAnimator) {
    pendingEvent = event
    return
  }
  if (!robotRig) return
  clearRockBreak()
  let destination: THREE.Vector3 | undefined
  let targetYaw: number | undefined
  let animation = event
  if (event.type === 'move' || event.type === 'forward' || event.type === 'back') {
    const previous = robotCell.value
    if (!previous) return
    const facing = gridDirection(robotRig.rotation.y)
    const sign = event.type === 'back' ? -1 : 1
    const x = event.to?.x ?? previous.x + facing.x * sign
    const y = event.to?.y ?? previous.y + facing.y * sign
    if (!Number.isInteger(x) || !Number.isInteger(y)) return
    const cell = props.stage.cells.find((item) => item.x === x && item.y === y)
    const canMove = Boolean(cell && cell.terrain === 'land' && event.success !== false)
    const next = canMove ? { x, y } : previous
    destination = new THREE.Vector3(
      next.x - (props.stage.width - 1) / 2,
      0.07,
      next.y - (props.stage.height - 1) / 2,
    )
    if (canMove) robotCell.value = next
    if (event.type === 'move') targetYaw = facingForStep(x - previous.x, y - previous.y) ?? undefined
    animation = { ...event, success: canMove }
  } else if (event.type === 'rotate') {
    if (event.direction !== 'left' && event.direction !== 'right') return
    targetYaw = robotRig.rotation.y + (event.direction === 'left' ? Math.PI / 2 : -Math.PI / 2)
  } else if (event.type === 'drill' && event.success !== false) {
    const rock = rockAhead()
    if (rock) rockBreak = createRockBreakEffect(rock)
  }
  if ((event.type === 'get' || event.type === 'pick_up') && event.success !== false && robotCell.value) {
    const facing = gridDirection(robotRig.rotation.y)
    const target = { x: robotCell.value.x + facing.x, y: robotCell.value.y + facing.y }
    const tile = board.children.find(
      (child) => child.userData.cellX === target.x && child.userData.cellY === target.y,
    )
    const luggage = tile?.getObjectByName('stage-luggage')
    if (luggage) luggage.visible = false
    setCarrying(true)
    robotAnimator.setCarrying(false)
    if (carriedLuggage) {
      const from = cargoPoint(target)
      const to = new THREE.Vector3(0, CARRIED_LUGGAGE_HEIGHT, 0)
      carriedLuggage.position.copy(from)
      cargoTransition = { from, to, start: null }
    }
  }
  if ((event.type === 'put' || event.type === 'release') && event.success !== false && carriedLuggage) {
    const facing = gridDirection(robotRig.rotation.y)
    const target = robotCell.value
      ? { x: robotCell.value.x + facing.x, y: robotCell.value.y + facing.y }
      : null
    cargoTransition = {
      from: carriedLuggage.position.clone(),
      to: target ? cargoPoint(target) : new THREE.Vector3(0, -0.2, -0.6),
      start: null,
    }
  }
  robotAnimator.play(animation, destination, targetYaw)
}

function resizeImmediately() {
  if (!renderer || !host.value) return
  const bounds = host.value.getBoundingClientRect()
  const width = Math.round(bounds.width)
  const height = Math.round(bounds.height)
  if (!width || !height) return
  if (width === canvasWidth && height === canvasHeight) return
  canvasWidth = width
  canvasHeight = height
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
  resizeTracking()
}

function animate(time: number): void {
  frame = requestAnimationFrame(animate)
  updateMarkerPlacement(time)
  advanceWater(time)
  robotAnimator?.update(time)
  if (cargoTransition && carriedLuggage) {
    cargoTransition.start ??= time
    const progress = Math.min(1, (time - cargoTransition.start) / 900)
    const eased = progress * progress * (3 - 2 * progress)
    carriedLuggage.position.lerpVectors(cargoTransition.from, cargoTransition.to, eased)
  }
  if (rockBreak) {
    rockBreakStart ??= time
    rockBreak.update(Math.min(1, (time - rockBreakStart) / 1000))
  }
  for (const target of animated) {
    target.object.rotation.y += target.kind === 'hover' ? 0.006 : 0.004
    if (target.kind === 'hover') target.object.position.y = target.baseY + Math.sin(time * 0.002) * 0.035
  }
  if (props.speech && robotRig && board?.visible && host.value) {
    board.updateMatrixWorld(true)
    camera.updateMatrixWorld()
    const point = new THREE.Vector3(0, 1.1, 0).applyMatrix4(robotRig.matrixWorld).project(camera)
    const horizontalInset = Math.min(
      host.value.clientWidth / 2,
      (speechBubble.value?.offsetWidth ?? 144) / 2 + 12,
    )
    const verticalInset = Math.min(
      host.value.clientHeight - 12,
      (speechBubble.value?.offsetHeight ?? 100) + 12,
    )
    speechPosition.value =
      point.z >= -1 && point.z <= 1
        ? {
            x: Math.max(
              horizontalInset,
              Math.min(
                host.value.clientWidth - horizontalInset,
                (point.x * 0.5 + 0.5) * host.value.clientWidth,
              ),
            ),
            y: Math.max(
              verticalInset,
              Math.min(host.value.clientHeight - 12, (-point.y * 0.5 + 0.5) * host.value.clientHeight),
            ),
          }
        : null
  } else if (speechPosition.value) speechPosition.value = null
  renderer.render(scene, camera)
}

function pointerDown(event: PointerEvent): void {
  if (props.mode === 'camera') return
  dragging = true
  lastPointer = [event.clientX, event.clientY]
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function pointerMove(event: PointerEvent): void {
  if (!dragging || props.mode !== 'preview') return
  yaw -= (event.clientX - lastPointer[0]) * 0.006
  pitch = THREE.MathUtils.clamp(pitch + (event.clientY - lastPointer[1]) * 0.006, 0.2, 1.35)
  lastPointer = [event.clientX, event.clientY]
  setView()
}

function wheel(event: WheelEvent): void {
  if (props.mode !== 'preview') return
  distance = THREE.MathUtils.clamp(distance + Math.sign(event.deltaY) * 0.35, 2.4, 9)
  setView()
}

onMounted(() => {
  if (!host.value) return
  mounted = true
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
    sun.castShadow = true
    sun.shadow.mapSize.set(1024, 1024)
    scene.add(sun)
    replaceStage()
    loadStageAssets().then(
      (loaded) => {
        if (!mounted) return
        stageAssets = loaded
        refreshVisuals()
      },
      (error) => {
        console.error('フィールド素材の読み込みに失敗しました', error)
        emit('error', 'フィールドを表示できませんでした。ページを再読み込みしてください。')
      },
    )
    loadRobot()
    resizeObserver = new ResizeObserver(resizeImmediately)
    resizeObserver.observe(host.value)
    resizeImmediately()
    frame = requestAnimationFrame(animate)
    startCamera()
  } catch (error) {
    console.error('3Dフィールドの起動に失敗しました', error)
    emit('error', 'フィールドを表示できませんでした。ページを再読み込みしてください。')
  }
})

watch(() => props.stage, replaceStage)
watch(() => props.animationEvent, playAnimationEvent)
watch(
  () => props.mode,
  () => {
    setView()
    startCamera()
  },
)
onBeforeUnmount(() => {
  mounted = false
  cancelAnimationFrame(frame)
  resizeObserver?.disconnect()
  stopCamera()
  robotAnimator?.stop()
  setCarrying(false)
  clearRockBreak()
  if (robotRig && board) board.remove(robotRig)
  robotModel?.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) return
    object.geometry?.dispose()
    if (Array.isArray(object.material)) object.material.forEach((material) => material.dispose())
    else object.material?.dispose()
  })
  if (board) disposeStageVisuals(board)
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <div
    ref="host"
    class="stage-scene"
    @pointerdown="pointerDown"
    @pointermove="pointerMove"
    @pointerup="dragging = false"
    @pointercancel="dragging = false"
    @wheel.prevent="wheel"
  >
    <div
      v-if="speech && speechPosition"
      ref="speechBubble"
      :key="speech.id"
      class="stage-robot-speech"
      :style="{ left: `${speechPosition.x}px`, top: `${speechPosition.y}px` }"
      :aria-label="speech.label"
      @pointerdown.stop
      @pointermove.stop
      @wheel.stop
    >
      <TerrainSpeechModel v-if="speech.terrain" :terrain="speech.terrain" />
      <div class="stage-robot-speech-items">
        <span v-for="(item, index) in speech.items" :key="index">{{ item }}</span>
        <span v-if="speech.items.length === 0">空</span>
      </div>
    </div>
    <RoomMarkerOverlay
      v-if="mode === 'camera'"
      :marker-state="markerState"
      :marker-active="markerActive"
      :marker-loading="markerLoading"
      :camera-ready="cameraReady"
      :placement-confirmed="placementConfirmed"
      :tracking-pose-available="trackingPoseAvailable"
      :calibrated="calibrated"
      @confirm="confirmPlacement"
    />
  </div>
</template>

<style>
.stage-robot-speech {
  position: absolute;
  z-index: 5;
  display: grid;
  justify-items: center;
  min-width: 120px;
  max-width: min(85%, 480px);
  padding: 12px 16px;
  transform: translate(-50%, -100%);
  border: 2px solid #356b61;
  border-radius: 14px;
  background: #fffdf2;
  box-shadow: 0 5px 18px rgb(0 0 0 / 24%);
  color: #143e39;
  pointer-events: auto;
}
.stage-robot-speech::after {
  position: absolute;
  bottom: -9px;
  left: calc(50% - 7px);
  width: 12px;
  height: 12px;
  transform: rotate(45deg);
  border-right: 2px solid #356b61;
  border-bottom: 2px solid #356b61;
  background: #fffdf2;
  content: '';
}
.stage-robot-speech-items {
  display: flex;
  max-width: 100%;
  gap: 6px;
  overflow-x: auto;
  font-size: 20px;
  font-weight: 700;
  white-space: nowrap;
  touch-action: pan-x;
}
.stage-robot-speech-items span {
  flex: none;
}
.stage-robot-speech-items span + span::before {
  margin-right: 6px;
  color: #6b8d83;
  content: '·';
}
.terrain-speech-model {
  width: 150px;
  height: 120px;
}
.terrain-speech-model canvas {
  display: block;
  width: 150px;
  height: 120px;
}
</style>
