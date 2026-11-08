import test from 'node:test'
import assert from 'node:assert/strict'
import { analyzePicolang } from '../src/features/picolang/language/analyzePicolang.ts'
import {
  block,
  defaultExpr,
  id,
  ident,
  newProgram,
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
    declare('x', 'integer', defaultExpr('bool')),
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
  programFunctions(program).other = fn([ret(defaultExpr('bool'))], 'integer')
  assert.equal(analyzePicolang(program).filter((d) => d.code === 'type_mismatch').length, 8)
})

test('user function argument counts and compound return types are checked', () => {
  const program = source(
    call('other'),
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
  )
  assert.equal(
    analyzePicolang(program).some((d) => d.severity === 'error'),
    false,
  )
})

test('incompatible nested stack elements are rejected', () => {
  const program = source(
    declare('items', stack(stack('bool')), call('new')),
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
