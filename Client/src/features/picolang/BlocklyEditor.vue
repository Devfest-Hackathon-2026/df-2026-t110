<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as Blockly from 'blockly'
import * as ja from 'blockly/msg/ja'
import { Icon } from '@iconify/vue'
import plusIcon from '@iconify-icons/lucide/plus'
import trashIcon from '@iconify-icons/lucide/trash-2'
import undoIcon from '@iconify-icons/lucide/undo-2'
import redoIcon from '@iconify-icons/lucide/redo-2'
import moveIcon from '@iconify-icons/lucide/move'
import mousePointerIcon from '@iconify-icons/lucide/mouse-pointer-2'
import gitBranchIcon from '@iconify-icons/lucide/git-branch'
import variableIcon from '@iconify-icons/lucide/variable'
import searchIcon from '@iconify-icons/lucide/search'
import bracesIcon from '@iconify-icons/lucide/braces'
import calculatorIcon from '@iconify-icons/lucide/calculator'
import repeatIcon from '@iconify-icons/lucide/repeat-2'
import shuffleIcon from '@iconify-icons/lucide/shuffle'
import listIcon from '@iconify-icons/lucide/list'
import typeIcon from '@iconify-icons/lucide/type'
import squareFunctionIcon from '@iconify-icons/lucide/square-function'
import focusIcon from '@iconify-icons/lucide/focus'
import zoomInIcon from '@iconify-icons/lucide/zoom-in'
import zoomOutIcon from '@iconify-icons/lucide/zoom-out'
import {
  keyType,
  loadFunction,
  picoToolbox,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
  typeKey,
} from './blocklyPico.ts'
import {
  block,
  defaultExpr,
  id,
  ident,
  isPicoProgram,
  newProgram,
  programFunctions,
  type PicoExpr,
  type PicoProgram,
  type PicoStatement,
} from './picolang.ts'
import { typeLabel } from './picoGuidance.ts'
import './blocklyEditor.css'

