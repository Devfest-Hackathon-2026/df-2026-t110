import type { components } from '../../../../api/schema.d.ts'
import { PicolangTraceReplay, type PicolangTrace } from '../../language/picolangTrace.ts'
import type { PicoProgram } from '../../language/picolang.ts'
import { replayFrames, type ReplayFrame } from './fieldHistory.ts'

export type PicolangTraceStep =
  { kind: 'field'; frame: ReplayFrame } | { kind: 'debug'; trace: PicolangTrace }

// Merge field animation and language state in the server's recorded order.
export function* picolangTraceSteps(history: unknown[], program: PicoProgram): Generator<PicolangTraceStep> {
  const frames = replayFrames(history)
  if (!frames.length) throw new Error('実行履歴に初期盤面がありません')
  const replay = new PicolangTraceReplay(program)
  let frameIndex = 0
  for (const item of history) {
    if (!item || typeof item !== 'object') continue
    const event = item as
      components['schemas']['field-event.schema'] | components['schemas']['picolang-event.schema']
    if (event.type === 'field') {
      if (event.state?.kind === 'initial' || (event.state?.kind === 'update' && frameIndex > 0))
        yield { kind: 'field', frame: frames[frameIndex++] }
    } else if (event.type === 'picolang') {
      const trace = replay.apply(event)
      if (trace) yield { kind: 'debug', trace }
    }
  }
}
