import {
  PICO_BUILTINS,
  programFunctions,
  walkPicoStatements,
  type PicoBlock,
  type PicoExpr,
  type PicoIdent,
  type PicoProgram,
  type PicoStatement,
  type PicoType,
  type PicoTypedProgram,
} from './picolang.ts'

export type PicoDiagnosticCode =
  | 'undefined_variable'
  | 'undefined_function'
  | 'shadowing_variable'
  | 'shadowed_variable'
  | 'unused_variable'
  | 'use_before_definition'
  | 'out_of_scope_reference'
  | 'type_mismatch'
  | 'argument_count'
  | 'cannot_infer_type'
  | 'function_conflict'
  | 'function_name_conflict'
  | 'invalid_name'
  | 'invalid_loop_control'

export interface PicoDiagnostic {
  code: PicoDiagnosticCode
  severity: 'error' | 'warning'
  functionName: string
  hintId: string
  relatedHintId?: string
  message: string
}

// Function signatures, stacks and options use the same first-order type constructors.
type Term = string | { kind: 'compound'; name: string; args: Term[] } | TypeVariable
interface TypeVariable {
  kind: 'variable'
  binding?: Term
}
const invalid = '<invalid>'
const compound = (name: string, args: Term[]): Term => ({ kind: 'compound', name, args })
const stack = (element: Term): Term => compound('stack', [element])
const functionType = (args: Term[], result: Term): Term => compound('function', [...args, result])
const term = (type: PicoType): Term =>
  typeof type === 'string' ? type : compound(type.kind, [term(type.elementType)])
function prune(type: Term): Term {
  if (typeof type !== 'string' && type.kind === 'variable' && type.binding !== undefined) {
    type.binding = prune(type.binding)
    return type.binding
  }
  return type
}
function occurs(variable: TypeVariable, type: Term): boolean {
  type = prune(type)
  return (
    type === variable ||
    (typeof type !== 'string' && type.kind === 'compound' && type.args.some((arg) => occurs(variable, arg)))
  )
}
function invalidate(type: Term): void {
  type = prune(type)
  if (typeof type === 'string') return
  if (type.kind === 'variable') type.binding = invalid
  else type.args.forEach(invalidate)
}
function unify(expected: Term, actual: Term): boolean {
  expected = prune(expected)
  actual = prune(actual)
  if (expected === actual) return true
  if (expected === invalid || actual === invalid) {
    invalidate(expected)
    invalidate(actual)
    return true
  }
  if (typeof expected !== 'string' && expected.kind === 'variable') {
    if (occurs(expected, actual)) return false
    expected.binding = actual
    return true
  }
  if (typeof actual !== 'string' && actual.kind === 'variable') return unify(actual, expected)
  return (
    typeof expected !== 'string' &&
    typeof actual !== 'string' &&
    expected.kind === 'compound' &&
    actual.kind === 'compound' &&
    expected.name === actual.name &&
    expected.args.length === actual.args.length &&
    expected.args.every((arg, index) => unify(arg, actual.args[index]))
  )
}
function typeName(type: Term): string {
  type = prune(type)
  if (typeof type === 'string') return type
  if (type.kind === 'variable') return '?'
  return `${type.name}<${type.args.map(typeName).join(', ')}>`
}

interface Scope {
  parent?: Scope
}
interface Binding {
  ident: PicoIdent
  type: Term
  scope: Scope
  order: number
  used: boolean
}
interface Reference {
  ident: PicoIdent
  scope: Scope
  order: number
}
function containsScope(ancestor: Scope, scope: Scope): boolean {
  for (let current: Scope | undefined = scope; current; current = current.parent)
    if (current === ancestor) return true
  return false
}

