import test from 'node:test'
import assert from 'node:assert/strict'
import { reactive } from 'vue'
import { normalizeStage } from '../src/features/stage/stage.ts'
import {
  applyReplayFrame,
  cloneReplayLayout,
  selectedReplayLayout,
  actionAnimation,
  actionSpeech,
} from '../src/features/picolang/editor/playback/fieldReplay.ts'
import { replayFrames } from '../src/features/match/matchReplay.ts'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import {
  receiveTutorialResult,
  saveTutorialProgram,
  startTutorialRun,
  TutorialRunApiError,
} from '../src/features/tutorial/tutorialRunApi.ts'

test('教材用プログラムの保存後、返されたIDとサーバー選択盤面で実行する', async () => {
  const originalFetch = globalThis.fetch
  const requests: Array<{ url: string; body: Record<string, unknown>; credentials: RequestCredentials }> = []
  globalThis.fetch = async (input, init) => {
    requests.push({
      url: String(input),
      body: JSON.parse(String(init?.body)) as Record<string, unknown>,
      credentials: init?.credentials as RequestCredentials,
    })
    const saved = requests.length === 1
    return new Response(
      JSON.stringify(saved ? { id: 'program-1' } : { runId: 'run-1', fieldId: 'variant-b' }),
      {
        status: saved ? 201 : 201,
        headers: { 'content-type': 'application/json' },
      },
    )
  }
  try {
    const signal = new AbortController().signal
    const source = newProgram()
    const programId = await saveTutorialProgram('lesson-1', '教材', source, signal)
    const run = await startTutorialRun('lesson-1', programId, signal)
    assert.deepEqual(run, { runId: 'run-1', fieldId: 'variant-b' })
    assert.equal(requests[0].url, '/api/program')
    assert.deepEqual(requests[0].body, { name: '教材', source, isBattle: false, tutorialID: 'lesson-1' })
    assert.equal(requests[1].url, '/api/tutorial/lesson-1/run')
    assert.deepEqual(requests[1].body, { programId: 'program-1' })
    assert.deepEqual(
      requests.map((request) => request.credentials),
      ['include', 'include'],
    )
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('認証されていない保存は401として伝える', async () => {
  const originalFetch = globalThis.fetch
  globalThis.fetch = async () => new Response('', { status: 401 })
  try {
    await assert.rejects(
      saveTutorialProgram('lesson-1', '教材', newProgram(), new AbortController().signal),
      (error: unknown) => error instanceof TutorialRunApiError && error.status === 401,
    )
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('SSEのresultを一度受け取って閉じ、リセット時には購読を中断する', async () => {
  const originalEventSource = globalThis.EventSource
  class FakeSource {
    static last: FakeSource
    closed = false
    handlers = new Map<string, (event: MessageEvent) => void>()
    onerror: (() => void) | null = null
    url: string
    options: EventSourceInit
    constructor(url: string, options: EventSourceInit) {
      this.url = url
      this.options = options
      FakeSource.last = this
    }
    addEventListener(name: string, handler: EventListenerOrEventListenerObject) {
      this.handlers.set(name, handler as (event: MessageEvent) => void)
    }
    close() {
      this.closed = true
    }
    emit(result: unknown) {
      this.handlers.get('result')?.({ data: JSON.stringify(result) } as MessageEvent)
    }
  }
  globalThis.EventSource = FakeSource as unknown as typeof EventSource
  try {
    const signal = new AbortController()
    const pending = receiveTutorialResult('run-1', signal.signal)
    assert.equal(FakeSource.last.url, '/api/run/run-1/events')
    assert.equal(FakeSource.last.options.withCredentials, true)
    const result = { history: [], summary: { cleared: true, tick: 2, reason: 'condition_met' } }
    FakeSource.last.emit(result)
    assert.deepEqual(await pending, result)
    assert.equal(FakeSource.last.closed, true)
    const cancelled = new AbortController()
    const interrupted = receiveTutorialResult('run-2', cancelled.signal)
    cancelled.abort()
    await assert.rejects(interrupted, { name: 'AbortError' })
    assert.equal(FakeSource.last.closed, true)
  } finally {
    globalThis.EventSource = originalEventSource
  }
})

test('サーバーが選んだ候補を使い、履歴の破壊と発話を教材シーンへ反映する', () => {
  const first = normalizeStage({ id: 'variant-a', map: ['S.G'] })
  const second = normalizeStage({ id: 'variant-b', map: ['S#G'] })
  const layouts = reactive([first, second])
  const { index, stage } = selectedReplayLayout(layouts, 'variant-b')
  assert.equal(index, 1)
  assert.equal(stage.cells[1].terrain, 'rock')
  assert.equal(cloneReplayLayout(layouts[0]).id, 'variant-a')
  const frames = replayFrames([
    {
      type: 'field',
      tick: 0,
      state: {
        kind: 'initial',
        terrain: [
          [
            { type: 'land', kind: { type: 'plain' } },
            { type: 'rock' },
            { type: 'land', kind: { type: 'goal' } },
          ],
        ],
        robots: [{ id: 'robot-1', position: { x: 0, y: 0 }, direction: 'east' }],
      },
    },
    {
      type: 'field',
      tick: 1,
      state: {
        kind: 'update',
        actions: [{ kind: 'drill', robotId: 'robot-1', success: true, target: { x: 1, y: 0 } }],
      },
    },
  ])
  assert.equal(applyReplayFrame(stage, frames[0]), false)
  assert.deepEqual(actionAnimation(frames[1].actions[0], 1), {
    id: 1,
    type: 'drill',
    success: true,
    to: undefined,
    direction: undefined,
    ticks: undefined,
  })
  assert.equal(applyReplayFrame(stage, frames[1]), true)
  assert.equal(stage.cells[1].terrain, 'land')
  assert.equal(stage.rows[0], '...')
  assert.equal(actionSpeech({ kind: 'print_integer', robotId: 'robot-1', integer: 0 }, 2)?.label, '0')
  assert.equal(
    actionSpeech({ kind: 'print_terrain', robotId: 'robot-1', terrain: { type: 'sea' } }, 3)?.terrain?.type,
    'sea',
  )
})
