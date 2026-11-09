import * as THREE from 'three'
import { stageAsset } from './stageAssets.ts'
import type { StageAssets, StageAssetName } from './stageAssets.ts'
import type { StageLayout, StageObject } from './stage.ts'

export interface StageAnimationTarget {
  object: THREE.Object3D
  kind: 'spin' | 'hover'
  baseY: number
}

const WATER_VERTEX = `
  uniform float uTime;
  varying vec2 vUv;
  void main() {
    vUv = uv;
    vec3 p = position;
    p.z += 0.018 * sin(p.x * 17.0 + uTime * 1.7) * cos(p.y * 13.0 - uTime * 1.2);
    gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
  }
`

const WATER_FRAGMENT = `
  uniform float uTime;
  varying vec2 vUv;
  float line(float x) {
    float f = fract(x);
    return 1.0 - smoothstep(0.0, 0.045, min(f, 1.0 - f));
  }
  void main() {
    float wave = 0.5 + 0.5 * sin(vUv.x * 19.0 + vUv.y * 13.0 - uTime * 1.6);
    float circuit = max(line(vUv.x * 7.0), line(vUv.y * 7.0));
    float ripple = 1.0 - smoothstep(0.015, 0.09, abs(length(vUv - vec2(0.5)) - fract(uTime * 0.18)));
    vec3 base = mix(vec3(0.015, 0.18, 0.22), vec3(0.04, 0.39, 0.46), wave);
    vec3 glow = vec3(0.28, 0.96, 0.86) * (circuit * 0.22 + ripple * 0.24);
    gl_FragColor = vec4(base + glow, 1.0);
  }
`

const material = (
  color: THREE.ColorRepresentation,
  metalness = 0.12,
  roughness = 0.48,
  emissive: THREE.ColorRepresentation = 0,
) => new THREE.MeshStandardMaterial({ color, metalness, roughness, emissive, emissiveIntensity: 0.2 })

const floorMaterial = material(0x638b82, 0.32)
const teamFloorColors = { 1: '#54ed65', 2: '#b077ed' } as const
type TerritoryTeam = keyof typeof teamFloorColors
const teamFloorMaterials = {
  1: material(teamFloorColors[1], 0.32),
  2: material(teamFloorColors[2], 0.32),
}
const tintedAssetMaterials = new WeakMap<
  THREE.MeshStandardMaterial,
  Map<TerritoryTeam, THREE.MeshStandardMaterial>
>()
const edgeMaterial = material(0x183d47, 0.55)
const circuitMaterial = material(0x8ce9c3, 0.25, 0.22, 0x4cae91)
const teamCircuitMaterials = {
  1: material(0x7cff82, 0.25, 0.22, 0x54ed65),
  2: material(0xd39cff, 0.25, 0.22, 0xb077ed),
}
const rockMaterial = material(0x8298a3, 0.36, 0.36)
const rockShade = material(0x455f70, 0.46, 0.42)
const waterMaterial = new THREE.ShaderMaterial({
  uniforms: { uTime: { value: 0 } },
  vertexShader: WATER_VERTEX,
  fragmentShader: WATER_FRAGMENT,
  side: THREE.DoubleSide,
})
const goldMaterial = material(0xffd888, 0.48, 0.23, 0xc98a32)
const parcelMaterial = material(0xc4f6a1, 0.36, 0.27, 0x84ca64)

function mesh(
  parent: THREE.Group,
  geometry: THREE.BufferGeometry,
  surface: THREE.Material,
  position: [number, number, number],
  rotation?: [number, number, number],
): THREE.Mesh {
  const object = new THREE.Mesh(geometry, surface)
  object.position.set(...position)
  if (rotation) object.rotation.set(...rotation)
  object.castShadow = true
  object.receiveShadow = true
  parent.add(object)
  return object
}

function label(parent: THREE.Group, letter: string, color: string, position: [number, number, number]): void {
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = 128
  const context = canvas.getContext('2d')
  if (!context) return
  context.fillStyle = color
  context.font = 'bold 84px sans-serif'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(letter, 64, 68)
  const texture = new THREE.CanvasTexture(canvas)
  const sign = new THREE.Sprite(
    new THREE.SpriteMaterial({ map: texture, transparent: true, depthWrite: false }),
  )
  sign.position.set(...position)
  sign.scale.set(0.35, 0.35, 1)
  parent.add(sign)
}

function territoryTeam(owner?: number): TerritoryTeam | undefined {
  return owner === 1 || owner === 2 ? owner : undefined
}

