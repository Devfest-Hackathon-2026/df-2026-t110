import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import type { components } from '../src/api/schema.d.ts'
import {
  newProgram,
  makeStatement,
  programFunctions,
  ident,
  id,
  block,
} from '../src/features/picolang/language/picolang.ts'
import { PicolangTraceReplay, traceProgramIndex } from '../src/features/picolang/language/picolangTrace.ts'
import { picolangTraceSteps } from '../src/features/picolang/editor/playback/picolangTimeline.ts'
import { TraceMemory } from '../src/features/picolang/language/picolangMemory.ts'
import { parseTutorialResult } from '../src/features/tutorial/tutorialRunApi.ts'

type Debug = components['schemas']['event.schema']
const debug = (event: Debug, interpreterId = 'robot-1') => ({ type: 'picolang', interpreterId, event })
const initial = {
  type: 'field',
  tick: 0,
  state: {
    kind: 'initial',
    terrain: [[{ type: 'land', kind: { type: 'plain' } }]],
    robots: [{ id: 'robot-1', position: { x: 0, y: 0 }, direction: 'east' }],
  },
}
function fixture() {
  const program = newProgram()
  const declaration = makeStatement('declare', programFunctions(program))
  assert.equal(declaration.kind, 'declare')
  declaration.ident.name = 'items'
  programFunctions(program).main.body.stmts.unshift(declaration)
  return { program, declaration, hintId: declaration.ident.hintId }
}

test('shared immutable and nested stacks retain values until their last reference is released', () => {
  const memory = new TraceMemory()
  const events: Debug[] = [
    { kind: 'memory_allocate', id: '1', value: { type: 'integer', value: 1 } },
    { kind: 'memory_allocate', id: '2', value: { type: 'list_node', prev: '0', value: '1' } },
    { kind: 'memory_release', id: '1' },
    { kind: 'memory_allocate', id: '3', value: { type: 'integer', value: 3 } },
    { kind: 'memory_allocate', id: '4', value: { type: 'list_node', prev: '2', value: '3' } },
    { kind: 'memory_release', id: '3' },
    { kind: 'memory_allocate', id: '5', value: { type: 'list_node', prev: '0', value: '2' } },
    { kind: 'memory_release', id: '2' },
  ]
  events.forEach((event) => memory.apply(event))
  assert.equal(memory.format('4'), '[1, 3]')
  assert.equal(memory.format('5'), '[[1]]')
  memory.apply({ kind: 'memory_release', id: '4' })
  assert.equal(memory.format('5'), '[[1]]')
  memory.apply({ kind: 'memory_release', id: '5' })
  assert.throws(() => memory.format('1'), /見つかりません/)
  assert.throws(() => memory.format('2'), /見つかりません/)
  assert.equal(memory.format('0'), '[]')
  assert.throws(() => memory.apply({ kind: 'memory_release', id: '0' }), /参照が不正/)
})

test('trace preserves field/debug order and recursive bindings; snapshots do not change retroactively', () => {
  const { program, declaration, hintId } = fixture()
  const steps = [
    ...picolangTraceSteps(
      [
        initial,
        debug({ kind: 'current_node', hintId: declaration.hintId }),
        debug({ kind: 'memory_allocate', id: '18446744073709551615', value: { type: 'integer', value: 7 } }),
        debug({ kind: 'variable_init', hintId, valueId: '18446744073709551615' }),
        { type: 'field', tick: 1, state: { kind: 'update', actions: [] } },
        debug({ kind: 'memory_allocate', id: '2', value: { type: 'bool', value: true } }),
        debug({ kind: 'variable_init', hintId, valueId: '2' }),
        debug({ kind: 'variable_update', hintId, valueId: '0' }),
        debug({ kind: 'variable_drop', hintId }),
        debug({ kind: 'variable_drop', hintId }),
        debug({
          kind: 'execution_finished',
          hintId: declaration.hintId,
          reason: 'step_limit',
          message: 'limit',
        }),
      ],
      program,
    ),
  ]
  assert.deepEqual(
    steps.map((step) => step.kind),
    ['field', 'debug', 'debug', 'field', 'debug', 'debug', 'debug', 'debug', 'debug'],
  )
  const traces = steps.filter((step) => step.kind === 'debug').map((step) => step.trace)
  assert.equal(traces[0].current?.kind, 'declare')
  assert.deepEqual(
    traces[1].variables.map((v) => v.value),
    ['7'],
  )
  assert.deepEqual(
    traces[2].variables.map((v) => [v.depth, v.value]),
    [
      [1, '7'],
      [2, 'true'],
    ],
  )
  assert.deepEqual(
    traces[3].variables.map((v) => v.value),
    ['7', '[]'],
  )
  assert.deepEqual(
    traces[4].variables.map((v) => v.value),
    ['7'],
  )
  assert.deepEqual(traces[5].variables, [])
  assert.equal(traces[6].finished?.reason, 'step_limit')
})

