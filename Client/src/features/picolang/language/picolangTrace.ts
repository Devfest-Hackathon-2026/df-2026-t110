import type { components } from '../../../api/schema.d.ts'
import { TraceMemory } from './picolangMemory.ts'
import type { PicoProgram } from './picolang.ts'

type DebugEvent = components['schemas']['event.schema']
type FinishedEvent = Extract<DebugEvent, { kind: 'execution_finished' }>
export type PicolangEvent = components['schemas']['picolang-event.schema']
export interface TraceNode {
  hintId: string
  functionName: string
  kind: string
  name?: string
  func?: string
  value?: boolean | number | null | string
}
export interface TraceVariable {
  hintId: string
  name: string
  functionName: string
  depth: number
  value: string
}
export interface PicolangTrace {
  current: TraceNode | null
  trail: TraceNode[]
  variables: TraceVariable[]
  changedHintId: string | null
  finished: FinishedEvent | null
}

export function traceProgramIndex(program: PicoProgram): {
  nodes: Map<string, TraceNode>
  variables: Map<string, { name: string; functionName: string }>
} {
  const nodes = new Map<string, TraceNode>()
  const variables = new Map<string, { name: string; functionName: string }>()
  function visit(value: unknown, functionName: string): void {
    if (Array.isArray(value)) {
      value.forEach((child) => visit(child, functionName))
      return
    }
    if (!value || typeof value !== 'object') return
    const node = value as Record<string, unknown>
    if (typeof node.hintId === 'string' && typeof node.kind === 'string') {
      nodes.set(node.hintId, {
        hintId: node.hintId,
        functionName,
        kind: node.kind,
        ...(typeof node.name === 'string' ? { name: node.name } : {}),
        ...(typeof node.func === 'string' ? { func: node.func } : {}),
        ...(node.kind === 'literal' ? { value: node.value as TraceNode['value'] } : {}),
      })
      if (node.kind === 'declare' || node.kind === 'for' || node.kind === 'param') {
        const ident = node.ident as { hintId: string; name: string }
        variables.set(ident.hintId, { name: ident.name, functionName })
      }
    }
    Object.values(node).forEach((child) => visit(child, functionName))
  }
  for (const [name, fn] of Object.entries(program.picolang.v0.program)) visit(fn, name)
  return { nodes, variables }
}

// Reconstruct language execution state independently of any editor or field.
export class PicolangTraceReplay {
  private index: ReturnType<typeof traceProgramIndex>
  private interpreters = new Map<
    string,
    {
      memory: TraceMemory
      bindings: Map<string, string[]>
      current: TraceNode | null
      trail: TraceNode[]
    }
  >()
  constructor(program: PicoProgram) {
    this.index = traceProgramIndex(program)
  }

  apply(event: PicolangEvent): PicolangTrace | null {
    let state = this.interpreters.get(event.interpreterId)
    if (!state) {
      state = { memory: new TraceMemory(), bindings: new Map(), current: null, trail: [] }
      this.interpreters.set(event.interpreterId, state)
    }
    const debug = event.event
    state.memory.apply(debug)
    if (debug.kind.startsWith('memory_')) return null
    let changedHintId: string | null = null
    let finished: FinishedEvent | null = null
    if (debug.kind === 'current_node') {
      state.current = this.index.nodes.get(debug.hintId) ?? null
      if (state.current) state.trail = [...state.trail.slice(-5), state.current]
    } else if (debug.kind === 'variable_init') {
      const stack = state.bindings.get(debug.hintId) ?? []
      stack.push(debug.valueId)
      state.bindings.set(debug.hintId, stack)
      changedHintId = debug.hintId
    } else if (debug.kind === 'variable_update' || debug.kind === 'variable_drop') {
      const stack = state.bindings.get(debug.hintId)
      if (!stack?.length) throw new Error('実行履歴の変数の対応が不正です')
      if (debug.kind === 'variable_update') stack[stack.length - 1] = debug.valueId
      else stack.pop()
      changedHintId = debug.hintId
    } else if (debug.kind === 'execution_finished') {
      finished = debug
      state.current = this.index.nodes.get(debug.hintId) ?? state.current
    } else throw new Error('未対応の実行履歴です')
    const variables: TraceVariable[] = []
    for (const [hintId, stack] of state.bindings) {
      const definition = this.index.variables.get(hintId)
      if (!definition) continue
      stack.forEach((id, i) =>
        variables.push({ hintId, ...definition, depth: i + 1, value: state.memory.format(id) }),
      )
    }
    return { current: state.current, trail: state.trail, variables, changedHintId, finished }
  }
}
