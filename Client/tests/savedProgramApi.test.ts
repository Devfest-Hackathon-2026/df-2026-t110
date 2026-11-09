import test from 'node:test'
import assert from 'node:assert/strict'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import {
  getSavedProgram,
  listTutorialPrograms,
  SavedProgramApiError,
  updateSavedProgram,
} from '../src/features/picolang/savedProgramApi.ts'

test('保存済みプログラムを取得・更新するとき所有者のCookieと正しい本文を送る', async () => {
  const originalFetch = globalThis.fetch
  const source = newProgram()
  const saved = { id: 'program-1', name: '改良版', source, isBattle: true }
  const requests: Array<{ path: string; init: RequestInit | undefined }> = []
  globalThis.fetch = async (input, init) => {
    requests.push({ path: String(input), init })
    return Response.json(saved)
  }
  try {
    assert.deepEqual(await getSavedProgram('program-1'), saved)
    assert.deepEqual(await updateSavedProgram('program-1', '改良版', source), saved)
    assert.deepEqual(
      requests.map((request) => request.path),
      ['/api/program/program-1', '/api/program/program-1'],
    )
    assert.equal(requests[0]?.init?.credentials, 'include')
    assert.equal(requests[0]?.init?.cache, 'no-store')
    assert.equal(requests[1]?.init?.method, 'PUT')
    assert.deepEqual(JSON.parse(String(requests[1]?.init?.body)), { name: '改良版', source })
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('教材の保存済み一覧を受け取り、権限エラーを区別する', async () => {
  const originalFetch = globalThis.fetch
  const source = newProgram()
  const saved = { id: 'program-2', name: '教材用', source, isBattle: false, tutorialID: 'first-step' }
  globalThis.fetch = async (input, init) => {
    assert.equal(String(input), '/api/tutorial/first-step/program')
    assert.equal(init?.credentials, 'include')
    return Response.json([saved])
  }
  try {
    assert.deepEqual(await listTutorialPrograms('first-step'), [saved])
  } finally {
    globalThis.fetch = originalFetch
  }
  globalThis.fetch = async () => new Response(null, { status: 403 })
  try {
    await assert.rejects(
      getSavedProgram('other-user'),
      (error: unknown) => error instanceof SavedProgramApiError && error.status === 403,
    )
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('安全に表現できない整数を含むソースを黙って丸めない', async () => {
  const originalFetch = globalThis.fetch
  const source = newProgram()
  source.picolang.v0.program.main!.body.stmts.unshift({
    kind: 'call',
    hintId: '00000000-0000-4000-8000-000000000001',
    func: 'print_integer',
    args: [{ kind: 'literal', hintId: '00000000-0000-4000-8000-000000000002', type: 'integer', value: 0 }],
  })
  const raw = JSON.stringify({ id: 'large', name: '大きい整数', source, isBattle: true }).replace(
    '"value":0',
    '"value":9007199254740993',
  )
  globalThis.fetch = async () => new Response(raw, { headers: { 'content-type': 'application/json' } })
  try {
    await assert.rejects(getSavedProgram('large'), /形式が不正/)
  } finally {
    globalThis.fetch = originalFetch
  }
})
