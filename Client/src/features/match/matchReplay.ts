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
}

export interface ReplayFrame {
  tick: number
  terrain: ReplayCell[][]
  robots: ReplayRobot[]
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
        actions?: Array<{
          kind: 'forward' | 'back' | 'rotate'
          robotId: string
          success?: boolean
          to?: { x: number; y: number }
          direction?: ReplayRobot['direction']
        }>
        removedRobotIds?: string[]
      }
    }

function isFieldEvent(value: unknown): value is FieldEvent {
  return value !== null && typeof value === 'object' && (value as { type?: unknown }).type === 'field'
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
      })
    } else if (event.state?.kind === 'update' && current) {
      const update = event.state
      const next: ReplayFrame = structuredClone(current)
      next.tick = event.tick
      for (const action of update.actions ?? []) {
        const robot = next.robots.find((item) => item.id === action.robotId)
        if (!robot) continue
        if ((action.kind === 'forward' || action.kind === 'back') && action.success && action.to)
          robot.position = action.to
        if (action.kind === 'rotate' && action.direction) robot.direction = action.direction
      }
      if (update.removedRobotIds?.length)
        next.robots = next.robots.filter((robot) => !update.removedRobotIds?.includes(robot.id))
      current = next
    } else continue
    if (current) frames.push(current)
  }
  return frames
}
