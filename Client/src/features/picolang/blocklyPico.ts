import * as Blockly from 'blockly'
import {
  PICO_BUILTINS,
  block as picoBlock,
  defaultExpr,
  id,
  ident,
  type PicoBlock,
  type PicoExpr,
  type PicoFunction,
  type PicoStatement,
  type PicoType,
} from './picolang.ts'
import { argumentLabel } from './picoGuidance.ts'

type State = Blockly.serialization.blocks.State
type Block = Blockly.Block

const colours: Record<string, string> = {
  動き: '#347ac4',
  操作: '#248b87',
  制御: '#be8b32',
  変数: '#bd7446',
  調べる: '#287d98',
  論理: '#7555ac',
  計算: '#549641',
  変換: '#a97737',
  乱数: '#6476ad',
  配列: '#8a67ab',
  表示: '#60866a',
  関数: '#875c98',
}
const typeChoices: [string, string][] = [
  ['整数', 'integer'],
  ['真理値', 'bool'],
  ['値なし', 'unit'],
  ['マスの種類', 'terrain'],
  ['向き', 'rotation'],
  ['整数の配列', 'vector:integer'],
  ['真理値の配列', 'vector:bool'],
  ['マスの配列', 'vector:terrain'],
  ['向きの配列', 'vector:rotation'],
]
const controlTypes = ['pico_if', 'pico_if_else', 'pico_while', 'pico_for']
let staticRegistered = false

