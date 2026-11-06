import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { armAngleAt } from '../src/features/stage/armController.ts'
import { createRobotAnimator } from '../src/features/stage/robotAnimator.ts'

function robot() {
  const rig = new THREE.Group()
  rig.position.y = 0.07
  rig.rotation.y = -Math.PI / 2
  const model = new THREE.Group()
  const pivots = [new THREE.Group(), new THREE.Group()]
  rig.add(model)
  pivots.forEach((pivot) => model.add(pivot))
  return { rig, model, pivots, animator: createRobotAnimator(rig, model, pivots) }
}

test('相互作用の中盤で腕が -90°相当になり、終了時に待機姿勢へ戻る', () => {
  const { rig, pivots, animator } = robot()
  assert.equal(armAngleAt(0, Math.PI / 6), 0)
  animator.play({ id: 1, type: 'pick_up' })
  animator.update(0)
  animator.update(500)
  for (const pivot of pivots) assert.ok(Math.abs(pivot.rotation.x - Math.PI / 6) < 0.001)
  assert.ok(rig.children.some((child) => child.name === 'runtime-pickup-field'))
  animator.update(1000)
  for (const pivot of pivots) assert.equal(pivot.rotation.x, 0)
  assert.ok(!rig.children.some((child) => child.name === 'runtime-pickup-field'))
})

test('左回転は1単位時間で90度進み、後退では向きを保つ', () => {
  const { rig, model, animator } = robot()
  animator.play({ id: 1, type: 'rotate', direction: 'left' }, undefined, 0)
  animator.update(0)
  animator.update(500)
  assert.ok(Math.abs(rig.rotation.y + Math.PI / 4) < 0.001)
  animator.update(1000)
  assert.ok(Math.abs(rig.rotation.y) < 0.001)

  const destination = new THREE.Vector3(0, 0.07, 1)
  animator.play({ id: 2, type: 'back' }, destination)
  animator.update(1100)
  animator.update(1600)
  assert.ok(rig.position.z > 0 && rig.position.z < 1)
  assert.ok(Math.abs(rig.rotation.y) < 0.001)
  animator.update(2100)
  assert.equal(rig.position.z, 1)
  assert.equal(model.rotation.x, 0)
})
