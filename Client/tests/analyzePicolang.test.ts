import test from 'node:test'
import assert from 'node:assert/strict'
import { analyzePicolang, inferPicolang } from '../src/features/picolang/language/analyzePicolang.ts'
import {
  block,
  defaultExpr,
  id,
  ident,
  isPicoDraft,
  isPicoProgram,
  newProgram,
  picoDraft,
  programFunctions,
  type PicoExpr,
  type PicoFunction,
  type PicoStatement,
  type PicoType,
} from '../src/features/picolang/language/picolang.ts'

const call = (func: string, ...args: PicoExpr[]): PicoExpr => ({ kind: 'call', hintId: id(), func, args })
const declare = (
  name: string,
  type: PicoType = 'integer',
  value = defaultExpr(type),
): Extract<PicoStatement, { kind: 'declare' }> => ({
  kind: 'declare',
  hintId: id(),
  ident: ident(name),
  type,
  value,
})
const ret = (value = defaultExpr('unit')): PicoStatement => ({ kind: 'return', hintId: id(), value })
const conditional = (stmts: PicoStatement[], other?: PicoStatement[]): PicoStatement => ({
  kind: 'if',
  hintId: id(),
  condition: defaultExpr('bool'),
  then: block(stmts),
  ...(other ? { else: block(other) } : {}),
})
const fn = (
  stmts: PicoStatement[],
  returnType: PicoType = 'unit',
  params: PicoFunction['params'] = [],
): PicoFunction => ({ kind: 'function', hintId: id(), params, returnType, body: block(stmts) })
const source = (...stmts: PicoStatement[]) => {
  const program = newProgram()
  programFunctions(program).main.body = block([...stmts, ret()])
  return program
}
const stack = (elementType: PicoType): PicoType => ({ kind: 'stack', elementType })

test('valid programs are analyzed without mutation and diagnostics are repeatable', () => {
  const program = source(declare('x'), call('print_integer', ident('x')))
  const before = structuredClone(program)
  assert.deepEqual(analyzePicolang(program), [])
  assert.deepEqual(analyzePicolang(program), [])
  assert.deepEqual(program, before)
})

test('undefined references report their hint and a later declaration in the same scope', () => {
  const use = ident('x')
  const declaration = declare('x')
  const diagnostics = analyzePicolang(source(call('print_integer', use), declaration))
  assert.deepEqual(
    diagnostics.filter((d) => d.hintId === use.hintId).map((d) => [d.code, d.severity]),
    [
      ['undefined_variable', 'error'],
      ['use_before_definition', 'warning'],
    ],
  )
  assert.equal(
    diagnostics.find((d) => d.code === 'use_before_definition')?.relatedHintId,
    declaration.ident.hintId,
  )
  assert.equal(
    diagnostics.some((d) => d.code === 'type_mismatch'),
    false,
  )
})

test('later declarations in ancestor scopes are distinguished from unrelated scopes', () => {
  const inner = ident('later')
  const sibling = ident('sibling')
  const diagnostics = analyzePicolang(
    source(
      conditional([call('print_integer', inner), call('print_integer', sibling)], [declare('sibling')]),
      declare('later'),
    ),
  )
  assert.equal(
    diagnostics.some((d) => d.hintId === inner.hintId && d.code === 'use_before_definition'),
    true,
  )
  assert.deepEqual(
    diagnostics.filter((d) => d.hintId === sibling.hintId).map((d) => d.code),
    ['undefined_variable'],
  )
})

test('a declaration is not visible in its own initializer', () => {
  const use = ident('x')
  const diagnostics = analyzePicolang(source(declare('x', 'integer', use)))
  assert.equal(
    diagnostics.some((d) => d.hintId === use.hintId && d.code === 'undefined_variable'),
    true,
  )
  assert.equal(
    diagnostics.some((d) => d.hintId === use.hintId && d.code === 'use_before_definition'),
    true,
  )
})

test('references to expired descendants get an out-of-scope warning', () => {
  const use = ident('local')
  const declaration = declare('local')
  const diagnostics = analyzePicolang(source(conditional([declaration]), call('print_integer', use)))
  assert.equal(
    diagnostics.find((d) => d.code === 'out_of_scope_reference')?.relatedHintId,
    declaration.ident.hintId,
  )
  assert.equal(
    diagnostics.some((d) => d.hintId === use.hintId && d.code === 'undefined_variable'),
    true,
  )
})

