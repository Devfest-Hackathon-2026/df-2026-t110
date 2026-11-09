import { PICO_BUILTIN_SIGNATURES, type PicoBuiltinName } from './builtins.gen.ts'
import type { PicoBuiltinSignature, PicoSignatureType } from './picoSignatures.ts'
export { PICO_BUILTIN_SIGNATURES, type PicoBuiltinName } from './builtins.gen.ts'
export type { PicoSignatureType } from './picoSignatures.ts'

export type PicoPrimitive = 'bool' | 'integer' | 'unit' | 'terrain' | 'rotation' | 'distance_target'
export const PICO_DISTANCE_TARGETS = [
  'robot',
  'ally',
  'enemy',
  'rock',
  'luggage',
  'sea',
  'goal',
  'territory',
] as const
export type PicoDistanceTarget = (typeof PICO_DISTANCE_TARGETS)[number]
export interface PicoTerrain {
  type: 'land' | 'rock' | 'sea'
  kind?: { type: 'plain' | 'goal' | 'checkpoint'; order?: number }
  hasLuggage?: boolean
  territory?: 'own' | 'enemy'
}
export type PicoValue =
  boolean | number | null | 'left' | 'right' | PicoDistanceTarget | PicoValue[] | PicoTerrain
export type PicoType = PicoPrimitive | { kind: 'stack' | 'option'; elementType: PicoType }
interface NodeBase {
  hintId: string
}
export interface PicoIdent extends NodeBase {
  kind: 'ident'
  name: string
}
export interface PicoLiteral extends NodeBase {
  kind: 'literal'
  type: 'bool' | 'integer' | 'unit' | 'rotation' | 'distance_target'
  value: boolean | number | null | 'left' | 'right' | PicoDistanceTarget
}
export interface PicoCall extends NodeBase {
  kind: 'call'
  func: string
  args: PicoExpr[]
}
export type PicoExpr = PicoIdent | PicoLiteral | PicoCall
export interface PicoBlock extends NodeBase {
  kind: 'block'
  stmts: PicoStatement[]
}
export type PicoStatement =
  | PicoExpr
  | (NodeBase & { kind: 'declare'; type?: PicoType; ident: PicoIdent; value: PicoExpr })
  | (NodeBase & { kind: 'assign'; ident: PicoIdent; value: PicoExpr })
  | (NodeBase & { kind: 'if'; condition: PicoExpr; then: PicoBlock; else?: PicoBlock })
  | (NodeBase & { kind: 'while'; condition: PicoExpr; body: PicoBlock })
  | (NodeBase & { kind: 'for'; ident: PicoIdent; start: PicoExpr; end: PicoExpr; body: PicoBlock })
  | (NodeBase & { kind: 'return'; value: PicoExpr })
  | (NodeBase & { kind: 'break' | 'continue' })
export interface PicoFunction extends NodeBase {
  kind: 'function'
  params: Array<NodeBase & { kind: 'param'; ident: PicoIdent; type?: PicoType }>
  returnType?: PicoType
  body: PicoBlock
}
export interface PicoProgram {
  picolangVersion: 0
  picolang: { v0: { program: Record<string, PicoFunction> } }
}
export type PicoTypedStatement =
  | Exclude<PicoStatement, { kind: 'declare' | 'if' | 'while' | 'for' }>
  | (Extract<PicoStatement, { kind: 'declare' }> & { type: PicoType })
  | (Extract<PicoStatement, { kind: 'if' }> & { then: PicoTypedBlock; else?: PicoTypedBlock })
  | (Extract<PicoStatement, { kind: 'while' | 'for' }> & { body: PicoTypedBlock })
