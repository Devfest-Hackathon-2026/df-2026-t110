import type { PicoValue } from '../picolang/language/executePicolang.ts'
import { PICO_DISTANCE_TARGETS, type PicoTerrain } from '../picolang/language/picolang.ts'
import type { StageCell, StageLayout, StageObject } from '../stage/stage.ts'
import type { AnimationEvent } from '../stage/robotAnimator.ts'
import { speechForValue, type TutorialSpeech } from './tutorialSpeech.ts'

type Direction = StageLayout['robot']['direction']
export type WorldAnimation = Omit<AnimationEvent, 'id'>
export interface WorldResult {
  value: PicoValue
  animation?: WorldAnimation
  changed?: boolean
  message?: string
  speech?: TutorialSpeech
}
const directions: Direction[] = ['north', 'east', 'south', 'west']
const vectors: Record<Direction, { x: number; y: number }> = {
  north: { x: 0, y: -1 },
  east: { x: 1, y: 0 },
  south: { x: 0, y: 1 },
  west: { x: -1, y: 0 },
}

export function createTutorialWorld(stage: StageLayout) {
  let speechId = 0
  const position = { ...stage.robot.position }
  let direction = stage.robot.direction
  let drillRemaining = stage.drillLimit ?? 99
  let held: StageObject | null = null
  const cellAt = (x: number, y: number): StageCell | undefined =>
    stage.cells.find((cell) => cell.x === x && cell.y === y)
  const front = (): StageCell | undefined =>
    cellAt(position.x + vectors[direction].x, position.y + vectors[direction].y)
  const terrain = (cell?: StageCell): PicoTerrain => {
    const type = cell?.terrain ?? 'sea'
    if (type !== 'land') return { type }
    const goal = cell?.objects.find((object) => object.type === 'goal')
    const checkpoint = cell?.objects.find((object) => object.type === 'checkpoint')
    return {
      type,
      kind: goal
        ? { type: 'goal' }
        : checkpoint
          ? { type: 'checkpoint', order: checkpoint.order ?? 0 }
          : { type: 'plain' },
      ...(cell?.objects.some((object) => object.type === 'luggage') ? { hasLuggage: true } : {}),
    }
  }
  const updateRow = (cell: StageCell): void => {
    const row = [...stage.rows[cell.y]]
    row[cell.x] = cell.symbol
    stage.rows[cell.y] = row.join('')
  }
  const face = (rotation: PicoValue): void => {
    if (rotation !== 'left' && rotation !== 'right') throw new Error('回転方向は left または right です')
    direction = directions[(directions.indexOf(direction) + (rotation === 'left' ? 3 : 1)) % 4]
  }
  function call(name: string, args: PicoValue[]): WorldResult {
    const target = front()
    const here = cellAt(position.x, position.y)
    if (name === 'search_front') return { value: terrain(target) }
    if (name === 'search_now') return { value: terrain(here) }
    if (name === 'drill_durability') return { value: drillRemaining }
    if (name === 'distance') {
      const delta = vectors[direction]
      for (let n = 1; n <= Math.max(stage.width, stage.height); n++) {
        const cell = cellAt(position.x + delta.x * n, position.y + delta.y * n)
        if (!cell) break
        if (cell.objects.some((object) => object.type === 'goal')) return { value: n }
      }
      return { value: -1 }
    }
    if (name.startsWith('is_')) {
      const value = args[0]
      if (!value || typeof value !== 'object' || Array.isArray(value) || !('type' in value))
        throw new Error('terrain が必要です')
      const kind = name.slice(3)
      return {
        value:
          kind === 'land' || kind === 'sea' || kind === 'rock'
            ? value.type === kind
            : kind === 'laggage' || kind === 'luggage'
              ? Boolean(value.hasLuggage)
              : value.kind?.type === kind,
      }
    }
    if (name === 'print_integer' || name === 'print_bool')
      return { value: null, speech: speechForValue(args[0], ++speechId) }
    if (name === 'print_unit' && args[0] === null)
      return { value: null, speech: speechForValue(args[0], ++speechId) }
    if (name === 'print_rotation' && (args[0] === 'left' || args[0] === 'right'))
      return { value: null, speech: speechForValue(args[0], ++speechId) }
    if (
      name === 'print_distance_target' &&
      typeof args[0] === 'string' &&
      PICO_DISTANCE_TARGETS.some((target) => target === args[0])
    )
      return { value: null, speech: speechForValue(args[0], ++speechId) }
    if (name === 'print_terrain' && args[0] && typeof args[0] === 'object' && !Array.isArray(args[0]))
      return { value: null, speech: speechForValue(args[0], ++speechId) }
    if (name === 'rotate') {
      face(args[0])
      return { value: null, animation: { type: 'rotate', direction: args[0] as 'left' | 'right' } }
    }
    if (name === 'wait') {
      if (typeof args[0] !== 'number' || !Number.isInteger(args[0]) || args[0] < 0)
        throw new Error('待機時間は0以上の整数にしてください')
      return { value: null, animation: { type: 'wait', ticks: args[0] } }
    }
    if (name === 'forward' || name === 'back') {
      const sign = name === 'forward' ? 1 : -1
      const destination = cellAt(
        position.x + vectors[direction].x * sign,
        position.y + vectors[direction].y * sign,
      )
      const success = destination?.terrain === 'land'
      if (success) {
        position.x = destination.x
        position.y = destination.y
      }
      return { value: success, animation: { type: name, success } }
    }
    if (name === 'drill') {
      const success = Boolean(target?.terrain === 'rock' && drillRemaining > 0)
      if (success && target) {
        target.terrain = 'land'
        target.symbol = '.'
        stage.rockCount--
        drillRemaining--
        updateRow(target)
      }
      return { value: success, animation: { type: 'drill', success }, changed: success }
    }
    if (name === 'pick_up') {
      const index = target?.objects.findIndex((object) => object.type === 'luggage') ?? -1
      const success = !held && index >= 0
      if (success && target) {
        held = target.objects.splice(index, 1)[0]
        target.symbol = '.'
        updateRow(target)
      }
      return { value: success, animation: { type: 'pick_up', success }, changed: success }
    }
    if (name === 'release') {
      const success = Boolean(
        held &&
        target &&
        target.terrain !== 'rock' &&
        !target.objects.some((object) => object.type === 'luggage'),
      )
      if (success && target) {
        if (target.terrain === 'land') {
          target.objects.push(held!)
          target.symbol = 'O'
          updateRow(target)
        }
        held = null
      }
      return { value: success, animation: { type: 'release', success }, changed: success }
    }
    throw new Error(`未対応の関数: ${name}`)
  }
  return {
    call,
    get position() {
      return { ...position }
    },
    get direction() {
      return direction
    },
    get held() {
      return held
    },
    get reachedGoal() {
      return Boolean(cellAt(position.x, position.y)?.objects.some((object) => object.type === 'goal'))
    },
  }
}
