import test from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import {
  diagnosticBlock,
  keyType,
  loadFunction,
  picoToolbox,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
  typeKey,
  updatePicoDiagnostics,
} from '../src/features/picolang/editor/blocklyPico.ts'
import {
  block,
  id,
  ident,
  defaultExpr,
  PICO_DISTANCE_TARGETS,
  isPicoProgram,
  isPicoDraft,
  makeStatement,
  newProgram,
  programFunctions,
  type PicoType,
} from '../src/features/picolang/language/picolang.ts'
import { analyzePicolang, inferPicolang } from '../src/features/picolang/language/analyzePicolang.ts'

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
    assert.equal(declaration.getField('TYPE'), null)
    refreshVariableBlocks(workspace)
    const empty = workspace.newBlock('pico_expr_new')
    const push = workspace.newBlock('pico_expr_push_back')
    const number = workspace.newBlock('pico_number')
    push.getInput('ARG0')!.connection!.connect(empty.outputConnection!)
    push.getInput('ARG1')!.connection!.connect(number.outputConnection!)
    declaration.getInput('VALUE')!.connection!.connect(push.outputConnection!)
    const result = readFunction(workspace, functions.main, functions)
    const statement = result.body.stmts[0]
    assert.equal(statement.kind, 'declare')
    assert.equal('type' in statement, false)
    assert.equal(statement.value.kind, 'call')
    assert.equal(statement.value.func, 'push_back')
    const analysis = inferPicolang({ ...program, picolang: { v0: { program: { main: result } } } })
    assert.deepEqual(analysis.types[statement.ident.hintId], { kind: 'stack', elementType: 'integer' })
    functions.main = result
    assert.equal(isPicoDraft(program), true)
    assert.equal(isPicoProgram(inferPicolang(program).program), true)
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
    refreshVariableBlocks(workspace)
    const declarationBlock = workspace.getAllBlocks(false).find((item) => item.type === 'pico_declare')
    assert.equal(declarationBlock?.getInputTargetBlock('TOKEN')?.isShadow(), true)
    const result = readFunction(workspace, functions.main, functions)
    functions.main = result
    assert.equal(isPicoDraft(program), true)
    assert.equal(isPicoProgram(inferPicolang(program).program), true)
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
    assert.equal(isPicoDraft(program), true)
    assert.equal(isPicoProgram(inferPicolang(program).program), true)
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

test('Boolean stack elements remain connected and type-check after a Blockly round trip', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  assert.equal(declaration.kind, 'declare')
  declaration.value = {
    kind: 'call',
    hintId: id(),
    func: 'push_back',
    args: [{ kind: 'call', hintId: id(), func: 'new', args: [] }, defaultExpr('bool')],
  }
  functions.main.body.stmts.unshift(declaration)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace)
    functions.main = readFunction(workspace, functions.main, functions)
    assert.equal(workspace.getTopBlocks(false).length, 1)
    assert.deepEqual(functions.main.body.stmts[0], declaration)
    assert.equal(
      analyzePicolang(program).some((d) => d.severity === 'error'),
      false,
    )
  } finally {
    workspace.dispose()
  }
})

test('same names in sibling scopes do not disconnect or change variable references', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  for (const type of ['bool', 'integer'] as const) {
    const declaration = makeStatement('declare', functions)
    assert.equal(declaration.kind, 'declare')
    declaration.value = defaultExpr(type)
    const conditional = makeStatement('if', functions)
    assert.equal(conditional.kind, 'if')
    conditional.then = block([
      declaration,
      { kind: 'call', hintId: id(), func: `print_${type}`, args: [ident('x')] },
    ])
    functions.main.body.stmts.unshift(conditional)
  }
  const before = structuredClone(functions.main)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace)
    functions.main = readFunction(workspace, functions.main, functions)
    assert.deepEqual(functions.main, before)
    assert.deepEqual(analyzePicolang(program), [])
  } finally {
    workspace.dispose()
  }
})