test('interpreter memories and bindings remain independent even with identical IDs', () => {
  const { program, hintId } = fixture()
  const steps = [
    ...picolangTraceSteps(
      [
        initial,
        debug({ kind: 'memory_allocate', id: '1', value: { type: 'integer', value: 1 } }, 'a'),
        debug({ kind: 'variable_init', hintId, valueId: '1' }, 'a'),
        debug({ kind: 'memory_allocate', id: '1', value: { type: 'integer', value: 9 } }, 'b'),
        debug({ kind: 'variable_init', hintId, valueId: '1' }, 'b'),
        debug({ kind: 'variable_update', hintId, valueId: '0' }, 'a'),
      ],
      program,
    ),
  ]
  assert.deepEqual(
    steps.filter((step) => step.kind === 'debug').map((step) => step.trace.variables[0].value),
    ['1', '9', '[]'],
  )
})

test('function parameters, local declarations, loops and nested nodes map back to source hints', () => {
  const { program, declaration, hintId } = fixture()
  const fns = programFunctions(program)
  const loop = makeStatement('for', fns)
  fns.main.body.stmts.unshift(loop)
  assert.equal(loop.kind, 'for')
  fns.helper = {
    kind: 'function',
    hintId: id(),
    params: [{ kind: 'param', hintId: id(), ident: ident('input') }],
    body: block([declaration]),
  }
  const index = traceProgramIndex(program)
  assert.deepEqual(index.variables.get(hintId), { name: 'items', functionName: 'helper' })
  assert.equal(index.variables.get(loop.ident.hintId)?.name, loop.ident.name)
  assert.equal(index.variables.get(fns.helper.params[0].ident.hintId)?.name, 'input')
  assert.equal(index.nodes.get(declaration.value.hintId)?.functionName, 'helper')
})

test('schema history can be replayed, while field-only histories remain compatible', () => {
  const history = JSON.parse(
    readFileSync(new URL('../../Schema/game_event/examples/history.json', import.meta.url), 'utf8'),
  )
  assert.ok([...picolangTraceSteps(history, newProgram())].some((step) => step.kind === 'debug'))
  assert.equal([...picolangTraceSteps([initial], newProgram())][0].kind, 'field')
  assert.throws(() => [...picolangTraceSteps([], newProgram())], /初期盤面/)
})

test('watch values preserve exact i64 digits from the SSE JSON', () => {
  const { program, hintId } = fixture()
  const result = parseTutorialResult(
    JSON.stringify({
      history: [
        initial,
        debug({ kind: 'memory_allocate', id: '1', value: { type: 'integer', value: 0 } }),
        debug({ kind: 'variable_init', hintId, valueId: '1' }),
      ],
      summary: { cleared: false, tick: 1, reason: 'all_stopped' },
    }).replace('"value":0', '"value":9223372036854775807'),
  )
  const last = [...picolangTraceSteps(result.history, program)].at(-1)!
  assert.equal(last.kind, 'debug')
  assert.equal(last.trace.variables[0].value, '9223372036854775807')
})

test('long stacks have bounded display and can be released without recursive stack overflow', () => {
  const memory = new TraceMemory()
  memory.apply({ kind: 'memory_allocate', id: '1', value: { type: 'bool', value: true } })
  let prev = '0'
  for (let i = 2; i < 5000; i++) {
    const id = String(i)
    memory.apply({ kind: 'memory_allocate', id, value: { type: 'list_node', prev, value: '1' } })
    if (prev !== '0') memory.apply({ kind: 'memory_release', id: prev })
    prev = id
  }
  assert.ok(memory.format(prev).length < 1000)
  assert.match(memory.format(prev), /…/)
  memory.apply({ kind: 'memory_release', id: prev })
  assert.equal(memory.format('1'), 'true')
  assert.throws(() => memory.format(prev), /見つかりません/)
})

test('language replay reconstructs nodes and variables without a field or editor', () => {
  const { program, declaration, hintId } = fixture()
  const replay = new PicolangTraceReplay(program)
  const apply = (event: Debug) => replay.apply({ type: 'picolang', interpreterId: 'robot-1', event })
  assert.equal(apply({ kind: 'current_node', hintId: declaration.hintId })?.current?.kind, 'declare')
  assert.equal(apply({ kind: 'memory_allocate', id: '1', value: { type: 'integer', value: 42 } }), null)
  assert.equal(apply({ kind: 'variable_init', hintId, valueId: '1' })?.variables[0].value, '42')
  assert.deepEqual(apply({ kind: 'variable_drop', hintId })?.variables, [])
})
