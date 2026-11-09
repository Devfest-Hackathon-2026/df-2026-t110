import { toRaw } from 'vue'
import type { ReplayAction, ReplayFrame } from './fieldHistory.ts'
import type { PicoValue } from '../../language/picolang.ts'
import type { AnimationEvent } from '../../../stage/robotAnimator.ts'
import type { StageLayout, StageObject } from '../../../stage/stage.ts'
import { speechForValue, type PicolangSpeech } from './picolangSpeech.ts'

export function cloneReplayLayout(layout: StageLayout): StageLayout {
  return structuredClone(toRaw(layout))
}

export function selectedReplayLayout(
  layouts: StageLayout[],
  fieldId: string,
): { index: number; stage: StageLayout } {
  const index = layouts.findIndex((layout) => layout.id === fieldId)
  if (index < 0) throw new Error(`選ばれたフィールド ${fieldId} が配置の候補にありません`)
  return { index, stage: cloneReplayLayout(layouts[index]) }
}

export function actionAnimation(action: ReplayAction, id: number): AnimationEvent | null {
  if (
    action.kind !== 'forward' &&
    action.kind !== 'back' &&
    action.kind !== 'rotate' &&
    action.kind !== 'drill' &&
    action.kind !== 'pick_up' &&
    action.kind !== 'release' &&
    action.kind !== 'wait'
  )
    return null
  return {
    id,
    type: action.kind,
    success: action.success,
    to: action.to,
    direction: action.rotation,
    ticks: action.ticks,
  }
}

export function actionSpeech(action: ReplayAction, id: number): PicolangSpeech | null {
  let value: PicoValue
  switch (action.kind) {
    case 'print_integer':
      if (typeof action.integer !== 'number') return null
      value = action.integer
      break
    case 'print_bool':
      if (typeof action.bool !== 'boolean') return null
      value = action.bool
      break
    case 'print_unit':
      value = null
      break
    case 'print_terrain':
      if (!action.terrain) return null
      value = action.terrain
      break
    case 'print_rotation':
      if (!action.rotation) return null
      value = action.rotation
      break
    case 'print_distance_target':
      if (!action.distanceTarget) return null
      value = action.distanceTarget
      break
    default:
      return null
  }
  return speechForValue(value, id)
}

export function applyReplayFrame(stage: StageLayout, frame: ReplayFrame): boolean {
  if (frame.terrain.length !== stage.height || frame.terrain.some((row) => row.length !== stage.width))
    throw new Error('実行履歴の盤面サイズが初期盤面と一致しません')
  let changed = false
  for (const cell of stage.cells) {
    const actual = frame.terrain[cell.y][cell.x]
    const objects: StageObject[] = [
      ...(actual.kind?.type === 'goal' ? [{ type: 'goal' as const }] : []),
      ...(actual.kind?.type === 'checkpoint'
        ? [{ type: 'checkpoint' as const, order: actual.kind.order }]
        : []),
      ...(actual.hasLuggage ? [{ type: 'luggage' as const }] : []),
    ]
    if (cell.terrain !== actual.type || JSON.stringify(cell.objects) !== JSON.stringify(objects))
      changed = true
    cell.terrain = actual.type
    cell.objects = objects
    cell.symbol = actual.type === 'rock' ? '#' : actual.type === 'sea' ? '_' : '.'
  }
  stage.rows = frame.terrain.map((row) =>
    row.map((cell) => (cell.type === 'rock' ? '#' : cell.type === 'sea' ? '_' : '.')).join(''),
  )
  return changed
}