const props = defineProps<{ tutorialId: string }>()
const emit = defineEmits<{ 'update:program': [program: PicoProgram] }>()
const host = ref<HTMLElement | null>(null)
const program = ref<PicoProgram>(newProgram())
const activeName = ref('main')
const showAddFunction = ref(false)
const newName = ref('')
const showJson = ref(false)
const jsonInput = ref('')
const message = ref('')
const history = ref<string[]>([])
const historyIndex = ref(0)
const canUndo = computed(() => historyIndex.value > 0)
const canRedo = computed(() => historyIndex.value < history.value.length - 1)
const functions = computed(() => programFunctions(program.value))
const activeFunction = computed(() => functions.value[activeName.value] ?? functions.value.main)
const signature = computed(
  () =>
    `${activeName.value}(${activeFunction.value.params.map((param) => `${param.ident.name}: ${typeLabel(param.type)}`).join(', ')}) → ${typeLabel(activeFunction.value.returnType)}`,
)
const typeOptions: [string, string][] = [
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
let workspace: Blockly.WorkspaceSvg | null = null
let observer: ResizeObserver | null = null
let loading = false
let scheduled = false
const categoryIcons = {
  動き: moveIcon,
  操作: mousePointerIcon,
  制御: gitBranchIcon,
  変数: variableIcon,
  調べる: searchIcon,
  論理: bracesIcon,
  計算: calculatorIcon,
  変換: repeatIcon,
  乱数: shuffleIcon,
  配列: listIcon,
  表示: typeIcon,
  関数: squareFunctionIcon,
}

function decorateToolboxCategories(): void {
  if (!host.value) return
  for (const category of host.value.querySelectorAll('.blocklyToolboxCategoryContainer')) {
    const content = category.querySelector('.blocklyTreeRowContentContainer')
    const label = category.querySelector('.blocklyToolboxCategoryLabel')
    const name = label?.textContent?.trim() as keyof typeof categoryIcons | undefined
    if (
      !content ||
      !label ||
      !name ||
      !categoryIcons[name] ||
      content.querySelector('.blockly-pico-category-icon')
    )
      continue
    const icon = categoryIcons[name]
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
    svg.classList.add('blockly-pico-category-icon')
    svg.setAttribute('viewBox', `0 0 ${icon.width} ${icon.height}`)
    svg.setAttribute('aria-hidden', 'true')
    svg.innerHTML = icon.body
    content.insertBefore(svg, label)
  }
}

function decorateWorkspaceControls(): void {
  if (!host.value) return
  const controls = [
    ['.blocklyZoomReset', focusIcon],
    ['.blocklyZoomIn', zoomInIcon],
    ['.blocklyZoomOut', zoomOutIcon],
    ['.blocklyTrash', trashIcon],
  ] as const
  for (const [selector, icon] of controls) {
    const button = host.value.querySelector(selector)
    if (!button || button.querySelector('.blockly-pico-control-icon')) continue
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
    svg.classList.add('blockly-pico-control-icon')
    svg.setAttribute('viewBox', '0 0 32 32')
    svg.setAttribute('width', '32')
    svg.setAttribute('height', '32')
    svg.setAttribute('aria-hidden', 'true')
    svg.innerHTML = `<circle cx="16" cy="16" r="15" fill="#e6f2ed" stroke="#5f9186"/><g transform="translate(4 4)">${icon.body}</g>`
    if (selector === '.blocklyTrash') {
      svg.setAttribute('x', '8')
      svg.setAttribute('y', '18')
    }
    button.appendChild(svg)
  }
}

function save(): void {
  const snapshot = JSON.stringify(program.value)
  if (snapshot !== history.value[historyIndex.value]) {
    history.value.splice(historyIndex.value + 1)
    history.value.push(snapshot)
    if (history.value.length > 101) history.value.shift()
    historyIndex.value = history.value.length - 1
  }
  emit('update:program', program.value)
  try {
    localStorage.setItem(`picolang:draft:${props.tutorialId}`, snapshot)
  } catch {
    /* Storage may be disabled. */
  }
}
function readWorkspace(): void {
  if (!workspace || loading) return
  loading = true
  try {
    refreshVariableBlocks(workspace, activeFunction.value.params)
    functions.value[activeName.value] = readFunction(workspace, activeFunction.value, functions.value)
    save()
  } finally {
    loading = false
  }
}
function onWorkspaceChange(event: Blockly.Events.Abstract): void {
  if (loading || event.isUiEvent || scheduled) return
  scheduled = true
  queueMicrotask(() => {
    scheduled = false
    readWorkspace()
  })
}
function loadActive(): void {
  if (!workspace) return
  loading = true
  try {
    registerPicoBlocks(functions.value)
    workspace.updateToolbox(picoToolbox(functions.value))
    decorateToolboxCategories()
    loadFunction(workspace, activeFunction.value, functions.value)
    refreshVariableBlocks(workspace, activeFunction.value.params)
    Blockly.svgResize(workspace)
    decorateWorkspaceControls()
  } finally {
    loading = false
  }
}
function selectFunction(name: string): void {
  readWorkspace()
  activeName.value = name
  loadActive()
}
function restore(index: number): void {
  const snapshot = history.value[index]
  if (!snapshot) return
  loading = true
  try {
    historyIndex.value = index
    program.value = JSON.parse(snapshot) as PicoProgram
    if (!functions.value[activeName.value]) activeName.value = 'main'
  } finally {
    loading = false
  }
  loadActive()
  save()
}
function undo(): void {
  if (canUndo.value) restore(historyIndex.value - 1)
}
function redo(): void {
  if (canRedo.value) restore(historyIndex.value + 1)
}
function onKey(event: KeyboardEvent): void {
  if (!(event.ctrlKey || event.metaKey) || event.altKey) return
  if (event.target instanceof Element && event.target.closest('input, textarea, [contenteditable]')) return
  if (event.key.toLowerCase() === 'z') {
    event.preventDefault()
    if (event.shiftKey) redo()
    else undo()
  } else if (event.key.toLowerCase() === 'y') {
    event.preventDefault()
    redo()
  }
}
function addFunction(): void {
  const name = newName.value.trim()
  if (!name || functions.value[name] || name === 'main') {
    message.value = '別の関数名を入力してください。'
    return
  }
  readWorkspace()
  functions.value[name] = {
    kind: 'function',
    hintId: id(),
    params: [],
    returnType: 'unit',
    body: block([{ kind: 'return', hintId: id(), value: defaultExpr('unit') }]),
  }
  showAddFunction.value = false
  newName.value = ''
  activeName.value = name
  save()
  loadActive()
}
function removeFunction(): void {
  if (activeName.value === 'main') return
  const name = activeName.value
  if (JSON.stringify(program.value).includes(`"func":"${name}"`)) {
    message.value = 'この関数を使っているブロックがあります。先に外してください。'
    return
  }
  delete functions.value[name]
  activeName.value = 'main'
  save()
  loadActive()
}
function visitExpr(expr: PicoExpr, fn: (expr: PicoExpr) => void): void {
  fn(expr)
  if (expr.kind === 'call') expr.args.forEach((arg) => visitExpr(arg, fn))
}
function visitStatement(stmt: PicoStatement, fn: (expr: PicoExpr) => void): void {
  if (stmt.kind === 'call' || stmt.kind === 'ident' || stmt.kind === 'literal') visitExpr(stmt, fn)
  else if (stmt.kind === 'declare' || stmt.kind === 'assign' || stmt.kind === 'return')
    visitExpr(stmt.value, fn)
  else if (stmt.kind === 'if') {
    visitExpr(stmt.condition, fn)
    stmt.then.stmts.forEach((child) => visitStatement(child, fn))
    stmt.else?.stmts.forEach((child) => visitStatement(child, fn))
  } else if (stmt.kind === 'while') {
    visitExpr(stmt.condition, fn)
    stmt.body.stmts.forEach((child) => visitStatement(child, fn))
  } else if (stmt.kind === 'for') {
    visitExpr(stmt.start, fn)
    visitExpr(stmt.end, fn)
    stmt.body.stmts.forEach((child) => visitStatement(child, fn))
  }
}
function updateFunctionCalls(name: string): void {
  const params = functions.value[name]?.params ?? []
  Object.values(functions.value).forEach((definition) =>
    definition.body.stmts.forEach((stmt) =>
      visitStatement(stmt, (expr) => {
        if (expr.kind !== 'call' || expr.func !== name) return
        expr.args.splice(params.length)
        while (expr.args.length < params.length) expr.args.push(defaultExpr(params[expr.args.length]!.type))
      }),
    ),
  )
}
function addParam(): void {
  readWorkspace()
  const fn = activeFunction.value
  let name = `arg${fn.params.length + 1}`
  while (fn.params.some((param) => param.ident.name === name)) name += '_'
  fn.params.push({ kind: 'param', hintId: id(), ident: ident(name), type: 'integer' })
  updateFunctionCalls(activeName.value)
  save()
  loadActive()
}
function removeParam(index: number): void {
  readWorkspace()
  activeFunction.value.params.splice(index, 1)
  updateFunctionCalls(activeName.value)
  save()
  loadActive()
}
function changeParamType(index: number, key: string): void {
  readWorkspace()
  activeFunction.value.params[index]!.type = keyType(key)
  Object.values(functions.value).forEach((definition) =>
    definition.body.stmts.forEach((stmt) =>
      visitStatement(stmt, (expr) => {
        if (expr.kind === 'call' && expr.func === activeName.value && expr.args[index])
          expr.args[index] = defaultExpr(activeFunction.value.params[index]!.type)
      }),
    ),
  )
  save()
  loadActive()
}
function changeReturnType(key: string): void {
  readWorkspace()
  const fn = activeFunction.value
  fn.returnType = keyType(key)
  const last = fn.body.stmts.at(-1)
  if (last?.kind === 'return') last.value = defaultExpr(fn.returnType)
  save()
  loadActive()
}
function renameParam(index: number, value: string): void {
  const name = value.trim()
  const params = activeFunction.value.params
  if (!name || params.some((param, i) => i !== index && param.ident.name === name)) {
    message.value = '受け取る値には重ならない名前を付けてください。'
    return
  }
  readWorkspace()
  const fn = activeFunction.value
  const oldName = fn.params[index]!.ident.name
  fn.params[index]!.ident.name = name
  fn.body.stmts.forEach((stmt) =>
    visitStatement(stmt, (expr) => {
      if (expr.kind === 'ident' && expr.name === oldName) expr.name = name
    }),
  )
  save()
  loadActive()
}
async function copyJson(): Promise<void> {
  try {
    await navigator.clipboard.writeText(JSON.stringify(program.value, null, 2))
    message.value = 'JSONをコピーしました。'
  } catch {
    message.value = 'コピーできませんでした。'
  }
}
function importJson(): void {
  try {
    const parsed: unknown = JSON.parse(jsonInput.value)
    if (!isPicoProgram(parsed)) throw new Error('Picolang v0 の正しいJSONを入力してください。')
    program.value = parsed
    activeName.value = 'main'
    save()
    loadActive()
    message.value = 'JSONを読み込みました。'
  } catch (error) {
    message.value = error instanceof Error ? error.message : '読み込めませんでした。'
  }
}

watch(
  () => props.tutorialId,
  (tutorialId) => {
    let loaded: PicoProgram = newProgram()
    try {
      const saved = localStorage.getItem(`picolang:draft:${tutorialId}`)
      const parsed: unknown = saved ? JSON.parse(saved) : null
      if (isPicoProgram(parsed)) loaded = parsed
    } catch {
      /* Use an empty program. */
    }
    program.value = loaded
    activeName.value = 'main'
    history.value = [JSON.stringify(loaded)]
    historyIndex.value = 0
    nextTick(() => {
      loadActive()
      save()
    })
  },
  { immediate: true },
)

onMounted(() => {
  if (!host.value) return
  Blockly.setLocale(
    Object.fromEntries(
      Object.entries(ja).filter((entry): entry is [string, string] => typeof entry[1] === 'string'),
    ),
  )
  registerPicoBlocks(functions.value)
  workspace = Blockly.inject(host.value, {
    toolbox: picoToolbox(functions.value),
    renderer: 'zelos',
    trashcan: true,
    sounds: false,
    move: { scrollbars: true, drag: true, wheel: true },
    zoom: { controls: true, wheel: true, startScale: 0.9, minScale: 0.5, maxScale: 1.6 },
  })
  workspace.addChangeListener(onWorkspaceChange)
  loadActive()
  save()
  observer = new ResizeObserver(() => {
    if (workspace) Blockly.svgResize(workspace)
  })
  observer.observe(host.value)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  document.removeEventListener('keydown', onKey)
  workspace?.dispose()
  workspace = null
})
</script>

<template>
  <div class="blockly-editor">
    <div class="blockly-editor-toolbar">
      <nav class="blockly-function-tabs" aria-label="関数を選択">
        <button
          v-for="name in Object.keys(functions)"
          :key="name"
          type="button"
          :aria-pressed="activeName === name"
          @click="selectFunction(name)"
        >
          {{ name }}
        </button>
        <button v-if="!showAddFunction" type="button" aria-label="関数を追加" @click="showAddFunction = true">
          <Icon :icon="plusIcon" />
        </button>
        <form v-else @submit.prevent="addFunction">
          <input v-model="newName" aria-label="新しい関数名" placeholder="関数名" /><button type="submit">
            追加
          </button>
        </form>
      </nav>
      <div class="blockly-history">
        <button type="button" aria-label="元に戻す" :disabled="!canUndo" @click="undo">
          <Icon :icon="undoIcon" />
        </button>
        <button type="button" aria-label="やり直す" :disabled="!canRedo" @click="redo">
          <Icon :icon="redoIcon" />
        </button>
      </div>
    </div>
    <div class="blockly-signature">
      <strong>{{ signature }}</strong>
      <label v-if="activeName !== 'main'"
        >返す値の種類
        <select
          :value="typeKey(activeFunction.returnType)"
          @change="changeReturnType(($event.target as HTMLSelectElement).value)"
        >
          <option v-for="[label, key] in typeOptions" :key="key" :value="key">{{ label }}</option>
        </select></label
      >
      <button v-if="activeName !== 'main'" type="button" @click="removeFunction">
        <Icon :icon="trashIcon" />関数を削除
      </button>
    </div>
    <div v-if="activeName !== 'main'" class="blockly-params">
      <span>受け取る値</span>
      <div v-for="(param, index) in activeFunction.params" :key="param.hintId">
        <input
          :value="param.ident.name"
          :aria-label="`受け取る値 ${index + 1} の名前`"
          @change="renameParam(index, ($event.target as HTMLInputElement).value)"
        />
        <select
          :value="typeKey(param.type)"
          :aria-label="`受け取る値 ${index + 1} の種類`"
          @change="changeParamType(index, ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="[label, key] in typeOptions" :key="key" :value="key">{{ label }}</option>
        </select>
        <button type="button" :aria-label="`受け取る値 ${index + 1} を削除`" @click="removeParam(index)">
          <Icon :icon="trashIcon" />
        </button>
      </div>
      <button type="button" @click="addParam"><Icon :icon="plusIcon" />受け取る値を追加</button>
    </div>
    <div ref="host" class="blockly-canvas" aria-label="Picolangのブロックエディタ"></div>
    <details
      class="blockly-json"
      :open="showJson"
      @toggle="showJson = ($event.target as HTMLDetailsElement).open"
    >
      <summary>Schema形式のJSON</summary>
      <button type="button" @click="copyJson">JSONをコピー</button>
      <pre>{{ JSON.stringify(program, null, 2) }}</pre>
      <label for="blockly-json-input">JSONを読み込む</label>
      <textarea
        id="blockly-json-input"
        v-model="jsonInput"
        placeholder="Picolang v0のJSONを貼り付け"
      ></textarea>
      <button type="button" @click="importJson">読み込む</button>
    </details>
    <p v-if="message" class="blockly-message" role="status">{{ message }}</p>
  </div>
</template>
