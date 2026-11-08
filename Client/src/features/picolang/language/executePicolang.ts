import type {
  PicoBlock,
  PicoExpr,
  PicoProgram,
  PicoStatement,
  PicoDistanceTarget,
  PicoTerrain,
} from './picolang.ts'

export type PicoValue =
  boolean | number | null | 'left' | 'right' | PicoDistanceTarget | PicoValue[] | PicoTerrain
export interface PicoRuntimeHost {
  call(name: string, args: PicoValue[], signal: AbortSignal): Promise<PicoValue>
}
class ReturnValue {
  readonly value: PicoValue
  constructor(value: PicoValue) {
    this.value = value
  }
}
class BreakLoop {}
class ContinueLoop {}

const number = (value: PicoValue): number => {
  if (typeof value !== 'number' || !Number.isSafeInteger(value)) throw new Error('整数が必要です')
  return value
}
const bool = (value: PicoValue): boolean => {
  if (typeof value !== 'boolean') throw new Error('真理値が必要です')
  return value
}
const stack = (value: PicoValue): PicoValue[] => {
  if (!Array.isArray(value)) throw new Error('stack が必要です')
  return value
}
const clone = (value: PicoValue): PicoValue =>
  Array.isArray(value) ? value.map(clone) : typeof value === 'object' && value !== null ? { ...value } : value

