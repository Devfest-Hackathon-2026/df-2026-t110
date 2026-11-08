import test from 'node:test'
import assert from 'node:assert/strict'
import {
  isPicoProgram,
  makeStatement,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'

test('生成したmainはPicolang v0のASTとして検証できる', () => {
  const source = newProgram()
  assert.equal(isPicoProgram(source), true)
  assert.equal(programFunctions(source).main.body.stmts.at(-1)?.kind, 'return')
  assert.equal(source.picolangVersion, 0)
})

test('入れ子の制御構文と組み込み関数は異なるhintIdを持つ', () => {
  const source = newProgram()
  const main = programFunctions(source).main
  const condition = makeStatement('if', programFunctions(source))
  assert.equal(condition.kind, 'if')
  if (condition.kind !== 'if') return
  condition.then.stmts.push(makeStatement('forward', programFunctions(source)))
  main.body.stmts.unshift(condition)
  assert.equal(isPicoProgram(source), true)

  const duplicate = structuredClone(source)
  const copiedIf = programFunctions(duplicate).main.body.stmts[0]
  if (copiedIf.kind === 'if') copiedIf.then.hintId = copiedIf.hintId
  assert.equal(isPicoProgram(duplicate), false)
})

test('必須returnとノード形状の破損を拒否する', () => {
  const source = newProgram()
  programFunctions(source).main.body.stmts.pop()
  assert.equal(isPicoProgram(source), false)

  const other = newProgram() as unknown as {
    picolang: { v0: { program: { main: { body: { extra: boolean } } } } }
  }
  other.picolang.v0.program.main.body.extra = true
  assert.equal(isPicoProgram(other), false)
})
