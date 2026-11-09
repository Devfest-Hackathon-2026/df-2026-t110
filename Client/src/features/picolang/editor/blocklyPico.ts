import { hasTypeParameter } from '../language/picoSignatures.ts'
import * as Blockly from 'blockly'
import { PICO_BUILTIN_OPTIONS } from './picoOptions.ts'
import {
  block as picoBlock,
  defaultExpr,
  id,
  ident,
  PICO_DISTANCE_TARGETS,
  type PicoBlock,
  type PicoCall,
  type PicoExpr,
  type PicoFunction,
  type PicoStatement,
  type PicoType,
  type PicoSignatureType,
} from '../language/picolang.ts'
import { argumentLabel, resultGuidance, typeLabel } from './picoGuidance.ts'
import type { PicoDiagnostic } from '../language/analyzePicolang.ts'

type State = Blockly.serialization.blocks.State
type Block = Blockly.Block

class EditorTextField extends Blockly.FieldTextInput {
  protected override widgetCreate_(): HTMLInputElement | HTMLTextAreaElement {
    const input = super.widgetCreate_()
    input.setAttribute('data-1p-ignore', '')
    return input
  }
}

class EditorNumberField extends Blockly.FieldNumber {
  protected override widgetCreate_(): HTMLInputElement {
    const input = super.widgetCreate_()
    input.setAttribute('data-1p-ignore', '')
    return input
  }
}

const colours: Record<string, string> = {
  動く: '#347ac4',
  オブジェクト: '#248b87',
  制御: '#be8b32',
  変数: '#bd7446',
  調べる: '#287d98',
  論理: '#7555ac',
  計算: '#549641',
  ランダム: '#6476ad',
  スタック: '#8a67ab',
  ヌラブル: '#95734b',
  発言: '#60866a',
  関数: '#875c98',
  値: '#65758b',
}
const controlTypes = ['pico_if', 'pico_if_else', 'pico_while', 'pico_for']
let staticRegistered = false

export function typeKey(type: PicoType): string {
  return typeof type === 'string' ? type : `${type.kind}:${typeKey(type.elementType)}`
}
export function keyType(key: string): PicoType {
  for (const kind of ['stack', 'option'] as const)
    if (key.startsWith(`${kind}:`)) return { kind, elementType: keyType(key.slice(kind.length + 1)) }
  return ['bool', 'integer', 'unit', 'terrain', 'rotation', 'distance_target'].includes(key)
    ? (key as PicoType)
    : 'integer'
}
function typeCheck(type: PicoSignatureType | undefined): string[] | null {
  if (type === undefined) return null
  if (typeof type !== 'string' && type.kind === 'type_parameter') return null
  if (hasTypeParameter(type)) return [(type as { kind: string }).kind]
  const key = typeKey(type as PicoType)
  return typeof type === 'string' ? [key] : [key, type.kind]
}
function nodeData(hintId: string, extra: Record<string, string> = {}): string {
  return JSON.stringify({ hintId, ...extra })
}
function meta(block: Block): Record<string, string> {
  let value: Record<string, string>
  try {
    value = JSON.parse(block.data ?? '{}') as Record<string, string>
  } catch {
    value = {}
  }
  if (!value.hintId) {
    value.hintId = id()
    block.data = JSON.stringify(value)
  }
  return value
}
function metaId(block: Block, key = 'hintId'): string {
  const value = meta(block)
  if (!value[key]) {
    value[key] = id()
    block.data = JSON.stringify(value)
  }
  return value[key]!
}
function field(block: Block, name: string, fallback = ''): string {
  return String(block.getFieldValue(name) ?? fallback)
}
function value(block: Block, input: string, functions: Record<string, PicoFunction>): PicoExpr {
  const child = block.getInputTargetBlock(input)
  if (child) return readExpr(child, functions)
  return { kind: 'missing', hintId: metaId(block, `empty_${input}`) }
}

