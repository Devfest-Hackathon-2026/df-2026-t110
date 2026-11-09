import test from 'node:test'
import assert from 'node:assert/strict'
import { getTutorial } from '../src/features/tutorial/tutorialApi.ts'

test('教材APIの本文が止まっても時間切れで読み込みを終了する', async () => {
  const originalFetch = globalThis.fetch
  const originalWindow = globalThis.window
  let active = false
  let onTimeout = () => {}
  Object.assign(globalThis, {
    window: {
      setTimeout(callback: () => void) {
        active = true
        onTimeout = callback
        return 1
      },
      clearTimeout() {
        active = false
      },
    },
  })
  globalThis.fetch = async (_input, init) => {
    assert.equal(init?.credentials, 'same-origin')
    return new Response(new ReadableStream({ start() {} }), {
      headers: { 'content-type': 'application/json' },
    })
  }
  try {
    const pending = getTutorial('first-step', new AbortController().signal)
    await new Promise<void>((resolve) => setImmediate(resolve))
    assert.equal(active, true, '本文の読み込み中はタイマーを維持する')
    onTimeout()
    await assert.rejects(pending, /20秒以内/)
  } finally {
    globalThis.fetch = originalFetch
    if (originalWindow === undefined) delete (globalThis as { window?: Window }).window
    else globalThis.window = originalWindow
  }
})

test('教材APIがログイン画面に転送された場合は認証を案内する', async () => {
  const originalFetch = globalThis.fetch
  const originalWindow = globalThis.window
  Object.assign(globalThis, {
    window: { setTimeout, clearTimeout },
  })
  globalThis.fetch = async () => {
    const response = new Response('<html>Login</html>', { headers: { 'content-type': 'text/html' } })
    Object.defineProperty(response, 'redirected', { value: true })
    return response
  }
  try {
    await assert.rejects(getTutorial('first-step', new AbortController().signal), /ログイン画面へ転送/)
  } finally {
    globalThis.fetch = originalFetch
    if (originalWindow === undefined) delete (globalThis as { window?: Window }).window
    else globalThis.window = originalWindow
  }
})
