// Development-only history shaped like Schema/game_event/field-event.schema.yaml.
export const replayDemoHistory: unknown[] = [
  {
    type: 'field',
    tick: 0,
    state: {
      kind: 'initial',
      terrain: [
        [
          { type: 'land', kind: { type: 'plain' } },
          { type: 'land', kind: { type: 'plain' }, hasLuggage: true },
          { type: 'rock' },
          { type: 'land', kind: { type: 'goal' } },
          { type: 'land', kind: { type: 'plain' } },
        ],
        [
          { type: 'land', kind: { type: 'plain' } },
          { type: 'land', kind: { type: 'plain' } },
          { type: 'sea' },
          { type: 'land', kind: { type: 'plain' } },
          { type: 'land', kind: { type: 'plain' } },
        ],
      ],
      robots: [
        { id: 'robot-1', position: { x: 0, y: 0 }, direction: 'east', drillDurability: 2, teamId: 1 },
        { id: 'robot-2', position: { x: 4, y: 0 }, direction: 'west', drillDurability: 1, teamId: 2 },
      ],
    },
  },
  {
    type: 'field',
    tick: 1,
    state: {
      kind: 'update',
      actions: [
        { kind: 'pick_up', robotId: 'robot-1', success: true, target: { x: 1, y: 0 } },
        { kind: 'rotate', robotId: 'robot-2', rotation: 'right', direction: 'north' },
      ],
    },
  },
  {
    type: 'field',
    tick: 2,
    state: {
      kind: 'update',
      actions: [
        { kind: 'forward', robotId: 'robot-1', success: true, to: { x: 1, y: 0 } },
        { kind: 'print_bool', robotId: 'robot-2', bool: true },
      ],
    },
  },
  {
    type: 'field',
    tick: 3,
    state: {
      kind: 'update',
      actions: [
        { kind: 'drill', robotId: 'robot-1', success: true, target: { x: 2, y: 0 }, drillDurability: 1 },
      ],
    },
  },
  {
    type: 'field',
    tick: 4,
    state: {
      kind: 'update',
      actions: [{ kind: 'forward', robotId: 'robot-1', success: true, to: { x: 2, y: 0 } }],
    },
  },
  {
    type: 'field',
    tick: 5,
    state: { kind: 'update', actions: [{ kind: 'print_integer', robotId: 'robot-1', integer: 3 }] },
  },
  {
    type: 'field',
    tick: 6,
    state: { kind: 'update', actions: [{ kind: 'wait', robotId: 'robot-1', ticks: 2 }] },
  },
  {
    type: 'field',
    tick: 8,
    state: {
      kind: 'update',
      actions: [{ kind: 'release', robotId: 'robot-1', success: true, target: { x: 3, y: 0 } }],
      removedRobotIds: ['robot-2'],
    },
  },
]
