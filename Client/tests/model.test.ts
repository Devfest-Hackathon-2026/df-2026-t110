import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('移植したGLBに腕・肩・目のObjectが含まれる', () => {
  const data = readFileSync(new URL('../public/robot.glb', import.meta.url))
  assert.equal(data.toString('ascii', 0, 4), 'glTF')
  const jsonLength = data.readUInt32LE(12)
  const gltf = JSON.parse(data.toString('utf8', 20, 20 + jsonLength))
  const names = new Set((gltf.nodes as Array<{ name: string }>).map((node) => node.name))
  for (const name of ['left_arm', 'right_arm', 'left_sholder', 'right_sholder', 'left_eye', 'right_eye']) {
    assert.ok(names.has(name), `${name} がありません`)
  }
})