test('branches and functions have independent scopes', () => {
  const program = source(
    conditional(
      [declare('x'), call('print_integer', ident('x'))],
      [declare('x'), call('print_integer', ident('x'))],
    ),
  )
  programFunctions(program).other = fn([call('print_integer', ident('x')), ret()])
  const diagnostics = analyzePicolang(program)
  assert.deepEqual(
    diagnostics.map((d) => [d.functionName, d.code]),
    [['other', 'undefined_variable']],
  )
})

test('shadowing errors point to the new declaration and warnings to the original', () => {
  const outer = declare('x')
  const inner = declare('x', 'bool')
  const diagnostics = analyzePicolang(source(outer, conditional([inner]), call('print_integer', ident('x'))))
  assert.equal(diagnostics.find((d) => d.code === 'shadowing_variable')?.hintId, inner.ident.hintId)
  assert.equal(diagnostics.find((d) => d.code === 'shadowed_variable')?.hintId, outer.ident.hintId)
  assert.equal(
    diagnostics.some((d) => d.hintId === outer.ident.hintId && d.code === 'unused_variable'),
    false,
  )
  assert.equal(
    diagnostics.some((d) => d.code === 'type_mismatch'),
    false,
  )
})

test('duplicate declarations in the same scope and parameters cannot shadow', () => {
  const param = { kind: 'param' as const, hintId: id(), ident: ident('p'), type: 'integer' as const }
  const program = source(declare('x'), declare('x'))
  programFunctions(program).other = fn([declare('p'), ret()], 'unit', [param])
  assert.equal(analyzePicolang(program).filter((d) => d.code === 'shadowing_variable').length, 2)
})

test('writes alone do not mark variables used, while reads in nested scopes do', () => {
  const x = declare('x')
  const y = declare('y')
  const diagnostics = analyzePicolang(
    source(
      x,
      y,
      { kind: 'assign', hintId: id(), ident: ident('x'), value: defaultExpr() },
      conditional([call('print_integer', ident('y'))]),
    ),
  )
  assert.deepEqual(
    diagnostics.filter((d) => d.code === 'unused_variable').map((d) => d.hintId),
    [x.ident.hintId],
  )
})

test('for bounds are evaluated before binding an integer loop variable', () => {
  const start = ident('i')
  const outside = ident('i')
  const loop: PicoStatement = {
    kind: 'for',
    hintId: id(),
    ident: ident('i'),
    start,
    end: defaultExpr(),
    body: block([call('print_integer', ident('i'))]),
  }
  const diagnostics = analyzePicolang(source(loop, call('print_integer', outside)))
  assert.deepEqual(
    diagnostics.filter((d) => d.code === 'undefined_variable').map((d) => d.hintId),
    [start.hintId, outside.hintId],
  )
  assert.equal(
    diagnostics.some((d) => d.hintId === outside.hintId && d.code === 'out_of_scope_reference'),
    true,
  )
  assert.equal(
    diagnostics.some((d) => d.code === 'unused_variable'),
    false,
  )
})

test('recursive and forward function calls resolve, undefined calls still analyze arguments', () => {
  const program = source(call('later'), call('missing', ident('unknown')))
  programFunctions(program).later = fn([call('later'), ret()])
  const diagnostics = analyzePicolang(program)
  assert.deepEqual(diagnostics.map((d) => d.code).sort(), ['undefined_function', 'undefined_variable'])
})

test('initializers, assignments, conditions, loop bounds, arguments, and returns are checked', () => {
  const program = source(
    declare('x', 'integer', call('add', defaultExpr('bool'), defaultExpr())),
    { kind: 'assign', hintId: id(), ident: ident('x'), value: defaultExpr('rotation') },
    { kind: 'if', hintId: id(), condition: defaultExpr(), then: block() },
    { kind: 'while', hintId: id(), condition: defaultExpr(), body: block() },
    {
      kind: 'for',
      hintId: id(),
      ident: ident('i'),
      start: defaultExpr('bool'),
      end: defaultExpr('unit'),
      body: block(),
    },
    call('print_integer', defaultExpr('bool')),
  )
  programFunctions(program).other = fn([ret(defaultExpr('bool')), ret(defaultExpr())])
  assert.equal(analyzePicolang(program).filter((d) => d.code === 'type_mismatch').length, 8)
})

