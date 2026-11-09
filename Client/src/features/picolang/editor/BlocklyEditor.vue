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
import shuffleIcon from '@iconify-icons/lucide/shuffle'
import listIcon from '@iconify-icons/lucide/list'
import typeIcon from '@iconify-icons/lucide/type'
import squareFunctionIcon from '@iconify-icons/lucide/square-function'
import boxIcon from '@iconify-icons/lucide/box'
import focusIcon from '@iconify-icons/lucide/focus'
import zoomInIcon from '@iconify-icons/lucide/zoom-in'
import zoomOutIcon from '@iconify-icons/lucide/zoom-out'
import {
  diagnosticBlock,
  loadFunction,
  picoToolbox,
  readFunction,
  refreshVariableBlocks,
  registerPicoBlocks,
  updatePicoDiagnostics,
} from './blocklyPico.ts'
import PicoDiagnostics from './PicoDiagnostics.vue'
import PicoExecutionTrace from './PicoExecutionTrace.vue'
import type { PicolangTrace, TraceNode } from '../language/picolangTrace.ts'
import { inferPicolang, type PicoDiagnostic } from '../language/analyzePicolang.ts'
import {
  block,
  defaultExpr,
  id,
  ident,
  isPicoDraft,
  picoDraft,
  newProgram,
  programFunctions,
  type PicoExpr,
  type PicoProgram,
  type PicoStatement,
} from '../language/picolang.ts'
import { typeLabel } from './picoGuidance.ts'
import './blocklyEditor.css'

