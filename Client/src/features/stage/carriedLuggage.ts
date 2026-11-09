import * as THREE from 'three'
import { stageAsset, type StageAssets } from './stageAssets.ts'

export const CARRIED_LUGGAGE_HEIGHT = 1.22

export function createCarriedLuggage(assets?: StageAssets): THREE.Group {
  const root = new THREE.Group()
  root.name = 'carried-luggage'
  const luggage = assets
    ? stageAsset(assets, 'luggage')
    : new THREE.Mesh(
        new THREE.OctahedronGeometry(0.23),
        new THREE.MeshStandardMaterial({ color: 0xc4f6a1, metalness: 0.36, roughness: 0.27 }),
      )
  const bounds = new THREE.Box3().setFromObject(luggage)
  if (!bounds.isEmpty()) {
    const size = bounds.getSize(new THREE.Vector3())
    const center = bounds.getCenter(new THREE.Vector3())
    const scale = 0.42 / Math.max(size.x, size.y, size.z)
    luggage.scale.multiplyScalar(scale)
    luggage.position.addScaledVector(center, -scale)
  }
  root.add(luggage)
  root.position.y = CARRIED_LUGGAGE_HEIGHT
  return root
}

export function disposeCarriedLuggage(root: THREE.Group): void {
  root.removeFromParent()
  root.traverse((object) => {
    if (!(object instanceof THREE.Mesh) || object.userData.sharedStageAsset) return
    object.geometry.dispose()
    const materials = Array.isArray(object.material) ? object.material : [object.material]
    for (const material of materials) material.dispose()
  })
}
