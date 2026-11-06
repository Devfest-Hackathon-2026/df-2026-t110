import * as THREE from 'three'
import { armAngleAt } from './armController.ts'
import { createEyeBeam } from './beamEffect.ts'
import { createTelekinesisEffect } from './telekinesisEffect.ts'

const TICK_MS = 1000
const DURATION = {
  move: TICK_MS,
  forward: TICK_MS,
  back: TICK_MS,
  rotate: TICK_MS,
  wait: TICK_MS,
  get: TICK_MS,
  pick_up: TICK_MS,
  put: TICK_MS,
  release: TICK_MS,
  drill: TICK_MS,
  clear: 1300,
} as const

export type ActionType = keyof typeof DURATION
export interface AnimationEvent {
  id: number
  type: ActionType
  to?: { x: number; y: number }
  direction?: 'left' | 'right'
  ticks?: number
  success?: boolean
}

interface ActiveAction {
  event: AnimationEvent
  startTime: number | null
  duration: number
  from: THREE.Vector3
  to: THREE.Vector3
  startYaw: number
  endYaw: number
}

const ease = (value: number): number => value * value * (3 - 2 * value)
const angleDifference = (from: number, to: number): number =>
  Math.atan2(Math.sin(to - from), Math.cos(to - from))

// The GLB has no clips. Animate the imported objects around runtime shoulder hinges.
export function createRobotAnimator(rig: THREE.Object3D, model: THREE.Object3D, pivots: THREE.Group[]) {
  const restPosition = model.position.clone()
  const restRotation = model.rotation.clone()
  const restRigY = rig.position.y
  let action: ActiveAction | null = null
  let effect: { update: (progress: number, time: number) => void; dispose: () => void } | null = null

  function stop(): void {
    effect?.dispose()
    effect = null
    action = null
    for (const pivot of pivots) pivot.rotation.x = 0
    rig.position.y = restRigY
    model.position.copy(restPosition)
    model.rotation.copy(restRotation)
  }

  function play(event: AnimationEvent | null, destination?: THREE.Vector3, targetYaw?: number): boolean {
    stop()
    if (!event || !(event.type in DURATION)) return false
    const ticks = Number.isFinite(event.ticks) ? Math.max(1, Math.floor(event.ticks ?? 1)) : 1
    action = {
      event,
      startTime: null,
      duration: event.type === 'wait' ? ticks * TICK_MS : DURATION[event.type],
      from: rig.position.clone(),
      to: destination?.clone() ?? rig.position.clone(),
      startYaw: rig.rotation.y,
      endYaw: targetYaw ?? rig.rotation.y,
    }
    if (event.success !== false) {
      if (event.type === 'drill') effect = createEyeBeam(rig, model)
      if (event.type === 'get' || event.type === 'pick_up') effect = createTelekinesisEffect(rig, false)
      if (event.type === 'put' || event.type === 'release') effect = createTelekinesisEffect(rig, true)
    }
    return true
  }

  function update(time: number): void {
    model.position.copy(restPosition)
    model.rotation.copy(restRotation)
    if (!action) {
      model.position.y += Math.sin(time * 0.002) * 0.012
      return
    }

    action.startTime ??= time
    const progress = Math.min(1, (time - action.startTime) / action.duration)
    const pulse = Math.sin(progress * Math.PI)
    const { event } = action

    if (event.type === 'move' || event.type === 'forward' || event.type === 'back') {
      if (action.from.distanceToSquared(action.to) < 0.0001) {
        model.position.z += (event.type === 'back' ? 0.045 : -0.045) * pulse
      } else {
        rig.position.lerpVectors(action.from, action.to, ease(progress))
        model.position.y += Math.sin(progress * Math.PI * 2) ** 2 * 0.024
      }
      model.rotation.x += (event.type === 'back' ? -0.06 : 0.06) * pulse
      rig.rotation.y = action.startYaw + angleDifference(action.startYaw, action.endYaw) * ease(progress)
    } else if (event.type === 'rotate') {
      rig.rotation.y = action.startYaw + (action.endYaw - action.startYaw) * ease(progress)
      model.rotation.z += (event.direction === 'left' ? -0.07 : 0.07) * pulse
    } else if (
      event.type === 'get' ||
      event.type === 'pick_up' ||
      event.type === 'put' ||
      event.type === 'release'
    ) {
      if (event.success !== false) {
        // +30° from the imported -120° pose reaches the requested -90° interaction pose.
        const angle = armAngleAt(progress * TICK_MS, Math.PI / 6)
        for (const pivot of pivots) pivot.rotation.x = angle
        model.rotation.x -= 0.045 * pulse
      }
    } else if (event.type === 'drill') {
      if (event.success !== false) {
        model.position.z += 0.04 * pulse
        model.rotation.x += 0.045 * pulse
      }
    } else if (event.type === 'clear') {
      rig.position.y = restRigY + pulse * 0.32
      rig.rotation.y = action.startYaw + progress * Math.PI * 2
      for (const pivot of pivots) pivot.rotation.x = (Math.PI / 6) * pulse
    } else {
      // Waiting keeps the same hovering rest pose.
      model.position.y += Math.sin(time * 0.002) * 0.012
    }

    effect?.update(progress, time)
    if (progress >= 1) {
      const finalPosition = action.to.clone()
      const finalYaw = action.endYaw
      stop()
      rig.position.copy(finalPosition)
      rig.rotation.y = finalYaw
    }
  }

  return { play, update, stop }
}
