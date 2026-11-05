<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { createStageVisuals, disposeStageVisuals } from './stageVisuals.js'

const props = defineProps({ stage: { type: Object, required: true }, mode: { type: String, required: true } })
const emit = defineEmits(['error'])
const host = ref(null)
const video = ref(null)
let renderer, scene, camera, board, animated = [], frame, resizeObserver, stream
let yaw = 0.55
let pitch = 0.65
let distance = 4.8
let dragging = false
let lastPointer = [0, 0]

function setView() {
  if (!camera || !board) return
  if (props.mode === 'camera') {
    camera.position.set(0, 0, 0)
    camera.rotation.set(0, 0, 0)
    board.rotation.set(0.9, 0, 0)
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
  if (board) {
    scene.remove(board)
    disposeStageVisuals(board)
  }
  const visuals = createStageVisuals(props.stage)
  board = visuals.root
  animated = visuals.animated
  scene.add(board)
  board.position.z = 0
  setView()
}

function resize() {
  if (!renderer || !host.value) return
  const { width, height } = host.value.getBoundingClientRect()
  if (!width || !height) return
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
}

function animate(time) {
  frame = requestAnimationFrame(animate)
  for (const [index, object] of animated.entries()) {
    if (object.geometry.type === 'PlaneGeometry') object.position.y = -0.035 + Math.sin(time * 0.0015 + index) * 0.012
    else if (object.geometry.type === 'OctahedronGeometry') {
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
    stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: { ideal: 'environment' } }, audio: false })
    if (props.mode !== 'camera') { stopCamera(); return }
    if (video.value) {
      video.value.srcObject = stream
      await video.value.play()
    }
  } catch (error) {
    emit('error', `カメラを開けませんでした: ${error.message}`)
  }
}

function stopCamera() {
  stream?.getTracks().forEach(track => track.stop())
  stream = null
  if (video.value) video.value.srcObject = null
}

function pointerDown(event) {
  if (props.mode === 'camera') {
    const rect = host.value.getBoundingClientRect()
    const depth = 3
    const halfHeight = Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * depth
    board.position.set(((event.clientX - rect.left) / rect.width * 2 - 1) * halfHeight * camera.aspect,
      (1 - (event.clientY - rect.top) / rect.height * 2) * halfHeight, -depth)
    return
  }
  dragging = true
  lastPointer = [event.clientX, event.clientY]
  event.currentTarget.setPointerCapture(event.pointerId)
}

function pointerMove(event) {
  if (!dragging || props.mode !== 'preview') return
  yaw -= (event.clientX - lastPointer[0]) * 0.006
  pitch = THREE.MathUtils.clamp(pitch + (event.clientY - lastPointer[1]) * 0.006, 0.2, 1.35)
  lastPointer = [event.clientX, event.clientY]
  setView()
}

function wheel(event) {
  if (props.mode !== 'preview') return
  distance = THREE.MathUtils.clamp(distance + Math.sign(event.deltaY) * 0.35, 2.4, 9)
  setView()
}

onMounted(() => {
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
    resizeObserver = new ResizeObserver(resize)
    resizeObserver.observe(host.value)
    resize()
    frame = requestAnimationFrame(animate)
    startCamera()
  } catch (error) {
    emit('error', `3Dステージを開始できませんでした: ${error.message}`)
  }
})

watch(() => props.stage, replaceStage)
watch(() => props.mode, () => { setView(); startCamera() })
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  resizeObserver?.disconnect()
  stopCamera()
  if (board) disposeStageVisuals(board)
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <div ref="host" class="stage-scene" @pointerdown="pointerDown" @pointermove="pointerMove" @pointerup="dragging = false" @pointercancel="dragging = false" @wheel.prevent="wheel">
    <video v-show="mode === 'camera'" ref="video" class="camera-feed" autoplay muted playsinline />
  </div>
</template>
