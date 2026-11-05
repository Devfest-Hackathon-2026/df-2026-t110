import * as Three from 'three'

function findEyes(model: Three.Object3D): Three.Object3D[] {
  const eyes = new Map<string, Three.Object3D>()
  model?.traverse((node) => {
    const name = node.name.toLowerCase().replace(/[^a-z0-9]/g, '')
    if (name === 'lefteye' || name === 'righteye') eyes.set(name, node)
  })
  return ['lefteye', 'righteye']
    .map((name) => eyes.get(name))
    .filter((eye): eye is Three.Object3D => Boolean(eye))
}

export function createEyeBeam(
  robot: Three.Object3D,
  model: Three.Object3D,
  THREE: typeof Three,
): (() => void) | null {
  if (!robot || !THREE) return null
  robot.updateWorldMatrix(true, true)
  const eyes = findEyes(model)
  const origins = eyes.map((eye) => {
    const center = new THREE.Box3().setFromObject(eye).getCenter(new THREE.Vector3())
    return robot.worldToLocal(center)
  })
  if (!origins.length) {
    origins.push(new THREE.Vector3(-0.13, 0.45, -0.3), new THREE.Vector3(0.13, 0.45, -0.3))
  }

  const group = new THREE.Group()
  group.name = 'runtime-eye-beam'
  const target = new THREE.Vector3(0, 0.35, -1)
  const bright = new THREE.MeshBasicMaterial({
    color: 0xd8ff64,
    transparent: true,
    opacity: 0.95,
    depthWrite: false,
  })
  const halo = new THREE.MeshBasicMaterial({
    color: 0x75f2e8,
    transparent: true,
    opacity: 0.25,
    depthWrite: false,
  })
  const sphereGeometry = new THREE.SphereGeometry(0.055, 10, 8)

  for (const origin of origins) {
    const vector = target.clone().sub(origin)
    const length = vector.length()
    const rays: Array<[number, Three.MeshBasicMaterial]> = [
      [0.012, bright],
      [0.035, halo],
    ]
    for (const [radius, material] of rays) {
      const ray = new THREE.Mesh(new THREE.CylinderGeometry(radius, radius, length, 8), material)
      ray.position.copy(origin).addScaledVector(vector, 0.5)
      ray.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), vector.clone().normalize())
      group.add(ray)
    }
    const eyeGlow = new THREE.Mesh(sphereGeometry, bright)
    eyeGlow.position.copy(origin)
    group.add(eyeGlow)
  }
  const impact = new THREE.Mesh(new THREE.SphereGeometry(0.12, 12, 8), halo)
  impact.position.copy(target)
  group.add(impact)
  robot.add(group)

  return () => {
    robot.remove(group)
    group.traverse((node) => {
      if (node instanceof THREE.Mesh) node.geometry.dispose()
    })
    bright.dispose()
    halo.dispose()
  }
}
