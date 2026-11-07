import * as THREE from 'three'

const IMPACT_AT = 0.29
const SHARD_COUNT = 14
const DUST_COUNT = 48

function rockBounds(rock: THREE.Object3D): THREE.Box3 {
  const bounds = new THREE.Box3()
  const parent = rock.parent
  if (!parent)
    return bounds.setFromCenterAndSize(new THREE.Vector3(0, 0.4, 0), new THREE.Vector3(0.7, 0.7, 0.7))
  parent.updateWorldMatrix(true, true)
  const inverse = parent.matrixWorld.clone().invert()
  rock.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) return
    object.geometry.computeBoundingBox()
    if (!object.geometry.boundingBox) return
    const local = object.geometry.boundingBox.clone()
    local.applyMatrix4(inverse.clone().multiply(object.matrixWorld))
    bounds.union(local)
  })
  if (bounds.isEmpty())
    bounds.setFromCenterAndSize(new THREE.Vector3(0, 0.4, 0), new THREE.Vector3(0.7, 0.7, 0.7))
  return bounds
}

export function createRockBreakEffect(rock: THREE.Object3D) {
  const parent = rock.parent
  if (!parent) throw new Error('岩が盤面に配置されていません')
  const container: THREE.Object3D = parent
  const bounds = rockBounds(rock)
  const center = bounds.getCenter(new THREE.Vector3())
  const size = bounds.getSize(new THREE.Vector3())
  const radius = THREE.MathUtils.clamp(Math.max(size.x, size.z) * 0.4, 0.18, 0.45)
  const height = THREE.MathUtils.clamp(size.y, 0.35, 0.9)
  const root = new THREE.Group()
  root.name = 'rock-break-effect'
  root.position.copy(center)
  root.visible = false
  container.add(root)

  const shardGeometry = new THREE.TetrahedronGeometry(1, 0)
  const shardMaterials = [0x8298a3, 0x526c7b, 0xb0bcc2].map(
    (color) => new THREE.MeshStandardMaterial({ color, roughness: 0.72, metalness: 0.16, transparent: true }),
  )
  const shards = Array.from({ length: SHARD_COUNT }, (_, index) => {
    const angle = index * 2.39996
    const spread = 0.12 + (index % 4) * 0.065
    const shard = new THREE.Mesh(shardGeometry, shardMaterials[index % shardMaterials.length]!)
    shard.position.set(Math.cos(angle) * spread, (index % 5) * 0.065 - height * 0.2, Math.sin(angle) * spread)
    shard.rotation.set(angle * 0.4, angle, angle * 0.2)
    shard.userData.start = shard.position.clone()
    shard.userData.rotation = shard.rotation.clone()
    shard.userData.velocity = new THREE.Vector3(
      Math.cos(angle) * (0.3 + (index % 3) * 0.11),
      0.3 + (index % 4) * 0.12,
      Math.sin(angle) * (0.3 + (index % 3) * 0.11),
    )
    shard.userData.size = radius * (0.31 + (index % 4) * 0.08)
    shard.scale.setScalar(shard.userData.size as number)
    root.add(shard)
    return shard
  })

  const dustPositions = new Float32Array(DUST_COUNT * 3)
  const dustColors = new Float32Array(DUST_COUNT * 3)
  const dustStarts: THREE.Vector3[] = []
  const dustVelocities: THREE.Vector3[] = []
  for (let index = 0; index < DUST_COUNT; index++) {
    const angle = index * 2.39996
    const distance = radius * Math.sqrt((index + 0.5) / DUST_COUNT)
    dustStarts.push(
      new THREE.Vector3(
        Math.cos(angle) * distance,
        ((index % 9) * height) / 11 - height * 0.32,
        Math.sin(angle) * distance,
      ),
    )
    dustVelocities.push(
      new THREE.Vector3(
        Math.cos(angle) * (0.22 + (index % 6) * 0.07),
        0.18 + (index % 7) * 0.07,
        Math.sin(angle) * (0.22 + (index % 6) * 0.07),
      ),
    )
    const color = new THREE.Color(index % 8 === 0 ? 0x9cf2d0 : index % 3 === 0 ? 0xb8c5c8 : 0x657f89)
    color.toArray(dustColors, index * 3)
  }
  const dustGeometry = new THREE.BufferGeometry()
  dustGeometry.setAttribute('position', new THREE.BufferAttribute(dustPositions, 3))
  dustGeometry.setAttribute('color', new THREE.BufferAttribute(dustColors, 3))
  const dustMaterial = new THREE.PointsMaterial({
    size: 0.075,
    vertexColors: true,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    sizeAttenuation: true,
  })
  root.add(new THREE.Points(dustGeometry, dustMaterial))
  let disposed = false

  function update(progress: number): void {
    if (disposed) return
    const burst = THREE.MathUtils.clamp((progress - IMPACT_AT) / (1 - IMPACT_AT), 0, 1)
    rock.visible = progress < IMPACT_AT
    root.visible = progress >= IMPACT_AT && burst < 1
    if (!root.visible) return
    const fade = Math.max(0, 1 - burst)
    for (const material of shardMaterials) material.opacity = Math.min(1, fade * 1.8)
    for (const shard of shards) {
      const start = shard.userData.start as THREE.Vector3
      const startRotation = shard.userData.rotation as THREE.Euler
      const velocity = shard.userData.velocity as THREE.Vector3
      shard.position.copy(start).addScaledVector(velocity, burst)
      shard.position.y -= 0.48 * burst * burst
      shard.rotation.set(
        startRotation.x + burst * 2.1,
        startRotation.y + burst * 1.3,
        startRotation.z + burst * 2.8,
      )
      shard.scale.setScalar((shard.userData.size as number) * Math.pow(fade, 0.75))
    }
    for (let index = 0; index < DUST_COUNT; index++) {
      const start = dustStarts[index]!
      const velocity = dustVelocities[index]!
      const offset = index * 3
      dustPositions[offset] = start.x + velocity.x * burst
      dustPositions[offset + 1] = start.y + velocity.y * burst - 0.12 * burst * burst
      dustPositions[offset + 2] = start.z + velocity.z * burst
    }
    dustGeometry.attributes.position!.needsUpdate = true
    dustMaterial.opacity = 0.85 * Math.sin(Math.PI * Math.min(1, burst * 1.2)) * fade
    dustMaterial.size = 0.075 + burst * 0.035
  }

  function dispose(): void {
    if (disposed) return
    disposed = true
    rock.visible = true
    container.remove(root)
    shardGeometry.dispose()
    shardMaterials.forEach((material) => material.dispose())
    dustGeometry.dispose()
    dustMaterial.dispose()
  }

  return { update, dispose }
}