export interface PicoTypedBlock extends PicoBlock {
  stmts: PicoTypedStatement[]
}
/** Execution JSON always carries concrete types required by the server schema. */
export interface PicoTypedFunction extends PicoFunction {
  params: Array<PicoFunction['params'][number] & { type: PicoType }>
  returnType: PicoType
  body: PicoTypedBlock
}
export interface PicoTypedProgram extends PicoProgram {
  picolang: { v0: { program: Record<string, PicoTypedFunction> } }
}
export interface PicoBuiltin {
  name: PicoBuiltinName
  args: readonly PicoSignatureType[]
  returns: PicoSignatureType
}
export const PICO_BUILTINS: PicoBuiltin[] = (Object.keys(PICO_BUILTIN_SIGNATURES) as PicoBuiltinName[]).map(
  (name) => ({
    name,
    args: PICO_BUILTIN_SIGNATURES[name].params,
    returns: PICO_BUILTIN_SIGNATURES[name].returnType,
  }),
)
export function builtinSignature(name: string): PicoBuiltinSignature | undefined {
  return Object.hasOwn(PICO_BUILTIN_SIGNATURES, name)
    ? PICO_BUILTIN_SIGNATURES[name as PicoBuiltinName]
    : undefined
}

export function samePicoType(left: PicoType | undefined, right: PicoType | undefined): boolean {
  if (!left || !right) return true
  if (typeof left === 'string' || typeof right === 'string') return left === right
  return left.kind === right.kind && samePicoType(left.elementType, right.elementType)
}

export const id = (): string => crypto.randomUUID()
export const ident = (name = 'x'): PicoIdent => ({ kind: 'ident', hintId: id(), name })
export const block = (stmts: PicoStatement[] = []): PicoBlock => ({ kind: 'block', hintId: id(), stmts })
export function defaultExpr(type: PicoSignatureType = 'integer'): PicoExpr {
  if (typeof type !== 'string' && type.kind === 'type_parameter') return defaultExpr('integer')
  if (typeof type !== 'string' && type.kind === 'option')
    return { kind: 'call', hintId: id(), func: 'some', args: [defaultExpr(type.elementType)] }
  if (typeof type !== 'string') return { kind: 'call', hintId: id(), func: 'new', args: [] }
  if (type === 'bool') return { kind: 'literal', hintId: id(), type, value: false }
  if (type === 'unit') return { kind: 'literal', hintId: id(), type, value: null }
  if (type === 'rotation') return { kind: 'literal', hintId: id(), type, value: 'left' }
  if (type === 'distance_target') return { kind: 'literal', hintId: id(), type, value: 'goal' }
  if (type === 'terrain') return makeCall('search_front')
  return { kind: 'literal', hintId: id(), type: 'integer', value: 0 }
}
export function makeCall(name: string, userFunctions: Record<string, PicoFunction> = {}): PicoCall {
  const option = PICO_BUILTINS.find((item) => item.name === name)
  const args =
    option?.args.map(defaultExpr) ?? userFunctions[name]?.params.map((param) => defaultExpr(param.type)) ?? []
  return { kind: 'call', hintId: id(), func: name, args }
}
export type StatementChoice =
  'declare' | 'assign' | 'if' | 'if-else' | 'while' | 'for' | 'return' | 'break' | 'continue'
