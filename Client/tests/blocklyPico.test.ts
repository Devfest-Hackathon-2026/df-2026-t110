import test from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import {
  keyType,
  loadFunction,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
  typeKey,
} from '../src/features/picolang/editor/blocklyPico.ts'
import {
  ident,
  isPicoProgram,
  makeStatement,
  newProgram,
  programFunctions,
  type PicoType,
} from '../src/features/picolang/language/picolang.ts'

test('入れ子のstack型をBlocklyの型キーと相互変換できる', () => {
  const type: PicoType = { kind: 'stack', elementType: { kind: 'stack', elementType: 'bool' } }
  assert.equal(typeKey(type), 'stack:stack:bool')
  assert.deepEqual(keyType('stack:stack:bool'), type)
})

test('Blocklyのstack型の変数と組み込み関数をASTに変換できる', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    const declaration = workspace.newBlock('pico_declare')
    declaration.setFieldValue('stack:integer', 'TYPE')
    refreshVariableBlocks(workspace, [])
    const empty = workspace.newBlock('pico_expr_new')
    declaration.getInput('VALUE')!.connection!.connect(empty.outputConnection!)
    const result = readFunction(workspace, functions.main, functions)
    const statement = result.body.stmts[0]
    assert.equal(statement.kind, 'declare')
    assert.deepEqual(statement.type, { kind: 'stack', elementType: 'integer' })
    assert.equal(statement.value.kind, 'call')
    assert.equal(statement.value.func, 'new')
    functions.main = result
    assert.equal(isPicoProgram(program), true)
  } finally {
    workspace.dispose()
  }
})

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
