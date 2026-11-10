import { ref, type Ref } from 'vue'
import * as THREE from 'three'
import { MarkerPoseStabilizer } from '../stage/markerPoseStabilizer.ts'
import {
  mountRoomMarkerTracker,
  type RoomMarkerId,
  type RoomMarkerState,
  type RoomMarkerTracker,
} from './roomMarkerTracking.ts'

interface PlacementOptions {
  host: Ref<HTMLElement | null>
  getBoard(): THREE.Group | null | undefined
  getCamera(): THREE.PerspectiveCamera | null | undefined
  isCamera(): boolean
  onError(message: string): void
  mountTracker?: typeof mountRoomMarkerTracker
}

function cameraErrorMessage(error: unknown): string {
  const name = error instanceof Error ? error.name : ''
  if (name === 'NotAllowedError') return 'カメラへのアクセスを許可して、もう一度お試しください。'
  if (name === 'NotFoundError') return 'カメラが見つかりません。接続を確認してください。'
  return 'カメラを起動できませんでした。接続とブラウザのカメラ設定を確認してください。'
}

/** Shared marker placement, camera projection and lifecycle for tutorial and battle scenes. */
export function useRoomMarkerPlacement(options: PlacementOptions) {
  const markerState = ref<RoomMarkerState>({ a: false, b: false })
  const markerActive = ref(false)
  const markerLoading = ref(false)
  const cameraReady = ref(false)
  const placementConfirmed = ref(false)
  const trackingPoseAvailable = ref(false)
  const calibrated = ref(false)
  let tracker: RoomMarkerTracker | null = null
  let generation = 0
  let controller: AbortController | null = null
  let anchorMarker: RoomMarkerId | null = null
  let alternateToAnchor: THREE.Matrix4 | null = null
  const markerPose = new MarkerPoseStabilizer()
  let lastPoseTime: number | null = null

  function stop(): void {
    generation++
    controller?.abort()
    controller = null
    tracker?.stop()
    tracker = null
    markerState.value = { a: false, b: false }
    markerActive.value = false
    markerLoading.value = false
    cameraReady.value = false
    placementConfirmed.value = false
    trackingPoseAvailable.value = false
    calibrated.value = false
    anchorMarker = null
    alternateToAnchor = null
    markerPose.reset()
    lastPoseTime = null
    const board = options.getBoard()
    if (board) board.visible = !options.isCamera()
    options.getCamera()?.updateProjectionMatrix()
  }

  function resize(): void {
    const scene = tracker?.scene as
      (HTMLElement & { systems?: { arjs?: { _arSession?: { onResize(): void } } } }) | undefined
    scene?.systems?.arjs?._arSession?.onResize()
  }

  async function start(): Promise<void> {
    stop()
    if (!options.isCamera() || !options.host.value) return
    markerLoading.value = true
    const board = options.getBoard()
    if (board) board.visible = false
    const currentGeneration = generation
    controller = new AbortController()
    const isCurrent = () => currentGeneration === generation && options.isCamera()
    const fail = (error: unknown) => {
      if (!isCurrent()) return
      stop()
      console.error('マーカー追跡を開始できませんでした', error)
      options.onError(cameraErrorMessage(error))
    }
    try {
      const mounted = await (options.mountTracker ?? mountRoomMarkerTracker)(
        options.host.value,
        {
          onChange: (state) => {
            if (isCurrent()) markerState.value = state
          },
          onCameraReady: () => {
            if (isCurrent()) cameraReady.value = true
          },
          onCameraError: fail,
        },
        controller.signal,
      )
      if (!isCurrent()) {
        mounted.stop()
        return
      }
      tracker = mounted
      markerLoading.value = false
      markerActive.value = true
      Object.assign(tracker.scene.style, {
        position: 'absolute',
        inset: '0',
        width: '100%',
        height: '100%',
        opacity: '0',
        pointerEvents: 'none',
      })
      tracker.scene.addEventListener('renderstart', resize, { once: true })
      resize()
    } catch (error) {
      fail(error)
    }
  }

  function detectedPose(id: RoomMarkerId): THREE.Matrix4 | null {
    if (!tracker || !markerState.value[id]) return null
    const marker = tracker.markers[id] as HTMLElement & {
      object3D?: { matrixWorld: { elements: number[] }; visible: boolean }
    }
    return marker.object3D?.visible
      ? new THREE.Matrix4().fromArray(marker.object3D.matrixWorld.elements)
      : null
  }

  function confirmPlacement(): void {
    if (!markerActive.value || (!markerState.value.a && !markerState.value.b)) return
    anchorMarker = markerState.value.a ? 'a' : 'b'
    alternateToAnchor = null
    calibrated.value = false
    markerPose.reset()
    lastPoseTime = null
    placementConfirmed.value = true
  }

  function retainBoardPose(previous: THREE.Group | null | undefined): boolean {
    const board = options.getBoard()
    if (!previous || !board || !options.isCamera() || !tracker || !placementConfirmed.value) return false
    board.position.copy(previous.position)
    board.quaternion.copy(previous.quaternion)
    board.scale.copy(previous.scale)
    board.visible = previous.visible
    return true
  }

  function update(time: number): void {
    const board = options.getBoard()
    const camera = options.getCamera()
    if (!tracker || !board || !camera) return
    const scene = tracker.scene as HTMLElement & {
      camera?: { projectionMatrix: { elements: number[] } }
    }
    if (scene.camera) {
      camera.projectionMatrix.fromArray(scene.camera.projectionMatrix.elements)
      const video = options.host.value?.querySelector<HTMLVideoElement>('.room-marker-video')
      const width = options.host.value?.clientWidth ?? 0
      const height = options.host.value?.clientHeight ?? 0
      if (video?.videoWidth && video.videoHeight && width && height) {
        const cover = Math.max(width / video.videoWidth, height / video.videoHeight)
        const scaleX = (video.videoWidth * cover) / width
        const scaleY = (video.videoHeight * cover) / height
        for (const index of [0, 4, 8, 12]) camera.projectionMatrix.elements[index] *= scaleX
        for (const index of [1, 5, 9, 13]) camera.projectionMatrix.elements[index] *= scaleY
      }
      camera.projectionMatrixInverse.copy(camera.projectionMatrix).invert()
    }
    if (!anchorMarker || !placementConfirmed.value) {
      board.visible = false
      trackingPoseAvailable.value = false
      lastPoseTime = null
      return
    }
    const alternateId = anchorMarker === 'a' ? 'b' : 'a'
    const anchorPose = detectedPose(anchorMarker)
    const alternatePose = detectedPose(alternateId)
    if (anchorPose && alternatePose) {
      alternateToAnchor = alternatePose.clone().invert().multiply(anchorPose)
      calibrated.value = true
    }
    const pose =
      anchorPose ?? (alternatePose && alternateToAnchor ? alternatePose.multiply(alternateToAnchor) : null)
    if (!pose) {
      if (lastPoseTime !== null && time - lastPoseTime < 300) return
      trackingPoseAvailable.value = false
      board.visible = false
      markerPose.reset()
      lastPoseTime = null
      return
    }
    lastPoseTime = time
    trackingPoseAvailable.value = true
    board.visible = true
    pose.decompose(board.position, board.quaternion, board.scale)
    markerPose.update(pose, time)
    board.position.copy(markerPose.position)
    board.quaternion.copy(markerPose.quaternion)
  }

  return {
    markerState,
    markerActive,
    markerLoading,
    cameraReady,
    placementConfirmed,
    trackingPoseAvailable,
    calibrated,
    confirmPlacement,
    retainBoardPose,
    resize,
    start,
    stop,
    update,
  }
}
