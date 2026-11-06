import * as THREE from 'three'

function findEyes(model: THREE.Object3D): THREE.Object3D[] {
  const eyes = new Map<string, THREE.Object3D>()
  model.traverse((node) => {
    const name = node.name.toLowerCase().replace(/[^a-z0-9]/g, '')
    if (name === 'lefteye' || name === 'righteye') eyes.set(name, node)
  })
  return ['lefteye', 'righteye']
    .map((name) => eyes.get(name))
    .filter((eye): eye is THREE.Object3D => Boolean(eye))
}

const clamp01 = (value: number): number => Math.min(1, Math.max(0, value))

export function createEyeBeam(robot: THREE.Object3D, model: THREE.Object3D) {
  robot.updateWorldMatrix(true, true)
  const origins = findEyes(model).map((eye) => {
    const center = new THREE.Box3().setFromObject(eye).getCenter(new THREE.Vector3())
    return robot.worldToLocal(center)
  })
  if (!origins.length) {
    origins.push(new THREE.Vector3(-0.13, 0.45, -0.3), new THREE.Vector3(0.13, 0.45, -0.3))
  }

  const root = new THREE.Group()
  root.name = 'runtime-eye-beam'
  root.visible = false
  const target = new THREE.Vector3(0, 0.46, -1)
  const core = new THREE.MeshBasicMaterial({
    color: 0xe9ff8c,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  })
  const aura = new THREE.MeshBasicMaterial({
    color: 0x75f2e8,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  })

  for (const origin of origins) {
    const vector = target.clone().sub(origin)
    for (const [radius, material] of [
      [0.012, core],
      [0.038, aura],
    ] as const) {
      const ray = new THREE.Mesh(new THREE.CylinderGeometry(radius, radius, vector.length(), 8), material)
      ray.position.copy(origin).addScaledVector(vector, 0.5)
      ray.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), vector.clone().normalize())
      root.add(ray)
    }
    const eyeGlow = new THREE.Mesh(new THREE.SphereGeometry(0.052, 10, 8), core)
    eyeGlow.position.copy(origin)
    root.add(eyeGlow)
  }

  const impact = new THREE.Mesh(new THREE.SphereGeometry(0.11, 12, 8), aura)
  impact.position.copy(target)
  root.add(impact)
  const shockwave = new THREE.Mesh(new THREE.TorusGeometry(0.15, 0.012, 6, 32), core)
  shockwave.position.copy(target)
  shockwave.rotation.x = 0.35
  root.add(shockwave)
  const shards = Array.from({ length: 5 }, (_, index) => {
    const shard = new THREE.Mesh(new THREE.OctahedronGeometry(0.035, 0), core)
    shard.userData.phase = (index / 5) * Math.PI * 2
    root.add(shard)
    return shard
  })
  robot.add(root)

  function update(progress: number, time: number): void {
    const charge = clamp01((progress - 0.12) / 0.15)
    const fade = clamp01((0.88 - progress) / 0.18)
    const strength = charge * fade
    root.visible = strength > 0.01
    core.opacity = 0.9 * strength
    aura.opacity = (0.22 + Math.sin(time * 0.04) * 0.06) * strength
    const impactPulse = 0.8 + Math.sin(time * 0.035) * 0.18
    impact.scale.setScalar(impactPulse)
    shockwave.scale.setScalar(0.7 + progress * 1.5)
    shockwave.rotation.z = time * 0.002
    for (const shard of shards) {
      const angle = shard.userData.phase as number
      const spread = 0.1 + progress * 0.22
      shard.position.set(
        target.x + Math.cos(angle) * spread,
        target.y + Math.sin(angle) * spread,
        target.z - 0.02 + Math.sin(time * 0.012 + angle) * 0.04,
      )
      shard.rotation.set(time * 0.002 + angle, angle, time * 0.003)
    }
  }

  function dispose(): void {
    robot.remove(root)
    root.traverse((node) => {
      if (node instanceof THREE.Mesh) node.geometry.dispose()
    })
    core.dispose()
    aura.dispose()
  }

  return { update, dispose }
}
