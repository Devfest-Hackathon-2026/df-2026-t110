import * as Three from 'three'

const ARM_NAMES = ['leftarm', 'rightarm']
const SHOULDER_NAMES = ['leftsholder', 'rightsholder']

function normalizedName(node: Three.Object3D): string {
  return node.name.toLowerCase().replace(/[^a-z0-9]/g, '')
}

function findNamedNodes(model: Three.Object3D, names: string[]): Array<Three.Object3D | null> {
  const found = new Map<string, Three.Object3D>()
  model.traverse((node) => {
    const name = normalizedName(node).replace('shoulder', 'sholder')
    for (const target of names) {
      if (!found.has(target) && (name === target || name.startsWith(`${target}0`))) found.set(target, node)
    }
  })
  return names.map((name) => found.get(name) ?? null)
}

// Blender の left_arm / right_arm を glTF のノード名から探す。
export function findArmNodes(model: Three.Object3D): Array<Three.Object3D | null> {
  return findNamedNodes(model, ARM_NAMES)
}

// 肩 Object の中心に実行時だけ回転軸を作り、現在の GLB 姿勢を待機姿勢として保つ。
export function createArmPivots(model: Three.Object3D, THREE: typeof Three) {
  const arms = findArmNodes(model)
  const shoulders = findNamedNodes(model, SHOULDER_NAMES)
  const pivots: Three.Group[] = []
  model.updateWorldMatrix(true, true)

  for (const [index, arm] of arms.entries()) {
    if (!arm) continue
    const shoulder = shoulders[index]
    const bounds = new THREE.Box3().setFromObject(shoulder ?? arm, true)
    if (bounds.isEmpty()) continue
    const hinge = new THREE.Vector3(
      (bounds.min.x + bounds.max.x) / 2,
      shoulder ? (bounds.min.y + bounds.max.y) / 2 : bounds.max.y,
      (bounds.min.z + bounds.max.z) / 2,
    )
    const pivot = new THREE.Group()
    pivot.name = `${arm.name}_runtime_shoulder`
    pivot.position.copy(model.worldToLocal(hinge))
    model.add(pivot)
    pivot.attach(arm)
    pivots.push(pivot)
  }

  return { pivots, found: arms.map(Boolean), shouldersFound: shoulders.map(Boolean) }
}

// 盤面の上面はロボット座標の y=0。全メッシュの最下点をそこに合わせる。
export function placeModelOnGround(model: Three.Object3D, THREE: typeof Three): number {
  model.updateWorldMatrix(true, true)
  const worldToModel = model.matrixWorld.clone().invert()
  const bounds = new THREE.Box3().makeEmpty()
  model.traverse((node) => {
    if (!(node instanceof THREE.Mesh) || !node.geometry) return
    if (!node.geometry.boundingBox) node.geometry.computeBoundingBox()
    const meshToModel = worldToModel.clone().multiply(node.matrixWorld)
    bounds.union(node.geometry.boundingBox.clone().applyMatrix4(meshToModel))
  })
  if (bounds.isEmpty()) return 0
  const offset = -bounds.min.y
  model.position.y += offset
  model.updateWorldMatrix(true, true)
  return offset
}

const INTERACTION_EVENTS = new Set(['get', 'put', 'score', 'drop', 'fill'])

export function hasSuccessfulInteraction(events: ReadonlyArray<{ robotId: string; kind: string }>): boolean {
  return events.some((event) => event.robotId === 'player' && INTERACTION_EVENTS.has(event.kind))
}

function ease(t: number): number {
  return (1 - Math.cos(Math.PI * t)) / 2
}

// GLBの腕は約 -120°。肩を +30°回すと、相互作用時に約 -90°を向く。
export function armAngleAt(elapsedMs: number, deltaRadians: number): number {
  if (elapsedMs <= 0 || elapsedMs >= 1000) return 0
  if (elapsedMs < 250) return deltaRadians * ease(elapsedMs / 250)
  if (elapsedMs < 700) return deltaRadians
  return deltaRadians * (1 - ease((elapsedMs - 700) / 300))
}
