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
  picoDraft,
  programFunctions,
  makeCall,
  type PicoType,
} from '../src/features/picolang/language/picolang.ts'
import { analyzePicolang, inferPicolang } from '../src/features/picolang/language/analyzePicolang.ts'

test('入れ子のstack型をBlocklyの型キーと相互変換できる', () => {
  const type: PicoType = { kind: 'stack', elementType: { kind: 'stack', elementType: 'bool' } }
  assert.equal(typeKey(type), 'stack:stack:bool')
  assert.deepEqual(keyType('stack:stack:bool'), type)
})

test('optionとstackの入れ子をBlocklyの型キーと相互変換できる', () => {
  for (const key of ['option:bool', 'option:stack:integer', 'stack:option:option:rotation'])
    assert.equal(typeKey(keyType(key)), key)
})

test('オプションカテゴリのブロックはoptionを受け取り、unwrapの返り値を固定しない', () => {
  const functions = programFunctions(newProgram())
  registerPicoBlocks(functions)
  const category = picoToolbox(functions).contents.find(
    (item) => item.kind === 'category' && 'name' in item && item.name === 'オプション',
  ) as Blockly.utils.toolbox.StaticCategoryInfo
  assert.ok(category)
  assert.equal(category.contents.length, 4)
  const workspace = new Blockly.Workspace()
  try {
    for (const name of ['unwrap', 'is_some']) {
      const type = `pico_expr_${name}`
      const entry = category.contents.find(
        (item) => item.kind === 'block' && 'type' in item && item.type === type,
      ) as Blockly.utils.toolbox.BlockInfo
      assert.ok(entry)
      const block = Blockly.serialization.blocks.append({ ...entry, type }, workspace)
      const input = block.getInput('ARG0')!.connection!
      assert.deepEqual(input.getCheck(), ['option'])
      assert.equal(block.getInputTargetBlock('ARG0')!.type, 'pico_expr_some')
      assert.deepEqual(block.outputConnection!.getCheck(), name === 'unwrap' ? null : ['bool'])
      for (const other of ['pico_number', 'pico_expr_new'])
        assert.equal(
          workspace.connectionChecker.doTypeChecks(input, workspace.newBlock(other).outputConnection!),
          false,
        )
    }
  } finally {
    workspace.dispose()
  }
})

test('option引数とboolのunwrapをBlocklyとAST・実行JSONの間で保持する', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const present = makeCall('is_some')
  present.args = [ident('input')]
  const unwrap = makeCall('unwrap')
  unwrap.args = [ident('input')]
  const print = makeCall('print_bool')
  print.args = [unwrap]
  functions.inspect = {
    kind: 'function',
    hintId: id(),
    params: [{ kind: 'param', hintId: id(), ident: ident('input') }],
    body: block([
      { kind: 'if', hintId: id(), condition: present, then: block([print]) },
      { kind: 'return', hintId: id(), value: ident('input') },
    ]),
  }
  const before = structuredClone(functions.inspect)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.inspect, functions)
    refreshVariableBlocks(workspace)
    assert.equal(workspace.getTopBlocks(false).length, 1)
    functions.inspect = readFunction(workspace, functions.inspect, functions)
    assert.deepEqual(functions.inspect, before)
    const analysis = inferPicolang(program)
    assert.deepEqual(analysis.diagnostics, [])
    assert.equal(isPicoProgram(analysis.program), true)
    assert.deepEqual(analysis.signatures.inspect, {
      params: [{ kind: 'option', elementType: 'bool' }],
      returns: { kind: 'option', elementType: 'bool' },
    })
  } finally {
    workspace.dispose()
  }
})

test('someとnoneのブロックを生成し、型を推論したプログラムとして往復できる', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  const declaration = makeStatement('declare', functions)
  assert.equal(declaration.kind, 'declare')
  declaration.ident = ident('maybe')
  declaration.value = makeCall('none')
  const some = makeCall('some')
  some.args = [defaultExpr('bool')]
  const assignment = makeStatement('assign', functions)
  assert.equal(assignment.kind, 'assign')
  assignment.ident = ident('maybe')
  assignment.value = some
  const unwrap = makeCall('unwrap')
  unwrap.args = [ident('maybe')]
  const print = makeCall('print_bool')
  print.args = [unwrap]
  functions.main.body.stmts.unshift(declaration, assignment, print)
  const before = structuredClone(functions.main)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    for (const name of ['some', 'none']) {
      const block = workspace.newBlock(`pico_expr_${name}`)
      assert.deepEqual(block.outputConnection!.getCheck(), ['option'])
      if (name === 'some') assert.equal(block.getInput('ARG0')!.connection!.getCheck(), null)
      else assert.equal(block.getInput('ARG0'), null)
    }
    loadFunction(workspace, functions.main, functions)
    refreshVariableBlocks(workspace)
    assert.equal(workspace.getTopBlocks(false).length, 1)
    functions.main = readFunction(workspace, functions.main, functions)
    assert.deepEqual(functions.main, before)
    const analysis = inferPicolang(program)
    assert.deepEqual(analysis.diagnostics, [])
    assert.deepEqual(analysis.types[declaration.ident.hintId], { kind: 'option', elementType: 'bool' })
    assert.equal(isPicoProgram(analysis.program), true)
  } finally {
    workspace.dispose()
  }
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

