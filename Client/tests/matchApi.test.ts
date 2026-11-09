import test from 'node:test'
import assert from 'node:assert/strict'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import {
  createMatch,
  createProgram,
  getBattlePrograms,
  parseBattleProgramIds,
  setBattlePrograms,
  subscribeMatchResult,
} from '../src/features/match/matchApi.ts'

test('3台の対戦用プログラムを登録し、同じ順序で対戦を作成する', async () => {
  const originalFetch = globalThis.fetch
  const source = newProgram()
  const ids: [string, string, string] = ['first', 'second', 'first']
  const requests: Array<{
    path: string
    method: string
    body: unknown
    credentials: RequestCredentials | undefined
  }> = []
  let getCount = 0
  globalThis.fetch = async (input, init) => {
    const path = String(input)
    requests.push({
      path,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
      credentials: init?.credentials,
    })
    if (path.endsWith('/battle/program') && !init?.method) {
      getCount++
      return getCount === 1 ? new Response(null, { status: 404 }) : Response.json(ids)
    }
    if (path.endsWith('/battle/program')) return new Response(null, { status: 204 })
    if (path.endsWith('/program'))
      return Response.json({ id: 'first', name: '前進', source, isBattle: true }, { status: 201 })
    return Response.json(
      {
        id: 'match-1',
        runId: 'run-1',
        fieldId: 'battle-1',
        selectedLayoutId: 'layout-1',
        opponentName: '相手',
        status: 'finished',
      },
      { status: 201 },
    )
  }
  try {
    assert.equal(await getBattlePrograms(), null)
    assert.deepEqual(await getBattlePrograms(), ids)
    await setBattlePrograms(ids)
    assert.equal((await createProgram({ name: '前進', source })).id, 'first')
    assert.equal((await createMatch(ids)).runId, 'run-1')
    assert.deepEqual(
      requests.map((request) => request.path),
      ['/api/battle/program', '/api/battle/program', '/api/battle/program', '/api/program', '/api/match'],
    )
    assert.deepEqual(requests[2].body, ids)
    assert.deepEqual(requests[3].body, { name: '前進', source, isBattle: true })
    assert.deepEqual(
      requests[4].body,
      ids.map((programId) => ({ programId })),
    )
    assert.ok(requests.every((request) => request.credentials === 'include'))
    assert.throws(() => parseBattleProgramIds(['a', 'b']), /形式/)
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('対戦の runId で結果イベントを受け取る', () => {
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
  }
  globalThis.EventSource = FakeSource as unknown as typeof EventSource
  try {
    let result: unknown
    const stop = subscribeMatchResult(
      'run-1',
      (value) => {
        result = value
      },
      () => assert.fail('unexpected SSE error'),
    )
    assert.equal(FakeSource.last.url, '/api/run/run-1/events')
    assert.equal(FakeSource.last.options.withCredentials, true)
    const received = { history: [], summary: { score: { player: 2, opponent: 1 } } }
    FakeSource.last.handlers.get('result')?.({ data: JSON.stringify(received) } as MessageEvent)
    assert.deepEqual(result, received)
    assert.equal(FakeSource.last.closed, true)
    stop()
  } finally {
    globalThis.EventSource = originalEventSource
  }
})
