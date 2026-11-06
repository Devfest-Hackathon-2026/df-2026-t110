<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { advanceWater, createStageVisuals, disposeStageVisuals } from './stageVisuals.ts'
import { loadStageAssets } from './stageAssets.ts'
import type { StageAssets } from './stageAssets.ts'
import type { StageAnimationTarget } from './stageVisuals.ts'
import { createArmPivots } from './armController.ts'
import { createRobotAnimator } from './robotAnimator.ts'
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
const video = ref<HTMLVideoElement | null>(null)
let renderer!: THREE.WebGLRenderer
let scene!: THREE.Scene
let camera!: THREE.PerspectiveCamera
let board!: THREE.Group
let animated: StageAnimationTarget[] = []
let stageAssets: StageAssets | null = null
let mounted = false
let frame = 0
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let canvasWidth = 0
let canvasHeight = 0
let stream: MediaStream | null = null
let robotRig: THREE.Group | null = null
let robotModel: THREE.Object3D | null = null
let robotAnimator: ReturnType<typeof createRobotAnimator> | null = null
let pendingEvent: AnimationEvent | null = null
const robotCell = ref<{ x: number; y: number } | null>(null)
defineExpose({ robotCell, reset: replaceStage, resetRobot, refreshVisuals })
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
    board.rotation.set(1, 0, 0)
    if (board.position.z !== -3) board.position.set(0, -0.05, -3)
  } else {
    const horizontal = Math.cos(pitch) * distance
    camera.position.set(Math.sin(yaw) * horizontal, Math.sin(pitch) * distance, Math.cos(yaw) * horizontal)
    camera.lookAt(0, 0, 0)
    board.position.set(0, 0, 0)
    board.rotation.set(0, 0, 0)
  }
}

function replaceStage() {
  if (!scene) return
  distance = props.previewDistance
  pendingEvent = null
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
  setView()
}

function refreshVisuals(): void {
  if (!scene || !board) return
  const currentYaw = robotRig?.rotation.y
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
  setView()
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
  robotAnimator?.stop()
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

function playAnimationEvent(event: AnimationEvent | null): void {
  if (!event) return
  if (!robotAnimator) {
    pendingEvent = event
    return
  }
  if (!robotRig) return
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
}

function resize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    resizeTimer = null
    resizeImmediately()
  }, 280)
}

function animate(time: number): void {
  frame = requestAnimationFrame(animate)
  advanceWater(time)
  robotAnimator?.update(time)
  for (const target of animated) {
    target.object.rotation.y += target.kind === 'hover' ? 0.006 : 0.004
    if (target.kind === 'hover') target.object.position.y = target.baseY + Math.sin(time * 0.002) * 0.035
  }
  renderer.render(scene, camera)
}

async function startCamera() {
  stopCamera()
  if (props.mode !== 'camera') return
  if (!navigator.mediaDevices?.getUserMedia) {
    emit('error', 'カメラを利用できません。localhost または HTTPS で開いてください。')
    return
  }
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'environment' } },
      audio: false,
    })
    if (props.mode !== 'camera') {
      stopCamera()
      return
    }
    if (video.value) {
      video.value.srcObject = stream
      await video.value.play()
    }
  } catch (error) {
    emit('error', `カメラを開けませんでした: ${error instanceof Error ? error.message : String(error)}`)
  }
}

function stopCamera() {
  stream?.getTracks().forEach((track) => track.stop())
  stream = null
  if (video.value) video.value.srcObject = null
}

function pointerDown(event: PointerEvent): void {
  if (props.mode === 'camera') {
    if (!host.value || !board) return
    const rect = host.value.getBoundingClientRect()
    const depth = 3
    const halfHeight = Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * depth
    board.position.set(
      (((event.clientX - rect.left) / rect.width) * 2 - 1) * halfHeight * camera.aspect,
      (1 - ((event.clientY - rect.top) / rect.height) * 2) * halfHeight,
      -depth,
    )
    return
  }
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
        replaceStage()
      },
      (error) =>
        emit(
          'error',
          `ステージのGLBを読み込めませんでした: ${error instanceof Error ? error.message : String(error)}`,
        ),
    )
    loadRobot()
    resizeObserver = new ResizeObserver(resize)
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
  if (resizeTimer) clearTimeout(resizeTimer)
  stopCamera()
  robotAnimator?.stop()
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
    <video v-show="mode === 'camera'" ref="video" class="camera-feed" autoplay muted playsinline />
  </div>
</template>
