import * as THREE from 'three'

const clamp01 = (value: number): number => Math.min(1, Math.max(0, value))
const smooth = (value: number): number => value * value * (3 - 2 * value)

// The parcel is in the cell ahead. Show a moving field around it without changing game state.
export function createTelekinesisEffect(robot: THREE.Object3D, releasing: boolean) {
  const color = releasing ? 0xffc76f : 0x86f5e4
  const root = new THREE.Group()
  root.name = releasing ? 'runtime-release-field' : 'runtime-pickup-field'
  const ringMaterial = new THREE.MeshBasicMaterial({
    color,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  })
  const moteMaterial = ringMaterial.clone()
  const rings = [0.19, 0.28].map((radius, index) => {
    const ring = new THREE.Mesh(new THREE.TorusGeometry(radius, 0.012, 6, 32), ringMaterial)
    ring.rotation.x = index === 0 ? 0.3 : -0.35
    root.add(ring)
    return ring
  })
  const motes = Array.from({ length: 6 }, (_, index) => {
    const mote = new THREE.Mesh(new THREE.OctahedronGeometry(0.025, 0), moteMaterial)
    mote.userData.phase = (index / 6) * Math.PI * 2
    root.add(mote)
    return mote
  })
  root.position.set(0, 0.38, releasing ? -0.48 : -0.92)
  root.visible = false
  robot.add(root)

  function update(progress: number, time: number): void {
    const strength = smooth(clamp01((progress - 0.2) / 0.16)) * smooth(clamp01((0.96 - progress) / 0.2))
    root.visible = strength > 0.01
    ringMaterial.opacity = 0.65 * strength
    moteMaterial.opacity = 0.9 * strength
    const travel = smooth(clamp01((progress - 0.4) / 0.36))
    root.position.z = releasing ? -0.48 - travel * 0.44 : -0.92 + travel * 0.44
    root.position.y = 0.38 + Math.sin(progress * Math.PI) * 0.08
    rings[0].rotation.z = time * 0.002
    rings[1].rotation.z = -time * 0.0016
    rings[0].scale.setScalar(0.9 + strength * 0.25)
    rings[1].scale.setScalar(1.1 - strength * 0.18)
    for (const mote of motes) {
      const phase = mote.userData.phase as number
      const angle = phase + time * 0.002
      mote.position.set(Math.cos(angle) * 0.22, Math.sin(angle) * 0.16, Math.sin(angle * 1.6) * 0.08)
      mote.scale.setScalar(0.65 + strength * 0.7)
    }
  }

  function dispose(): void {
    robot.remove(root)
    root.traverse((node) => {
      if (node instanceof THREE.Mesh) node.geometry.dispose()
    })
    ringMaterial.dispose()
    moteMaterial.dispose()
  }

  return { update, dispose }
}
