import * as THREE from 'three'

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

const material = (color, metalness = 0.12, roughness = 0.48, emissive = 0) =>
  new THREE.MeshStandardMaterial({ color, metalness, roughness, emissive, emissiveIntensity: 0.2 })

const floorMaterial = material(0x638b82, 0.32)
const edgeMaterial = material(0x183d47, 0.55)
const circuitMaterial = material(0x8ce9c3, 0.25, 0.22, 0x4cae91)
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

function mesh(parent, geometry, surface, position, rotation) {
  const object = new THREE.Mesh(geometry, surface)
  object.position.set(...position)
  if (rotation) object.rotation.set(...rotation)
  object.castShadow = true
  object.receiveShadow = true
  parent.add(object)
  return object
}

function label(parent, letter, color, position) {
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = 128
  const context = canvas.getContext('2d')
  context.fillStyle = color
  context.font = 'bold 84px sans-serif'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(letter, 64, 68)
  const texture = new THREE.CanvasTexture(canvas)
  const sign = new THREE.Sprite(new THREE.SpriteMaterial({ map: texture, transparent: true, depthWrite: false }))
  sign.position.set(...position)
  sign.scale.set(0.35, 0.35, 1)
  parent.add(sign)
}

function addFloor(group) {
  mesh(group, new THREE.BoxGeometry(0.96, 0.2, 0.96), edgeMaterial, [0, -0.1, 0])
  mesh(group, new THREE.BoxGeometry(0.88, 0.055, 0.88), floorMaterial, [0, 0.025, 0])
  for (const offset of [-0.36, 0.36]) {
    mesh(group, new THREE.BoxGeometry(0.03, 0.008, 0.52), circuitMaterial, [offset, 0.057, 0])
    mesh(group, new THREE.BoxGeometry(0.52, 0.008, 0.03), circuitMaterial, [0, 0.057, offset])
  }
}

function addRock(group, seed) {
  for (let index = 0; index < 4; index++) {
    const angle = seed * 2.41 + index * 1.57
    const radius = index === 0 ? 0 : 0.21
    const height = index === 0 ? 0.7 : 0.32 + (seed + index) % 3 * 0.08
    const crystal = mesh(group, new THREE.IcosahedronGeometry(0.25, 0), index % 2 ? rockShade : rockMaterial,
      [Math.cos(angle) * radius, 0.13 + height / 2, Math.sin(angle) * radius])
    crystal.scale.set(index === 0 ? 1.25 : 0.72, height / 0.45, index === 0 ? 1.15 : 0.72)
    crystal.rotation.y = angle
  }
}

function addSea(group) {
  mesh(group, new THREE.BoxGeometry(0.96, 0.2, 0.96), edgeMaterial, [0, -0.1, 0])
  mesh(group, new THREE.PlaneGeometry(0.84, 0.84, 20, 20), waterMaterial, [0, 0.035, 0], [-Math.PI / 2, 0, 0])
  for (const offset of [-0.25, 0.05, 0.34]) {
    mesh(group, new THREE.TorusGeometry(0.12, 0.008, 5, 32), circuitMaterial,
      [offset, 0.055, offset * 0.4], [-Math.PI / 2, 0, 0])
  }
}

function addMarker(group, symbol, animated) {
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
    parcel.rotation.y = 0.4
    animated.push(parcel)
  } else if (symbol === 'X') {
    mesh(group, new THREE.CylinderGeometry(0.29, 0.34, 0.08, 8), goldMaterial, [0, 0.1, 0])
    mesh(group, new THREE.TorusGeometry(0.22, 0.025, 7, 4), goldMaterial, [0, 0.15, 0], [-Math.PI / 2, 0, Math.PI / 4])
    label(group, 'X', '#fff3bb', [0, 0.57, 0])
  }
}

export function createStageVisuals(stage) {
  const root = new THREE.Group()
  const animated = []
  for (const cell of stage.cells) {
    const tile = new THREE.Group()
    tile.position.set(cell.x - (stage.width - 1) / 2, 0, cell.y - (stage.height - 1) / 2)
    if (cell.symbol === '_') addSea(tile)
    else {
      addFloor(tile)
      if (cell.symbol === '#') addRock(tile, cell.x + cell.y * stage.width)
      else addMarker(tile, cell.symbol, animated)
    }
    root.add(tile)
  }
  // The board stays a manageable size in the camera view even for long tutorial maps.
  root.scale.setScalar(2.15 / Math.max(stage.width, stage.height))
  return { root, animated }
}

export function disposeStageVisuals(root) {
  root.traverse(object => {
    if (object.isSprite) {
      object.material.map?.dispose()
      object.material.dispose()
    } else if (object.isMesh) {
      object.geometry.dispose()
    }
  })
}

export function advanceWater(time) {
  waterMaterial.uniforms.uTime.value = time * 0.001
}