function tintedFloorAssetMaterial(
  source: THREE.MeshStandardMaterial,
  team: TerritoryTeam,
): THREE.MeshStandardMaterial {
  let variants = tintedAssetMaterials.get(source)
  if (!variants) {
    variants = new Map()
    tintedAssetMaterials.set(source, variants)
  }
  let tinted = variants.get(team)
  if (!tinted) {
    tinted = source.clone()
    const color = teamFloorColors[team]
    tinted.color.set(color)
    tinted.emissive.set(color)
    tinted.emissiveIntensity = source.name === 'RF_ceramic-deck' ? 0.1 : source.emissiveIntensity
    variants.set(team, tinted)
  }
  return tinted
}

function addFloorAsset(group: THREE.Group, assets: StageAssets, team?: TerritoryTeam): void {
  const instance = placeAsset(group, assets, 'floor')
  if (!team) return
  instance.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) return
    const tint = (surface: THREE.Material): THREE.Material =>
      surface instanceof THREE.MeshStandardMaterial &&
      (surface.name === 'RF_ceramic-deck' || surface.name === 'RF_mint-light')
        ? tintedFloorAssetMaterial(surface, team)
        : surface
    object.material = Array.isArray(object.material) ? object.material.map(tint) : tint(object.material)
  })
}

function addFloor(group: THREE.Group, team?: TerritoryTeam): void {
  mesh(group, new THREE.BoxGeometry(0.96, 0.2, 0.96), edgeMaterial, [0, -0.1, 0])
  mesh(
    group,
    new THREE.BoxGeometry(0.88, 0.055, 0.88),
    team ? teamFloorMaterials[team] : floorMaterial,
    [0, 0.025, 0],
  )
  for (const offset of [-0.36, 0.36]) {
    const circuit = team ? teamCircuitMaterials[team] : circuitMaterial
    mesh(group, new THREE.BoxGeometry(0.03, 0.008, 0.52), circuit, [offset, 0.057, 0])
    mesh(group, new THREE.BoxGeometry(0.52, 0.008, 0.03), circuit, [0, 0.057, offset])
  }
}

function addRock(group: THREE.Group, seed: number): void {
  for (let index = 0; index < 4; index++) {
    const angle = seed * 2.41 + index * 1.57
    const radius = index === 0 ? 0 : 0.21
    const height = index === 0 ? 0.7 : 0.32 + ((seed + index) % 3) * 0.08
    const crystal = mesh(
      group,
      new THREE.IcosahedronGeometry(0.25, 0),
      index % 2 ? rockShade : rockMaterial,
      [Math.cos(angle) * radius, 0.13 + height / 2, Math.sin(angle) * radius],
    )
    crystal.scale.set(index === 0 ? 1.25 : 0.72, height / 0.45, index === 0 ? 1.15 : 0.72)
    crystal.rotation.y = angle
  }
}

function addSea(group: THREE.Group, hasFrame = false): void {
  if (!hasFrame) mesh(group, new THREE.BoxGeometry(0.96, 0.2, 0.96), edgeMaterial, [0, -0.1, 0])
  mesh(group, new THREE.PlaneGeometry(0.84, 0.84, 20, 20), waterMaterial, [0, 0.035, 0], [-Math.PI / 2, 0, 0])
  for (const offset of [-0.25, 0.05, 0.34]) {
    mesh(
      group,
      new THREE.TorusGeometry(0.12, 0.008, 5, 32),
      circuitMaterial,
      [offset, 0.055, offset * 0.4],
      [-Math.PI / 2, 0, 0],
    )
  }
}

function placeAsset(group: THREE.Group, assets: StageAssets, name: StageAssetName): THREE.Group {
  const instance = stageAsset(assets, name)
  group.add(instance)
  return instance
}

function addSchemaObject(
  group: THREE.Group,
  object: StageObject,
  assets: StageAssets,
  animated: StageAnimationTarget[],
): void {
  const model = placeAsset(group, assets, object.type === 'dock' ? 'dock' : object.type)
  if (object.type === 'luggage') {
    model.name = 'stage-luggage'
    animated.push({ object: model, kind: 'hover', baseY: model.position.y })
  } else if (object.type === 'checkpoint') {
    const core = model.getObjectByName('RF_checkpoint-core')
    if (core) animated.push({ object: core, kind: 'hover', baseY: core.position.y })
    label(group, String(object.order ?? '?'), '#c9e9ff', [0, 0.88, 0])
  } else if (object.type === 'goal') {
    const halo = model.getObjectByName('RF_goal-halo')
    if (halo) animated.push({ object: halo, kind: 'spin', baseY: halo.position.y })
  }
}

