import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { createRockBreakEffect } from '../src/features/stage/rockBreakEffect.ts'

test('ビーム着弾時に岩が破片へ切り替わり、チリが消えた後に片付く', () => {
  const tile = new THREE.Group()
  const rock = new THREE.Group()
  rock.add(new THREE.Mesh(new THREE.IcosahedronGeometry(0.35), new THREE.MeshStandardMaterial()))
  tile.add(rock)
  const effect = createRockBreakEffect(rock)
  const burst = tile.getObjectByName('rock-break-effect')
  assert.ok(burst)
  effect.update(0.2)
  assert.equal(rock.visible, true)
  assert.equal(burst.visible, false)
  effect.update(0.4)
  assert.equal(rock.visible, false)
  assert.equal(burst.visible, true)
  assert.ok(burst.children.some((child) => child instanceof THREE.Points))
  effect.update(1)
  assert.equal(burst.visible, false)
  effect.dispose()
  assert.equal(rock.visible, true)
  assert.equal(tile.getObjectByName('rock-break-effect'), undefined)
})
