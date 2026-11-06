export type PicoPrimitive = 'bool' | 'integer' | 'unit' | 'terrain' | 'rotation'
export type PicoType = PicoPrimitive | { kind: 'vector'; elementType: PicoType }
interface NodeBase {
  hintId: string
}
export interface PicoIdent extends NodeBase {
  kind: 'ident'
  name: string
}
export interface PicoLiteral extends NodeBase {
  kind: 'literal'
  type: 'bool' | 'integer' | 'unit' | 'rotation'
  value: boolean | number | null | 'left' | 'right'
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
  | (NodeBase & { kind: 'declare'; type: PicoType; ident: PicoIdent; value: PicoExpr })
  | (NodeBase & { kind: 'assign'; ident: PicoIdent; value: PicoExpr })
  | (NodeBase & { kind: 'if'; condition: PicoExpr; then: PicoBlock; else?: PicoBlock })
  | (NodeBase & { kind: 'while'; condition: PicoExpr; body: PicoBlock })
  | (NodeBase & { kind: 'for'; ident: PicoIdent; start: PicoExpr; end: PicoExpr; body: PicoBlock })
  | (NodeBase & { kind: 'return'; value: PicoExpr })
  | (NodeBase & { kind: 'break' | 'continue' })
export interface PicoFunction extends NodeBase {
  kind: 'function'
  params: Array<NodeBase & { kind: 'param'; ident: PicoIdent; type: PicoType }>
  returnType: PicoType
  body: PicoBlock
}
export interface PicoProgram {
  picolangVersion: 0
  picolang: { v0: { program: Record<string, PicoFunction> } }
}
export interface PicoFunctionOption {
  name: string
  label: string
  category: string
  args: PicoType[]
  returns: PicoType
}
const vectorInteger: PicoType = { kind: 'vector', elementType: 'integer' }
const builtin = (
  name: string,
  label: string,
  category: string,
  args: PicoType[],
  returns: PicoType,
): PicoFunctionOption => ({ name, label, category, args, returns })
export const PICO_BUILTINS: PicoFunctionOption[] = [
  builtin('forward', '前進する', '動き', [], 'bool'),
  builtin('back', '後退する', '動き', [], 'bool'),
  builtin('rotate', '回転する', '動き', ['rotation'], 'unit'),
  builtin('wait', '待つ', '動き', ['integer'], 'unit'),
  builtin('pick_up', '拾う', '操作', [], 'bool'),
  builtin('release', '置く', '操作', [], 'bool'),
  builtin('drill', '岩を壊す', '操作', [], 'bool'),
  builtin('drill_durability', 'ドリル残量', '調べる', [], 'integer'),
  builtin('distance', '距離', '調べる', [], 'integer'),
  builtin('search_front', '前を調べる', '調べる', [], 'terrain'),
  builtin('search_now', '現在地を調べる', '調べる', [], 'terrain'),
  builtin('is_land', '陸？', '調べる', ['terrain'], 'bool'),
  builtin('is_goal', 'ゴール？', '調べる', ['terrain'], 'bool'),
  builtin('is_checkpoint', 'チェックポイント？', '調べる', ['terrain'], 'bool'),
  builtin('is_laggage', '荷物？', '調べる', ['terrain'], 'bool'),
  builtin('is_sea', '海？', '調べる', ['terrain'], 'bool'),
  builtin('is_rock', '岩？', '調べる', ['terrain'], 'bool'),
  builtin('not', 'ではない', '論理', ['bool'], 'bool'),
  builtin('and', 'かつ', '論理', ['bool', 'bool'], 'bool'),
  builtin('or', 'または', '論理', ['bool', 'bool'], 'bool'),
  builtin('bool_eq', '真偽が等しい', '論理', ['bool', 'bool'], 'bool'),
  builtin('bool_neq', '真偽が異なる', '論理', ['bool', 'bool'], 'bool'),
  builtin('add', '足す', '計算', ['integer', 'integer'], 'integer'),
  builtin('minus', '符号を反転', '計算', ['integer'], 'integer'),
  builtin('sub', '引く', '計算', ['integer', 'integer'], 'integer'),
  builtin('mul', '掛ける', '計算', ['integer', 'integer'], 'integer'),
  builtin('div', '割る', '計算', ['integer', 'integer'], 'integer'),
  builtin('integer_eq', '数が等しい', '計算', ['integer', 'integer'], 'bool'),
  builtin('integer_neq', '数が異なる', '計算', ['integer', 'integer'], 'bool'),
  builtin('gt', 'より大きい', '計算', ['integer', 'integer'], 'bool'),
  builtin('lt', 'より小さい', '計算', ['integer', 'integer'], 'bool'),
  builtin('geq', '以上', '計算', ['integer', 'integer'], 'bool'),
  builtin('leq', '以下', '計算', ['integer', 'integer'], 'bool'),
  builtin('bool_to_integer', '真偽を数に', '変換', ['bool'], 'integer'),
  builtin('integer_to_bool', '数を真偽に', '変換', ['integer'], 'bool'),
  builtin('random_bool', 'ランダムな真偽', '乱数', [], 'bool'),
  builtin('random_integer', 'ランダムな整数', '乱数', ['integer', 'integer'], 'integer'),
  builtin('new', '空の配列', '配列', [], vectorInteger),
  builtin('push_back', '末尾に追加', '配列', [vectorInteger, 'integer'], vectorInteger),
  builtin('pop_back', '末尾を削除', '配列', [vectorInteger], vectorInteger),
  builtin('get_back', '末尾を読む', '配列', [vectorInteger], 'integer'),
  builtin('size', '配列の長さ', '配列', [vectorInteger], 'integer'),
  builtin('print_integer', '整数を表示', '表示', ['integer'], 'unit'),
  builtin('print_bool', '真偽を表示', '表示', ['bool'], 'unit'),
]
export const id = (): string => crypto.randomUUID()
export const ident = (name = 'x'): PicoIdent => ({ kind: 'ident', hintId: id(), name })
export const block = (stmts: PicoStatement[] = []): PicoBlock => ({ kind: 'block', hintId: id(), stmts })
export function defaultExpr(type: PicoType = 'integer'): PicoExpr {
  if (typeof type !== 'string') return { kind: 'call', hintId: id(), func: 'new', args: [] }
  if (type === 'bool') return { kind: 'literal', hintId: id(), type, value: false }
  if (type === 'unit') return { kind: 'literal', hintId: id(), type, value: null }
  if (type === 'rotation') return { kind: 'literal', hintId: id(), type, value: 'left' }
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
      return { kind: 'declare', hintId: id(), type: 'integer', ident: ident(), value: defaultExpr() }
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
const primitiveTypes: PicoPrimitive[] = ['bool', 'integer', 'unit', 'terrain', 'rotation']
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
        value.kind === 'vector' &&
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
    default:
      return false
  }
}
function validBlock(value: unknown, seen: Set<string>): value is PicoBlock {
  return (
    object(value) &&
    node(value, 'block', ['stmts'], seen) &&
    Array.isArray(value.stmts) &&
    value.stmts.every((stmt) => validStmt(stmt, seen))
  )
}
function validStmt(value: unknown, seen: Set<string>): boolean {
  if (!object(value)) return false
  switch (value.kind) {
    case 'literal':
    case 'ident':
    case 'call':
      return validExpr(value, seen)
    case 'declare':
      return (
        node(value, 'declare', ['type', 'ident', 'value'], seen) &&
        type(value.type) &&
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
        validBlock(value.then, seen) &&
        (value.else === undefined || validBlock(value.else, seen))
      )
    case 'while':
      return (
        node(value, 'while', ['condition', 'body'], seen) &&
        validExpr(value.condition, seen) &&
        validBlock(value.body, seen)
      )
    case 'for':
      return (
        node(value, 'for', ['ident', 'start', 'end', 'body'], seen) &&
        validIdent(value.ident, seen) &&
        validExpr(value.start, seen) &&
        validExpr(value.end, seen) &&
        validBlock(value.body, seen)
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
export function isPicoProgram(value: unknown): value is PicoProgram {
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
      !node(definition, 'function', ['params', 'returnType', 'body'], seen) ||
      !Array.isArray(definition.params) ||
      !type(definition.returnType)
    )
      return false
    for (const param of definition.params) {
      if (
        !object(param) ||
        !node(param, 'param', ['ident', 'type'], seen) ||
        !validIdent(param.ident, seen) ||
        !type(param.type)
      )
        return false
    }
    if (!validBlock(definition.body, seen)) return false
    if (definition.body.stmts.at(-1)?.kind !== 'return') return false
  }
  return (
    functions.main.returnType === 'unit' &&
    Array.isArray(functions.main.params) &&
    functions.main.params.length === 0
  )
}
