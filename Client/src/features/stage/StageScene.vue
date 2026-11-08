<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
import { MarkerPoseStabilizer } from './markerPoseStabilizer.ts'
import {
  mountRoomMarkerTracker,
  type RoomMarkerId,
  type RoomMarkerState,
  type RoomMarkerTracker,
} from '../rooms/roomMarkerTracking.ts'
import type { StageCell, StageLayout } from './stage.ts'
import type { AnimationEvent } from './robotAnimator.ts'

const props = withDefaults(
  defineProps<{
    stage: StageLayout
    mode: 'preview' | 'camera'
    animationEvent?: AnimationEvent | null
    previewDistance?: number
  }>(),
  { animationEvent: null, previewDistance: 4.8 },
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
let markerTracker: RoomMarkerTracker | null = null
let markerGeneration = 0
const markerState = ref<RoomMarkerState>({ a: false, b: false })
const markerSeen = computed(() => markerState.value.a || markerState.value.b)
const markerCount = computed(() => Number(markerState.value.a) + Number(markerState.value.b))
const markerActive = ref(false)
const markerLoading = ref(false)
const cameraReady = ref(false)
const placementConfirmed = ref(false)
const trackingPoseAvailable = ref(false)
const calibrated = ref(false)
let anchorMarker: RoomMarkerId | null = null
let alternateToAnchor: THREE.Matrix4 | null = null
const markerPose = new MarkerPoseStabilizer()
let lastMarkerPoseTime: number | null = null
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
    if (markerTracker || markerLoading.value) {
      board.visible = false
      return
    }
    board.rotation.set(1, 0, 0)
    if (board.position.z !== -3) board.position.set(0, -0.05, -3)
  } else {
    const horizontal = Math.cos(pitch) * distance
    camera.position.set(Math.sin(yaw) * horizontal, Math.sin(pitch) * distance, Math.cos(yaw) * horizontal)
    camera.lookAt(0, 0, 0)
    board.position.set(0, 0, 0)
    board.rotation.set(0, 0, 0)
    board.scale.set(1, 1, 1)
  }
}

function retainCameraBoardPose(previous: THREE.Group | undefined): boolean {
  if (!previous || props.mode !== 'camera' || !markerTracker || !placementConfirmed.value) return false
  board.position.copy(previous.position)
  board.quaternion.copy(previous.quaternion)
  board.scale.copy(previous.scale)
  board.visible = previous.visible
  return true
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
      if (!found.every(Boolean)) emit('error', 'GLBの left_arm / right_arm を検出できませんでした')
      if (pendingEvent) {
        playAnimationEvent(pendingEvent)
        pendingEvent = null
      }
    },
    undefined,
    (error) =>
      emit(
        'error',
        `ロボットGLBを読み込めませんでした: ${error instanceof Error ? error.message : String(error)}`,
      ),
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
  const trackingScene = markerTracker?.scene as
    | (HTMLElement & {
        systems?: { arjs?: { _arSession?: { onResize(): void } } }
      })
    | undefined
  trackingScene?.systems?.arjs?._arSession?.onResize()
}

function detectedMarkerPose(id: RoomMarkerId): THREE.Matrix4 | null {
  if (!markerTracker || !markerState.value[id]) return null
  const marker = markerTracker.markers[id] as HTMLElement & {
    object3D?: { matrixWorld: { elements: number[] }; visible: boolean }
  }
  return marker.object3D?.visible ? new THREE.Matrix4().fromArray(marker.object3D.matrixWorld.elements) : null
}

function confirmPlacement(): void {
  if (!markerActive.value || !markerSeen.value) return
  anchorMarker = markerState.value.a ? 'a' : 'b'
  alternateToAnchor = null
  calibrated.value = false
  markerPose.reset()
  lastMarkerPoseTime = null
  placementConfirmed.value = true
}