const props = defineProps<{
  tutorialId: string
  execution?: PicolangTrace | null
  running?: boolean
  playbackReady?: boolean
  paused?: boolean
  nodeInterval?: number
}>()
const emit = defineEmits<{
  'update:program': [program: PicoProgram | null]
  pause: []
  step: []
  interval: [value: number]
}>()
const host = ref<HTMLElement | null>(null)
const program = ref<PicoProgram>(picoDraft(newProgram()))
const activeName = ref('main')
const showAddFunction = ref(false)
const newName = ref('')
const showJson = ref(false)
const jsonInput = ref('')
const message = ref('')
const messageSeverity = ref<'error' | 'success'>('error')
function showMessage(text: string, severity: 'error' | 'success' = 'error'): void {
  message.value = text
  messageSeverity.value = severity
}
const history = ref<string[]>([])
const historyIndex = ref(0)
const canUndo = computed(() => historyIndex.value > 0)
const canRedo = computed(() => historyIndex.value < history.value.length - 1)
const functions = computed(() => programFunctions(program.value))
const analysis = computed(() => inferPicolang(program.value))
const diagnostics = computed(() => analysis.value.diagnostics)
const executionJson = computed(() =>
  analysis.value.program ? JSON.stringify(analysis.value.program, null, 2) : null,
)
const activeFunction = computed(() => functions.value[activeName.value] ?? functions.value.main)
const signature = computed(() => {
  const inferred = analysis.value.signatures[activeName.value]
  return `${activeName.value}(${activeFunction.value.params.map((param, index) => `${param.ident.name}: ${typeLabel(inferred?.params[index])}`).join(', ')}) → ${typeLabel(inferred?.returns)}`
})
let workspace: Blockly.WorkspaceSvg | null = null
let observer: ResizeObserver | null = null
let loading = false
let scheduled = false
const categoryIcons = {
  動く: moveIcon,
  オブジェクト: mousePointerIcon,
  制御: gitBranchIcon,
  変数: variableIcon,
  調べる: searchIcon,
  論理: bracesIcon,
  計算: calculatorIcon,
  ランダム: shuffleIcon,
  スタック: listIcon,
  ヌラブル: boxIcon,
  発言: typeIcon,
  関数: squareFunctionIcon,
  値: boxIcon,
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
  emit('update:program', analysis.value.program)
  try {
    localStorage.setItem(`picolang:draft:${props.tutorialId}`, snapshot)
  } catch {
    /* Storage may be disabled. */
  }
}
function readWorkspace(): void {
  if (!workspace || loading || props.running) return
  loading = true
  try {
    refreshVariableBlocks(workspace)
    functions.value[activeName.value] = readFunction(workspace, activeFunction.value, functions.value)
    save()
  } finally {
    loading = false
  }
}
function onWorkspaceChange(event: Blockly.Events.Abstract): void {
  if (loading || props.running || event.isUiEvent || scheduled) return
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
    refreshVariableBlocks(workspace)
    updatePicoDiagnostics(
      workspace,
      diagnostics.value.filter((item) => item.functionName === activeName.value),
      analysis.value.types,
    )
    Blockly.svgResize(workspace)
    decorateWorkspaceControls()
    lockWorkspace()
    highlightExecution()
  } finally {
    loading = false
  }
}
function lockWorkspace(): void {
  if (!workspace) return
  Blockly.Events.disable()
  try {
    workspace.setIsReadOnly(Boolean(props.running))
    workspace.getToolbox()?.setVisible(!props.running)
    if (props.running) {
      Blockly.hideChaff()
      workspace.getFlyout()?.hide()
    }
  } finally {
    Blockly.Events.enable()
  }
}
function highlightExecution(): void {
  if (!workspace) return
  const node = props.execution?.current
  const target = node?.functionName === activeName.value ? diagnosticBlock(workspace, node.hintId) : null
  workspace.highlightBlock(target?.id ?? null)
  if (props.running && target instanceof Blockly.BlockSvg)
    workspace.scrollBoundsIntoView(target.getBoundingRectangleWithoutChildren())
}
function locateExecution(node: TraceNode): void {
  if (activeName.value !== node.functionName && functions.value[node.functionName])
    selectFunction(node.functionName)
  if (!workspace) return
  const target = diagnosticBlock(workspace, node.hintId)
  if (target) workspace.centerOnBlock(target.id)
}
watch(
  () => props.running,
  () => lockWorkspace(),
)
watch(
  () => props.execution,
  (trace) => {
    const name = trace?.current?.functionName
    if (props.running && name && activeName.value !== name && functions.value[name]) selectFunction(name)
    highlightExecution()
  },
)
function selectFunction(name: string): void {
  readWorkspace()
  activeName.value = name
  loadActive()
}
function selectDiagnostic(diagnostic: PicoDiagnostic): void {
  if (activeName.value !== diagnostic.functionName) selectFunction(diagnostic.functionName)
  if (!workspace) return
  const target = diagnosticBlock(workspace, diagnostic.hintId)
  if (target instanceof Blockly.BlockSvg) {
    workspace.centerOnBlock(target.id)
    target.select()
  }
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
  if (props.running) return
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
    showMessage('別の関数名を入力してください。')
    return
  }
  readWorkspace()
  functions.value[name] = {
    kind: 'function',
    hintId: id(),
    params: [],
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
    showMessage('この関数を使っているブロックがあります。先に外してください。')
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
        while (expr.args.length < params.length) expr.args.push(defaultExpr())
      }),
    ),
  )
}
function addParam(): void {
  readWorkspace()
  const fn = activeFunction.value
  let name = `arg${fn.params.length + 1}`
  while (fn.params.some((param) => param.ident.name === name)) name += '_'
  fn.params.push({ kind: 'param', hintId: id(), ident: ident(name) })
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
function renameParam(index: number, value: string): void {
  const name = value.trim()
  const params = activeFunction.value.params
  if (!name || params.some((param, i) => i !== index && param.ident.name === name)) {
    showMessage('受け取る値には重ならない名前を付けてください。')
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
    if (!executionJson.value) {
      showMessage('エラーを解消するとJSONを出力できます。')
      return
    }
    await navigator.clipboard.writeText(executionJson.value)
    showMessage('JSONをコピーしました。', 'success')
  } catch {
    showMessage('コピーできませんでした。')
  }
}
function importJson(): void {
  try {
    const parsed: unknown = JSON.parse(jsonInput.value)
    if (!isPicoDraft(parsed)) throw new Error('Picolang v0 の正しいJSONを入力してください。')
    program.value = picoDraft(parsed)
    activeName.value = 'main'
    save()
    loadActive()
    showMessage('JSONを読み込みました。', 'success')
  } catch (error) {
    showMessage(error instanceof Error ? error.message : '読み込めませんでした。')
  }
}

watch(analysis, (result) => {
  if (!workspace) return
  updatePicoDiagnostics(
    workspace,
    result.diagnostics.filter((item) => item.functionName === activeName.value),
    result.types,
  )
})

watch(
  () => props.tutorialId,
  (tutorialId) => {
    let loaded: PicoProgram = picoDraft(newProgram())
    try {
      const saved = localStorage.getItem(`picolang:draft:${tutorialId}`)
      const parsed: unknown = saved ? JSON.parse(saved) : null
      if (isPicoDraft(parsed)) loaded = picoDraft(parsed)
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
    if (workspace) {
      Blockly.svgResize(workspace)
      highlightExecution()
    }
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
        <button
          v-if="!showAddFunction"
          type="button"
          aria-label="関数を追加"
          :disabled="running"
          @click="showAddFunction = true"
        >
          <Icon :icon="plusIcon" />
        </button>
        <form v-else @submit.prevent="addFunction">
          <input
            v-model="newName"
            :disabled="running"
            aria-label="新しい関数名"
            placeholder="関数名"
            data-1p-ignore
          /><button type="submit" :disabled="running">追加</button>
        </form>
      </nav>
      <div class="blockly-history">
        <button type="button" aria-label="元に戻す" :disabled="running || !canUndo" @click="undo">
          <Icon :icon="undoIcon" />
        </button>
        <button type="button" aria-label="やり直す" :disabled="running || !canRedo" @click="redo">
          <Icon :icon="redoIcon" />
        </button>
      </div>
    </div>
    <div class="blockly-signature">
      <strong>{{ signature }}</strong>
      <button v-if="activeName !== 'main' && !running" type="button" @click="removeFunction">
        <Icon :icon="trashIcon" />関数を削除
      </button>
    </div>
    <div v-if="activeName !== 'main' && !running" class="blockly-params">
      <span>受け取る値</span>
      <div v-for="(param, index) in activeFunction.params" :key="param.hintId">
        <input
          :value="param.ident.name"
          :disabled="running"
          :aria-label="`受け取る値 ${index + 1} の名前`"
          data-1p-ignore
          @change="renameParam(index, ($event.target as HTMLInputElement).value)"
        />
        <span>{{ typeLabel(analysis.signatures[activeName]?.params[index]) }}</span>
        <button
          type="button"
          :disabled="running"
          :aria-label="`受け取る値 ${index + 1} を削除`"
          @click="removeParam(index)"
        >
          <Icon :icon="trashIcon" />
        </button>
      </div>
      <button type="button" :disabled="running" @click="addParam">
        <Icon :icon="plusIcon" />受け取る値を追加
      </button>
    </div>
    <div ref="host" class="blockly-canvas" aria-label="Picolangのブロックエディタ"></div>
    <PicoExecutionTrace
      v-if="running || execution"
      :trace="execution ?? null"
      :running="Boolean(running)"
      :ready="Boolean(playbackReady)"
      :paused="Boolean(paused)"
      :interval="nodeInterval ?? 300"
      @pause="emit('pause')"
      @step="emit('step')"
      @interval="emit('interval', $event)"
      @locate="locateExecution"
    />
    <PicoDiagnostics :diagnostics="diagnostics" @select="selectDiagnostic" />
    <details
      class="blockly-json"
      :open="showJson"
      @toggle="showJson = ($event.target as HTMLDetailsElement).open"
    >
      <summary>プログラムのJSON</summary>
      <button type="button" :disabled="!executionJson" @click="copyJson">JSONをコピー</button>
      <pre>{{ executionJson ?? 'エラーを直すと、プログラムのJSONを表示できます。' }}</pre>
      <label for="blockly-json-input">JSONを読み込む</label>
      <textarea
        id="blockly-json-input"
        v-model="jsonInput"
        :disabled="running"
        placeholder="Picolang v0のJSONを貼り付け"
        data-1p-ignore
      ></textarea>
      <button type="button" :disabled="running" @click="importJson">読み込む</button>
    </details>
    <p v-if="message && !running" class="blockly-message" :data-severity="messageSeverity" role="status">
      {{ message }}
    </p>
  </div>
</template>
