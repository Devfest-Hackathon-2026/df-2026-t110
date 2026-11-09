import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { MarkerPoseStabilizer } from '../src/features/stage/markerPoseStabilizer.ts'

function pose(x: number, degrees: number): THREE.Matrix4 {
  return new THREE.Matrix4().compose(
    new THREE.Vector3(x, 0, 0),
    new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), THREE.MathUtils.degToRad(degrees)),
    new THREE.Vector3(1, 1, 1),
  )
}

test('small camera shake leaves the rendered marker pose still', () => {
  const stabilizer = new MarkerPoseStabilizer()
  stabilizer.update(pose(0, 0), 0)
  for (let frame = 1; frame <= 30; frame++) {
    const direction = frame % 2 === 0 ? 1 : -1
    stabilizer.update(pose(direction * 0.006, direction * 0.8), frame * 16)
  }
  assert.equal(stabilizer.position.x, 0)
  assert.equal(stabilizer.quaternion.angleTo(new THREE.Quaternion()), 0)
})

test('intentional movement follows smoothly and a new placement snaps into place', () => {
  const stabilizer = new MarkerPoseStabilizer()
  stabilizer.update(pose(0, 0), 0)
  stabilizer.update(pose(0.08, 10), 16)
  assert.ok(stabilizer.position.x > 0)
  assert.ok(stabilizer.position.x < 0.08)
  for (let frame = 2; frame <= 60; frame++) stabilizer.update(pose(0.08, 10), frame * 16)
  assert.ok(stabilizer.position.x > 0.06)
  assert.ok(stabilizer.quaternion.angleTo(new THREE.Quaternion()) > THREE.MathUtils.degToRad(7))

  stabilizer.reset()
  stabilizer.update(pose(-0.5, -45), 1000)
  assert.equal(stabilizer.position.x, -0.5)
  assert.ok(stabilizer.quaternion.angleTo(new THREE.Quaternion()) > THREE.MathUtils.degToRad(44))
})
