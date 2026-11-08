import test from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import {
  loadFunction,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
} from '../src/features/picolang/editor/blocklyPico.ts'
import {
  ident,
  isPicoProgram,
  makeStatement,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'

test('Blocklyとの往復で制御構文と変数宣言をPicolang ASTに保つ', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  const conditional = makeStatement('if', functions)
  assert.equal(declaration.kind, 'declare')
  assert.equal(conditional.kind, 'if')
  declaration.ident = ident('steps')
  conditional.then.stmts.push(makeStatement('forward', functions))
  functions.main.body.stmts.unshift(declaration, conditional)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace, [])
    const declarationBlock = workspace.getAllBlocks(false).find((item) => item.type === 'pico_declare')
    assert.equal(declarationBlock?.getInputTargetBlock('TOKEN')?.isShadow(), true)
    const result = readFunction(workspace, functions.main, functions)
    functions.main = result
    assert.equal(isPicoProgram(program), true)
    assert.deepEqual(
      result.body.stmts.map((item) => item.kind),
      ['declare', 'if', 'return'],
    )
    assert.equal(
      result.body.stmts[1]?.kind === 'if' &&
        result.body.stmts[1].then.stmts[0]?.kind === 'call' &&
        result.body.stmts[1].then.stmts[0].func,
      'forward',
    )
  } finally {
    workspace.dispose()
  }
})