export interface PicoInferredSignature {
  params: Array<PicoType | undefined>
  returns: PicoType | undefined
}
export interface PicoAnalysis {
  diagnostics: PicoDiagnostic[]
  signatures: Record<string, PicoInferredSignature>
  types: Record<string, PicoType | undefined>
  program: PicoTypedProgram | null
}
function resolvedType(type: Term): PicoType | undefined {
  type = prune(type)
  if (typeof type === 'string') return type === invalid ? undefined : (type as PicoType)
  if (type.kind === 'variable') return undefined
  const elementType = resolvedType(type.args[0])
  return elementType && (type.name === 'stack' || type.name === 'option')
    ? { kind: type.name, elementType }
    : undefined
}

/** Share monomorphic function signatures across every body and call, including recursive calls. */
export function inferPicolang(program: PicoProgram): PicoAnalysis {
  const diagnostics: PicoDiagnostic[] = []
  const functions = programFunctions(program)
  const builtins = new Map(PICO_BUILTINS.map((builtin) => [builtin.name as string, builtin]))
  const bindingTypes = new Map<string, Term>()
  const signatures = new Map<string, { params: Term[]; result: Term }>()
  const variables: Array<{ variable: TypeVariable; hintId: string; functionName: string }> = []
  function fresh(functionName: string, hintId: string): TypeVariable {
    const variable: TypeVariable = { kind: 'variable' }
    variables.push({ variable, hintId, functionName })
    return variable
  }
  for (const [name, fn] of Object.entries(functions))
    signatures.set(name, {
      params: fn.params.map((param) => fresh(name, param.ident.hintId)),
      result: name === 'main' ? 'unit' : fresh(name, fn.hintId),
    })

  for (const functionName of Object.keys(functions).sort()) {
    const fn = functions[functionName]
    const bindings: Binding[] = []
    const unresolved: Reference[] = []
    const active = new Map<string, Binding>()
    let order = 0
    const report = (
      code: PicoDiagnosticCode,
      severity: PicoDiagnostic['severity'],
      hintId: string,
      message: string,
      relatedHintId?: string,
    ): void => {
      diagnostics.push({ code, severity, functionName, hintId, message, relatedHintId })
    }
    function expect(hintId: string, expected: Term, actual: Term): void {
      const want = typeName(expected),
        got = typeName(actual)
      if (unify(expected, actual)) return
      report(
        'type_mismatch',
        'error',
        hintId,
        `型が合いません。${want} が必要ですが、${got} になっています。`,
      )
      invalidate(expected)
      invalidate(actual)
    }
    function declare(ident: PicoIdent, type: Term, scope: Scope): Binding | undefined {
      const binding: Binding = { ident, type, scope, order: order++, used: false }
      bindings.push(binding)
      bindingTypes.set(ident.hintId, type)
      const previous = active.get(ident.name)
      if (!ident.name.trim()) report('invalid_name', 'error', ident.hintId, '変数の名前を入力してください。')
      else if (previous) {
        report(
          'shadowing_variable',
          'error',
          ident.hintId,
          `変数「${ident.name}」はすでに使われています。別の名前にしてください。`,
          previous.ident.hintId,
        )
        report(
          'shadowed_variable',
          'warning',
          previous.ident.hintId,
          `変数「${ident.name}」と同じ名前の変数が後で作られています。`,
          ident.hintId,
        )
      } else if (builtins.has(ident.name) || signatures.has(ident.name)) {
        report(
          'function_name_conflict',
          'error',
          ident.hintId,
          `「${ident.name}」は関数の名前として使われています。`,
        )
      } else {
        active.set(ident.name, binding)
        return binding
      }
    }
    function reference(ident: PicoIdent, scope: Scope, read: boolean): Term {
      const position = order++
      const binding = active.get(ident.name)
      if (binding) {
        if (read) binding.used = true
        return binding.type
      }
      report(
        'undefined_variable',
        'error',
        ident.hintId,
        `変数「${ident.name}」はここでは定義されていません。`,
      )
      unresolved.push({ ident, scope, order: position })
      return invalid
    }
    function signature(name: string, hintId: string): Term | undefined {
      const builtin = builtins.get(name)
      if (!builtin) {
        const signature = signatures.get(name)
        return signature ? functionType(signature.params, signature.result) : undefined
      }
      if (['new', 'push_back', 'pop_back', 'get_back', 'size'].includes(name)) {
        const variable = fresh(functionName, hintId)
        const value = stack(variable)
        switch (name) {
          case 'new':
            return functionType([], value)
          case 'push_back':
            return functionType([value, variable], value)
          case 'pop_back':
            return functionType([value], value)
          case 'get_back':
            return functionType([value], variable)
          default:
            return functionType([value], 'integer')
        }
      }
      if (['some', 'none', 'unwrap', 'is_some'].includes(name)) {
        const variable = fresh(functionName, hintId)
        const value = compound('option', [variable])
        switch (name) {
          case 'some':
            return functionType([variable], value)
          case 'none':
            return functionType([], value)
          case 'unwrap':
            return functionType([value], variable)
          default:
            return functionType([value], 'bool')
        }
      }
      return functionType(builtin.args.map(term), term(builtin.returns))
    }
    function expression(expr: PicoExpr, scope: Scope): Term {
      if (expr.kind === 'literal') return expr.type
      if (expr.kind === 'ident') return reference(expr, scope, true)
      const args = expr.args.map((arg) => expression(arg, scope))
      const callee = signature(expr.func, expr.hintId)
      if (!callee || typeof callee === 'string' || callee.kind !== 'compound') {
        report('undefined_function', 'error', expr.hintId, `関数「${expr.func}」が定義されていません。`)
        args.forEach(invalidate)
        return invalid
      }
      const params = callee.args.slice(0, -1)
      const result = callee.args.at(-1)!
      if (params.length !== args.length) {
        report(
          'argument_count',
          'error',
          expr.hintId,
          `関数「${expr.func}」には ${params.length} 個の引数が必要ですが、${args.length} 個渡されています。`,
        )
        if (builtins.has(expr.func)) invalidate(callee)
        args.forEach(invalidate)
        return invalid
      }
      args.forEach((arg, index) => {
        if (index < params.length) expect(expr.args[index].hintId, params[index], arg)
      })
      return result
    }
    function visitBlock(block: PicoBlock, parent: Scope, loopDepth: number): void {
      const scope: Scope = { parent }
      const locals: Binding[] = []
      for (const stmt of block.stmts) statement(stmt, scope, loopDepth, locals)
      for (const binding of locals) active.delete(binding.ident.name)
    }
    function statement(stmt: PicoStatement, scope: Scope, loopDepth: number, locals: Binding[]): void {
      switch (stmt.kind) {
        case 'literal':
        case 'ident':
        case 'call':
          expression(stmt, scope)
          break
        case 'declare': {
          const actual = expression(stmt.value, scope)
          const inferred = fresh(functionName, stmt.ident.hintId)
          expect(stmt.value.hintId, inferred, actual)
          const binding = declare(stmt.ident, inferred, scope)
          if (binding) locals.push(binding)
          break
        }
        case 'assign':
          expect(stmt.value.hintId, reference(stmt.ident, scope, false), expression(stmt.value, scope))
          break
        case 'return':
          expect(stmt.value.hintId, signatures.get(functionName)!.result, expression(stmt.value, scope))
          break
        case 'if':
          expect(stmt.condition.hintId, 'bool', expression(stmt.condition, scope))
          visitBlock(stmt.then, scope, loopDepth)
          if (stmt.else) visitBlock(stmt.else, scope, loopDepth)
          break
        case 'while':
          expect(stmt.condition.hintId, 'bool', expression(stmt.condition, scope))
          visitBlock(stmt.body, scope, loopDepth + 1)
          break
        case 'for': {
          expect(stmt.start.hintId, 'integer', expression(stmt.start, scope))
          expect(stmt.end.hintId, 'integer', expression(stmt.end, scope))
          const loopScope: Scope = { parent: scope }
          const binding = declare(stmt.ident, 'integer', loopScope)
          visitBlock(stmt.body, loopScope, loopDepth + 1)
          if (binding) active.delete(binding.ident.name)
          break
        }
        case 'break':
        case 'continue':
          if (loopDepth === 0)
            report('invalid_loop_control', 'error', stmt.hintId, 'この命令はくり返しの中で使ってください。')
      }
    }

    if (!functionName.trim()) report('invalid_name', 'error', fn.hintId, '関数の名前を入力してください。')
    if (builtins.has(functionName))
      report('function_conflict', 'error', fn.hintId, `「${functionName}」は組み込み関数の名前です。`)
    const parameterScope: Scope = {}
    fn.params.forEach((param, index) =>
      declare(param.ident, signatures.get(functionName)!.params[index], parameterScope),
    )
    if (functionName === 'main' && fn.params.length)
      report('argument_count', 'error', fn.hintId, 'main 関数は引数を受け取れません。')
    visitBlock(fn.body, parameterScope, 0)
    // Classify unresolved uses after collecting declarations, including future ones.
    for (const ref of unresolved) {
      const candidates = bindings.filter((binding) => binding.ident.name === ref.ident.name)
      const future = candidates.find(
        (binding) => binding.order > ref.order && containsScope(binding.scope, ref.scope),
      )
      const expired = candidates.find(
        (binding) =>
          binding.order < ref.order && binding.scope !== ref.scope && containsScope(ref.scope, binding.scope),
      )
      if (future)
        report(
          'use_before_definition',
          'warning',
          ref.ident.hintId,
          `変数「${ref.ident.name}」は、この場所より後で作られています。`,
          future.ident.hintId,
        )
      else if (expired)
        report(
          'out_of_scope_reference',
          'warning',
          ref.ident.hintId,
          `変数「${ref.ident.name}」は内側のブロックで作られているため、ここでは使えません。`,
          expired.ident.hintId,
        )
    }
    for (const binding of bindings)
      if (!binding.used)
        report(
          'unused_variable',
          'warning',
          binding.ident.hintId,
          `変数「${binding.ident.name}」の値が使われていません。`,
        )
  }
  // Resolve only after every function and call has contributed constraints.
  const reported = new Set<TypeVariable>()
  function reportUnresolved(type: Term, functionName: string, hintId: string): void {
    const root = prune(type)
    if (typeof root === 'string') return
    if (root.kind === 'compound') {
      root.args.forEach((arg) => reportUnresolved(arg, functionName, hintId))
      return
    }
    if (reported.has(root)) return
    reported.add(root)
    diagnostics.push({
      code: 'cannot_infer_type',
      severity: 'error',
      functionName,
      hintId,
      message: '型を決められません。関数本体や呼び出し先で値を使い、型が分かる値と組み合わせてください。',
    })
  }
  for (const { variable, functionName, hintId } of variables) reportUnresolved(variable, functionName, hintId)
  const inferred = Object.fromEntries(
    [...signatures].map(([name, signature]) => [
      name,
      {
        params: signature.params.map(resolvedType),
        returns: resolvedType(signature.result),
      },
    ]),
  )
  const types = Object.fromEntries([...bindingTypes].map(([hintId, type]) => [hintId, resolvedType(type)]))
  let compiled: PicoTypedProgram | null = null
  if (!diagnostics.some((item) => item.severity === 'error')) {
    compiled = JSON.parse(JSON.stringify(program)) as PicoTypedProgram
    for (const [name, fn] of Object.entries(compiled.picolang.v0.program)) {
      fn.returnType = inferred[name].returns!
      fn.params.forEach((param, index) => {
        param.type = inferred[name].params[index]!
      })
      walkPicoStatements(fn.body, (stmt) => {
        if (stmt.kind === 'declare') stmt.type = types[stmt.ident.hintId]!
      })
    }
  }
  return { diagnostics, signatures: inferred, types, program: compiled }
}

/** Analyze without mutating the authoring AST. */
export function analyzePicolang(program: PicoProgram): PicoDiagnostic[] {
  return inferPicolang(program).diagnostics
}
