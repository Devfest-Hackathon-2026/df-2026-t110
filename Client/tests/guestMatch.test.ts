import test from 'node:test'
import assert from 'node:assert/strict'
import { runGuestProgram } from '../src/features/match/guestMatch.ts'
import { makeStatement, newProgram, programFunctions } from '../src/features/picolang/picolang.ts'

test('ゲスト練習は選んだPicolangプログラムでゴール到達を判定する', async () => {
  const source = newProgram()
  assert.equal(await runGuestProgram('first-step', source), false)
  programFunctions(source).main.body.stmts.unshift(makeStatement('forward', programFunctions(source)))
  assert.equal(await runGuestProgram('first-step', source), true)
  await assert.rejects(runGuestProgram('unknown-field', source), /ローカル練習/)
})
