import test from 'node:test'
import assert from 'node:assert/strict'
import { shallowRef } from 'vue'
import * as THREE from 'three'
import { useRoomMarkerPlacement } from '../src/features/rooms/useRoomMarkerPlacement.ts'
import type {
  RoomMarkerCallbacks,
  RoomMarkerId,
  RoomMarkerState,
  RoomMarkerTracker,
} from '../src/features/rooms/roomMarkerTracking.ts'

function fixture(pending = false) {
  let board = new THREE.Group()
  let cameraMode = true
  const camera = new THREE.PerspectiveCamera(48, 1, 0.05, 100)
  const previewProjection = camera.projectionMatrix.clone()
  const trackingProjection = new THREE.PerspectiveCamera(60, 4 / 3, 0.01, 100).projectionMatrix
  const scene = Object.assign(new EventTarget(), {
    style: {},
    camera: { projectionMatrix: trackingProjection },
    systems: { arjs: { _arSession: { onResize: () => resizeCount++ } } },
  })
  const video = { videoWidth: 640, videoHeight: 480 }
  const host = shallowRef({
    clientWidth: 400,
    clientHeight: 600,
    querySelector: () => video,
  } as unknown as HTMLElement)
  const markers = { a: new THREE.Group(), b: new THREE.Group() }
  let callbacks: RoomMarkerCallbacks
  let signal: AbortSignal | undefined
  let stopCount = 0
  let resizeCount = 0
  const errors: string[] = []
  let resolveMount: (tracker: RoomMarkerTracker) => void
  const tracker = {
    scene: scene as unknown as HTMLElement,
    markers: {
      a: { object3D: markers.a },
      b: { object3D: markers.b },
    } as unknown as RoomMarkerTracker['markers'],
    stop: () => stopCount++,
  }
  const placement = useRoomMarkerPlacement({
    host,
    getBoard: () => board,
    getCamera: () => camera,
    isCamera: () => cameraMode,
    onError: (message) => errors.push(message),
    mountTracker: (_host, handlers, abortSignal) => {
      callbacks = handlers
      signal = abortSignal
      return pending
        ? new Promise((resolve) => {
            resolveMount = resolve
          })
        : Promise.resolve(tracker)
    },
  })
  return {
    placement,
    camera,
    host,
    previewProjection,
    trackingProjection,
    errors,
    get board() {
      return board
    },
    get stopCount() {
      return stopCount
    },
    get resizeCount() {
      return resizeCount
    },
    get signal() {
      return signal
    },
    detect(state: RoomMarkerState) {
      callbacks.onChange(state)
    },
    ready() {
      callbacks.onCameraReady?.()
    },
    fail(error: Error) {
      callbacks.onCameraError?.(error)
    },
    pose(id: RoomMarkerId, matrix: THREE.Matrix4) {
      markers[id].matrixWorld.copy(matrix)
    },
    replaceBoard() {
      const previous = board
      board = new THREE.Group()
      return placement.retainBoardPose(previous)
    },
    finishMount() {
      resolveMount(tracker)
    },
    preview() {
      cameraMode = false
      placement.stop()
    },
  }
}

function pose(x: number, rotation = 0): THREE.Matrix4 {
  return new THREE.Matrix4().compose(
    new THREE.Vector3(x, 0.1, -2),
    new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), rotation),
    new THREE.Vector3(1, 1, 1),
  )
}

for (const anchor of ['a', 'b'] as const) {
  test(`placement on ${anchor} waits for confirmation and uses the calibrated alternate marker`, async () => {
    const f = fixture()
    const alternate = anchor === 'a' ? 'b' : 'a'
    await f.placement.start()
    assert.equal(f.board.visible, false)
    f.ready()
    assert.equal(f.placement.cameraReady.value, true)
    f.placement.confirmPlacement()
    assert.equal(f.placement.placementConfirmed.value, false)

    f.pose(anchor, pose(0.3, 0.2))
    f.pose(alternate, pose(0.8, -0.4))
    f.detect({ a: anchor === 'a', b: anchor === 'b' })
    f.placement.update(0)
    assert.equal(f.board.visible, false)
    f.placement.confirmPlacement()
    f.placement.update(16)
    assert.equal(f.board.visible, true)
    assert.equal(f.placement.calibrated.value, false)
    assert.ok(f.board.position.distanceTo(new THREE.Vector3(0.3, 0.1, -2)) < 1e-10)

    // Seeing an uncalibrated alternate alone must not move the field to that marker.
    f.detect({ a: alternate === 'a', b: alternate === 'b' })
    f.placement.update(400)
    assert.equal(f.board.visible, false)

    f.detect({ a: true, b: true })
    f.placement.update(416)
    assert.equal(f.placement.calibrated.value, true)
    const confirmedPosition = f.board.position.clone()
    const confirmedRotation = f.board.quaternion.clone()
    f.detect({ a: alternate === 'a', b: alternate === 'b' })
    f.placement.update(432)
    assert.equal(f.board.visible, true)
    assert.ok(f.board.position.distanceTo(confirmedPosition) < 1e-10)
    assert.ok(f.board.quaternion.angleTo(confirmedRotation) < 1e-7)
  })
}