export function callType(name: string, statement: boolean): string {
  return `pico_${statement ? 'stmt' : 'expr'}_${encodeURIComponent(name)}`
}
function callName(block: Block): string {
  return decodeURIComponent(block.type.slice(block.type.indexOf('_', 5) + 1))
}
function callSignature(
  name: string,
  functions: Record<string, PicoFunction>,
): {
  args: readonly (PicoSignatureType | undefined)[]
  returns: PicoSignatureType | undefined
  category: string
  label: string
  blockMessage?: string
} {
  const builtin = PICO_BUILTIN_OPTIONS.find((item) => item.name === name)
  if (builtin) return builtin
  const fn = functions[name]
  return fn
    ? { args: fn.params.map((param) => param.type), returns: fn.returnType, category: '関数', label: name }
    : { args: [], returns: 'unit', category: '関数', label: name }
}

function defineStaticBlocks(): void {
  if (staticRegistered) return
  staticRegistered = true
  Blockly.Blocks.pico_number = {
    init() {
      this.appendDummyInput()
        .appendField('整数')
        .appendField(new EditorNumberField(0, undefined, undefined, 1), 'VALUE')
      this.setOutput(true, 'integer')
      this.setColour(colours.値)
    },
  }
  Blockly.Blocks.pico_bool = {
    init() {
      this.appendDummyInput()
        .appendField('真理値')
        .appendField(
          new Blockly.FieldDropdown([
            ['はい', 'true'],
            ['いいえ', 'false'],
          ]),
          'VALUE',
        )
      this.setOutput(true, 'bool')
      this.setColour(colours.値)
    },
  }
  Blockly.Blocks.pico_rotation = {
    init() {
      this.appendDummyInput()
        .appendField('向き')
        .appendField(
          new Blockly.FieldDropdown([
            ['左', 'left'],
            ['右', 'right'],
          ]),
          'VALUE',
        )
      this.setOutput(true, 'rotation')
      this.setColour(colours.値)
    },
  }
  Blockly.Blocks.pico_unit = {
    init() {
      this.appendDummyInput().appendField('値なし')
      this.setOutput(true, 'unit')
      this.setColour(colours.値)
    },
  }
  Blockly.Blocks.pico_distance_target = {
    init() {
      this.appendDummyInput()
        .appendField('探索対象')
        .appendField(
          new Blockly.FieldDropdown(
            PICO_DISTANCE_TARGETS.map((target) => [
              {
                robot: 'ロボット',
                ally: '味方のロボット',
                enemy: '相手のロボット',
                rock: '岩',
                luggage: '荷物',
                sea: '海',
                goal: 'ゴール',
                territory: '領地',
              }[target],
              target,
            ]),
          ),
          'VALUE',
        )
      this.setOutput(true, 'distance_target')
      this.setColour(colours.値)
    },
  }
  Blockly.Blocks.pico_ident = {
    init() {
      this.appendDummyInput().appendField('変数').appendField(new EditorTextField('x'), 'NAME')
      this.setOutput(true)
      this.setColour(colours.変数)
    },
  }
  Blockly.Blocks.pico_declare = {
    init() {
      this.appendValueInput('TOKEN').appendField('未確定', 'INFERRED_TYPE').appendField('型の')
      this.appendDummyInput().appendField('を').appendField(new EditorTextField('x'), 'NAME')
      this.appendValueInput('VALUE').appendField('という名前で')
      this.appendDummyInput().appendField('に初期化する')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.変数)
      this.setTooltip('小さな変数ブロックをドラッグすると、別の入力欄でこの変数を使えます。')
    },
  }
  Blockly.Blocks.pico_assign = {
    init() {
      this.appendDummyInput().appendField(new EditorTextField('x'), 'NAME')
      this.appendValueInput('VALUE').appendField('を')
      this.appendDummyInput().appendField('に変える')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.変数)
    },
  }
  Blockly.Blocks.pico_if = {
    init() {
      this.appendValueInput('CONDITION').setCheck('bool').appendField('もし')
      this.appendStatementInput('THEN').appendField('なら')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_if_else = {
    init() {
      this.appendValueInput('CONDITION').setCheck('bool').appendField('もし')
      this.appendDummyInput().appendField('なら')
      this.appendStatementInput('THEN')
      this.appendDummyInput().appendField('そうではないなら')
      this.appendStatementInput('ELSE')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_while = {
    init() {
      this.appendValueInput('CONDITION').setCheck('bool')
      this.appendDummyInput().appendField('の間')
      this.appendStatementInput('BODY')
      this.appendDummyInput().appendField('を繰り返す')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_for = {
    init() {
      this.appendValueInput('TOKEN')
      this.appendDummyInput().appendField('を').appendField(new EditorTextField('i'), 'NAME')
      this.appendValueInput('START').setCheck('integer').appendField('という名前で')
      this.appendValueInput('END').setCheck('integer').appendField('から')
      this.appendDummyInput().appendField('-1まで')
      this.appendStatementInput('BODY')
      this.appendDummyInput().appendField('を繰り返す')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_return = {
    init() {
      this.appendValueInput('VALUE')
      this.appendDummyInput().appendField('を返す')
      this.setInputsInline(true)
      this.setPreviousStatement(true)
      this.setColour(colours.関数)
    },
  }
  Blockly.Blocks.pico_break = {
    init() {
      this.appendDummyInput().appendField('くり返しを抜ける')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_continue = {
    init() {
      this.appendDummyInput().appendField('次のくり返しへ')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_expr_statement = {
    init() {
      this.appendValueInput('VALUE').appendField('この値を計算する')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.計算)
    },
  }
}

export function registerPicoBlocks(functions: Record<string, PicoFunction>): void {
  defineStaticBlocks()
  for (const option of PICO_BUILTIN_OPTIONS) defineCall(option.name, functions)
  for (const name of Object.keys(functions)) if (name !== 'main') defineCall(name, functions)
}
function defineCall(name: string, functions: Record<string, PicoFunction>): void {
  const signature = callSignature(name, functions)
  const userFunction = Object.hasOwn(functions, name)
  const inputCheck = (type: PicoSignatureType | undefined): string[] | null =>
    userFunction ? null : typeCheck(type)
  const missing = !PICO_BUILTIN_OPTIONS.some((item) => item.name === name) && !Object.hasOwn(functions, name)
  for (const statement of [false, true]) {
    Blockly.Blocks[callType(name, statement)] = {
      init() {
        if (signature.blockMessage) {
          this.jsonInit({
            message0: signature.blockMessage,
            args0: signature.args.map((type, index) => ({
              type: 'input_value',
              name: `ARG${index}`,
              check: inputCheck(type),
            })),
            inputsInline: true,
          })
        } else {
          this.appendDummyInput().appendField(signature.label)
          signature.args.forEach((type, index) =>
            this.appendValueInput(`ARG${index}`)
              .setCheck(inputCheck(type))
              .appendField(argumentLabel(name, index, functions)),
          )
        }
        if (statement) {
          this.setPreviousStatement(true)
          this.setNextStatement(true)
        } else this.setOutput(true, missing || userFunction ? null : typeCheck(signature.returns))
        this.setColour(colours[signature.category] ?? colours.関数)
        this.setTooltip(
          [
            resultGuidance[name],
            statement && !['動く', 'オブジェクト', '発言'].includes(signature.category)
              ? 'この行だけでは返した値を使いません。変数や条件の入力欄で使ってください。'
              : '',
          ]
            .filter(Boolean)
            .join(' '),
        )
      },
      // Keep unresolved calls editable so imported programs can display their
      // diagnostics, including references inside the unknown call's arguments.
      loadExtraState(state: { argumentCount: number }) {
        if (!missing) return
        for (let index = 0; index < state.argumentCount; index++)
          this.appendValueInput(`ARG${index}`).appendField(`引数 ${index + 1}`)
      },
      saveExtraState() {
        return missing
          ? {
              argumentCount: this.inputList.filter((input: Blockly.Input) => /^ARG\d+$/.test(input.name))
                .length,
            }
          : null
      },
    }
  }
}

function callState(expr: PicoCall, statement: boolean, functions: Record<string, PicoFunction>): State {
  const missing =
    !PICO_BUILTIN_OPTIONS.some((item) => item.name === expr.func) && !Object.hasOwn(functions, expr.func)
  defineCall(expr.func, functions)
  return withValueInputs(
    {
      type: callType(expr.func, statement),
      data: nodeData(expr.hintId),
      ...(missing ? { extraState: { argumentCount: expr.args.length } } : {}),
    },
    Object.fromEntries(expr.args.map((arg, i) => [`ARG${i}`, arg])),
    functions,
  )
}

/** Empty value inputs keep their diagnostic IDs without creating a value block. */
function withValueInputs(
  state: State,
  expressions: Record<string, PicoExpr>,
  functions: Record<string, PicoFunction>,
): State {
  const data = JSON.parse(state.data ?? '{}') as Record<string, string>
  state.inputs ??= {}
  for (const [input, expr] of Object.entries(expressions)) {
    if (expr.kind === 'missing') {
      data[`empty_${input}`] = expr.hintId
      state.inputs[input] = {}
    } else state.inputs[input] = { block: exprState(expr, functions) }
  }
  state.data = JSON.stringify(data)
  return state
}

function exprState(
  expr: Exclude<PicoExpr, { kind: 'missing' }>,
  functions: Record<string, PicoFunction>,
): State {
  if (expr.kind === 'ident')
    return { type: 'pico_ident', data: nodeData(expr.hintId), fields: { NAME: expr.name } }
  if (expr.kind === 'literal') {
    const type = {
      integer: 'pico_number',
      bool: 'pico_bool',
      rotation: 'pico_rotation',
      unit: 'pico_unit',
      distance_target: 'pico_distance_target',
    }[expr.type]
    return {
      type,
      data: nodeData(expr.hintId),
      fields: expr.type === 'unit' ? {} : { VALUE: expr.type === 'bool' ? String(expr.value) : expr.value },
    }
  }
  return callState(expr, false, functions)
}
function stmtState(stmt: PicoStatement, functions: Record<string, PicoFunction>): State {
  if (stmt.kind === 'call') return callState(stmt, true, functions)
  if (stmt.kind === 'literal' || stmt.kind === 'ident' || stmt.kind === 'missing')
    return withValueInputs(
      { type: 'pico_expr_statement', data: nodeData(stmt.hintId) },
      { VALUE: stmt },
      functions,
    )
  if (stmt.kind === 'declare')
    return withValueInputs(
      {
        type: 'pico_declare',
        data: nodeData(stmt.hintId, { identId: stmt.ident.hintId }),
        fields: { NAME: stmt.ident.name },
        inputs: { TOKEN: { shadow: exprState(ident(stmt.ident.name), functions) } },
      },
      { VALUE: stmt.value },
      functions,
    )
  if (stmt.kind === 'assign')
    return withValueInputs(
      {
        type: 'pico_assign',
        data: nodeData(stmt.hintId, { identId: stmt.ident.hintId }),
        fields: { NAME: stmt.ident.name },
      },
      { VALUE: stmt.value },
      functions,
    )
  if (stmt.kind === 'if')
    return withValueInputs(
      {
        type: stmt.else ? 'pico_if_else' : 'pico_if',
        data: nodeData(stmt.hintId, {
          thenId: stmt.then.hintId,
          ...(stmt.else ? { elseId: stmt.else.hintId } : {}),
        }),
        inputs: {
          ...(stmt.then.stmts.length ? { THEN: { block: chainState(stmt.then.stmts, functions)! } } : {}),
          ...(stmt.else?.stmts.length ? { ELSE: { block: chainState(stmt.else.stmts, functions)! } } : {}),
        },
      },
      { CONDITION: stmt.condition },
      functions,
    )
  if (stmt.kind === 'while' || stmt.kind === 'for')
    return withValueInputs(
      {
        type: stmt.kind === 'for' ? 'pico_for' : 'pico_while',
        data: nodeData(stmt.hintId, {
          bodyId: stmt.body.hintId,
          ...(stmt.kind === 'for' ? { identId: stmt.ident.hintId } : {}),
        }),
        fields: stmt.kind === 'for' ? { NAME: stmt.ident.name } : {},
        inputs: {
          ...(stmt.kind === 'for' ? { TOKEN: { shadow: exprState(ident(stmt.ident.name), functions) } } : {}),
          ...(stmt.body.stmts.length ? { BODY: { block: chainState(stmt.body.stmts, functions)! } } : {}),
        },
      },
      stmt.kind === 'while' ? { CONDITION: stmt.condition } : { START: stmt.start, END: stmt.end },
      functions,
    )
  if (stmt.kind === 'return')
    return withValueInputs(
      { type: 'pico_return', data: nodeData(stmt.hintId) },
      { VALUE: stmt.value },
      functions,
    )
  return { type: `pico_${stmt.kind}`, data: nodeData(stmt.hintId) }
}
function chainState(stmts: PicoStatement[], functions: Record<string, PicoFunction>): State | null {
  if (!stmts.length) return null
  const [first, ...rest] = stmts
  const state = stmtState(first!, functions)
  const next = chainState(rest, functions)
  if (next) state.next = { block: next }
  return state
}

export function loadFunction(
  workspace: Blockly.Workspace,
  fn: PicoFunction,
  functions: Record<string, PicoFunction>,
): void {
  workspace.clear()
  const chain = chainState(fn.body.stmts, functions)
  if (chain) Blockly.serialization.blocks.append({ ...chain, x: 220, y: 36 }, workspace)
}

function readExpr(block: Block, functions: Record<string, PicoFunction>): PicoExpr {
  if (block.type === 'pico_ident')
    return { kind: 'ident', hintId: metaId(block), name: field(block, 'NAME', 'x') }
  if (block.type === 'pico_number')
    return {
      kind: 'literal',
      hintId: metaId(block),
      type: 'integer',
      value: Number(field(block, 'VALUE', '0')),
    }
  if (block.type === 'pico_bool')
    return { kind: 'literal', hintId: metaId(block), type: 'bool', value: field(block, 'VALUE') === 'true' }
  if (block.type === 'pico_rotation')
    return {
      kind: 'literal',
      hintId: metaId(block),
      type: 'rotation',
      value: field(block, 'VALUE') === 'right' ? 'right' : 'left',
    }
  if (block.type === 'pico_unit') return { kind: 'literal', hintId: metaId(block), type: 'unit', value: null }
  if (block.type === 'pico_distance_target')
    return {
      kind: 'literal',
      hintId: metaId(block),
      type: 'distance_target',
      value: field(block, 'VALUE') as (typeof PICO_DISTANCE_TARGETS)[number],
    }
  const name = callName(block)
  return {
    kind: 'call',
    hintId: metaId(block),
    func: name,
    args: block.inputList
      .filter((input) => /^ARG\d+$/.test(input.name))
      .map((input) => value(block, input.name, functions)),
  }
}
function readStack(first: Block | null, functions: Record<string, PicoFunction>): PicoStatement[] {
  const statements: PicoStatement[] = []
  for (let current = first; current; current = current.getNextBlock()) {
    const hintId = metaId(current)
    if (current.type === 'pico_declare') {
      statements.push({
        kind: 'declare',
        hintId,
        ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'x') },
        value: value(current, 'VALUE', functions),
      })
    } else if (current.type === 'pico_assign') {
      statements.push({
        kind: 'assign',
        hintId,
        ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'x') },
        value: value(current, 'VALUE', functions),
      })
    } else if (current.type === 'pico_if' || current.type === 'pico_if_else') {
      const then = picoBlock(readStack(current.getInputTargetBlock('THEN'), functions))
      then.hintId = metaId(current, 'thenId')
      let otherwise: PicoBlock | undefined
      if (current.type === 'pico_if_else') {
        otherwise = picoBlock(readStack(current.getInputTargetBlock('ELSE'), functions))
        otherwise.hintId = metaId(current, 'elseId')
      }
      statements.push({
        kind: 'if',
        hintId,
        condition: value(current, 'CONDITION', functions),
        then,
        ...(otherwise ? { else: otherwise } : {}),
      })
    } else if (current.type === 'pico_while' || current.type === 'pico_for') {
      const body = picoBlock(readStack(current.getInputTargetBlock('BODY'), functions))
      body.hintId = metaId(current, 'bodyId')
      if (current.type === 'pico_while')
        statements.push({
          kind: 'while',
          hintId,
          condition: value(current, 'CONDITION', functions),
          body,
        })
      else
        statements.push({
          kind: 'for',
          hintId,
          ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'i') },
          start: value(current, 'START', functions),
          end: value(current, 'END', functions),
          body,
        })
    } else if (current.type === 'pico_return')
      statements.push({ kind: 'return', hintId, value: value(current, 'VALUE', functions) })
    else if (current.type === 'pico_break' || current.type === 'pico_continue')
      statements.push({ kind: current.type === 'pico_break' ? 'break' : 'continue', hintId })
    else if (current.type === 'pico_expr_statement')
      statements.push(
        current.getInputTargetBlock('VALUE')
          ? readExpr(current.getInputTargetBlock('VALUE')!, functions)
          : value(current, 'VALUE', functions),
      )
    else if (current.type.startsWith('pico_stmt_')) statements.push(readExpr(current, functions))
  }
  return statements
}
export function readFunction(
  workspace: Blockly.Workspace,
  original: PicoFunction,
  functions: Record<string, PicoFunction>,
): PicoFunction {
  const tops = workspace.getTopBlocks(true).filter((item) => !!item.previousConnection && !item.isShadow())
  const statements = tops.flatMap((top) => readStack(top, functions))
  const returns = statements.filter((item) => item.kind === 'return')
  const originalReturn = original.body.stmts.find((item) => item.kind === 'return')
  const lastReturn = returns.at(-1) ??
    originalReturn ?? {
      kind: 'return' as const,
      hintId: id(),
      value: defaultExpr('unit'),
    }
  const body = picoBlock([...statements.filter((item) => item !== lastReturn), lastReturn])
  body.hintId = original.body.hintId
  return { ...original, body }
}

function toolboxBlock(
  type: string,
  args: readonly (PicoSignatureType | undefined)[] = [],
  functions: Record<string, PicoFunction> = {},
): Blockly.utils.toolbox.BlockInfo {
  const defaults: Record<string, PicoType> = {
    pico_declare: 'integer',
    pico_assign: 'integer',
    pico_if: 'bool',
    pico_if_else: 'bool',
    pico_while: 'bool',
    pico_for: 'integer',
    pico_return: 'unit',
  }
  const inputNames: Record<string, string> = {
    pico_declare: 'VALUE',
    pico_assign: 'VALUE',
    pico_if: 'CONDITION',
    pico_if_else: 'CONDITION',
    pico_while: 'CONDITION',
    pico_for: 'START',
    pico_return: 'VALUE',
  }
  const inputs: Record<string, Blockly.serialization.blocks.ConnectionState> = Object.fromEntries(
    args.map((arg, i) => [`ARG${i}`, { block: exprState(defaultExpr(arg), functions) }]),
  )
  if (defaults[type]) inputs[inputNames[type]!] = { block: exprState(defaultExpr(defaults[type]), functions) }
  if (type === 'pico_for') {
    inputs.END = { block: exprState(defaultExpr('integer'), functions) }
    inputs.TOKEN = { shadow: exprState(ident('i'), functions) }
  }
  if (type === 'pico_declare') inputs.TOKEN = { shadow: exprState(ident('x'), functions) }
  return {
    kind: 'block',
    type,
    inputs,
  }
}
export function picoToolbox(functions: Record<string, PicoFunction>): Blockly.utils.toolbox.ToolboxInfo {
  const categories: Blockly.utils.toolbox.StaticCategoryInfo[] = []
  for (const category of [
    '動く',
    'オブジェクト',
    '制御',
    '変数',
    '調べる',
    '論理',
    '計算',
    'ランダム',
    'スタック',
    'ヌラブル',
    '発言',
    '関数',
    '値',
  ]) {
    const contents: Blockly.utils.toolbox.ToolboxItemInfo[] = []
    if (category === '制御')
      for (const type of [...controlTypes, 'pico_break', 'pico_continue', 'pico_return'])
        contents.push(toolboxBlock(type))
    else if (category === '変数')
      for (const type of ['pico_declare', 'pico_assign', 'pico_ident']) contents.push(toolboxBlock(type))
    else if (category === '値')
      for (const type of ['pico_number', 'pico_bool', 'pico_rotation', 'pico_unit', 'pico_distance_target'])
        contents.push(toolboxBlock(type))
    else if (category === '関数') {
      for (const name of Object.keys(functions).filter((item) => item !== 'main')) {
        contents.push(
          toolboxBlock(
            callType(name, true),
            functions[name]!.params.map((param) => param.type),
            functions,
          ),
        )
        contents.push(
          toolboxBlock(
            callType(name, false),
            functions[name]!.params.map((param) => param.type),
            functions,
          ),
        )
      }
    } else {
      for (const builtin of PICO_BUILTIN_OPTIONS.filter((item) => item.category === category))
        contents.push(
          toolboxBlock(
            callType(builtin.name, ['動く', 'オブジェクト', '発言'].includes(category)),
            builtin.args,
            functions,
          ),
        )
    }
    categories.push({
      kind: 'category',
      name: category,
      colour: colours[category],
      contents,
    } as Blockly.utils.toolbox.StaticCategoryInfo)
  }
  return { kind: 'categoryToolbox', contents: categories }
}

export function refreshVariableBlocks(workspace: Blockly.Workspace): void {
  for (const block of workspace.getAllBlocks(false)) {
    if (block.type === 'pico_declare' || block.type === 'pico_for') {
      const name = field(block, 'NAME', block.type === 'pico_for' ? 'i' : 'x')
      block.getInput('VALUE')?.connection?.setCheck(null)
      const input = block.getInput('TOKEN')?.connection
      const token = block.getInputTargetBlock('TOKEN')
      if (token?.isShadow() && field(token, 'NAME') !== name) token.setFieldValue(name, 'NAME')
      if (!token && input) {
        const reporter = Blockly.serialization.blocks.append(
          { type: 'pico_ident', fields: { NAME: name } },
          workspace,
        )
        reporter.setShadow(true)
        input.setShadowState({ type: 'pico_ident', fields: { NAME: name } })
        if (reporter.outputConnection) input.connect(reporter.outputConnection)
      }
    }
  }
  for (const block of workspace.getAllBlocks(false)) {
    // Analysis checks scopes and types without disconnecting references because
    // an unrelated declaration elsewhere in the workspace has the same name.
    if (block.type === 'pico_ident') block.outputConnection?.setCheck(null)
    if (block.type === 'pico_assign') block.getInput('VALUE')?.connection?.setCheck(null)
  }
}

export function diagnosticBlock(workspace: Blockly.Workspace, hintId: string): Block | undefined {
  return workspace.getAllBlocks(false).find((block) => Object.values(meta(block)).includes(hintId))
}

export function updatePicoDiagnostics(
  workspace: Blockly.Workspace,
  diagnostics: PicoDiagnostic[],
  types: Record<string, PicoType | undefined> = {},
): void {
  const blocks = workspace.getAllBlocks(false)
  const byHint = new Map<string, Block>()
  for (const block of blocks) for (const hint of Object.values(meta(block))) byHint.set(hint, block)
  const messages = new Map<Block, string[]>()
  const severities = new Map<Block, PicoDiagnostic['severity']>()
  for (const diagnostic of diagnostics) {
    const block = byHint.get(diagnostic.hintId)
    if (!block) continue
    const items = messages.get(block) ?? []
    items.push(`${diagnostic.severity === 'error' ? 'エラー' : '警告'}：${diagnostic.message}`)
    messages.set(block, items)
    if (severities.get(block) !== 'error') severities.set(block, diagnostic.severity)
  }
  for (const block of blocks) {
    block.setWarningText(messages.get(block)?.join('\n') ?? null, 'pico-analysis')
    if (block instanceof Blockly.BlockSvg) {
      const root = block.getSvgRoot()
      const severity = severities.get(block)
      if (severity) root.dataset.picoDiagnostic = severity
      else delete root.dataset.picoDiagnostic
    }
    if (block.type === 'pico_declare')
      block.setFieldValue(typeLabel(types[metaId(block, 'identId')]), 'INFERRED_TYPE')
  }
}