test('user function argument counts and compound return types are checked', () => {
  const program = source(
    call('other'),
    declare('result', stack('bool'), call('other', defaultExpr())),
    call('print_bool', call('get_back', ident('result'))),
    call('other', defaultExpr('bool')),
    call('other', defaultExpr(), defaultExpr()),
  )
  programFunctions(program).other = fn([ret(call('new'))], stack('bool'), [
    { kind: 'param', hintId: id(), ident: ident('n'), type: 'integer' },
  ])
  const diagnostics = analyzePicolang(program)
  assert.equal(diagnostics.filter((d) => d.code === 'argument_count').length, 2)
  assert.equal(diagnostics.filter((d) => d.code === 'type_mismatch').length, 1)
  assert.equal(
    diagnostics.some((d) => d.code === 'cannot_infer_type'),
    false,
  )
})

test('stack type parameters are fresh for each call and support nested stacks', () => {
  const program = source(
    declare('bools', stack('bool'), call('new')),
    declare('ints', stack('integer'), call('push_back', call('new'), defaultExpr())),
    declare('nested', stack(stack('bool')), call('push_back', call('new'), ident('bools'))),
    call('print_bool', call('get_back', call('get_back', ident('nested')))),
    call('print_integer', call('size', call('pop_back', ident('ints')))),
  )
  assert.deepEqual(analyzePicolang(program), [])
})

test('surrounding types constrain nested stack calls after arguments have been visited', () => {
  const program = source(
    declare('nested', stack(stack('rotation')), call('push_back', call('new'), call('new'))),
    declare('rotation', 'rotation', call('get_back', call('new'))),
    call('print_rotation', call('get_back', call('get_back', ident('nested')))),
    call('print_rotation', ident('rotation')),
  )
  assert.equal(
    analyzePicolang(program).some((d) => d.severity === 'error'),
    false,
  )
})

test('incompatible nested stack elements are rejected', () => {
  const program = source(
    declare('items', stack(stack('bool')), call('new')),
    call('print_bool', call('get_back', call('get_back', ident('items')))),
    call('push_back', ident('items'), call('push_back', call('new'), defaultExpr())),
  )
  assert.equal(analyzePicolang(program).filter((d) => d.code === 'type_mismatch').length, 1)
})

test('unconstrained element types are diagnosed without guessing integer', () => {
  for (const expr of [call('new'), call('size', call('new')), call('get_back', call('new'))]) {
    const diagnostics = analyzePicolang(source(expr))
    assert.equal(diagnostics.filter((d) => d.code === 'cannot_infer_type').length, 1)
  }
})

test('an earlier error suppresses secondary inference errors', () => {
  for (const expr of [
    call('push_back', defaultExpr(), call('new')),
    call('missing', call('new')),
    call('new', defaultExpr()),
  ]) {
    const diagnostics = analyzePicolang(source(expr))
    assert.equal(diagnostics.filter((d) => d.severity === 'error').length, 1)
    assert.equal(
      diagnostics.some((d) => d.code === 'cannot_infer_type'),
      false,
    )
  }
})

test('function conflicts and loop controls match server restrictions', () => {
  const program = source(
    declare('forward'),
    { kind: 'break', hintId: id() },
    { kind: 'continue', hintId: id() },
    {
      kind: 'while',
      hintId: id(),
      condition: defaultExpr('bool'),
      body: block([{ kind: 'break', hintId: id() }]),
    },
  )
  programFunctions(program).forward = fn([ret()])
  const diagnostics = analyzePicolang(program)
  assert.equal(diagnostics.filter((d) => d.code === 'function_conflict').length, 1)
  assert.equal(diagnostics.filter((d) => d.code === 'function_name_conflict').length, 1)
  assert.equal(diagnostics.filter((d) => d.code === 'invalid_loop_control').length, 2)
})

const inferredFunction = (names: string[], ...stmts: PicoStatement[]): PicoFunction => ({
  kind: 'function',
  hintId: id(),
  params: names.map((name) => ({ kind: 'param', hintId: id(), ident: ident(name) })),
  body: block(stmts),
})

