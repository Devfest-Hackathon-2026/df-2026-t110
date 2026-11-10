export type RoomMarkerId = 'a' | 'b'

export interface RoomMarkerState {
  a: boolean
  b: boolean
}

export interface RoomMarkerTracker {
  scene: HTMLElement
  markers: Record<RoomMarkerId, HTMLElement>
  stop(): void
}

export interface RoomMarkerCallbacks {
  onChange(state: RoomMarkerState): void
  onCameraReady?(): void
  onCameraError?(error: unknown): void
}

const MARKER_SIZE_METERS = 0.14
const PATTERN_URLS: Record<RoomMarkerId, string> = {
  a: '/markers/marker-hiro.patt',
  b: '/markers/marker-kanji.patt',
}

let libraryLoad: Promise<void> | undefined

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = src
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`ARライブラリを読み込めませんでした: ${src}`))
    document.head.append(script)
  })
}

function loadLibraries(): Promise<void> {
  libraryLoad ??= loadScript('/vendor/aframe-1.6.0.min.js')
    .then(() => loadScript('/vendor/aframe-ar-3.4.8.js'))
    .catch((error: unknown) => {
      libraryLoad = undefined
      throw error
    })
  return libraryLoad
}

/** Mounts the two printed room markers. The caller owns the camera UI and room placement. */
export async function mountRoomMarkerTracker(
  container: HTMLElement,
  callbacks: RoomMarkerCallbacks,
  signal?: AbortSignal,
): Promise<RoomMarkerTracker> {
  if (!window.isSecureContext || !navigator.mediaDevices?.getUserMedia) {
    throw new Error('カメラにはlocalhostまたはHTTPSが必要です')
  }

  await loadLibraries()
  signal?.throwIfAborted()

  const scene = document.createElement('a-scene')
  scene.setAttribute('embedded', '')
  scene.setAttribute('vr-mode-ui', 'enabled: false')
  scene.setAttribute('device-orientation-permission-ui', 'enabled: false')
  scene.setAttribute('renderer', 'antialias: true; alpha: true; precision: medium')
  scene.setAttribute('arjs', 'sourceType: webcam; debugUIEnabled: false')

  const state: RoomMarkerState = { a: false, b: false }
  const markers = {} as Record<RoomMarkerId, HTMLElement>
  const listeners: Array<() => void> = []
  const bodyStyle = {
    width: document.body.style.width,
    height: document.body.style.height,
    marginLeft: document.body.style.marginLeft,
    marginTop: document.body.style.marginTop,
  }

  for (const id of ['a', 'b'] as const) {
    const marker = document.createElement('a-marker')
    marker.setAttribute('type', 'pattern')
    marker.setAttribute('url', PATTERN_URLS[id])
    marker.setAttribute('size', String(MARKER_SIZE_METERS))
    marker.setAttribute('emitevents', 'true')
    const found = () => {
      state[id] = true
      callbacks.onChange({ ...state })
    }
    const lost = () => {
      state[id] = false
      callbacks.onChange({ ...state })
    }
    marker.addEventListener('markerFound', found)
    marker.addEventListener('markerLost', lost)
    listeners.push(() => {
      marker.removeEventListener('markerFound', found)
      marker.removeEventListener('markerLost', lost)
    })
    markers[id] = marker
    scene.append(marker)
  }

  const camera = document.createElement('a-entity')
  camera.setAttribute('camera', '')
  scene.append(camera)

  let cameraVideo: HTMLVideoElement | undefined
  const onCameraReady = (event: Event) => {
    const component = event instanceof CustomEvent ? event.detail?.component : undefined
    if (component instanceof HTMLVideoElement) {
      cameraVideo = component
      cameraVideo.classList.add('room-marker-video')
      container.prepend(cameraVideo)
    }
    callbacks.onCameraReady?.()
  }
  const onCameraError = (event: Event) => {
    const detail = event instanceof CustomEvent ? event.detail : undefined
    callbacks.onCameraError?.(detail?.error ?? event)
  }
  window.addEventListener('arjs-video-loaded', onCameraReady)
  window.addEventListener('camera-error', onCameraError)
  document.addEventListener('camera-error', onCameraError)
  listeners.push(() => {
    window.removeEventListener('arjs-video-loaded', onCameraReady)
    window.removeEventListener('camera-error', onCameraError)
    document.removeEventListener('camera-error', onCameraError)
  })

  document.body.classList.add('room-marker-tracking')
  container.append(scene)
  callbacks.onChange({ ...state })

  return {
    scene,
    markers,
    stop() {
      for (const removeListener of listeners) removeListener()
      const arjsSystem = (
        scene as HTMLElement & {
          systems?: { arjs?: { arToolkitSource?: { domElement?: HTMLVideoElement } } }
        }
      ).systems?.arjs
      const video = cameraVideo ?? arjsSystem?.arToolkitSource?.domElement
      if (video?.srcObject instanceof MediaStream) {
        for (const track of video.srcObject.getTracks()) track.stop()
        video.srcObject = null
      }
      cameraVideo?.remove()
      ;(scene as HTMLElement & { pause?: () => void }).pause?.()
      scene.remove()
      document.body.classList.remove('room-marker-tracking')
      Object.assign(document.body.style, bodyStyle)
    },
  }
}
