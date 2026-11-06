import * as THREE from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'

export const STAGE_ASSET_NAMES = [
  'floor',
  'rock',
  'goal',
  'checkpoint',
  'luggage',
  'dock',
  'start',
  'sea-frame',
] as const

export type StageAssetName = (typeof STAGE_ASSET_NAMES)[number]
export type StageAssets = Record<StageAssetName, THREE.Group>

let loading: Promise<StageAssets> | null = null

export function loadStageAssets(): Promise<StageAssets> {
  loading ??= Promise.all(
    STAGE_ASSET_NAMES.map(async (name) => {
      const gltf = await new GLTFLoader().loadAsync(`/stage-assets/${name}.glb`)
      gltf.scene.traverse((node) => {
        if (node instanceof THREE.Mesh) {
          // Clones share geometry and materials. Board replacement must not dispose them.
          node.userData.sharedStageAsset = true
          node.castShadow = true
          node.receiveShadow = true
        }
      })
      return [name, gltf.scene] as const
    }),
  ).then((entries) => Object.fromEntries(entries) as StageAssets)
  return loading
}

export function stageAsset(assets: StageAssets, name: StageAssetName): THREE.Group {
  return assets[name].clone(true)
}