test('function body and all call sites infer a signature without annotations', () => {
  const program = source(call('print_integer', call('increment', defaultExpr())))
  programFunctions(program).increment = inferredFunction(['n'], ret(call('add', ident('n'), defaultExpr())))
  const before = structuredClone(program)
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.increment, { params: ['integer'], returns: 'integer' })
  assert.ok(analysis.program)
  assert.equal(isPicoProgram(analysis.program), true)
  assert.deepEqual(program, before)
  assert.equal(programFunctions(analysis.program).increment.params[0].type, 'integer')
  assert.equal(programFunctions(analysis.program).increment.returnType, 'integer')
})

test('a caller constrains an identity function and export does not reuse imported annotations', () => {
  const program = source(call('print_bool', call('identity', defaultExpr('bool'))))
  const identity = inferredFunction(['x'], ret(ident('x')))
  identity.params[0].type = 'integer'
  identity.returnType = 'rotation'
  programFunctions(program).identity = identity
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.identity, { params: ['bool'], returns: 'bool' })
  assert.equal(programFunctions(analysis.program!).identity.returnType, 'bool')
  assert.equal(identity.returnType, 'rotation')
})

test('function constraints are resolved across forward calls and mutual recursion', () => {
  const program = source(call('print_integer', call('alpha', defaultExpr())))
  const functions = programFunctions(program)
  functions.alpha = inferredFunction(['n'], ret(call('zeta', ident('n'))))
  functions.zeta = inferredFunction(
    ['n'],
    conditional([ret(call('alpha', ident('n')))]),
    ret(call('add', ident('n'), defaultExpr())),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.alpha, { params: ['integer'], returns: 'integer' })
  assert.deepEqual(analysis.signatures.zeta, analysis.signatures.alpha)
  const reversed = {
    ...program,
    picolang: { v0: { program: Object.fromEntries(Object.entries(functions).reverse()) } },
  }
  assert.deepEqual(inferPicolang(reversed).signatures, analysis.signatures)
})

test('recursive calls share parameter and result constraints with their function body', () => {
  const program = source(call('print_integer', call('factorial', defaultExpr())))
  programFunctions(program).factorial = inferredFunction(
    ['n'],
    conditional([ret(defaultExpr())]),
    ret(call('mul', ident('n'), call('factorial', call('sub', ident('n'), defaultExpr())))),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.factorial, { params: ['integer'], returns: 'integer' })
})

test('nested stack results are resolved after constraints from later functions', () => {
  const program = source(
    declare('items', stack(stack('rotation')), call('make_nested')),
    call('print_rotation', call('get_back', call('get_back', ident('items')))),
  )
  const functions = programFunctions(program)
  functions.make_nested = inferredFunction([], ret(call('wrap', call('new'))))
  functions.wrap = inferredFunction(['x'], ret(call('push_back', call('new'), ident('x'))))
  const analysis = inferPicolang(program)
  assert.equal(
    analysis.diagnostics.some((d) => d.severity === 'error'),
    false,
  )
  assert.deepEqual(analysis.signatures.make_nested, { params: [], returns: stack(stack('rotation')) })
  assert.deepEqual(analysis.signatures.wrap, {
    params: [stack('rotation')],
    returns: stack(stack('rotation')),
  })
  assert.equal(isPicoProgram(analysis.program), true)
})

test('incompatible calls to one function cannot be generalized independently', () => {
  const program = source(
    call('print_integer', call('identity', defaultExpr())),
    call('print_bool', call('identity', defaultExpr('bool'))),
  )
  programFunctions(program).identity = inferredFunction(['x'], ret(ident('x')))
  const analysis = inferPicolang(program)
  assert.ok(analysis.diagnostics.some((d) => d.code === 'type_mismatch'))
  assert.equal(analysis.program, null)
})

test('incompatible return paths fail inference and prevent export', () => {
  const program = source(call('choose'))
  programFunctions(program).choose = inferredFunction(
    [],
    conditional([ret(defaultExpr())]),
    ret(defaultExpr('bool')),
  )
  const analysis = inferPicolang(program)
  assert.ok(analysis.diagnostics.some((d) => d.code === 'type_mismatch'))
  assert.equal(analysis.program, null)
})

