const ARM_NAMES = ['leftarm', 'rightarm']
const SHOULDER_NAMES = ['leftsholder', 'rightsholder']

function normalizedName(node) {
  return node.name.toLowerCase().replace(/[^a-z0-9]/g, '')
}

function findNamedNodes(model, names) {
  const found = new Map()
  model.traverse(node => {
    const name = normalizedName(node).replace('shoulder', 'sholder')
    for (const target of names) {
      if (!found.has(target) && (name === target || name.startsWith(`${target}0`))) found.set(target, node)
    }
  })
  return names.map(name => found.get(name) ?? null)
}

// Blender の left_arm / right_arm を glTF のノード名から探す。
export function findArmNodes(model) {
  return findNamedNodes(model, ARM_NAMES)
}

// 肩 Object の中心に実行時だけ回転軸を作り、現在の GLB 姿勢を待機姿勢として保つ。
export function createArmPivots(model, THREE) {
  const arms = findArmNodes(model)
  const shoulders = findNamedNodes(model, SHOULDER_NAMES)
  const pivots = []
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
export function placeModelOnGround(model, THREE) {
  model.updateWorldMatrix(true, true)
  const worldToModel = model.matrixWorld.clone().invert()
  const bounds = new THREE.Box3().makeEmpty()
  model.traverse(node => {
    if (!node.isMesh || !node.geometry) return
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

export function hasSuccessfulInteraction(events) {
  return events.some(event => event.robotId === 'player' && INTERACTION_EVENTS.has(event.kind))
}

function ease(t) {
  return (1 - Math.cos(Math.PI * t)) / 2
}

// 読み込んだ姿勢からの差分角。約 0.8 秒で 1 tick 内に収める。
export function armAngleAt(elapsedMs, deltaRadians) {
  if (elapsedMs <= 0 || elapsedMs >= 800) return 0
  if (elapsedMs < 280) return deltaRadians * ease(elapsedMs / 280)
  if (elapsedMs < 420) return deltaRadians
  return deltaRadians * (1 - ease((elapsedMs - 420) / 380))
}
