import assert from 'node:assert/strict'
import test from 'node:test'
import * as THREE from 'three'
import { replayFrameToStage, type ReplayFrame } from '../src/features/match/matchReplay.ts'
import { createStageVisuals, disposeStageVisuals } from '../src/features/stage/stageVisuals.ts'
import type { StageAssets } from '../src/features/stage/stageAssets.ts'

test('対戦の陣地属性を床モデルのテクスチャに反映し、通常床の材質は変えない', () => {
  const frame: ReplayFrame = {
    tick: 0,
    terrain: [
      [
        { type: 'land', kind: { type: 'plain' }, territoryOwner: 1 },
        { type: 'land', kind: { type: 'plain' }, territoryOwner: 2 },
        { type: 'land', kind: { type: 'plain' } },
      ],
    ],
    robots: [],
    actions: [],
    removedRobotIds: [],
  }
  const stage = replayFrameToStage(frame)
  assert.deepEqual(
    stage.cells.map((cell) => cell.territoryOwner),
    [1, 2, undefined],
  )

  const texture = new THREE.Texture()
  const deckMaterial = new THREE.MeshStandardMaterial({ map: texture, name: 'RF_ceramic-deck' })
  const floor = new THREE.Group()
  floor.add(new THREE.Mesh(new THREE.BoxGeometry(1, 0.2, 1), deckMaterial))
  const assets = { floor } as StageAssets
  const { root } = createStageVisuals(stage, assets)
  const surfaces = root.children.map(
    (tile) =>
      (tile.children[0] as THREE.Group).children[0] as THREE.Mesh<
        THREE.BoxGeometry,
        THREE.MeshStandardMaterial
      >,
  )

  assert.equal(surfaces[0].geometry, surfaces[1].geometry)
  assert.equal(surfaces[0].material.map, texture)
  assert.equal(surfaces[1].material.map, texture)
  assert.notEqual(surfaces[0].material, deckMaterial)
  assert.notEqual(surfaces[1].material, deckMaterial)
  assert.notEqual(surfaces[0].material.color.getHex(), surfaces[1].material.color.getHex())
  assert.equal(surfaces[2].material, deckMaterial)
  assert.equal(deckMaterial.color.getHex(), 0xffffff)

  disposeStageVisuals(root)
  floor.children[0].traverse((object) => {
    if (object instanceof THREE.Mesh) object.geometry.dispose()
  })
  deckMaterial.dispose()
  texture.dispose()
})
