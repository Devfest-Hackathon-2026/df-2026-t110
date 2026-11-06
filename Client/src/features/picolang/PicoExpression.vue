<script setup lang="ts">
import { computed, ref } from 'vue'
import { expressionFromDrop, PICO_VARIABLE_MIME } from './picoDrag.ts'
import {
  PICO_BUILTINS,
  defaultExpr,
  ident,
  makeCall,
  type PicoExpr,
  type PicoFunction,
  type PicoType,
  type PicoVariableOption,
} from './picolang.ts'

const props = defineProps<{
  node: PicoExpr
  functions: Record<string, PicoFunction>
  variables?: PicoVariableOption[]
  statement?: boolean
  defaultCall?: string
  expectedType?: PicoType
}>()
const emit = defineEmits<{ replace: [node: PicoExpr] }>()
const dragActive = ref(false)
const dropError = ref('')
const mode = computed(() => (props.node.kind === 'literal' ? props.node.type : props.node.kind))
const identifierName = computed(() => (props.node.kind === 'ident' ? props.node.name : ''))
const options = computed(() => [
  ...PICO_BUILTINS,
  ...Object.entries(props.functions).map(([name, definition]) => ({
    name,
    label: name,
    category: '関数',
    args: definition.params.map((param) => param.type),
    returns: definition.returnType,
  })),
])
const callLabel = computed(() => {
  const node = props.node
  return node.kind === 'call'
    ? (options.value.find((option) => option.name === node.func)?.label ?? node.func)
    : ''
})
function changeMode(event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  if (value === 'ident') emit('replace', ident(props.variables?.[0]?.name ?? 'x'))
  else if (value === 'call') emit('replace', makeCall(props.defaultCall ?? 'forward', props.functions))
  else emit('replace', defaultExpr(value as 'integer' | 'bool' | 'unit' | 'rotation'))
}
function changeFunction(event: Event): void {
  emit('replace', makeCall((event.target as HTMLSelectElement).value, props.functions))
}
function setLiteral(value: string): void {
  if (props.node.kind !== 'literal') return
  if (props.node.type === 'integer')
    emit('replace', { ...props.node, value: Number.isSafeInteger(Number(value)) ? Number(value) : 0 })
  if (props.node.type === 'bool') emit('replace', { ...props.node, value: value === 'true' })
  if (props.node.type === 'rotation') emit('replace', { ...props.node, value: value as 'left' | 'right' })
}
function setName(value: string): void {
  if (props.node.kind === 'ident') emit('replace', { ...props.node, name: value })
}
function replaceArg(index: number, expr: PicoExpr): void {
  if (props.node.kind === 'call')
    emit('replace', { ...props.node, args: props.node.args.map((arg, i) => (i === index ? expr : arg)) })
}
function argumentType(index: number): PicoType | undefined {
  const node = props.node
  if (node.kind !== 'call') return undefined
  return options.value.find((item) => item.name === node.func)?.args[index]
}
function dragOver(event: DragEvent): void {
  if (
    props.statement ||
    !event.dataTransfer?.types.some(
      (type) => type === PICO_VARIABLE_MIME || type === 'application/x-picolang-choice',
    )
  )
    return
  event.preventDefault()
  event.stopPropagation()
  dragActive.value = true
}
function dropExpression(event: DragEvent): void {
  if (props.statement) return
  event.preventDefault()
  event.stopPropagation()
  dragActive.value = false
  const expression = expressionFromDrop(
    event.dataTransfer,
    props.expectedType,
    props.functions,
    props.variables ?? [],
  )
  if (expression) {
    dropError.value = ''
    emit('replace', expression)
  } else dropError.value = 'この入力欄には置けません。値の種類を確認してください。'
}
</script>

<template>
  <span
    class="pico-expression"
    :class="[
      `pico-expression--${mode}`,
      { 'pico-expression--statement': statement, 'pico-expression--drop-target': dragActive },
    ]"
    @dragover="dragOver"
    @dragleave="dragActive = false"
    @drop="dropExpression"
  >
    <select
      v-if="!statement"
      class="pico-expression-kind"
      :value="mode"
      aria-label="式の種類"
      @change="changeMode"
    >
      <option value="integer">数</option>
      <option value="bool">真偽</option>
      <option value="unit">()</option>
      <option value="rotation">向き</option>
      <option value="ident" :disabled="!variables?.length">変数</option>
      <option value="call">関数</option>
    </select>
    <template v-if="node.kind === 'literal'">
      <input
        v-if="node.type === 'integer'"
        type="number"
        step="1"
        :value="node.value ?? 0"
        aria-label="整数"
        @change="setLiteral(($event.target as HTMLInputElement).value)"
      />
      <select
        v-else-if="node.type === 'bool'"
        :value="String(node.value)"
        aria-label="真偽"
        @change="setLiteral(($event.target as HTMLSelectElement).value)"
      >
        <option value="true">true</option>
        <option value="false">false</option>
      </select>
      <select
        v-else-if="node.type === 'rotation'"
        :value="node.value"
        aria-label="回転方向"
        @change="setLiteral(($event.target as HTMLSelectElement).value)"
      >
        <option value="left">左</option>
        <option value="right">右</option>
      </select>
      <span v-else>()</span>
    </template>
    <select
      v-else-if="node.kind === 'ident' && variables?.length"
      :value="identifierName"
      aria-label="参照する変数"
      @change="setName(($event.target as HTMLSelectElement).value)"
    >
      <option
        v-if="!variables.some((variable) => variable.name === identifierName)"
        :value="identifierName"
        disabled
      >
        {{ identifierName }}（ここでは使えません）
      </option>
      <option v-for="variable in variables" :key="variable.name" :value="variable.name">
        {{ variable.name }}
      </option>
    </select>
    <input
      v-else-if="node.kind === 'ident'"
      :value="node.name"
      aria-label="変数名"
      class="pico-name-input"
      @input="setName(($event.target as HTMLInputElement).value)"
    />
    <template v-else>
      <strong v-if="statement" class="pico-command-label">{{ callLabel }}</strong>
      <select v-else :value="node.func" :aria-label="`関数: ${callLabel}`" @change="changeFunction">
        <optgroup
          v-for="category in [...new Set(options.map((option) => option.category))]"
          :key="category"
          :label="category"
        >
          <option
            v-for="option in options.filter((item) => item.category === category)"
            :key="option.name"
            :value="option.name"
          >
            {{ option.label }}
          </option>
        </optgroup>
      </select>
      <span v-if="node.args.length" class="pico-arguments">
        <PicoExpression
          v-for="(arg, index) in node.args"
          :key="arg.hintId"
          :node="arg"
          :functions="functions"
          :variables="variables"
          :expected-type="argumentType(index)"
          @replace="replaceArg(index, $event)"
        />
      </span>
    </template>
    <small v-if="dropError" class="pico-expression-drop-error" role="status">{{ dropError }}</small>
  </span>
</template>
