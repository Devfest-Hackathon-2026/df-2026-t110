import test from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import {
  keyType,
  loadFunction,
  picoToolbox,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
  typeKey,
} from '../src/features/picolang/editor/blocklyPico.ts'
import {
  ident,
  defaultExpr,
  PICO_DISTANCE_TARGETS,
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

test('値カテゴリから各型のブロックを作成し、値をASTと往復できる', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  registerPicoBlocks(functions)
  const toolbox = picoToolbox(functions)
  const category = toolbox.contents.find(
    (item) => item.kind === 'category' && 'name' in item && item.name === '値',
  ) as Blockly.utils.toolbox.StaticCategoryInfo
  assert.ok(category)
  const cases = [
    { blockType: 'pico_number', type: 'integer', field: -42, value: -42 },
    { blockType: 'pico_bool', type: 'bool', field: 'true', value: true },
    { blockType: 'pico_rotation', type: 'rotation', field: 'right', value: 'right' },
    { blockType: 'pico_unit', type: 'unit', field: undefined, value: null },
    { blockType: 'pico_distance_target', type: 'distance_target', field: 'enemy', value: 'enemy' },
  ]
  for (const { blockType, type, field, value } of cases) {
    const workspace = new Blockly.Workspace()
    try {
      const entry = category.contents.find(
        (item) => item.kind === 'block' && 'type' in item && item.type === blockType,
      ) as Blockly.utils.toolbox.BlockInfo
      assert.ok(entry)
      const literal = Blockly.serialization.blocks.append({ ...entry, type: blockType }, workspace)
      assert.equal(literal.isShadow(), false)
      assert.deepEqual(literal.outputConnection?.getCheck(), [type])
      if (field !== undefined) literal.setFieldValue(field, 'VALUE')
      const statement = workspace.newBlock('pico_expr_statement')
      statement.getInput('VALUE')!.connection!.connect(literal.outputConnection!)
      const result = readFunction(workspace, functions.main, functions)
      const expression = result.body.stmts[0]
      assert.equal(expression.kind, 'literal')
      assert.equal(expression.type, type)
      assert.equal(expression.value, value)
      loadFunction(workspace, result, functions)
      assert.deepEqual(readFunction(workspace, result, functions), result)
    } finally {
      workspace.dispose()
    }
  }
})

test('ツールボックスの初期リテラルを取り外し、別の入力へ接続し直せる', () => {
  const functions = programFunctions(newProgram())
  registerPicoBlocks(functions)
  const toolbox = picoToolbox(functions)
  for (const [categoryName, blockType, input] of [
    ['動き', 'pico_stmt_wait', 'ARG0'],
    ['動き', 'pico_stmt_rotate', 'ARG0'],
    ['制御', 'pico_if', 'CONDITION'],
    ['制御', 'pico_return', 'VALUE'],
    ['制御', 'pico_for', 'END'],
    ['変数', 'pico_declare', 'VALUE'],
  ]) {
    const workspace = new Blockly.Workspace()
    try {
      const category = toolbox.contents.find(
        (item) => item.kind === 'category' && 'name' in item && item.name === categoryName,
      ) as Blockly.utils.toolbox.StaticCategoryInfo
      const entry = category.contents.find(
        (item) => item.kind === 'block' && 'type' in item && item.type === blockType,
      ) as Blockly.utils.toolbox.BlockInfo
      const original = Blockly.serialization.blocks.append({ ...entry, type: blockType }, workspace)
      const literal = original.getInputTargetBlock(input)!
      assert.ok(literal)
      assert.equal(literal.isShadow(), false)
      literal.unplug()
      assert.equal(original.getInputTargetBlock(input), null)
      assert.equal(original.getInput(input)!.connection!.getShadowState(), null)
      const target = workspace.newBlock(blockType)
      target.getInput(input)!.connection!.connect(literal.outputConnection!)
      assert.equal(target.getInputTargetBlock(input), literal)
    } finally {
      workspace.dispose()
    }
  }
})

test('読み込んだプログラムのリテラルもshadowを伴わない独立したブロックになる', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  assert.equal(declaration.kind, 'declare')
  declaration.type = 'bool'
  declaration.value = defaultExpr('bool')
  functions.main.body.stmts.unshift(
    declaration,
    makeStatement('if', functions),
    makeStatement('for', functions),
    makeStatement('rotate', functions),
    makeStatement('wait', functions),
  )
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    assert.deepEqual(readFunction(workspace, functions.main, functions), functions.main)
    const literals = workspace
      .getAllBlocks(false)
      .filter((item) => ['pico_number', 'pico_bool', 'pico_rotation', 'pico_unit'].includes(item.type))
    assert.equal(literals.length, 7)
    for (const literal of literals) {
      assert.equal(literal.isShadow(), false)
      const parent = literal.getParent()!
      const input = parent.inputList.find((item) => item.connection?.targetBlock() === literal)!
      assert.equal(input.connection!.getShadowState(), null)
      literal.unplug()
      assert.equal(input.connection!.targetBlock(), null)
    }
  } finally {
    workspace.dispose()
  }
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

test('全探索対象をprint_distance_targetのBlockly入力とASTで保持する', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const prints = PICO_DISTANCE_TARGETS.map((target) => {
    const statement = makeStatement('print_distance_target', functions)
    assert.equal(statement.kind, 'call')
    const arg = statement.args[0]
    assert.equal(arg.kind, 'literal')
    arg.value = target
    return statement
  })
  functions.main.body.stmts.unshift(...prints)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    const result = readFunction(workspace, functions.main, functions)
    assert.deepEqual(result, functions.main)
    functions.main = result
    assert.equal(isPicoProgram(program), true)
    const first = result.body.stmts[0]
    assert.equal(first.kind, 'call')
    const arg = first.args[0]
    assert.equal(arg.kind, 'literal')
    arg.value = 'invalid' as typeof arg.value
    assert.equal(isPicoProgram(program), false)
  } finally {
    workspace.dispose()
  }
})