test('空の引数は保存・再読込後もエラーになり、値を入れると警告が消えて実行できる', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  functions.main.body.stmts.unshift(makeCall('print_integer'))
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    const print = workspace.getTopBlocks(false)[0]!
    print.getInputTargetBlock('ARG0')!.dispose()
    functions.main = readFunction(workspace, functions.main, functions)
    const analysis = inferPicolang(program)
    assert.equal(analysis.program, null)
    assert.equal(isPicoDraft(program), true)
    assert.equal(isPicoProgram(program), false)
    assert.deepEqual(
      analysis.diagnostics.map((d) => d.code),
      ['missing_argument'],
    )
    const diagnostic = analysis.diagnostics[0]!
    assert.equal(diagnosticBlock(workspace, diagnostic.hintId), print)
    assert.deepEqual(readFunction(workspace, functions.main, functions), functions.main)

    const saved = picoDraft(JSON.parse(JSON.stringify(program)))
    assert.equal(isPicoDraft(saved), true)
    const savedFunctions = programFunctions(saved)
    loadFunction(workspace, savedFunctions.main, savedFunctions)
    const restored = diagnosticBlock(workspace, diagnostic.hintId)!
    assert.equal(restored.type, 'pico_stmt_print_integer')
    assert.equal(restored.getInputTargetBlock('ARG0'), null)
    assert.deepEqual(readFunction(workspace, savedFunctions.main, savedFunctions), savedFunctions.main)
    assert.deepEqual(inferPicolang(saved).diagnostics, analysis.diagnostics)

    let warning: string | null = null
    restored.setWarningText = (text) => {
      warning = text
    }
    updatePicoDiagnostics(workspace, analysis.diagnostics)
    assert.match(warning!, /エラー：入力欄が空/)
    const number = workspace.newBlock('pico_number')
    restored.getInput('ARG0')!.connection!.connect(number.outputConnection!)
    savedFunctions.main = readFunction(workspace, savedFunctions.main, savedFunctions)
    const repaired = inferPicolang(saved)
    assert.deepEqual(repaired.diagnostics, [])
    assert.equal(isPicoProgram(repaired.program), true)
    updatePicoDiagnostics(workspace, repaired.diagnostics)
    assert.equal(warning, null)
  } finally {
    workspace.dispose()
  }
})

test('制御・変数・返り値・式の空欄も、それぞれのブロックに対応するエラーとして保持する', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  registerPicoBlocks(functions)
  for (const [type, inputs] of [
    ['pico_if', ['CONDITION']],
    ['pico_if_else', ['CONDITION']],
    ['pico_while', ['CONDITION']],
    ['pico_for', ['START', 'END']],
    ['pico_declare', ['VALUE']],
    ['pico_assign', ['VALUE']],
    ['pico_return', ['VALUE']],
    ['pico_expr_statement', ['VALUE']],
    ['pico_stmt_add', ['ARG0', 'ARG1']],
  ] as const) {
    const workspace = new Blockly.Workspace()
    try {
      const empty = workspace.newBlock(type)
      if (type === 'pico_assign') {
        const declaration = workspace.newBlock('pico_declare')
        declaration
          .getInput('VALUE')!
          .connection!.connect(workspace.newBlock('pico_number').outputConnection!)
        declaration.nextConnection!.connect(empty.previousConnection!)
      }
      functions.main = readFunction(workspace, programFunctions(newProgram()).main, functions)
      const analysis = inferPicolang(program)
      const missing = analysis.diagnostics.filter((d) => d.code === 'missing_argument')
      assert.equal(missing.length, inputs.length, type)
      assert.equal(analysis.program, null, type)
      assert.equal(
        analysis.diagnostics.some((d) => d.code === 'type_mismatch'),
        false,
        type,
      )
      assert.equal(isPicoDraft(program), true, type)
      for (const diagnostic of missing) assert.equal(diagnosticBlock(workspace, diagnostic.hintId), empty)
      const before = structuredClone(functions.main)
      loadFunction(workspace, functions.main, functions)
      functions.main = readFunction(workspace, functions.main, functions)
      assert.deepEqual(functions.main, before, type)
      assert.deepEqual(inferPicolang(program).diagnostics, analysis.diagnostics, type)
    } finally {
      workspace.dispose()
    }
  }
})

test('自作関数と入れ子の多相関数でも空の引数を検出する', () => {
  const program = newProgram()
  const functions = programFunctions(program)
  functions.consume = {
    kind: 'function',
    hintId: id(),
    params: [{ kind: 'param', hintId: id(), ident: ident('input') }],
    body: block([makeCall('print_integer'), { kind: 'return', hintId: id(), value: defaultExpr('unit') }]),
  }
  functions.consume.body.stmts[0] = {
    kind: 'call',
    hintId: id(),
    func: 'print_integer',
    args: [ident('input')],
  }
  const nested = makeCall('print_bool')
  nested.args = [makeCall('is_some')]
  functions.main.body.stmts.unshift(makeCall('consume', functions), nested)
  registerPicoBlocks(functions)
  const workspace = new Blockly.Workspace()
  try {
    loadFunction(workspace, functions.main, functions)
    for (const type of ['pico_stmt_consume', 'pico_expr_is_some']) {
      const target = workspace.getAllBlocks(false).find((item) => item.type === type)!
      target.getInputTargetBlock('ARG0')!.dispose()
    }
    functions.main = readFunction(workspace, functions.main, functions)
    const analysis = inferPicolang(program)
    assert.deepEqual(
      analysis.diagnostics.map((d) => d.code),
      ['missing_argument', 'missing_argument'],
    )
    assert.equal(analysis.program, null)
    const before = structuredClone(functions.main)
    loadFunction(workspace, functions.main, functions)
    assert.deepEqual(readFunction(workspace, functions.main, functions), before)
    for (const diagnostic of analysis.diagnostics) assert.ok(diagnosticBlock(workspace, diagnostic.hintId))
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