test('unresolved signatures are diagnosed even for uncalled functions', () => {
  const program = source()
  programFunctions(program).identity = inferredFunction(['x'], ret(ident('x')))
  const analysis = inferPicolang(program)
  assert.ok(analysis.diagnostics.some((d) => d.code === 'cannot_infer_type' && d.functionName === 'identity'))
  assert.equal(analysis.signatures.identity.params[0], undefined)
  assert.equal(analysis.signatures.identity.returns, undefined)
  assert.equal(analysis.program, null)
})

test('occurs check rejects infinitely nested types in recursive calls', () => {
  const program = source()
  programFunctions(program).grow = inferredFunction(
    ['x'],
    ret(call('grow', call('push_back', call('new'), ident('x')))),
  )
  const analysis = inferPicolang(program)
  assert.ok(analysis.diagnostics.some((d) => d.code === 'type_mismatch'))
  assert.equal(analysis.program, null)
})

test('main remains a zero-argument function returning unit', () => {
  const program = source()
  programFunctions(program).main.body.stmts = [ret(defaultExpr())]
  assert.ok(analyzePicolang(program).some((d) => d.code === 'type_mismatch'))
  programFunctions(program).main = inferredFunction(['x'], ret(defaultExpr('unit')))
  assert.ok(analyzePicolang(program).some((d) => d.code === 'argument_count'))
})

test('annotation-free drafts persist and typed execution JSON round trips without freezing inferred types', () => {
  const original = source(call('print_bool', call('identity', defaultExpr('bool'))))
  programFunctions(original).identity = inferredFunction(['x'], ret(ident('x')))
  const draft = picoDraft(original)
  assert.equal(isPicoDraft(draft), true)
  assert.equal(isPicoProgram(draft), false)
  const serialized = JSON.parse(JSON.stringify(draft))
  assert.equal(isPicoDraft(serialized), true)
  const compiled = inferPicolang(serialized).program
  assert.ok(compiled)
  assert.equal(isPicoProgram(JSON.parse(JSON.stringify(compiled))), true)
  const imported = picoDraft(compiled)
  assert.equal('returnType' in programFunctions(imported).identity, false)
  assert.equal('type' in programFunctions(imported).identity.params[0], false)
  assert.deepEqual(inferPicolang(imported).program, compiled)
})

test('every primitive function type is inferred and emitted without manual annotations', () => {
  for (const type of ['integer', 'bool', 'unit', 'terrain', 'rotation', 'distance_target'] as const) {
    const program = source(call(`print_${type}`, call('identity', defaultExpr(type))))
    programFunctions(program).identity = inferredFunction(['x'], ret(ident('x')))
    const analysis = inferPicolang(program)
    assert.deepEqual(analysis.diagnostics, [], type)
    assert.deepEqual(analysis.signatures.identity, { params: [type], returns: type }, type)
    assert.ok(analysis.program, type)
    assert.equal(isPicoProgram(analysis.program), true, type)
  }
})

test('assignment to a parameter constrains its type and the return type', () => {
  const program = source()
  programFunctions(program).set_flag = inferredFunction(
    ['x'],
    { kind: 'assign', hintId: id(), ident: ident('x'), value: defaultExpr('bool') },
    ret(ident('x')),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.set_flag, { params: ['bool'], returns: 'bool' })
})

const local = (name: string, value: PicoExpr): Extract<PicoStatement, { kind: 'declare' }> => ({
  kind: 'declare',
  hintId: id(),
  ident: ident(name),
  value,
})

test('local declarations infer every primitive from the initializer and export concrete types', () => {
  for (const type of ['integer', 'bool', 'unit', 'terrain', 'rotation', 'distance_target'] as const) {
    const declaration = local('x', defaultExpr(type))
    const program = source(declaration, call(`print_${type}`, ident('x')))
    const before = structuredClone(program)
    const analysis = inferPicolang(program)
    assert.deepEqual(analysis.diagnostics, [], type)
    assert.equal(analysis.types[declaration.ident.hintId], type)
    assert.equal(isPicoProgram(analysis.program), true)
    const compiled = programFunctions(analysis.program!).main.body.stmts[0]
    assert.equal(compiled.kind, 'declare')
    assert.equal(compiled.type, type)
    assert.deepEqual(program, before)
    assert.equal('type' in declaration, false)
  }
})

test('an empty stack gets its element type from a later assignment', () => {
  const declaration = local('items', call('new'))
  const program = source(
    declaration,
    {
      kind: 'assign',
      hintId: id(),
      ident: ident('items'),
      value: call('push_back', ident('items'), defaultExpr('bool')),
    },
    call('print_bool', call('get_back', ident('items'))),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.types[declaration.ident.hintId], stack('bool'))
  assert.equal(isPicoProgram(analysis.program), true)
})

