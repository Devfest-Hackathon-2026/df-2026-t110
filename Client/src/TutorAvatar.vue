<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'

const host = ref(null)
const failed = ref(false)
let renderer, scene, camera, model, observer, frame
let disposed = false
let baseY = 0

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
  if (!renderer || !scene) return
  if (model && document.visibilityState === 'visible') {
    model.position.y = baseY + Math.sin(time * 0.002) * 0.045
    model.rotation.y = 0.2 + Math.sin(time * 0.0008) * 0.08
  }
  renderer.render(scene, camera)
}

onMounted(() => {
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
    frame = requestAnimationFrame(animate)

    new GLTFLoader().load('/robot.glb', gltf => {
      if (disposed) return
      model = gltf.scene
      model.traverse(node => {
        if (!node.isMesh || !/^(Body|left_arm|right_arm)$/.test(node.name)) return
        node.material = node.material.clone()
        node.material.color.set('#ee783d')
        node.material.emissive.set('#8f360f')
        node.material.emissiveIntensity = 0.12
      })
      const bounds = new THREE.Box3().setFromObject(model)
      const size = bounds.getSize(new THREE.Vector3())
      const center = bounds.getCenter(new THREE.Vector3())
      const scale = Math.min(1.2 / size.y, 1.3 / Math.max(size.x, size.z))
      model.scale.setScalar(scale)
      baseY = -bounds.min.y * scale
      model.position.set(-center.x * scale, baseY, -center.z * scale)
      scene.add(model)
    }, undefined, () => { failed.value = true })
  } catch {
    failed.value = true
  }
})

onBeforeUnmount(() => {
  disposed = true
  cancelAnimationFrame(frame)
  observer?.disconnect()
  model?.traverse(node => {
    if (!node.isMesh) return
    node.geometry?.dispose()
    if (Array.isArray(node.material)) node.material.forEach(material => material.dispose())
    else node.material?.dispose()
  })
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <div ref="host" class="tutor-avatar" role="img" aria-label="オレンジ色の案内ロボット">
    <span v-if="failed" class="tutor-avatar-fallback">案内役</span>
  </div>
</template>
