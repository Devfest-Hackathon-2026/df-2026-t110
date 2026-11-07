import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { parseMatchResult } from '../src/features/match/matchApi.ts'
import {
  executionReasonText,
  isExecutionFailure,
  robotExecutionResults,
} from '../src/features/match/matchExecutionResults.ts'
import { replayFrames } from '../src/features/match/matchReplay.ts'
import { useMatchStore } from '../src/features/match/useMatchStore.ts'

const history: unknown[] = JSON.parse(
  readFileSync(new URL('../../Schema/game_event/examples/execution-results.json', import.meta.url), 'utf8'),
)

test('schema history exposes successful and failed robot executions with their source', () => {
  const results = robotExecutionResults(history)
  assert.deepEqual(
    results.map((result) => result.reason),
    [
      'completed',
      'interrupted',
      'runtime_error',
      'division_by_zero',
      'step_limit',
      'memory_limit',
      'call_depth_limit',
    ],
  )
  assert.deepEqual(results.map(isExecutionFailure), [false, false, true, true, true, true, true])
  assert.equal(results[3].interpreterId, 'robot-4')
  assert.equal(results[3].hintId, '00000000-0000-4000-8000-000000000004')
  assert.match(executionReasonText(results[0]), /正常に終了/)
  assert.match(executionReasonText(results[1]), /打ち切られ/)
  assert.match(executionReasonText(results[2]), /nonempty vector/)
  assert.match(executionReasonText(results[3]), /0で割る/)
  assert.match(executionReasonText(results[4]), /ステップ数/)
  assert.match(executionReasonText(results[5]), /メモリ使用量/)
  assert.match(executionReasonText(results[6]), /呼び出しの深さ/)
  const frames = replayFrames(history)
  assert.equal(frames.length, 2)
  assert.equal(frames[0].robots.length, 7)
  assert.deepEqual(frames[1].robots, [])
})

test('match result store retains execution endings after robots are removed and clears them for a new field', () => {
  setActivePinia(createPinia())
  const store = useMatchStore()
  assert.equal(store.executionResults.length, 0)
  store.result = parseMatchResult({ history, summary: { score: { player: 1, opponent: 0 } } })
  assert.equal(store.executionResults.length, 7)
  assert.equal(store.executionResults[0].reason, 'completed')
  store.setField('another-field')
  assert.deepEqual(store.executionResults, [])
})

test('unrelated and malformed history does not create execution results', () => {
  const event = {
    kind: 'execution_finished',
    hintId: '00000000-0000-4000-8000-000000000001',
    reason: 'completed',
    message: 'program completed',
  }
  assert.deepEqual(
    robotExecutionResults([
      null,
      [],
      'invalid',
      { type: 'field', state: {} },
      { type: 'picolang', interpreterId: 'a', event: { kind: 'current_node' } },
      { type: 'picolang', interpreterId: '', event },
      { type: 'picolang', interpreterId: 'a', event: { ...event, hintId: null } },
      { type: 'picolang', interpreterId: 'a', event: { ...event, reason: 1 } },
      { type: 'picolang', interpreterId: 'a', event: { ...event, message: '' } },
      { type: 'picolang', interpreterId: 'a', event: null },
    ]),
    [],
  )
})

test('unknown future reasons retain the server explanation', () => {
  const [result] = robotExecutionResults([
    {
      type: 'picolang',
      interpreterId: 'a',
      event: {
        kind: 'execution_finished',
        hintId: '00000000-0000-4000-8000-000000000001',
        reason: 'future_reason',
        message: 'Additional explanation',
      },
    },
  ])
  assert.match(executionReasonText(result), /Additional explanation/)
})
