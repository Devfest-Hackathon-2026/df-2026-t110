<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as THREE from 'three'
import type { PicoTerrain } from '../picolang/language/picolang.ts'
import { loadStageAssets } from '../stage/stageAssets.ts'
import { createStageVisuals, disposeStageVisuals } from '../stage/stageVisuals.ts'
import type { StageLayout, StageObject } from '../stage/stage.ts'

const props = defineProps<{ terrain: PicoTerrain }>()
const host = ref<HTMLElement | null>(null)
let renderer: THREE.WebGLRenderer | null = null
let root: THREE.Group | null = null
let disposed = false

onMounted(async () => {
  if (!host.value) return
  try {
    renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    renderer.setSize(116, 94, false)
    host.value.append(renderer.domElement)
    const objects: StageObject[] = []
    if (props.terrain.kind?.type === 'goal') objects.push({ type: 'goal' })
    if (props.terrain.kind?.type === 'checkpoint')
      objects.push({ type: 'checkpoint', order: props.terrain.kind.order })
    if (props.terrain.hasLuggage) objects.push({ type: 'luggage' })
    const stage: StageLayout = {
      id: 'speech-terrain',
      title: '',
      label: '',
      rows: ['.'],
      width: 1,
      height: 1,
      rockCount: Number(props.terrain.type === 'rock'),
      cells: [{ x: 0, y: 0, symbol: '.', terrain: props.terrain.type, objects, start: false }],
      robot: { position: { x: 0, y: 0 }, direction: 'east' },
    }
    let assets
    try {
      assets = await loadStageAssets()
    } catch {
      /* Fall back to procedural terrain. */
    }
    if (disposed || !renderer) return
    root = createStageVisuals(stage, assets).root
    const scene = new THREE.Scene()
    scene.add(root)
    scene.add(new THREE.HemisphereLight(0xffffff, 0x456472, 2.8))
    const light = new THREE.DirectionalLight(0xffe5b6, 2.2)
    light.position.set(2, 4, 3)
    scene.add(light)
    const camera = new THREE.PerspectiveCamera(40, 116 / 94, 0.1, 20)
    camera.position.set(2.8, 2.4, 3.1)
    camera.lookAt(0, 0.35, 0)
    renderer.render(scene, camera)
  } catch {
    // The accompanying text remains available if WebGL cannot create another context.
  }
})

onBeforeUnmount(() => {
  disposed = true
  if (root) disposeStageVisuals(root)
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<template>
  <div ref="host" class="terrain-speech-model" aria-hidden="true"></div>
</template>
