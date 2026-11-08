<script setup lang="ts">
import { computed, ref } from 'vue'
import { Icon } from '@iconify/vue'
import arrowUpIcon from '@iconify-icons/lucide/arrow-up'
import arrowDownIcon from '@iconify-icons/lucide/arrow-down'
import trashIcon from '@iconify-icons/lucide/trash-2'
import plusIcon from '@iconify-icons/lucide/plus'
import PicoCondition from './PicoCondition.vue'
import PicoExpression from './PicoExpression.vue'
import { typeLabel } from './picoGuidance.ts'
import { dragVariable, endPicoDrag } from './picoDrag.ts'
import { PICO_BUILTIN_OPTIONS, type PicoVariableOption } from './picoOptions.ts'
import {
  block as makeBlock,
  defaultExpr,
  makeStatement,
  type PicoBlock,
  type PicoFunction,
  type PicoStatement,
  type PicoType,
} from '../language/picolang.ts'

defineOptions({ name: 'PicoBlock' })
const props = defineProps<{
  block: PicoBlock
  functions: Record<string, PicoFunction>
  variables?: PicoVariableOption[]
  inferredTypes?: Record<string, PicoType | undefined>
  returnType?: PicoType
  loopDepth?: number
  terminalReturn?: boolean
  emptyHint?: string
}>()
const statements = computed(() => props.block.stmts)
const choice = ref('forward')
const dragInsert = ref<{ index: number; after: boolean } | null>(null)
const controlChoices = [
  { value: 'if', label: 'もし' },
  { value: 'if-else', label: 'もし／でなければ' },
  { value: 'while', label: '〜の間くり返す' },
  { value: 'for', label: '数を1ずつ増やしてくり返す' },
  { value: 'declare', label: '変数を作る' },
  { value: 'assign', label: '変数の値を変える' },
  { value: 'return', label: 'この関数から返す' },
  { value: 'break', label: 'くり返しを抜ける' },
  { value: 'continue', label: '次のくり返しへ' },
]
function availableChoices(index: number): typeof controlChoices {
  return controlChoices.filter(
    (item) =>
      !(['break', 'continue'].includes(item.value) && !props.loopDepth) &&
      !(item.value === 'assign' && !variablesBefore(index).length),
  )
}
function createStatement(
  value: string,
  index: number,
  visible = variablesBefore(index),
  loopDepth = props.loopDepth ?? 0,
): PicoStatement | null {
  if (['break', 'continue'].includes(value) && !loopDepth) return null
  if (value === 'assign' && !visible.length) return null
  const statement = makeStatement(value, props.functions)
  if (statement.kind === 'assign') {
    statement.ident.name = visible[0]!.name
    statement.value = defaultExpr(visible[0]!.type)
  }
  if (statement.kind === 'return') statement.value = defaultExpr(props.returnType ?? 'unit')
  if (statement.kind === 'declare' || statement.kind === 'for') {
    const base = statement.kind === 'for' ? 'i' : 'x'
    let name = base
    for (let suffix = 2; visible.some((item) => item.name === name); suffix++) name = `${base}${suffix}`
    statement.ident.name = name
  }
  return statement
}
function variablesBefore(index: number): PicoVariableOption[] {
  const visible = new Map<string, PicoVariableOption>()
  for (const variable of props.variables ?? []) if (variable.name.trim()) visible.set(variable.name, variable)
  for (const statement of statements.value.slice(0, index)) {
    if (statement.kind === 'declare' && statement.ident.name.trim())
      visible.set(statement.ident.name, {
        name: statement.ident.name,
        type: props.inferredTypes?.[statement.ident.hintId],
      })
  }
  return [...visible.values()]
}
function append(value = choice.value): void {
  const last = statements.value.at(-1)
  const index =
    props.terminalReturn && last?.kind === 'return' ? statements.value.length - 1 : statements.value.length
  const statement = createStatement(value, index)
  if (!statement) return
  if (props.terminalReturn && last?.kind === 'return') statements.value.splice(index, 0, statement)
  else statements.value.push(statement)
}
function move(index: number, delta: number): void {
  const target = index + delta
  if (target < 0 || target >= statements.value.length) return
  if (
    props.terminalReturn &&
    (statements.value[index]?.kind === 'return' || statements.value[target]?.kind === 'return')
  )
    return
  const [item] = statements.value.splice(index, 1)
  statements.value.splice(target, 0, item)
}
function drop(event: DragEvent): void {
  event.preventDefault()
  event.stopPropagation()
  const value = event.dataTransfer?.getData('application/x-picolang-choice')
  if (
    value &&
    (controlChoices.some((item) => item.value === value) ||
      PICO_BUILTIN_OPTIONS.some((item) => item.name === value) ||
      value in props.functions)
  )
    append(value)
}
function dragOverAt(index: number, event: DragEvent): void {
  if (!event.dataTransfer?.types.includes('application/x-picolang-choice')) return
  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect()
  dragInsert.value = { index, after: event.clientY >= bounds.top + bounds.height / 2 }
}
function dropAt(index: number, event: DragEvent): void {
  event.preventDefault()
  event.stopPropagation()
  const value = event.dataTransfer?.getData('application/x-picolang-choice')
  const after = dragInsert.value?.index === index && dragInsert.value.after
  dragInsert.value = null
  if (
    !value ||
    !(
      controlChoices.some((item) => item.value === value) ||
      PICO_BUILTIN_OPTIONS.some((item) => item.name === value) ||
      value in props.functions
    )
  )
    return
  const position = Math.min(
    index + (after ? 1 : 0),
    props.terminalReturn ? statements.value.length - 1 : statements.value.length,
  )
  const statement = createStatement(value, position)
  if (statement) statements.value.splice(position, 0, statement)
}
function dropInto(
  target: PicoBlock,
  index: number,
  event: DragEvent,
  insideLoop = false,
  loopVariable?: string,
): void {
  event.preventDefault()
  event.stopPropagation()
  const value = event.dataTransfer?.getData('application/x-picolang-choice')
  if (
    value &&
    (controlChoices.some((item) => item.value === value) ||
      PICO_BUILTIN_OPTIONS.some((item) => item.name === value) ||
      value in props.functions)
  ) {
    const visible = variablesBefore(index)
    if (loopVariable) visible.push({ name: loopVariable, type: 'integer' })
    const statement = createStatement(value, index, visible, (props.loopDepth ?? 0) + (insideLoop ? 1 : 0))
    if (statement) target.stmts.push(statement)
  }
}
function changeAssignmentTarget(
  stmt: Extract<PicoStatement, { kind: 'assign' }>,
  index: number,
  name: string,
): void {
  const target = variablesBefore(index).find((variable) => variable.name === name)
  if (!target) return
  stmt.ident.name = name
  stmt.value = defaultExpr(target.type)
}
function replace(index: number, node: PicoStatement): void {
  statements.value[index] = node
}
function addElse(stmt: Extract<PicoStatement, { kind: 'if' }>): void {
  stmt.else = makeBlock()
}
function remove(index: number): void {
  if (props.terminalReturn && statements.value[index]?.kind === 'return') return
  statements.value.splice(index, 1)
}
</script>

