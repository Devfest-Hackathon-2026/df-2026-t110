<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'

const props = withDefaults(defineProps<{ variant?: 'orange' | 'green' | 'purple'; animated?: boolean }>(), {
  variant: 'orange',
  animated: true,
})
const host = ref<HTMLElement | null>(null)
const failed = ref(false)
let renderer!: THREE.WebGLRenderer
let scene!: THREE.Scene
let camera!: THREE.PerspectiveCamera
let model: THREE.Object3D | null = null
let observer: ResizeObserver | null = null
let frame = 0
let disposed = false
let baseY = 0

function resize() {
  if (!renderer || !host.value) return
  const { width, height } = host.value.getBoundingClientRect()
  if (!width || !height) return
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
  if (!props.animated) renderer.render(scene, camera)
}

function animate(time: number): void {
  frame = requestAnimationFrame(animate)
  if (!renderer || !scene) return
  if (model && document.visibilityState === 'visible') {
    model.position.y = baseY + Math.sin(time * 0.002) * 0.045
    model.rotation.y = 0.2 + Math.sin(time * 0.0008) * 0.08
  }
  renderer.render(scene, camera)
}

onMounted(() => {
  if (!host.value) return
  try {
    renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5))
    renderer.setClearColor(0x000000, 0)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    renderer.toneMapping = THREE.ACESFilmicToneMapping
    host.value.append(renderer.domElement)
    scene = new THREE.Scene()
    camera = new THREE.PerspectiveCamera(40, 1, 0.1, 20)
    camera.position.set(1.1, 1.45, -3.5)
    camera.lookAt(0, 0.5, 0)
    scene.add(new THREE.HemisphereLight(0xffffff, 0xa2b2a4, 2.4))
    const key = new THREE.DirectionalLight(0xffefda, 2.8)
    key.position.set(-2, 4, 5)
    scene.add(key)
    observer = new ResizeObserver(resize)
    observer.observe(host.value)
    resize()
    if (props.animated) frame = requestAnimationFrame(animate)

    new GLTFLoader().load(
      '/robot.glb',
      (gltf) => {
        if (disposed) return
        model = gltf.scene
        model.traverse((node) => {
          if (!(node instanceof THREE.Mesh) || !/^(Body|left_arm|right_arm)$/.test(node.name)) return
          if (!(node.material instanceof THREE.MeshStandardMaterial)) return
          if (props.variant !== 'green') {
            node.material = node.material.clone()
            node.material.color.set(props.variant === 'purple' ? '#9860d7' : '#ee783d')
            node.material.emissive.set(props.variant === 'purple' ? '#492471' : '#8f360f')
            node.material.emissiveIntensity = 0.12
          }
        })
        const bounds = new THREE.Box3().setFromObject(model)
        const size = bounds.getSize(new THREE.Vector3())
        const center = bounds.getCenter(new THREE.Vector3())
        const scale = Math.min(1.2 / size.y, 1.3 / Math.max(size.x, size.z))
        model.scale.setScalar(scale)
        baseY = -bounds.min.y * scale
        model.position.set(-center.x * scale, baseY, -center.z * scale)
        scene.add(model)
        if (!props.animated) renderer.render(scene, camera)
      },
      undefined,
      () => {
        failed.value = true
      },
    )
  } catch {
    failed.value = true
  }
})

onBeforeUnmount(() => {
  disposed = true
  cancelAnimationFrame(frame)
  observer?.disconnect()
  model?.traverse((node) => {
    if (!(node instanceof THREE.Mesh)) return
    node.geometry?.dispose()
    if (Array.isArray(node.material)) node.material.forEach((material) => material.dispose())
    else node.material?.dispose()
  })
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <div
    ref="host"
    class="tutor-avatar"
    role="img"
    :aria-label="`${props.variant === 'green' ? '緑' : props.variant === 'purple' ? '紫' : 'オレンジ'}色のロボット`"
  >
    <span v-if="failed" class="tutor-avatar-fallback">ロボット</span>
  </div>
</template>