export async function executePicolang(
  source: PicoProgram,
  host: PicoRuntimeHost,
  signal: AbortSignal,
  maxSteps = 10000,
): Promise<void> {
  const functions = source.picolang.v0.program
  let steps = 0
  let depth = 0
  const step = async (): Promise<void> => {
    if (signal.aborted) throw new DOMException('実行を中断しました', 'AbortError')
    if (++steps > maxSteps) throw new Error('実行ステップ数の上限に達しました')
    if (steps % 128 === 0) await new Promise<void>((resolve) => setTimeout(resolve, 0))
  }

  async function invoke(name: string, args: PicoValue[]): Promise<PicoValue> {
    await step()
    const definition = functions[name]
    if (definition) {
      if (++depth > 64) throw new Error('関数呼び出しが深すぎます')
      try {
        if (args.length !== definition.params.length) throw new Error(`${name} の引数の数が異なります`)
        const scopes = [
          new Map(definition.params.map((param, index) => [param.ident.name, clone(args[index])] as const)),
        ]
        try {
          await runBlock(definition.body, scopes, 0)
          return null
        } catch (error) {
          if (error instanceof ReturnValue) return clone(error.value)
          throw error
        }
      } finally {
        depth--
      }
    }
    if (name === 'not') return !bool(args[0])
    if (name === 'and') return bool(args[0]) && bool(args[1])
    if (name === 'or') return bool(args[0]) || bool(args[1])
    if (name === 'bool_eq' || name === 'integer_eq') return args[0] === args[1]
    if (name === 'bool_neq' || name === 'integer_neq') return args[0] !== args[1]
    if (name === 'add') return number(args[0]) + number(args[1])
    if (name === 'minus') return -number(args[0])
    if (name === 'sub') return number(args[0]) - number(args[1])
    if (name === 'mul') return number(args[0]) * number(args[1])
    if (name === 'div') {
      const divisor = number(args[1])
      if (divisor === 0) throw new Error('0では割れません')
      return Math.trunc(number(args[0]) / divisor)
    }
    if (name === 'gt') return number(args[0]) > number(args[1])
    if (name === 'lt') return number(args[0]) < number(args[1])
    if (name === 'geq') return number(args[0]) >= number(args[1])
    if (name === 'leq') return number(args[0]) <= number(args[1])
    if (name === 'random_bool') return Math.random() < 0.5
    if (name === 'random_integer') {
      const start = number(args[0]),
        end = number(args[1])
      if (end < start) throw new Error('乱数の範囲が逆です')
      return start + Math.floor(Math.random() * (end - start + 1))
    }
    if (name === 'new') return []
    if (name === 'push_back') return [...stack(args[0]).map(clone), clone(args[1])]
    if (name === 'pop_back') {
      const items = stack(args[0])
      if (!items.length) throw new Error('空のstackから取り出せません')
      return items.slice(0, -1).map(clone)
    }
    if (name === 'get_back') {
      const items = stack(args[0])
      if (!items.length) throw new Error('空のstackから取り出せません')
      return clone(items.at(-1)!)
    }
    if (name === 'size') return stack(args[0]).length
    return host.call(name, args, signal)
  }

  async function evaluate(expr: PicoExpr, scopes: Map<string, PicoValue>[]): Promise<PicoValue> {
    await step()
    if (expr.kind === 'literal') return expr.value
    if (expr.kind === 'ident') {
      for (let i = scopes.length - 1; i >= 0; i--)
        if (scopes[i].has(expr.name)) return clone(scopes[i].get(expr.name)!)
      throw new Error(`変数 ${expr.name} が見つかりません`)
    }
    const args: PicoValue[] = []
    for (const arg of expr.args) args.push(await evaluate(arg, scopes))
    return invoke(expr.func, args)
  }

  async function runBlock(
    body: PicoBlock,
    scopes: Map<string, PicoValue>[],
    loopDepth: number,
  ): Promise<void> {
    await step()
    scopes.push(new Map())
    try {
      for (const stmt of body.stmts) await runStatement(stmt, scopes, loopDepth)
    } finally {
      scopes.pop()
    }
  }

  async function runStatement(
    stmt: PicoStatement,
    scopes: Map<string, PicoValue>[],
    loopDepth: number,
  ): Promise<void> {
    await step()
    if (stmt.kind === 'call' || stmt.kind === 'literal' || stmt.kind === 'ident') {
      await evaluate(stmt, scopes)
      return
    }
    if (stmt.kind === 'declare') {
      if (scopes.some((scope) => scope.has(stmt.ident.name)))
        throw new Error(`変数 ${stmt.ident.name} は既にあります`)
      scopes.at(-1)!.set(stmt.ident.name, clone(await evaluate(stmt.value, scopes)))
      return
    }
    if (stmt.kind === 'assign') {
      const scope = [...scopes].reverse().find((item) => item.has(stmt.ident.name))
      if (!scope) throw new Error(`変数 ${stmt.ident.name} が見つかりません`)
      scope.set(stmt.ident.name, clone(await evaluate(stmt.value, scopes)))
      return
    }
    if (stmt.kind === 'if') {
      if (bool(await evaluate(stmt.condition, scopes))) await runBlock(stmt.then, scopes, loopDepth)
      else if (stmt.else) await runBlock(stmt.else, scopes, loopDepth)
      return
    }
    if (stmt.kind === 'while') {
      while (bool(await evaluate(stmt.condition, scopes))) {
        try {
          await runBlock(stmt.body, scopes, loopDepth + 1)
        } catch (error) {
          if (error instanceof BreakLoop) break
          if (!(error instanceof ContinueLoop)) throw error
        }
      }
      return
    }
    if (stmt.kind === 'for') {
      const start = number(await evaluate(stmt.start, scopes)),
        end = number(await evaluate(stmt.end, scopes))
      const loopScope = new Map<string, PicoValue>()
      scopes.push(loopScope)
      try {
        for (let index = start; index < end; index++) {
          loopScope.set(stmt.ident.name, index)
          try {
            await runBlock(stmt.body, scopes, loopDepth + 1)
          } catch (error) {
            if (error instanceof BreakLoop) break
            if (!(error instanceof ContinueLoop)) throw error
          }
        }
      } finally {
        scopes.pop()
      }
      return
    }
    if (stmt.kind === 'return') throw new ReturnValue(await evaluate(stmt.value, scopes))
    if (loopDepth === 0) throw new Error(`${stmt.kind} はくり返しの中で使います`)
    if (stmt.kind === 'break') throw new BreakLoop()
    throw new ContinueLoop()
  }
  await invoke('main', [])
}
