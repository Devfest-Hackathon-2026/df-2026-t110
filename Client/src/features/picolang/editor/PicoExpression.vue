<script setup lang="ts">
import { computed, ref } from 'vue'
import { Icon } from '@iconify/vue'
import infoIcon from '@iconify-icons/lucide/info'
import { expressionFromDrop, PICO_VARIABLE_MIME } from './picoDrag.ts'
import { PICO_BUILTIN_OPTIONS, type PicoVariableOption } from './picoOptions.ts'
import { argumentLabel, resultGuidance, typeLabel } from './picoGuidance.ts'
import {
  defaultExpr,
  ident,
  makeCall,
  samePicoType,
  PICO_DISTANCE_TARGETS,
  type PicoExpr,
  type PicoFunction,
  type PicoType,
} from '../language/picolang.ts'

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
const showInfo = ref(false)
const infoTrigger = ref<HTMLButtonElement | null>(null)
const infoFlip = ref(false)
const infoWidth = ref(260)
function positionInfo(): void {
  const trigger = infoTrigger.value
  const workspace = trigger?.closest('.pico-workspace')
  if (!trigger || !workspace) return
  const bounds = workspace.getBoundingClientRect()
  infoWidth.value = Math.min(260, bounds.width - 16, window.innerWidth * 0.7)
  infoFlip.value = trigger.getBoundingClientRect().left + infoWidth.value > bounds.right - 8
}
function toggleInfo(): void {
  positionInfo()
  showInfo.value = !showInfo.value
}
const mode = computed(() => (props.node.kind === 'literal' ? props.node.type : props.node.kind))
const identifierName = computed(() => (props.node.kind === 'ident' ? props.node.name : ''))
const options = computed(() => [
  ...PICO_BUILTIN_OPTIONS,
  ...Object.entries(props.functions)
    .filter(([name]) => name !== 'main')
    .map(([name, definition]) => ({
      name,
      label: name,
      category: '関数',
      args: definition.params.map((param) => param.type),
      returns: definition.returnType,
    })),
])
const availableVariables = computed(() =>
  (props.variables ?? []).filter((variable) => samePicoType(props.expectedType, variable.type)),
)
const availableCalls = computed(() =>
  options.value.filter(
    (option) =>
      samePicoType(props.expectedType, option.returns) ||
      (option.name === 'new' && typeof props.expectedType === 'object'),
  ),
)
const literalTypes = computed(() =>
  (['integer', 'bool', 'unit', 'rotation', 'distance_target'] as const).filter((type) =>
    samePicoType(props.expectedType, type),
  ),
)
const invalidType = computed(() => {
  if (!props.expectedType) return false
  const node = props.node
  if (node.kind === 'literal') return !samePicoType(props.expectedType, node.type)
  if (node.kind === 'ident') return !availableVariables.value.some((item) => item.name === node.name)
  return !availableCalls.value.some((item) => item.name === node.func)
})
const callLabel = computed(() => {
  const node = props.node
  return node.kind === 'call'
    ? (options.value.find((option) => option.name === node.func)?.label ?? node.func)
    : ''
})
const standaloneNote = computed(() => {
  const node = props.node
  if (!props.statement || node.kind !== 'call') return ''
  if (resultGuidance[node.func]) return `${resultGuidance[node.func]} この行だけでは返した値を使いません。`
  const category = PICO_BUILTIN_OPTIONS.find((item) => item.name === node.func)?.category
  return category && !['動き', '操作', '表示'].includes(category)
    ? 'この行だけでは返した値を使いません。変数や条件の入力欄に入れてください。'
    : ''
})
function changeMode(event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  if (value === 'ident' && availableVariables.value[0])
    emit('replace', ident(availableVariables.value[0].name))
  else if (value === 'call' && availableCalls.value.length) {
    const name = availableCalls.value.some((option) => option.name === props.defaultCall)
      ? props.defaultCall!
      : availableCalls.value[0]!.name
    emit('replace', makeCall(name, props.functions))
  } else if (literalTypes.value.some((type) => type === value))
    emit('replace', defaultExpr(value as 'integer' | 'bool' | 'unit' | 'rotation' | 'distance_target'))
}
function changeFunction(event: Event): void {
  const name = (event.target as HTMLSelectElement).value
  if (availableCalls.value.some((option) => option.name === name))
    emit('replace', makeCall(name, props.functions))
}
function setLiteral(value: string): void {
  if (props.node.kind !== 'literal') return
  if (props.node.type === 'integer')
    emit('replace', { ...props.node, value: Number.isSafeInteger(Number(value)) ? Number(value) : 0 })
  if (props.node.type === 'bool') emit('replace', { ...props.node, value: value === 'true' })
  if (props.node.type === 'rotation') emit('replace', { ...props.node, value: value as 'left' | 'right' })
  if (props.node.type === 'distance_target')
    emit('replace', { ...props.node, value: value as (typeof PICO_DISTANCE_TARGETS)[number] })
}
function setName(value: string): void {
  if (props.node.kind === 'ident' && availableVariables.value.some((item) => item.name === value))
    emit('replace', { ...props.node, name: value })
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
      aria-label="値の作り方"
      @change="changeMode"
    >
      <option v-if="literalTypes.includes('integer')" value="integer">数</option>
      <option v-if="literalTypes.includes('bool')" value="bool">はい／いいえ</option>
      <option v-if="literalTypes.includes('unit')" value="unit">値なし</option>
      <option v-if="literalTypes.includes('rotation')" value="rotation">向き</option>
      <option v-if="literalTypes.includes('distance_target')" value="distance_target">探索対象</option>
      <option value="ident" :disabled="!availableVariables.length">変数</option>
      <option value="call" :disabled="!availableCalls.length">命令の答え</option>
    </select>
    <template v-if="node.kind === 'literal'">
      <input
        v-if="node.type === 'integer'"
        type="number"
        step="1"
        :value="node.value ?? 0"
        aria-label="整数"
        data-1p-ignore
        @change="setLiteral(($event.target as HTMLInputElement).value)"
      />
      <select
        v-else-if="node.type === 'bool'"
        :value="String(node.value)"
        aria-label="真偽"
        @change="setLiteral(($event.target as HTMLSelectElement).value)"
      >
        <option value="true">はい</option>
        <option value="false">いいえ</option>
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
      <select
        v-else-if="node.type === 'distance_target'"
        :value="node.value"
        aria-label="探索対象"
        @change="setLiteral(($event.target as HTMLSelectElement).value)"
      >
        <option v-for="target in PICO_DISTANCE_TARGETS" :key="target" :value="target">{{ target }}</option>
      </select>
      <span v-else>値なし</span>
    </template>
    <select
      v-else-if="node.kind === 'ident' && availableVariables.length"
      :value="identifierName"
      aria-label="参照する変数"
      @change="setName(($event.target as HTMLSelectElement).value)"
    >
      <option
        v-if="!availableVariables.some((variable) => variable.name === identifierName)"
        :value="identifierName"
        disabled
      >
        {{ identifierName }}（ここでは使えません）
      </option>
      <option v-for="variable in availableVariables" :key="variable.name" :value="variable.name">
        {{ variable.name }}
      </option>
    </select>
    <span v-else-if="node.kind === 'ident'" class="pico-expression-invalid"
      >{{ node.name }}（この位置では使えません）</span
    >
    <template v-else>
      <strong v-if="statement" class="pico-command-label">{{ callLabel }}</strong>
      <select v-else :value="node.func" :aria-label="`答えを使う命令: ${callLabel}`" @change="changeFunction">
        <optgroup
          v-for="category in [...new Set(availableCalls.map((option) => option.category))]"
          :key="category"
          :label="category"
        >
          <option
            v-for="option in availableCalls.filter((item) => item.category === category)"
            :key="option.name"
            :value="option.name"
          >
            {{ option.label }}
          </option>
        </optgroup>
      </select>
      <span v-if="node.args.length" class="pico-arguments">
        <span v-for="(arg, index) in node.args" :key="arg.hintId" class="pico-argument">
          <small
            >{{ argumentLabel(node.func, index, functions)
            }}<template v-if="argumentType(index)">（{{ typeLabel(argumentType(index)!) }}）</template></small
          >
          <PicoExpression
            :node="arg"
            :functions="functions"
            :variables="variables"
            :expected-type="argumentType(index)"
            @replace="replaceArg(index, $event)"
          />
        </span>
      </span>
      <span
        v-if="standaloneNote"
        class="pico-info"
        :class="{ 'pico-info--open': showInfo, 'pico-info--flip': infoFlip }"
        :style="{ '--pico-info-width': `${infoWidth}px` }"
        @mouseenter="positionInfo"
      >
        <button
          ref="infoTrigger"
          type="button"
          class="pico-info-trigger"
          :aria-label="`${callLabel}の説明`"
          :aria-expanded="showInfo"
          :aria-controls="`pico-info-${node.hintId}`"
          @click="toggleInfo"
          @blur="showInfo = false"
          @keydown.esc="showInfo = false"
        >
          <Icon :icon="infoIcon" aria-hidden="true" />
        </button>
        <small :id="`pico-info-${node.hintId}`" class="pico-info-popover" role="tooltip">
          {{ standaloneNote }}
        </small>
      </span>
    </template>
    <small v-if="dropError" class="pico-expression-drop-error" role="status">{{ dropError }}</small>
    <small v-if="invalidType" class="pico-expression-drop-error" role="status">
      ここには{{ typeLabel(expectedType!) }}が必要です。
      <button type="button" @click="emit('replace', defaultExpr(expectedType!))">直す</button>
    </small>
  </span>
</template>