function updateMarkerPlacement(time: number): void {
  if (!board || !anchorMarker || !placementConfirmed.value) {
    if (board) board.visible = false
    trackingPoseAvailable.value = false
    lastMarkerPoseTime = null
    return
  }
  const alternateId = anchorMarker === 'a' ? 'b' : 'a'
  const anchorPose = detectedMarkerPose(anchorMarker)
  const alternatePose = detectedMarkerPose(alternateId)
  if (anchorPose && alternatePose) {
    alternateToAnchor = alternatePose.clone().invert().multiply(anchorPose)
    calibrated.value = true
  }
  const pose =
    anchorPose ?? (alternatePose && alternateToAnchor ? alternatePose.multiply(alternateToAnchor) : null)
  if (!pose) {
    if (lastMarkerPoseTime !== null && time - lastMarkerPoseTime < 300) return
    trackingPoseAvailable.value = false
    board.visible = false
    markerPose.reset()
    lastMarkerPoseTime = null
    return
  }
  lastMarkerPoseTime = time
  trackingPoseAvailable.value = true
  board.visible = true
  board.matrix.copy(pose)
  board.matrix.decompose(board.position, board.quaternion, board.scale)
  markerPose.update(pose, time)
  board.position.copy(markerPose.position)
  board.quaternion.copy(markerPose.quaternion)
  board.scale.multiplyScalar(1)
}

function animate(time: number): void {
  frame = requestAnimationFrame(animate)
  if (markerTracker && board) {
    const trackingCamera = markerTracker.scene as HTMLElement & {
      camera?: { projectionMatrix: { elements: number[] } }
    }
    if (trackingCamera.camera) {
      camera.projectionMatrix.fromArray(trackingCamera.camera.projectionMatrix.elements)
      const cameraVideo = host.value?.querySelector<HTMLVideoElement>('.room-marker-video')
      if (cameraVideo?.videoWidth && cameraVideo.videoHeight && canvasWidth && canvasHeight) {
        const cover = Math.max(canvasWidth / cameraVideo.videoWidth, canvasHeight / cameraVideo.videoHeight)
        const scaleX = (cameraVideo.videoWidth * cover) / canvasWidth
        const scaleY = (cameraVideo.videoHeight * cover) / canvasHeight
        for (const index of [0, 4, 8, 12]) camera.projectionMatrix.elements[index] *= scaleX
        for (const index of [1, 5, 9, 13]) camera.projectionMatrix.elements[index] *= scaleY
      }
      camera.projectionMatrixInverse.copy(camera.projectionMatrix).invert()
    }
    updateMarkerPlacement(time)
  }
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
  renderer.render(scene, camera)
}

async function startCamera() {
  stopMarkerTracking()
  if (props.mode !== 'camera') return
  markerLoading.value = true
  cameraReady.value = false
  board.visible = false
  const generation = markerGeneration
  try {
    const tracker = await mountRoomMarkerTracker(host.value!, {
      onChange: (state) => {
        markerState.value = state
      },
      onCameraReady: () => {
        if (generation === markerGeneration) cameraReady.value = true
      },
      onCameraError: (error) => emit('error', `マーカー追跡のカメラを開けませんでした: ${String(error)}`),
    })
    if (generation !== markerGeneration || props.mode !== 'camera') {
      tracker.stop()
      return
    }
    markerTracker = tracker
    markerLoading.value = false
    markerActive.value = true
    tracker.scene.style.position = 'absolute'
    tracker.scene.style.inset = '0'
    tracker.scene.style.width = '100%'
    tracker.scene.style.height = '100%'
    tracker.scene.style.opacity = '0'
    tracker.scene.style.pointerEvents = 'none'
    tracker.scene.addEventListener(
      'renderstart',
      () => {
        const sceneWithAR = tracker.scene as HTMLElement & {
          systems?: { arjs?: { _arSession?: { onResize(): void } } }
        }
        sceneWithAR.systems?.arjs?._arSession?.onResize()
      },
      { once: true },
    )
    return
  } catch (error) {
    if (generation !== markerGeneration || props.mode !== 'camera') return
    markerLoading.value = false
    emit(
      'error',
      `マーカー追跡を開始できませんでした: ${error instanceof Error ? error.message : String(error)}`,
    )
  }
}