test('a local variable carries constraints between an initializer, function calls, and a return', () => {
  const program = source(call('print_rotation', call('wrap_local', call('new'))))
  const declaration = local('item', call('get_back', ident('items')))
  programFunctions(program).wrap_local = inferredFunction(['items'], declaration, ret(ident('item')))
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.signatures.wrap_local, { params: [stack('rotation')], returns: 'rotation' })
  assert.equal(analysis.types[declaration.ident.hintId], 'rotation')
})

test('later references infer nested stack elements without declaration annotations', () => {
  const declaration = local('items', call('push_back', call('new'), call('new')))
  const program = source(
    declaration,
    call('print_distance_target', call('get_back', call('get_back', ident('items')))),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.deepEqual(analysis.types[declaration.ident.hintId], stack(stack('distance_target')))
})

test('different lexical bindings with the same name have independent inferred types', () => {
  const integer = local('x', defaultExpr())
  const bool = local('x', defaultExpr('bool'))
  const program = source(
    conditional([integer, call('print_integer', ident('x'))], [bool, call('print_bool', ident('x'))]),
  )
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.equal(analysis.types[integer.ident.hintId], 'integer')
  assert.equal(analysis.types[bool.ident.hintId], 'bool')
})

test('inconsistent assignments, incompatible uses, and infinite local types prevent export', () => {
  for (const program of [
    source(local('x', defaultExpr()), {
      kind: 'assign',
      hintId: id(),
      ident: ident('x'),
      value: defaultExpr('bool'),
    }),
    source(local('x', defaultExpr('bool')), call('print_integer', ident('x'))),
    source(local('x', call('new')), {
      kind: 'assign',
      hintId: id(),
      ident: ident('x'),
      value: call('push_back', ident('x'), ident('x')),
    }),
  ]) {
    const analysis = inferPicolang(program)
    assert.ok(analysis.diagnostics.some((d) => d.code === 'type_mismatch'))
    assert.equal(analysis.program, null)
  }
})

test('an unused or size-only empty stack is not assigned a guessed element type', () => {
  for (const program of [
    source(local('items', call('new'))),
    source(local('items', call('new')), call('size', ident('items'))),
  ]) {
    const analysis = inferPicolang(program)
    assert.ok(analysis.diagnostics.some((d) => d.code === 'cannot_infer_type'))
    assert.equal(analysis.program, null)
  }
})

test('old annotations are stripped and inferred types are filled in every nested lexical block', () => {
  const declarations = ['a', 'b', 'c', 'd'].map((name) => declare(name, 'integer', defaultExpr('bool')))
  const use = (index: number) => [
    declarations[index],
    call('print_bool', ident(declarations[index].ident.name)),
  ]
  const program = source(
    conditional(use(0), use(1)),
    { kind: 'while', hintId: id(), condition: defaultExpr('bool'), body: block(use(2)) },
    {
      kind: 'for',
      hintId: id(),
      ident: ident('i'),
      start: defaultExpr(),
      end: defaultExpr(),
      body: block(use(3)),
    },
  )
  const draft = picoDraft(program)
  assert.equal(isPicoDraft(draft), true)
  assert.equal(isPicoProgram(draft), false)
  const analysis = inferPicolang(draft)
  assert.equal(
    analysis.diagnostics.some((d) => d.severity === 'error'),
    false,
  )
  assert.equal(isPicoProgram(analysis.program), true)
  assert.deepEqual(
    declarations.map((d) => analysis.types[d.ident.hintId]),
    ['bool', 'bool', 'bool', 'bool'],
  )
  assert.deepEqual(picoDraft(analysis.program!), draft)
  assert.deepEqual(inferPicolang(picoDraft(analysis.program!)).program, analysis.program)
  const compiled = analysis.program!
  const first = programFunctions(compiled).main.body.stmts[0]
  assert.equal(first.kind, 'if')
  const missingType = first.then.stmts[0]
  assert.equal(missingType.kind, 'declare')
  delete missingType.type
  assert.equal(isPicoProgram(compiled), false)
  assert.equal(isPicoDraft(compiled), true)
})
