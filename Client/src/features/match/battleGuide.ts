// Keep these guide values and the example in sync with Server/infra/matchrun/layouts.json.
// The example is battle-1-v2-seed-165-layout-01: # = rock, O = luggage, 1–3 = robot slots.
export const battleGuide = {
  robotCount: 3,
  turns: 160,
  drillLimit: 2,
  luggagePoints: 1,
  discardPenalty: 1,
} as const

export const battleFieldExample = [
  '....##..##....',
  '...O......O...',
  '..............',
  '....#.OO.#....',
  '....#....#....',
  '1##..#..#..##1',
  '2....O..O....2',
  '3..#......#..3',
  '.O##......##O.',
  '.#..O....O..#.',
  '......##......',
] as const
