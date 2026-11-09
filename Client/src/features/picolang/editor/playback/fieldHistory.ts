import { formatPicoValue } from '../../language/formatPicoValue.ts'
import type { PicoTerrain, PicoDistanceTarget } from '../../language/picolang.ts'

export interface ReplayCell {
  type: 'land' | 'rock' | 'sea'
  kind?: { type: 'plain' | 'goal' | 'checkpoint'; order?: number }
  hasLuggage?: boolean
  territoryOwner?: number
}

export interface ReplayRobot {
  id: string
  position: { x: number; y: number }
  direction: 'north' | 'south' | 'west' | 'east'
  carryingLuggage?: boolean
  drillDurability?: number
  teamId?: number
  speech?: string
}

export interface ReplayAction {
  kind:
    | 'forward'
    | 'back'
    | 'rotate'
    | 'wait'
    | 'drill'
    | 'pick_up'
    | 'release'
    | 'print_integer'
    | 'print_bool'
    | 'print_unit'
    | 'print_terrain'
    | 'print_rotation'
    | 'print_distance_target'
  robotId: string
  success?: boolean
  to?: { x: number; y: number }
  rotation?: 'left' | 'right'
  direction?: ReplayRobot['direction']
  target?: { x: number; y: number }
  discarded?: boolean
  drillDurability?: number
  ticks?: number
  integer?: number
  bool?: boolean
  terrain?: PicoTerrain
  distanceTarget?: PicoDistanceTarget
}

export interface ReplayFrame {
  tick: number
  terrain: ReplayCell[][]
  robots: ReplayRobot[]
  actions: ReplayAction[]
  removedRobotIds: string[]
}

export function replayPrintMessage(action: ReplayAction): string | undefined {
  switch (action.kind) {
    case 'print_integer':
      return typeof action.integer === 'number' ? formatPicoValue(action.integer) : undefined
    case 'print_bool':
      return typeof action.bool === 'boolean' ? formatPicoValue(action.bool) : undefined
    case 'print_unit':
      return formatPicoValue(null)
    case 'print_terrain':
      return action.terrain ? formatPicoValue(action.terrain) : undefined
    case 'print_rotation':
      return action.rotation ? formatPicoValue(action.rotation) : undefined
    case 'print_distance_target':
      return action.distanceTarget ? formatPicoValue(action.distanceTarget) : undefined
  }
}

type FieldEvent =
  | {
      type: 'field'
      tick: number
      state: { kind: 'initial'; terrain: ReplayCell[][]; robots: ReplayRobot[] }
    }
  | {
      type: 'field'
      tick: number
      state: {
        kind: 'update'
        actions?: ReplayAction[]
        removedRobotIds?: string[]
      }
    }

function isFieldEvent(value: unknown): value is FieldEvent {
  return value !== null && typeof value === 'object' && (value as { type?: unknown }).type === 'field'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

// Accept a MatchResult, a field-event array, or BoardOutput objects from Go.
export function parseBoardEventHistory(json: string): unknown[] {
  const source: unknown = JSON.parse(json)
  const history: unknown[] = []

  function collect(value: unknown): void {
    if (Array.isArray(value)) {
      for (const item of value) collect(item)
    } else if (isFieldEvent(value)) {
      history.push(value)
    } else if (isRecord(value)) {
      const events = value.history ?? value.events ?? value.Events
      if (Array.isArray(events)) collect(events)
    }
  }

  collect(source)
  if (!history.some((event) => isFieldEvent(event) && event.state?.kind === 'initial'))
    throw new Error('初期盤面のfieldイベントがありません。Board.Start()のイベントも含めてください。')
  if (replayFrames(history).length < 1) throw new Error('再生できるfieldイベントがありません。')
  return history
}

export function replayFrames(history: unknown[]): ReplayFrame[] {
  const frames: ReplayFrame[] = []
  let current: ReplayFrame | null = null
  for (const event of history) {
    if (!isFieldEvent(event)) continue
    if (event.state?.kind === 'initial') {
      if (!Array.isArray(event.state.terrain) || !Array.isArray(event.state.robots))
        throw new Error('盤面の初期状態が不正です')
      current = structuredClone({
        tick: event.tick,
        terrain: event.state.terrain,
        robots: event.state.robots,
        actions: [],
        removedRobotIds: [],
      })
    } else if (event.state?.kind === 'update' && current) {
      const update = event.state
      const next: ReplayFrame = structuredClone(current)
      next.tick = event.tick
      next.actions = structuredClone(update.actions ?? [])
      next.removedRobotIds = [...(update.removedRobotIds ?? [])]
      for (const robot of next.robots) robot.speech = undefined
      for (const action of update.actions ?? []) {
        const robot = next.robots.find((item) => item.id === action.robotId)
        if (!robot) continue
        if ((action.kind === 'forward' || action.kind === 'back') && action.success && action.to)
          robot.position = action.to
        if (action.kind === 'rotate' && action.direction) robot.direction = action.direction
        if (action.kind === 'drill') {
          if (typeof action.drillDurability === 'number') robot.drillDurability = action.drillDurability
          if (action.success && action.target) {
            const cell = next.terrain[action.target.y]?.[action.target.x]
            if (cell?.type === 'rock')
              next.terrain[action.target.y][action.target.x] = { type: 'land', kind: { type: 'plain' } }
          }
        }
        const message = replayPrintMessage(action)
        if (message !== undefined) robot.speech = message
        if (action.kind === 'release' && action.success && action.discarded) robot.carryingLuggage = false
        if ((action.kind === 'pick_up' || action.kind === 'release') && action.success && action.target) {
          const { x, y } = action.target
          const cell = next.terrain[y]?.[x]
          if (cell?.type === 'land') {
            cell.hasLuggage = action.kind === 'release'
            robot.carryingLuggage = action.kind === 'pick_up'
          }
        }
      }
      if (update.removedRobotIds?.length)
        next.robots = next.robots.filter((robot) => !update.removedRobotIds?.includes(robot.id))
      current = next
    } else continue
    if (current) frames.push(current)
  }
  return frames
}
import type { StageLayout } from '../../../stage/stage.ts'

export function replayFrameToStage(frame: ReplayFrame): StageLayout {
  const cells = frame.terrain.flatMap((row, y) =>
    row.map((cell, x) => ({
      x,
      y,
      symbol: cell.type === 'rock' ? '#' : cell.type === 'sea' ? '_' : '.',
      terrain: cell.type,
      objects: [
        ...(cell.kind?.type === 'goal'
          ? [{ type: 'goal' as const }]
          : cell.kind?.type === 'checkpoint'
            ? [{ type: 'checkpoint' as const, order: cell.kind.order }]
            : []),
        ...(cell.hasLuggage ? [{ type: 'luggage' as const }] : []),
      ],
      start: false,
    })),
  )
  const first = frame.robots[0]
  const start = cells.find((cell) => cell.x === first?.position.x && cell.y === first?.position.y)
  if (start) start.start = true
  const width = frame.terrain[0]?.length ?? 0
  return {
    id: `replay-${frame.tick}`,
    title: `ターン ${frame.tick}`,
    rows: frame.terrain.map((row) =>
      row.map((cell) => (cell.type === 'rock' ? '#' : cell.type === 'sea' ? '_' : '.')).join(''),
    ),
    width,
    height: frame.terrain.length,
    rockCount: cells.filter((cell) => cell.terrain === 'rock').length,
    cells,
    label: '',
    robot: { position: first?.position ?? { x: 0, y: 0 }, direction: first?.direction ?? 'east' },
  }
}