<template>
  <div class="pico-block-list" @dragover.stop.prevent @drop="drop">
    <p v-if="!statements.length" class="pico-drop-hint">
      {{ emptyHint ?? 'ここへ命令を置く' }}
    </p>
    <div
      v-for="(stmt, index) in statements"
      :key="stmt.hintId"
      class="pico-statement"
      :class="[
        `pico-statement--${stmt.kind}`,
        dragInsert?.index === index
          ? dragInsert.after
            ? 'pico-statement--drop-after'
            : 'pico-statement--drop-before'
          : '',
      ]"
      @dragover.stop.prevent="dragOverAt(index, $event)"
      @drop.stop.prevent="dropAt(index, $event)"
      @dragleave="dragInsert = null"
    >
      <div class="pico-statement-toolbar">
        <div>
          <button
            type="button"
            :disabled="index === 0"
            :aria-label="`${index + 1}番目を上へ`"
            @click="move(index, -1)"
          >
            <Icon :icon="arrowUpIcon" aria-hidden="true" />
          </button>
          <button
            type="button"
            :disabled="index === statements.length - 1"
            :aria-label="`${index + 1}番目を下へ`"
            @click="move(index, 1)"
          >
            <Icon :icon="arrowDownIcon" aria-hidden="true" />
          </button>
          <button
            type="button"
            :disabled="terminalReturn && stmt.kind === 'return'"
            :aria-label="`${index + 1}番目を削除`"
            @click="remove(index)"
          >
            <Icon :icon="trashIcon" aria-hidden="true" />
          </button>
        </div>
      </div>
      <div class="pico-statement-content">
        <div v-if="stmt.kind === 'if'" class="pico-control-shell">
          <div class="pico-control-head">
            <strong>もし</strong>
            <span class="pico-control-brace" aria-hidden="true">{</span>
          </div>
          <div class="pico-control-settings">
            <span class="pico-control-field-label">実行する条件</span>
            <PicoCondition
              :node="stmt.condition"
              :functions="functions"
              :variables="variablesBefore(index)"
              @replace="stmt.condition = $event"
            />
            <p class="pico-condition-help">答えが「はい」なら、下の命令を実行します。</p>
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="もしの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.then, index, $event)"
          >
            <span class="pico-scope-label">条件が真のとき</span>
            <PicoBlock
              :inferred-types="inferredTypes"
              :block="stmt.then"
              :functions="functions"
              :variables="variablesBefore(index)"
              :return-type="returnType"
              :loop-depth="loopDepth"
              empty-hint="条件が成り立ったときの命令をここに追加"
            />
          </div>
          <template v-if="stmt.else">
            <div class="pico-control-middle">
              <span aria-hidden="true">}</span><strong>でなければ</strong><span aria-hidden="true">{</span>
            </div>
            <div
              class="pico-control-scope"
              role="group"
              aria-label="でなければの中の処理"
              @dragover.stop.prevent
              @drop="dropInto(stmt.else, index, $event)"
            >
              <span class="pico-scope-label">条件が偽のとき</span>
              <PicoBlock
                :inferred-types="inferredTypes"
                :block="stmt.else"
                :functions="functions"
                :variables="variablesBefore(index)"
                :return-type="returnType"
                :loop-depth="loopDepth"
                empty-hint="条件が成り立たないときの命令をここに追加"
              />
            </div>
          </template>
          <button v-else type="button" class="pico-control-add-else" @click="addElse(stmt)">
            <Icon :icon="plusIcon" aria-hidden="true" />でなければ
          </button>
          <div class="pico-control-tail" aria-hidden="true">}</div>
        </div>
        <div v-else-if="stmt.kind === 'while'" class="pico-control-shell">
          <div class="pico-control-head">
            <strong>条件が成り立つ間くり返す</strong>
            <span class="pico-control-brace" aria-hidden="true">{</span>
          </div>
          <div class="pico-control-settings">
            <span class="pico-control-field-label">くり返す条件</span>
            <PicoCondition
              :node="stmt.condition"
              :functions="functions"
              :variables="variablesBefore(index)"
              @replace="stmt.condition = $event"
            />
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="whileの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.body, index, $event, true)"
          >
            <span class="pico-scope-label">くり返す処理</span>
            <PicoBlock
              :inferred-types="inferredTypes"
              :block="stmt.body"
              :functions="functions"
              :variables="variablesBefore(index)"
              :return-type="returnType"
              :loop-depth="(loopDepth ?? 0) + 1"
            />
          </div>
          <div class="pico-control-tail" aria-hidden="true">}</div>
        </div>
        <div v-else-if="stmt.kind === 'for'" class="pico-control-shell">
          <div class="pico-control-head">
            <strong>数を1ずつ増やしてくり返す</strong>
            <span class="pico-control-brace" aria-hidden="true">{</span>
          </div>
          <div class="pico-control-settings pico-control-settings--for">
            <div class="pico-control-field">
              <label :for="`pico-loop-name-${stmt.hintId}`">変数名</label>
              <input
                :id="`pico-loop-name-${stmt.hintId}`"
                v-model="stmt.ident.name"
                class="pico-name-input"
                placeholder="i"
                data-1p-ignore
              />
              <button
                type="button"
                class="pico-variable-token"
                draggable="true"
                :aria-label="`変数 ${stmt.ident.name} をドラッグ`"
                @dragstart="dragVariable($event, stmt.ident.name)"
                @dragend="endPicoDrag"
              >
                {{ stmt.ident.name }}
              </button>
            </div>
            <div class="pico-control-field" role="group" aria-label="最初の値">
              <span class="pico-control-field-label">最初の値</span>
              <PicoExpression
                :node="stmt.start"
                :functions="functions"
                :variables="variablesBefore(index)"
                expected-type="integer"
                @replace="stmt.start = $event"
              />
            </div>
            <div class="pico-control-field" role="group" aria-label="終わりの値。この値は含まない">
              <span class="pico-control-field-label">終わりの値 <small>含まない</small></span>
              <PicoExpression
                :node="stmt.end"
                :functions="functions"
                :variables="variablesBefore(index)"
                expected-type="integer"
                @replace="stmt.end = $event"
              />
            </div>
            <p class="pico-control-help">変数を最初の値から1ずつ増やし、終わりの値の手前で止めます。</p>
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="forの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.body, index, $event, true, stmt.ident.name)"
          >
            <span class="pico-scope-label">くり返す処理</span>
            <PicoBlock
              :inferred-types="inferredTypes"
              :block="stmt.body"
              :functions="functions"
              :variables="[...variablesBefore(index), { name: stmt.ident.name, type: 'integer' }]"
              :return-type="returnType"
              :loop-depth="(loopDepth ?? 0) + 1"
            />
          </div>
          <div class="pico-control-tail" aria-hidden="true">}</div>
        </div>
        <PicoExpression
          v-else-if="stmt.kind === 'call' || stmt.kind === 'literal' || stmt.kind === 'ident'"
          :node="stmt"
          :functions="functions"
          :variables="variablesBefore(index)"
          :statement="stmt.kind === 'call'"
          @replace="replace(index, $event)"
        />
        <template v-else-if="stmt.kind === 'declare'">
          <span>変数を作る</span>
          <span>{{ typeLabel(inferredTypes?.[stmt.ident.hintId]) }}</span>
          <input v-model="stmt.ident.name" class="pico-name-input" aria-label="変数名" data-1p-ignore />
          <span>に入れる値</span>
          <PicoExpression
            :node="stmt.value"
            :functions="functions"
            :variables="variablesBefore(index)"
            @replace="stmt.value = $event"
          />
          <small class="pico-variable-tip">
            <button
              type="button"
              class="pico-variable-token"
              draggable="true"
              :aria-label="`変数 ${stmt.ident.name} をドラッグ`"
              @dragstart="dragVariable($event, stmt.ident.name)"
              @dragend="endPicoDrag"
            >
              {{ stmt.ident.name || '変数' }}
            </button>
            を入力欄へドラッグして使えます。
          </small>
        </template>
        <template v-else-if="stmt.kind === 'assign'">
          <span>変数</span>
          <select
            :value="
              variablesBefore(index).some((variable) => variable.name === stmt.ident.name)
                ? stmt.ident.name
                : ''
            "
            aria-label="値を変える変数"
            @change="changeAssignmentTarget(stmt, index, ($event.target as HTMLSelectElement).value)"
          >
            <option value="" disabled>変数を選ぶ</option>
            <option v-for="variable in variablesBefore(index)" :key="variable.name" :value="variable.name">
              {{ variable.name }}
            </option>
          </select>
          <span>をこの値に変える</span>
          <PicoExpression
            :node="stmt.value"
            :functions="functions"
            :variables="variablesBefore(index)"
            :expected-type="
              variablesBefore(index).find((variable) => variable.name === stmt.ident.name)?.type
            "
            @replace="stmt.value = $event"
          />
        </template>
        <template v-else-if="stmt.kind === 'return'">
          <span>この値を返す</span>
          <PicoExpression
            :node="stmt.value"
            :functions="functions"
            :variables="variablesBefore(index)"
            :expected-type="returnType"
            @replace="stmt.value = $event"
          />
        </template>
        <span v-else>{{ stmt.kind === 'break' ? 'くり返しを抜ける' : '次のくり返しへ' }}</span>
      </div>
    </div>
    <div class="pico-add-row">
      <select v-model="choice" aria-label="追加する命令">
        <optgroup label="動きと操作">
          <option
            v-for="item in PICO_BUILTIN_OPTIONS.filter((item) => ['動き', '操作'].includes(item.category))"
            :key="item.name"
            :value="item.name"
          >
            {{ item.label }}
          </option>
        </optgroup>
        <optgroup label="制御と変数">
          <option v-for="item in availableChoices(statements.length)" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </optgroup>
        <optgroup label="組み込み関数">
          <option
            v-for="item in PICO_BUILTIN_OPTIONS.filter((item) => !['動き', '操作'].includes(item.category))"
            :key="item.name"
            :value="item.name"
          >
            {{ item.label }}
          </option>
        </optgroup>
        <optgroup v-if="Object.keys(functions).length > 1" label="自分の関数">
          <option
            v-for="name in Object.keys(functions).filter((name) => name !== 'main')"
            :key="name"
            :value="name"
          >
            {{ name }}
          </option>
        </optgroup>
      </select>
      <button type="button" @click="append()"><Icon :icon="plusIcon" aria-hidden="true" />追加</button>
    </div>
  </div>
</template>
