import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { armAngleAt } from '../src/features/stage/armController.ts'
import { createRobotAnimator, HOLD_ARM_ANGLE } from '../src/features/stage/robotAnimator.ts'

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

test('荷物を拾うと腕を上げ、運搬中の移動でも保持し、置くと戻す', () => {
  const { rig, pivots, animator } = robot()
  assert.equal(armAngleAt(0, Math.PI / 6), 0)
  animator.play({ id: 1, type: 'pick_up', success: true })
  animator.update(0)
  animator.update(500)
  for (const pivot of pivots) assert.ok(Math.abs(pivot.rotation.x - Math.PI / 6) < 0.001)
  assert.ok(rig.children.some((child) => child.name === 'runtime-pickup-field'))
  animator.update(1000)
  for (const pivot of pivots) assert.ok(Math.abs(pivot.rotation.x - HOLD_ARM_ANGLE) < 0.001)
  assert.ok(!rig.children.some((child) => child.name === 'runtime-pickup-field'))

  animator.play({ id: 2, type: 'forward', success: true }, new THREE.Vector3(1, 0.07, 0))
  animator.update(1100)
  animator.update(1600)
  for (const pivot of pivots) assert.ok(Math.abs(pivot.rotation.x - HOLD_ARM_ANGLE) < 0.001)
  animator.update(2100)

  animator.play({ id: 3, type: 'release', success: true })
  animator.update(2200)
  animator.update(2700)
  for (const pivot of pivots) assert.ok(Math.abs(pivot.rotation.x - Math.PI / 6) < 0.001)
  animator.update(3200)
  for (const pivot of pivots) assert.equal(pivot.rotation.x, 0)
})

test('再生位置の変更に合わせて荷物保持姿勢を復元する', () => {
  const { pivots, animator } = robot()
  animator.setCarrying(true)
  for (const pivot of pivots) assert.equal(pivot.rotation.x, HOLD_ARM_ANGLE)
  animator.stop()
  for (const pivot of pivots) assert.equal(pivot.rotation.x, HOLD_ARM_ANGLE)
  animator.setCarrying(false)
  for (const pivot of pivots) assert.equal(pivot.rotation.x, 0)
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