test('brief detection loss keeps placement, longer loss hides the board and reacquisition restores it', async () => {
  const f = fixture()
  await f.placement.start()
  f.detect({ a: true, b: false })
  f.pose('a', pose(0.2))
  f.placement.confirmPlacement()
  f.placement.update(0)
  f.detect({ a: false, b: false })
  f.placement.update(299)
  assert.equal(f.board.visible, true)
  f.placement.update(300)
  assert.equal(f.board.visible, false)
  assert.equal(f.placement.trackingPoseAvailable.value, false)

  f.pose('a', pose(1))
  f.detect({ a: true, b: false })
  f.placement.update(400)
  assert.equal(f.board.visible, true)
  assert.equal(f.placement.trackingPoseAvailable.value, true)
  assert.equal(f.board.position.x, 1)
})

test('replay frame replacement retains placement and visibility even during marker loss', async () => {
  const f = fixture()
  await f.placement.start()
  f.detect({ a: true, b: false })
  f.pose('a', pose(0.4, 0.5))
  f.placement.confirmPlacement()
  f.placement.update(0)
  const previous = f.board.clone()
  assert.equal(f.replaceBoard(), true)
  assert.deepEqual(f.board.position, previous.position)
  assert.deepEqual(f.board.quaternion.toArray(), previous.quaternion.toArray())
  assert.deepEqual(f.board.scale, previous.scale)
  assert.equal(f.board.visible, true)
  f.detect({ a: false, b: false })
  f.placement.update(301)
  assert.equal(f.replaceBoard(), true)
  assert.equal(f.board.visible, false)
  f.preview()
  assert.equal(f.replaceBoard(), false)
})

test('camera projection matches video cover cropping and resets when returning to 3D', async () => {
  const f = fixture()
  await f.placement.start()
  f.placement.update(0)
  assert.equal(f.camera.projectionMatrix.elements[0], f.trackingProjection.elements[0] * 2)
  assert.equal(f.camera.projectionMatrix.elements[5], f.trackingProjection.elements[5])
  const identity = f.camera.projectionMatrix.clone().multiply(f.camera.projectionMatrixInverse)
  identity.elements.forEach((value, i) => assert.ok(Math.abs(value - (i % 5 === 0 ? 1 : 0)) < 1e-10))
  f.placement.update(16)
  assert.equal(f.camera.projectionMatrix.elements[0], f.trackingProjection.elements[0] * 2)
  // Resizing the viewport changes which video dimension is cropped.
  Object.assign(f.host.value!, { clientWidth: 800, clientHeight: 400 })
  f.placement.resize()
  assert.ok(f.resizeCount >= 2)
  f.placement.update(32)
  assert.equal(f.camera.projectionMatrix.elements[0], f.trackingProjection.elements[0])
  assert.equal(f.camera.projectionMatrix.elements[5], f.trackingProjection.elements[5] * 1.5)
  f.preview()
  assert.deepEqual(f.camera.projectionMatrix, f.previewProjection)
  assert.equal(f.stopCount, 1)
  assert.equal(f.placement.markerActive.value, false)
  assert.equal(f.placement.placementConfirmed.value, false)
})

test('stopping while libraries load aborts startup and ignores late camera callbacks', async () => {
  const f = fixture(true)
  const started = f.placement.start()
  assert.equal(f.placement.markerLoading.value, true)
  f.preview()
  assert.equal(f.signal?.aborted, true)
  f.ready()
  f.detect({ a: true, b: true })
  f.fail(new Error('late error'))
  f.finishMount()
  await started
  assert.equal(f.stopCount, 1)
  assert.equal(f.placement.markerLoading.value, false)
  assert.equal(f.placement.markerActive.value, false)
  assert.equal(f.placement.cameraReady.value, false)
  assert.deepEqual(f.placement.markerState.value, { a: false, b: false })
  assert.deepEqual(f.errors, [])
})

test('a camera error releases the tracker, reports the problem and allows retry', async (t) => {
  t.mock.method(console, 'error', () => {})
  const f = fixture()
  await f.placement.start()
  f.fail(new DOMException('Denied', 'NotAllowedError'))
  assert.equal(f.stopCount, 1)
  assert.equal(f.placement.markerActive.value, false)
  assert.match(f.errors[0], /カメラへのアクセスを許可/)
  await f.placement.start()
  assert.equal(f.placement.markerActive.value, true)
})
