import test from 'node:test'
import assert from 'node:assert/strict'
import { newProgram } from '../src/features/picolang/language/picolang.ts'
import {
  generateProgram,
  ProgramGenerationApiError,
} from '../src/features/program-generation/programGenerationApi.ts'

test('認証Cookieを含めて依頼を送り、Picolangを受け取る', async () => {
  const originalFetch = globalThis.fetch
  const program = newProgram()
  globalThis.fetch = async (input, init) => {
    assert.equal(input, '/api/program/create-with-vibe')
    assert.equal(init?.method, 'POST')
    assert.equal(init?.credentials, 'include')
    assert.equal(
      init?.headers && (init.headers as Record<string, string>)['Content-Type'],
      'application/json',
    )
    assert.deepEqual(JSON.parse(String(init?.body)), {
      prompt: 'ゴールへ進む',
      fieldId: 'first-step',
      isBattle: false,
    })
    return Response.json(program, { status: 201 })
  }
  try {
    assert.deepEqual(
      await generateProgram(
        { prompt: 'ゴールへ進む', fieldId: 'first-step', isBattle: false },
        new AbortController().signal,
      ),
      program,
    )
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('認証エラーと不正な成功レスポンスを区別する', async () => {
  const originalFetch = globalThis.fetch
  try {
    globalThis.fetch = async () => new Response(null, { status: 401 })
    await assert.rejects(
      generateProgram({ prompt: '前進' }, new AbortController().signal),
      (error: unknown) => error instanceof ProgramGenerationApiError && error.status === 401,
    )
    globalThis.fetch = async () => Response.json({ invalid: true }, { status: 201 })
    await assert.rejects(
      generateProgram({ prompt: '前進' }, new AbortController().signal),
      /生成したプログラムを使用できませんでした/,
    )
  } finally {
    globalThis.fetch = originalFetch
  }
})