export function makeStatement(
  choice: StatementChoice | string,
  functions: Record<string, PicoFunction>,
): PicoStatement {
  switch (choice) {
    case 'declare':
      return { kind: 'declare', hintId: id(), ident: ident(), value: defaultExpr() }
    case 'assign':
      return { kind: 'assign', hintId: id(), ident: ident(), value: defaultExpr() }
    case 'if':
      return { kind: 'if', hintId: id(), condition: defaultExpr('bool'), then: block() }
    case 'if-else':
      return { kind: 'if', hintId: id(), condition: defaultExpr('bool'), then: block(), else: block() }
    case 'while':
      return { kind: 'while', hintId: id(), condition: defaultExpr('bool'), body: block() }
    case 'for':
      return {
        kind: 'for',
        hintId: id(),
        ident: ident('i'),
        start: defaultExpr(),
        end: defaultExpr(),
        body: block(),
      }
    case 'return':
      return { kind: 'return', hintId: id(), value: defaultExpr('unit') }
    case 'break':
      return { kind: 'break', hintId: id() }
    case 'continue':
      return { kind: 'continue', hintId: id() }
    default:
      return makeCall(choice, functions)
  }
}
export function newProgram(): PicoProgram {
  return {
    picolangVersion: 0,
    picolang: {
      v0: {
        program: {
          main: {
            kind: 'function',
            hintId: id(),
            params: [],
            returnType: 'unit',
            body: block([makeStatement('return', {})]),
          },
        },
      },
    },
  }
}
export function programFunctions(program: PicoProgram): Record<string, PicoFunction> {
  return program.picolang.v0.program
}
type JsonObject = Record<string, unknown>
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const primitiveTypes: PicoPrimitive[] = ['bool', 'integer', 'unit', 'terrain', 'rotation', 'distance_target']
const object = (value: unknown): value is JsonObject =>
  !!value && typeof value === 'object' && !Array.isArray(value)
const exact = (value: JsonObject, keys: string[]): boolean =>
  Object.keys(value).every((key) => keys.includes(key)) && keys.every((key) => key in value)