function stopCamera() {
  stopMarkerTracking()
}

function stopMarkerTracking(): void {
  markerGeneration++
  markerTracker?.stop()
  markerTracker = null
  markerState.value = { a: false, b: false }
  markerActive.value = false
  markerLoading.value = false
  cameraReady.value = false
  placementConfirmed.value = false
  trackingPoseAvailable.value = false
  calibrated.value = false
  anchorMarker = null
  alternateToAnchor = null
  markerPose.reset()
  lastMarkerPoseTime = null
  if (board) board.visible = true
  if (camera) camera.updateProjectionMatrix()
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
      (error) =>
        emit(
          'error',
          `ステージのGLBを読み込めませんでした: ${error instanceof Error ? error.message : String(error)}`,
        ),
    )
    loadRobot()
    resizeObserver = new ResizeObserver(resizeImmediately)
    resizeObserver.observe(host.value)
    resizeImmediately()
    frame = requestAnimationFrame(animate)
    startCamera()
  } catch (error) {
    emit(
      'error',
      `3Dステージを開始できませんでした: ${error instanceof Error ? error.message : String(error)}`,
    )
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
    <div v-if="mode === 'camera' && (markerLoading || markerActive)" class="marker-overlay">
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
        <button v-else-if="!placementConfirmed" type="button" @click.stop="confirmPlacement">
          配置を確定
        </button>
        <p v-else-if="!trackingPoseAvailable" role="status">
          基準マーカーを映し、2枚を一度同時に映してください。
        </p>
        <p v-else role="status">もう1枚のマーカーも一度同時に映すと、見失ったときに切り替えられます。</p>
      </div>
    </div>
  </div>
</template>

<style>
/* AR.js keeps a resize listener after the scene is removed and can resize body again. */
body {
  width: auto !important;
  height: auto !important;
  margin: 0 !important;
}
.stage-scene .room-marker-video {
  position: absolute !important;
  inset: 0 !important;
  margin: 0 !important;
  width: 100% !important;
  height: 100% !important;
  object-fit: cover;
  z-index: 0 !important;
  pointer-events: none;
}
.stage-scene .marker-overlay {
  position: absolute;
  z-index: 4;
  top: max(8px, env(safe-area-inset-top));
  right: max(8px, env(safe-area-inset-right));
  display: flex;
  width: min(280px, calc(100% - 16px));
  max-height: calc(100% - max(8px, env(safe-area-inset-top)) - 8px);
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
.stage-scene .marker-status {
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
.stage-scene .marker-status strong,
.stage-scene .marker-status span {
  padding: 3px 6px;
}
.stage-scene .marker-status strong {
  grid-column: 1 / -1;
}
.stage-scene .marker-status span {
  border-radius: 4px;
  background: #405047;
}
.stage-scene .marker-status span.detected {
  background: #b5f7d0;
  color: #172c24;
}
.stage-scene .marker-guidance {
  flex: none;
  margin: 0;
  padding: 10px 14px;
  border-radius: 8px;
  background: #172c24;
  color: white;
}
.stage-scene .marker-guidance a {
  color: #b5f7d0;
}
.stage-scene .marker-guidance p {
  margin: 0;
}
.stage-scene .marker-guidance button {
  border: 0;
  border-radius: 6px;
  padding: 8px 12px;
  background: #b5f7d0;
  color: #172c24;
  font: inherit;
  cursor: pointer;
}
@media (max-width: 600px) {
  .stage-scene .marker-overlay {
    width: min(280px, calc(100% - 104px));
  }
}
</style>