function addMarker(group: THREE.Group, symbol: string, animated: THREE.Mesh[]): void {
  if (symbol === 'S') {
    mesh(group, new THREE.CylinderGeometry(0.26, 0.31, 0.08, 8), circuitMaterial, [0, 0.11, 0])
    label(group, 'S', '#ebffbd', [0, 0.53, 0])
  } else if (symbol === 'G') {
    mesh(group, new THREE.CylinderGeometry(0.28, 0.32, 0.1, 8), goldMaterial, [0, 0.12, 0])
    const ring = mesh(group, new THREE.TorusGeometry(0.22, 0.045, 8, 36), goldMaterial, [0, 0.43, 0])
    animated.push(ring)
    label(group, 'G', '#fff3bb', [0, 0.85, 0])
  } else if (symbol === 'O') {
    const parcel = mesh(group, new THREE.OctahedronGeometry(0.27), parcelMaterial, [0, 0.39, 0])
    parcel.name = 'stage-luggage'
    parcel.rotation.y = 0.4
    animated.push(parcel)
  } else if (symbol === 'X') {
    mesh(group, new THREE.CylinderGeometry(0.29, 0.34, 0.08, 8), goldMaterial, [0, 0.1, 0])
    mesh(
      group,
      new THREE.TorusGeometry(0.22, 0.025, 7, 4),
      goldMaterial,
      [0, 0.15, 0],
      [-Math.PI / 2, 0, Math.PI / 4],
    )
    label(group, 'X', '#fff3bb', [0, 0.57, 0])
  }
}

export function createStageVisuals(
  stage: StageLayout,
  assets?: StageAssets,
): {
  root: THREE.Group
  animated: StageAnimationTarget[]
} {
  const root = new THREE.Group()
  const animated: StageAnimationTarget[] = []
  const fallbackAnimated: THREE.Mesh[] = []
  for (const cell of stage.cells) {
    const tile = new THREE.Group()
    tile.userData.cellX = cell.x
    tile.userData.cellY = cell.y
    tile.position.set(cell.x - (stage.width - 1) / 2, 0, cell.y - (stage.height - 1) / 2)
    if (cell.terrain === 'sea') {
      if (assets) placeAsset(tile, assets, 'sea-frame')
      addSea(tile, Boolean(assets))
    } else {
      const team = territoryTeam(cell.territoryOwner)
      if (assets) addFloorAsset(tile, assets, team)
      else addFloor(tile, team)
      if (cell.terrain === 'rock') {
        const rock = new THREE.Group()
        rock.name = 'stage-rock'
        rock.userData.cellX = cell.x
        rock.userData.cellY = cell.y
        tile.add(rock)
        if (assets) {
          const model = placeAsset(rock, assets, 'rock')
          model.rotation.y = ((cell.x + cell.y * stage.width) % 7) * 0.28
        } else addRock(rock, cell.x + cell.y * stage.width)
      } else if (assets) {
        if (cell.start) {
          placeAsset(tile, assets, 'start')
        }
        for (const object of cell.objects) addSchemaObject(tile, object, assets, animated)
      } else {
        if (cell.start) addMarker(tile, 'S', fallbackAnimated)
        for (const object of cell.objects) {
          const symbol = { goal: 'G', checkpoint: 'G', luggage: 'O', dock: 'X' }[object.type]
          addMarker(tile, symbol, fallbackAnimated)
          if (object.type === 'checkpoint') label(tile, String(object.order ?? '?'), '#c9e9ff', [0, 0.9, 0])
        }
      }
    }
    root.add(tile)
  }
  for (const object of fallbackAnimated)
    animated.push({
      object,
      kind: object.geometry.type === 'OctahedronGeometry' ? 'hover' : 'spin',
      baseY: object.position.y,
    })
  // The board stays a manageable size in the camera view even for long tutorial maps.
  root.scale.setScalar(2.15 / Math.max(stage.width, stage.height))
  return { root, animated }
}

export function disposeStageVisuals(root: THREE.Group): void {
  root.traverse((object) => {
    if (object instanceof THREE.Sprite) {
      object.material.map?.dispose()
      object.material.dispose()
    } else if (object instanceof THREE.Mesh && !object.userData.sharedStageAsset) {
      object.geometry.dispose()
    }
  })
}

export function advanceWater(time: number): void {
  waterMaterial.uniforms.uTime.value = time * 0.001
}
