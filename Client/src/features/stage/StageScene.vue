<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { advanceWater, createStageVisuals, disposeStageVisuals } from './stageVisuals.ts'
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
let animated: THREE.Mesh[] = []
let frame = 0
let resizeObserver: ResizeObserver | null = null
let stream: MediaStream | null = null
let robotRig: THREE.Group | null = null
let robotModel: THREE.Object3D | null = null
let robotAnimator: ReturnType<typeof createRobotAnimator> | null = null
let pendingEvent: AnimationEvent | null = null
const robotCell = ref<{ x: number; y: number } | null>(null)
defineExpose({ robotCell })
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
  const visuals = createStageVisuals(props.stage)
  board = visuals.root
  animated = visuals.animated
  scene.add(board)
  const start = props.stage.cells.find((cell) => cell.symbol === 'S')
  robotCell.value = start ? { x: start.x, y: start.y } : null
  if (robotRig) {
    board.add(robotRig)
    positionRobot(start)
  }
  board.position.z = 0
  setView()
}

function positionRobot(cell: StageCell | { x: number; y: number } | null | undefined): void {
  if (!robotRig || !cell) return
  robotRig.position.set(cell.x - (props.stage.width - 1) / 2, 0.07, cell.y - (props.stage.height - 1) / 2)
  robotRig.rotation.y = -Math.PI / 2
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

function playAnimationEvent(event: AnimationEvent | null): void {
  if (!event) return
  if (!robotAnimator) {
    pendingEvent = event
    return
  }
  if (!robotRig) return
  let destination
  if (event.type === 'move') {
    const { x, y } = event.to ?? {}
    if (typeof x !== 'number' || typeof y !== 'number') return
    const cell = props.stage.cells.find((item) => item.x === x && item.y === y)
    if (!cell || ['#', '_'].includes(cell.symbol)) return
    const previous = robotCell.value
    if (!previous) return
    if (x > previous.x) robotRig.rotation.y = -Math.PI / 2
    else if (x < previous.x) robotRig.rotation.y = Math.PI / 2
    else if (y > previous.y) robotRig.rotation.y = Math.PI
    else if (y < previous.y) robotRig.rotation.y = 0
    destination = new THREE.Vector3(x - (props.stage.width - 1) / 2, 0.07, y - (props.stage.height - 1) / 2)
    robotCell.value = { x, y }
  }
  robotAnimator.play(event, destination)
}

function resize() {
  if (!renderer || !host.value) return
  const { width, height } = host.value.getBoundingClientRect()
  if (!width || !height) return
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
}

function animate(time: number): void {
  frame = requestAnimationFrame(animate)
  advanceWater(time)
  robotAnimator?.update(time)
  for (const object of animated) {
    if (object.geometry.type === 'OctahedronGeometry') {
      object.rotation.y += 0.006
      object.position.y = 0.39 + Math.sin(time * 0.002) * 0.035
    } else object.rotation.y += 0.004
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
    loadRobot()
    resizeObserver = new ResizeObserver(resize)
    resizeObserver.observe(host.value)
    resize()
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
  cancelAnimationFrame(frame)
  resizeObserver?.disconnect()
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