export function typeKey(type: PicoType): string {
  return typeof type === 'string' ? type : `vector:${typeKey(type.elementType)}`
}
export function keyType(key: string): PicoType {
  return key.startsWith('vector:')
    ? { kind: 'vector', elementType: keyType(key.slice(7)) }
    : ['bool', 'integer', 'unit', 'terrain', 'rotation'].includes(key)
      ? (key as PicoType)
      : 'integer'
}
function typeCheck(type: PicoType): string[] {
  const key = typeKey(type)
  return typeof type === 'string' ? [key] : [key, 'vector']
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
function value(
  block: Block,
  input: string,
  fallback: PicoType,
  functions: Record<string, PicoFunction>,
): PicoExpr {
  const child = block.getInputTargetBlock(input)
  if (child) return readExpr(child, functions)
  const expression = defaultExpr(fallback)
  expression.hintId = metaId(block, `empty_${input}`)
  return expression
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
): { args: PicoType[]; returns: PicoType; category: string; label: string } {
  const builtin = PICO_BUILTINS.find((item) => item.name === name)
  if (builtin)
    return { args: builtin.args, returns: builtin.returns, category: builtin.category, label: builtin.label }
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
      this.appendDummyInput().appendField(new Blockly.FieldNumber(0, undefined, undefined, 1), 'VALUE')
      this.setOutput(true, 'integer')
      this.setColour(colours.計算)
    },
  }
  Blockly.Blocks.pico_bool = {
    init() {
      this.appendDummyInput().appendField(
        new Blockly.FieldDropdown([
          ['はい', 'true'],
          ['いいえ', 'false'],
        ]),
        'VALUE',
      )
      this.setOutput(true, 'bool')
      this.setColour(colours.論理)
    },
  }
  Blockly.Blocks.pico_rotation = {
    init() {
      this.appendDummyInput().appendField(
        new Blockly.FieldDropdown([
          ['左', 'left'],
          ['右', 'right'],
        ]),
        'VALUE',
      )
      this.setOutput(true, 'rotation')
      this.setColour(colours.動き)
    },
  }
  Blockly.Blocks.pico_unit = {
    init() {
      this.appendDummyInput().appendField('値なし')
      this.setOutput(true, 'unit')
      this.setColour(colours.関数)
    },
  }
  Blockly.Blocks.pico_ident = {
    init() {
      this.appendDummyInput().appendField('変数').appendField(new Blockly.FieldTextInput('x'), 'NAME')
      this.setOutput(true)
      this.setColour(colours.変数)
    },
  }
  Blockly.Blocks.pico_declare = {
    init() {
      this.appendDummyInput()
        .appendField(new Blockly.FieldDropdown(typeChoices), 'TYPE')
        .appendField('の変数')
        .appendField(new Blockly.FieldTextInput('x'), 'NAME')
        .appendField('を作る')
      this.appendValueInput('VALUE').setCheck('integer').appendField('最初の値')
      this.appendValueInput('TOKEN').appendField('この変数を使う')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.変数)
      this.setTooltip('下の小さな変数ブロックをドラッグすると、別の入力欄でこの変数を使えます。')
    },
  }
  Blockly.Blocks.pico_assign = {
    init() {
      this.appendDummyInput().appendField('変数').appendField(new Blockly.FieldTextInput('x'), 'NAME')
      this.appendValueInput('VALUE').appendField('をこの値に変える')
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
      this.appendStatementInput('THEN').appendField('なら')
      this.appendStatementInput('ELSE').appendField('でなければ')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_while = {
    init() {
      this.appendValueInput('CONDITION').setCheck('bool').appendField('この条件の間くり返す')
      this.appendStatementInput('BODY').appendField('すること')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_for = {
    init() {
      this.appendDummyInput()
        .appendField('数')
        .appendField(new Blockly.FieldTextInput('i'), 'NAME')
        .appendField('を1ずつ増やす')
      this.appendValueInput('START').setCheck('integer').appendField('はじめ')
      this.appendValueInput('END').setCheck('integer').appendField('ここまで（含まない）')
      this.appendStatementInput('BODY').appendField('すること')
      this.setPreviousStatement(true)
      this.setNextStatement(true)
      this.setColour(colours.制御)
    },
  }
  Blockly.Blocks.pico_return = {
    init() {
      this.appendValueInput('VALUE').appendField('この値を返す')
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
  for (const option of PICO_BUILTINS) defineCall(option.name, functions)
  for (const name of Object.keys(functions)) if (name !== 'main') defineCall(name, functions)
}
function defineCall(name: string, functions: Record<string, PicoFunction>): void {
  const signature = callSignature(name, functions)
  for (const statement of [false, true]) {
    Blockly.Blocks[callType(name, statement)] = {
      init() {
        this.appendDummyInput().appendField(signature.label)
        signature.args.forEach((type, index) =>
          this.appendValueInput(`ARG${index}`)
            .setCheck(typeCheck(type))
            .appendField(argumentLabel(name, index, functions)),
        )
        if (statement) {
          this.setPreviousStatement(true)
          this.setNextStatement(true)
        } else this.setOutput(true, typeCheck(signature.returns))
        this.setColour(colours[signature.category] ?? colours.関数)
        this.setTooltip(
          statement && !['動き', '操作', '表示'].includes(signature.category)
            ? 'この行だけでは返した値を使いません。変数や条件の入力欄で使ってください。'
            : '',
        )
      },
    }
  }
}

function exprState(expr: PicoExpr, functions: Record<string, PicoFunction>): State {
  if (expr.kind === 'ident')
    return { type: 'pico_ident', data: nodeData(expr.hintId), fields: { NAME: expr.name } }
  if (expr.kind === 'literal') {
    const type = { integer: 'pico_number', bool: 'pico_bool', rotation: 'pico_rotation', unit: 'pico_unit' }[
      expr.type
    ]
    return {
      type,
      data: nodeData(expr.hintId),
      fields: expr.type === 'unit' ? {} : { VALUE: expr.type === 'bool' ? String(expr.value) : expr.value },
    }
  }
  const signature = callSignature(expr.func, functions)
  return {
    type: callType(expr.func, false),
    data: nodeData(expr.hintId),
    inputs: Object.fromEntries(
      expr.args.map((arg, i) => [
        `ARG${i}`,
        {
          block: exprState(arg, functions),
          shadow: exprState(defaultExpr(signature.args[i] ?? 'integer'), functions),
        },
      ]),
    ),
  }
}
function stmtState(stmt: PicoStatement, functions: Record<string, PicoFunction>): State {
  if (stmt.kind === 'call') {
    const signature = callSignature(stmt.func, functions)
    return {
      type: callType(stmt.func, true),
      data: nodeData(stmt.hintId),
      inputs: Object.fromEntries(
        stmt.args.map((arg, i) => [
          `ARG${i}`,
          {
            block: exprState(arg, functions),
            shadow: exprState(defaultExpr(signature.args[i] ?? 'integer'), functions),
          },
        ]),
      ),
    }
  }
  if (stmt.kind === 'literal' || stmt.kind === 'ident')
    return {
      type: 'pico_expr_statement',
      data: nodeData(stmt.hintId),
      inputs: { VALUE: { block: exprState(stmt, functions) } },
    }
  if (stmt.kind === 'declare')
    return {
      type: 'pico_declare',
      data: nodeData(stmt.hintId, { identId: stmt.ident.hintId }),
      fields: { TYPE: typeKey(stmt.type), NAME: stmt.ident.name },
      inputs: {
        VALUE: {
          block: exprState(stmt.value, functions),
          shadow: exprState(defaultExpr(stmt.type), functions),
        },
        TOKEN: { shadow: exprState(ident(stmt.ident.name), functions) },
      },
    }
  if (stmt.kind === 'assign')
    return {
      type: 'pico_assign',
      data: nodeData(stmt.hintId, { identId: stmt.ident.hintId }),
      fields: { NAME: stmt.ident.name },
      inputs: { VALUE: { block: exprState(stmt.value, functions) } },
    }
  if (stmt.kind === 'if')
    return {
      type: stmt.else ? 'pico_if_else' : 'pico_if',
      data: nodeData(stmt.hintId, {
        thenId: stmt.then.hintId,
        ...(stmt.else ? { elseId: stmt.else.hintId } : {}),
      }),
      inputs: {
        CONDITION: {
          block: exprState(stmt.condition, functions),
          shadow: exprState(defaultExpr('bool'), functions),
        },
        ...(stmt.then.stmts.length ? { THEN: { block: chainState(stmt.then.stmts, functions)! } } : {}),
        ...(stmt.else?.stmts.length ? { ELSE: { block: chainState(stmt.else.stmts, functions)! } } : {}),
      },
    }
  if (stmt.kind === 'while' || stmt.kind === 'for') {
    const inputs: Record<string, Blockly.serialization.blocks.ConnectionState> = {}
    if (stmt.kind === 'while')
      inputs.CONDITION = {
        block: exprState(stmt.condition, functions),
        shadow: exprState(defaultExpr('bool'), functions),
      }
    else {
      inputs.START = {
        block: exprState(stmt.start, functions),
        shadow: exprState(defaultExpr('integer'), functions),
      }
      inputs.END = {
        block: exprState(stmt.end, functions),
        shadow: exprState(defaultExpr('integer'), functions),
      }
    }
    if (stmt.body.stmts.length) inputs.BODY = { block: chainState(stmt.body.stmts, functions)! }
    return {
      type: stmt.kind === 'for' ? 'pico_for' : 'pico_while',
      data: nodeData(stmt.hintId, {
        bodyId: stmt.body.hintId,
        ...(stmt.kind === 'for' ? { identId: stmt.ident.hintId } : {}),
      }),
      fields: stmt.kind === 'for' ? { NAME: stmt.ident.name } : {},
      inputs,
    }
  }
  if (stmt.kind === 'return')
    return {
      type: 'pico_return',
      data: nodeData(stmt.hintId),
      inputs: { VALUE: { block: exprState(stmt.value, functions) } },
    }
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
  const name = callName(block)
  const signature = callSignature(name, functions)
  return {
    kind: 'call',
    hintId: metaId(block),
    func: name,
    args: signature.args.map((type, index) => value(block, `ARG${index}`, type, functions)),
  }
}
function readStack(
  first: Block | null,
  functions: Record<string, PicoFunction>,
  returnType: PicoType,
): PicoStatement[] {
  const statements: PicoStatement[] = []
  for (let current = first; current; current = current.getNextBlock()) {
    const hintId = metaId(current)
    if (current.type === 'pico_declare') {
      const type = keyType(field(current, 'TYPE', 'integer'))
      statements.push({
        kind: 'declare',
        hintId,
        type,
        ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'x') },
        value: value(current, 'VALUE', type, functions),
      })
    } else if (current.type === 'pico_assign') {
      statements.push({
        kind: 'assign',
        hintId,
        ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'x') },
        value: value(current, 'VALUE', 'integer', functions),
      })
    } else if (current.type === 'pico_if' || current.type === 'pico_if_else') {
      const then = picoBlock(readStack(current.getInputTargetBlock('THEN'), functions, returnType))
      then.hintId = metaId(current, 'thenId')
      let otherwise: PicoBlock | undefined
      if (current.type === 'pico_if_else') {
        otherwise = picoBlock(readStack(current.getInputTargetBlock('ELSE'), functions, returnType))
        otherwise.hintId = metaId(current, 'elseId')
      }
      statements.push({
        kind: 'if',
        hintId,
        condition: value(current, 'CONDITION', 'bool', functions),
        then,
        ...(otherwise ? { else: otherwise } : {}),
      })
    } else if (current.type === 'pico_while' || current.type === 'pico_for') {
      const body = picoBlock(readStack(current.getInputTargetBlock('BODY'), functions, returnType))
      body.hintId = metaId(current, 'bodyId')
      if (current.type === 'pico_while')
        statements.push({
          kind: 'while',
          hintId,
          condition: value(current, 'CONDITION', 'bool', functions),
          body,
        })
      else
        statements.push({
          kind: 'for',
          hintId,
          ident: { kind: 'ident', hintId: metaId(current, 'identId'), name: field(current, 'NAME', 'i') },
          start: value(current, 'START', 'integer', functions),
          end: value(current, 'END', 'integer', functions),
          body,
        })
    } else if (current.type === 'pico_return')
      statements.push({ kind: 'return', hintId, value: value(current, 'VALUE', returnType, functions) })
    else if (current.type === 'pico_break' || current.type === 'pico_continue')
      statements.push({ kind: current.type === 'pico_break' ? 'break' : 'continue', hintId })
    else if (current.type === 'pico_expr_statement')
      statements.push(
        current.getInputTargetBlock('VALUE')
          ? readExpr(current.getInputTargetBlock('VALUE')!, functions)
          : value(current, 'VALUE', 'unit', functions),
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
  const statements = tops.flatMap((top) => readStack(top, functions, original.returnType))
  const returns = statements.filter((item) => item.kind === 'return')
  const originalReturn = original.body.stmts.find((item) => item.kind === 'return')
  const lastReturn = returns.at(-1) ??
    originalReturn ?? {
      kind: 'return' as const,
      hintId: id(),
      value: defaultExpr(original.returnType),
    }
  const body = picoBlock([...statements.filter((item) => item !== lastReturn), lastReturn])
  body.hintId = original.body.hintId
  return { ...original, body }
}

function toolboxBlock(
  type: string,
  args: PicoType[] = [],
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
  const inputs = Object.fromEntries(
    args.map((arg, i) => [`ARG${i}`, { shadow: exprState(defaultExpr(arg), functions) }]),
  )
  if (defaults[type])
    inputs[inputNames[type]!] = { shadow: exprState(defaultExpr(defaults[type]), functions) }
  if (type === 'pico_for') inputs.END = { shadow: exprState(defaultExpr('integer'), functions) }
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
    '動き',
    '操作',
    '制御',
    '変数',
    '調べる',
    '論理',
    '計算',
    '変換',
    '乱数',
    '配列',
    '表示',
    '関数',
  ]) {
    const contents: Blockly.utils.toolbox.ToolboxItemInfo[] = []
    if (category === '制御')
      for (const type of [...controlTypes, 'pico_break', 'pico_continue', 'pico_return'])
        contents.push(toolboxBlock(type))
    else if (category === '変数')
      for (const type of ['pico_declare', 'pico_assign', 'pico_ident']) contents.push(toolboxBlock(type))
    else if (category === '関数') {
      contents.push(toolboxBlock('pico_unit'))
      for (const name of Object.keys(functions).filter((item) => item !== 'main')) {
        contents.push(
          toolboxBlock(
            callType(name, true),
            functions[name]!.params.map((param) => param.type),
            functions,
          ),
        )
        if (functions[name]!.returnType !== 'unit')
          contents.push(
            toolboxBlock(
              callType(name, false),
              functions[name]!.params.map((param) => param.type),
              functions,
            ),
          )
      }
    } else {
      for (const builtin of PICO_BUILTINS.filter((item) => item.category === category))
        contents.push(
          toolboxBlock(
            callType(builtin.name, ['動き', '操作', '表示'].includes(category)),
            builtin.args,
            functions,
          ),
        )
      if (category === '計算') contents.unshift(toolboxBlock('pico_number'))
      if (category === '論理') contents.unshift(toolboxBlock('pico_bool'))
      if (category === '動き') contents.push(toolboxBlock('pico_rotation'))
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

export function refreshVariableBlocks(workspace: Blockly.Workspace, params: PicoFunction['params']): void {
  const variables = new Map(params.map((param) => [param.ident.name, param.type]))
  for (const block of workspace.getAllBlocks(false)) {
    if (block.type === 'pico_declare') {
      const name = field(block, 'NAME', 'x')
      const type = keyType(field(block, 'TYPE', 'integer'))
      variables.set(name, type)
      block.getInput('VALUE')?.connection?.setCheck(typeCheck(type))
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
    if (block.type === 'pico_ident')
      block.outputConnection?.setCheck(
        variables.has(field(block, 'NAME')) ? typeCheck(variables.get(field(block, 'NAME'))!) : null,
      )
    if (block.type === 'pico_assign')
      block
        .getInput('VALUE')
        ?.connection?.setCheck(
          variables.has(field(block, 'NAME')) ? typeCheck(variables.get(field(block, 'NAME'))!) : null,
        )
  }
}