const name = (value: unknown): value is string => typeof value === 'string' && value.length > 0
function type(value: unknown): boolean {
  return typeof value === 'string'
    ? primitiveTypes.includes(value as PicoPrimitive)
    : object(value) &&
        exact(value, ['kind', 'elementType']) &&
        (value.kind === 'stack' || value.kind === 'option') &&
        type(value.elementType)
}
function node(value: JsonObject, kind: string, keys: string[], seen: Set<string>): boolean {
  if (!exact(value, ['kind', 'hintId', ...keys]) || value.kind !== kind) return false
  if (typeof value.hintId !== 'string' || !uuidPattern.test(value.hintId) || seen.has(value.hintId))
    return false
  seen.add(value.hintId)
  return true
}
function validIdent(value: unknown, seen: Set<string>): boolean {
  return object(value) && node(value, 'ident', ['name'], seen) && name(value.name)
}
function validExpr(value: unknown, seen: Set<string>): boolean {
  if (!object(value)) return false
  if (value.kind === 'ident') return validIdent(value, seen)
  if (value.kind === 'call')
    return (
      node(value, 'call', ['func', 'args'], seen) &&
      name(value.func) &&
      Array.isArray(value.args) &&
      value.args.every((arg) => validExpr(arg, seen))
    )
  if (value.kind !== 'literal' || !node(value, 'literal', ['type', 'value'], seen)) return false
  switch (value.type) {
    case 'bool':
      return typeof value.value === 'boolean'
    case 'integer':
      return Number.isInteger(value.value)
    case 'unit':
      return value.value === null
    case 'rotation':
      return value.value === 'left' || value.value === 'right'
    case 'distance_target':
      return PICO_DISTANCE_TARGETS.includes(value.value as PicoDistanceTarget)
    default:
      return false
  }
}
function validBlock(value: unknown, seen: Set<string>, draft: boolean): value is PicoBlock {
  return (
    object(value) &&
    node(value, 'block', ['stmts'], seen) &&
    Array.isArray(value.stmts) &&
    value.stmts.every((stmt) => validStmt(stmt, seen, draft))
  )
}
function validStmt(value: unknown, seen: Set<string>, draft: boolean): boolean {
  if (!object(value)) return false
  switch (value.kind) {
    case 'literal':
    case 'ident':
    case 'call':
      return validExpr(value, seen)
    case 'declare':
      return (
        node(value, 'declare', [...('type' in value ? ['type'] : []), 'ident', 'value'], seen) &&
        ((draft && !('type' in value)) || type(value.type)) &&
        validIdent(value.ident, seen) &&
        validExpr(value.value, seen)
      )
    case 'assign':
      return (
        node(value, 'assign', ['ident', 'value'], seen) &&
        validIdent(value.ident, seen) &&
        validExpr(value.value, seen)
      )
    case 'if':
      return (
        node(
          value,
          'if',
          value.else === undefined ? ['condition', 'then'] : ['condition', 'then', 'else'],
          seen,
        ) &&
        validExpr(value.condition, seen) &&
        validBlock(value.then, seen, draft) &&
        (value.else === undefined || validBlock(value.else, seen, draft))
      )
    case 'while':
      return (
        node(value, 'while', ['condition', 'body'], seen) &&
        validExpr(value.condition, seen) &&
        validBlock(value.body, seen, draft)
      )
    case 'for':
      return (
        node(value, 'for', ['ident', 'start', 'end', 'body'], seen) &&
        validIdent(value.ident, seen) &&
        validExpr(value.start, seen) &&
        validExpr(value.end, seen) &&
        validBlock(value.body, seen, draft)
      )
    case 'return':
      return node(value, 'return', ['value'], seen) && validExpr(value.value, seen)
    case 'break':
    case 'continue':
      return node(value, value.kind, [], seen)
    default:
      return false
  }
}
function validProgram(value: unknown, draft: boolean): boolean {
  if (!object(value) || !exact(value, ['picolangVersion', 'picolang']) || value.picolangVersion !== 0)
    return false
  if (
    !object(value.picolang) ||
    !exact(value.picolang, ['v0']) ||
    !object(value.picolang.v0) ||
    !exact(value.picolang.v0, ['program'])
  )
    return false
  const functions = value.picolang.v0.program
  if (!object(functions) || !object(functions.main)) return false
  const seen = new Set<string>()
  for (const [functionName, definition] of Object.entries(functions)) {
    if (
      !name(functionName) ||
      !object(definition) ||
      !node(
        definition,
        'function',
        ['params', ...('returnType' in definition ? ['returnType'] : []), 'body'],
        seen,
      ) ||
      !Array.isArray(definition.params) ||
      (!(draft && !('returnType' in definition)) && !type(definition.returnType))
    )
      return false
    for (const param of definition.params) {
      if (
        !object(param) ||
        !node(param, 'param', ['ident', ...('type' in param ? ['type'] : [])], seen) ||
        !validIdent(param.ident, seen) ||
        (!(draft && !('type' in param)) && !type(param.type))
      )
        return false
    }
    if (!validBlock(definition.body, seen, draft)) return false
    if (definition.body.stmts.at(-1)?.kind !== 'return') return false
  }
  return (
    (draft || functions.main.returnType === 'unit') &&
    Array.isArray(functions.main.params) &&
    functions.main.params.length === 0
  )
}

export function isPicoProgram(value: unknown): value is PicoTypedProgram {
  return validProgram(value, false)
}
export function isPicoDraft(value: unknown): value is PicoProgram {
  return validProgram(value, true)
}
/** Discard imported annotations: they are output of inference, never constraints on editing. */
export function picoDraft(program: PicoProgram): PicoProgram {
  const draft = JSON.parse(JSON.stringify(program)) as PicoProgram
  for (const fn of Object.values(programFunctions(draft))) {
    delete fn.returnType
    for (const param of fn.params) delete param.type
    walkPicoStatements(fn.body, (stmt) => {
      if (stmt.kind === 'declare') delete stmt.type
    })
  }
  return draft
}

/** Visit declarations in every nested lexical block as well as top-level statements. */
export function walkPicoStatements(body: PicoBlock, visit: (stmt: PicoStatement) => void): void {
  for (const stmt of body.stmts) {
    visit(stmt)
    if (stmt.kind === 'if') {
      walkPicoStatements(stmt.then, visit)
      if (stmt.else) walkPicoStatements(stmt.else, visit)
    } else if (stmt.kind === 'while' || stmt.kind === 'for') walkPicoStatements(stmt.body, visit)
  }
}
