import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import {
  compatibleSignatureTypes,
  instantiateSignature,
  type PicoBuiltinSignature,
  type PicoSignatureType,
} from '../src/features/picolang/language/picoSignatures.ts'
import {
  PICO_BUILTIN_SIGNATURES,
  PICO_BUILTINS,
  builtinSignature,
  defaultExpr,
  makeCall,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'
import { PICO_BUILTIN_OPTIONS } from '../src/features/picolang/editor/picoOptions.ts'
import { builtinArguments } from '../src/features/picolang/editor/picoGuidance.ts'
import { inferPicolang } from '../src/features/picolang/language/analyzePicolang.ts'
import { expressionFromDrop } from '../src/features/picolang/editor/picoDrag.ts'

test('generated signatures exactly match the authoritative header', async () => {
  const source = JSON.parse(
    await readFile(new URL('../../lang/builtins.picoheader.json', import.meta.url), 'utf8'),
  )
  assert.deepEqual(PICO_BUILTIN_SIGNATURES, source)
  assert.deepEqual(new Set(PICO_BUILTIN_OPTIONS.map((option) => option.name)), new Set(Object.keys(source)))
  assert.equal(builtinSignature('toString'), undefined)
  assert.equal(builtinSignature('__proto__'), undefined)
})

test('signature parameters share names only within each instantiation', () => {
  const a: PicoSignatureType = { kind: 'type_parameter', name: 'A' }
  const b: PicoSignatureType = { kind: 'type_parameter', name: 'B' }
  const signature: PicoBuiltinSignature = {
    typeParameters: ['A', 'B'],
    params: [{ kind: 'stack', elementType: { kind: 'option', elementType: a } }, b, a],
    returnType: { kind: 'option', elementType: b },
  }
  type Variable = { name: string }
  type Term = string | Variable | { kind: string; element: Term }
  const instances: Variable[] = []
  const instantiate = () =>
    instantiateSignature<Term>(
      signature,
      (name) => {
        const variable = { name }
        instances.push(variable)
        return variable
      },
      (primitive) => primitive,
      (kind, element) => ({ kind, element }),
    )
  const first = instantiate()
  const second = instantiate()
  assert.equal(instances.length, 4)
  assert.equal(first.params[2], instances[0])
  assert.equal(first.params[1], instances[1])
  assert.deepEqual(first.params[0], { kind: 'stack', element: { kind: 'option', element: instances[0] } })
  assert.deepEqual(first.result, { kind: 'option', element: instances[1] })
  assert.notEqual(first.params[2], first.params[1])
  assert.notEqual(first.params[2], second.params[2])
  assert.notEqual(first.params[1], second.params[1])
})

test('UI metadata covers argument counts and block message placeholders', () => {
  for (const option of PICO_BUILTIN_OPTIONS) {
    assert.equal(builtinArguments[option.name]?.length ?? 0, option.args.length, option.name)
    if (option.blockMessage) {
      const placeholders = [...option.blockMessage.matchAll(/%(\d+)/g)]
        .map((match) => Number(match[1]))
        .sort((a, b) => a - b)
      assert.deepEqual(
        placeholders,
        option.args.map((_, i) => i + 1),
        option.name,
      )
    }
  }
})

test('calls use header arity and independent generic types', () => {
  for (const builtin of PICO_BUILTINS) assert.equal(makeCall(builtin.name).args.length, builtin.args.length)
  assert.equal(makeCall('distance').args[0]?.kind, 'literal')
  assert.equal(makeCall('is_luggage').args.length, 1)
  const program = newProgram()
  const statements = programFunctions(program).main.body.stmts
  for (const literal of [defaultExpr('bool'), defaultExpr('integer')]) {
    const some = makeCall('some')
    some.args = [literal]
    const unwrap = makeCall('unwrap')
    unwrap.args = [some]
    const print = makeCall(
      literal.kind === 'literal' && literal.type === 'bool' ? 'print_bool' : 'print_integer',
    )
    print.args = [unwrap]
    statements.unshift(print)
  }
  for (const name of ['is_own_territory', 'is_enemy_territory', 'is_luggage'])
    statements.unshift(makeCall(name))
  const analysis = inferPicolang(program)
  assert.deepEqual(analysis.diagnostics, [])
  assert.ok(analysis.program)
})

test('generic return matching and drag candidates do not default T to integer', () => {
  const functions = programFunctions(newProgram())
  for (const name of ['new', 'push_back', 'pop_back']) {
    const expected: PicoSignatureType = { kind: 'stack', elementType: 'bool' }
    assert.equal(compatibleSignatureTypes(expected, builtinSignature(name)!.returnType), true)
    const transfer = {
      getData: (key: string) => (key === 'application/x-picolang-choice' ? name : ''),
    } as DataTransfer
    assert.ok(expressionFromDrop(transfer, expected, functions, []))
  }
  assert.equal(compatibleSignatureTypes('bool', builtinSignature('size')!.returnType), false)
  assert.deepEqual(defaultExpr({ kind: 'type_parameter', name: 'T' }).kind, 'literal')
})
