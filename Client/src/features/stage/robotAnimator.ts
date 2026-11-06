import * as THREE from 'three'
import { armAngleAt } from './armController.ts'
import { createEyeBeam } from './beamEffect.ts'

const DURATION = { move: 750, drill: 650, get: 800, put: 800, clear: 1300 }
type ActionType = keyof typeof DURATION
export interface AnimationEvent {
  id: number
  type: ActionType
  to?: { x: number; y: number }
}
const ease = (value: number): number => value * value * (3 - 2 * value)

// GLBにアニメーションクリップはないので、Object名から作った肩軸に差分回転を加える。
export function createRobotAnimator(rig: THREE.Object3D, model: THREE.Object3D, pivots: THREE.Group[]) {
  const restingModelY = model.position.y
  let action: {
    type: ActionType
    startTime: number | null
    from: THREE.Vector3
    to: THREE.Vector3
    baseYaw: number
  } | null = null
  let removeBeam: (() => void) | null = null

  function stop() {
    removeBeam?.()
    removeBeam = null
    action = null
    for (const pivot of pivots) pivot.rotation.x = 0
    rig.position.y = 0.07
    model.position.y = restingModelY
  }

  function play(event: AnimationEvent | null, destination?: THREE.Vector3): boolean {
    stop()
    if (!event || !(event.type in DURATION)) return false
    action = {
      type: event.type,
      startTime: null,
      from: rig.position.clone(),
      to: destination?.clone() ?? rig.position.clone(),
      baseYaw: rig.rotation.y,
    }
    if (event.type === 'drill') removeBeam = createEyeBeam(rig, model, THREE)
    return true
  }

  function update(time: number): void {
    if (!action) {
      model.position.y = restingModelY + Math.sin(time * 0.002) * 0.012
      return
    }
    action.startTime ??= time
    const progress = Math.min(1, (time - action.startTime) / DURATION[action.type])
    if (action.type === 'move') rig.position.lerpVectors(action.from, action.to, ease(progress))
    if (action.type === 'get' || action.type === 'put') {
      const angle = armAngleAt(time - action.startTime, Math.PI / 6)
      for (const pivot of pivots) pivot.rotation.x = angle
    }
    if (action.type === 'clear') {
      rig.position.y = 0.07 + Math.sin(progress * Math.PI) * 0.36
      rig.rotation.y = action.baseYaw + progress * Math.PI * 2
    }
    if (progress === 1) {
      const finalPosition = action.to.clone()
      const finalYaw = action.baseYaw
      stop()
      rig.position.copy(finalPosition)
      rig.rotation.y = finalYaw
    }
  }

  return { play, update, stop }
}