test('diagnostics locate declaration and assignment identifier hints and clear stale warnings', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  assert.equal(declaration.kind, 'declare')
  const assignment = makeStatement('assign', functions)
  assert.equal(assignment.kind, 'assign')
  assignment.ident = ident('missing')
  functions.main.body.stmts.unshift(declaration, assignment)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    const messages = new Map<string, string | null>()
    for (const item of workspace.getAllBlocks(false))
      item.setWarningText = (text) => {
        messages.set(item.id, text)
      }
    const diagnostics = analyzePicolang(program)
    updatePicoDiagnostics(workspace, diagnostics)
    const declared = diagnosticBlock(workspace, declaration.ident.hintId)!
    const assigned = diagnosticBlock(workspace, assignment.ident.hintId)!
    assert.equal(declared.type, 'pico_declare')
    assert.equal(assigned.type, 'pico_assign')
    assert.match(messages.get(declared.id)!, /警告/)
    assert.match(messages.get(assigned.id)!, /エラー/)
    updatePicoDiagnostics(workspace, [])
    assert.equal(messages.get(declared.id), null)
    assert.equal(messages.get(assigned.id), null)
  } finally {
    workspace.dispose()
  }
})

test('undefined function blocks retain arguments of different arities for diagnostics', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  functions.main.body.stmts.unshift(
    { kind: 'call', hintId: id(), func: 'unknown', args: [] },
    { kind: 'call', hintId: id(), func: 'unknown', args: [ident('missing')] },
    {
      kind: 'call',
      hintId: id(),
      func: 'print_integer',
      args: [
        { kind: 'call', hintId: id(), func: 'unknown', args: [defaultExpr('bool'), defaultExpr('rotation')] },
      ],
    },
  )
  const before = structuredClone(functions.main)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    functions.main = readFunction(workspace, functions.main, functions)
    assert.deepEqual(functions.main, before)
    const diagnostics = analyzePicolang(program)
    assert.equal(diagnostics.filter((d) => d.code === 'undefined_function').length, 3)
    assert.equal(diagnostics.filter((d) => d.code === 'undefined_variable').length, 1)
  } finally {
    workspace.dispose()
  }
})

test('annotation-free user functions retain arguments and return expressions through Blockly', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  functions.identity = {
    kind: 'function',
    hintId: id(),
    params: [{ kind: 'param', hintId: id(), ident: ident('x') }],
    body: block([{ kind: 'return', hintId: id(), value: ident('x') }]),
  }
  const statement = {
    kind: 'call' as const,
    hintId: id(),
    func: 'print_bool',
    args: [{ kind: 'call' as const, hintId: id(), func: 'identity', args: [defaultExpr('bool')] }],
  }
  functions.main.body.stmts.unshift(statement)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace)
    assert.deepEqual(readFunction(workspace, functions.main, functions), functions.main)
    assert.equal(workspace.getTopBlocks(false).length, 1)
    const call = workspace.getAllBlocks(false).find((item) => item.type === 'pico_expr_identity')!
    assert.equal(call.getInputTargetBlock('ARG0')!.type, 'pico_bool')
    const toolbox = picoToolbox(functions)
    const category = toolbox.contents.find(
      (item) => item.kind === 'category' && 'name' in item && item.name === '関数',
    ) as Blockly.utils.toolbox.StaticCategoryInfo
    assert.ok(
      category.contents.some(
        (item) => item.kind === 'block' && 'type' in item && item.type === 'pico_expr_identity',
      ),
    )
    loadFunction(workspace, functions.identity, functions)
    assert.deepEqual(readFunction(workspace, functions.identity, functions), functions.identity)
    assert.deepEqual(analyzePicolang(program), [])
  } finally {
    workspace.dispose()
  }
})

test('declaration blocks display inferred stack types without a type selector or changing connections', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  assert.equal(declaration.kind, 'declare')
  declaration.value = { kind: 'call', hintId: id(), func: 'new', args: [] }
  functions.main.body.stmts.unshift(declaration, {
    kind: 'call',
    hintId: id(),
    func: 'print_bool',
    args: [
      {
        kind: 'call',
        hintId: id(),
        func: 'get_back',
        args: [ident(declaration.ident.name)],
      },
    ],
  })
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace)
    const block = workspace.getAllBlocks(false).find((item) => item.type === 'pico_declare')!
    assert.equal(block.getField('TYPE'), null)
    const initializer = block.getInputTargetBlock('VALUE')
    const analysis = inferPicolang(program)
    updatePicoDiagnostics(workspace, analysis.diagnostics, analysis.types)
    assert.equal(block.getFieldValue('INFERRED_TYPE'), '真理値のスタック')
    assert.equal(block.getInputTargetBlock('VALUE'), initializer)
    assert.deepEqual(readFunction(workspace, functions.main, functions), functions.main)
    assert.equal('type' in declaration, false)
    updatePicoDiagnostics(workspace, [])
    assert.equal(block.getFieldValue('INFERRED_TYPE'), '未確定')
  } finally {
    workspace.dispose()
  }
})
