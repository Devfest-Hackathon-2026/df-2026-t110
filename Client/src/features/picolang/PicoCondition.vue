<script setup lang="ts">
import { computed, ref } from 'vue'
import PicoExpression from './PicoExpression.vue'
import { ident, makeCall, type PicoExpr, type PicoFunction, type PicoVariableOption } from './picolang.ts'

const props = defineProps<{
  node: PicoExpr
  functions: Record<string, PicoFunction>
  variables: PicoVariableOption[]
}>()
const emit = defineEmits<{ replace: [node: PicoExpr] }>()
const rawMode = ref(false)
const comparisons = [
  { value: 'integer_eq', label: '＝' },
  { value: 'integer_neq', label: '≠' },
  { value: 'gt', label: '＞' },
  { value: 'lt', label: '＜' },
  { value: 'geq', label: '≧' },
  { value: 'leq', label: '≦' },
]
const mode = computed(() => {
  if (rawMode.value) return 'custom'
  const node = props.node
  if (node.kind === 'call') {
    const front = node.args[0]
    if (
      ['is_rock', 'is_sea', 'is_goal'].includes(node.func) &&
      front?.kind === 'call' &&
      front.func === 'search_front'
    )
      return `front:${node.func}`
    const variable = node.args[0]
    const number = node.args[1]
    if (
      comparisons.some((item) => item.value === node.func) &&
      variable?.kind === 'ident' &&
      number?.kind === 'literal' &&
      number.type === 'integer' &&
      props.variables.some((item) => item.name === variable.name && item.type === 'integer')
    )
      return `compare:${variable.name}`
  }
  if (
    node.kind === 'ident' &&
    props.variables.some((item) => item.name === node.name && item.type === 'bool')
  )
    return `bool:${node.name}`
  return 'custom'
})
const comparison = computed(() => {
  const node = props.node
  return mode.value.startsWith('compare:') && node.kind === 'call' ? node : null
})
const comparisonName = computed(() =>
  comparison.value?.args[0]?.kind === 'ident' ? comparison.value.args[0].name : '',
)
const comparisonNumber = computed(() => {
  const number = comparison.value?.args[1]
  return number?.kind === 'literal' && number.type === 'integer' ? Number(number.value) : 0
})

function changeMode(event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  if (value === 'custom') {
    rawMode.value = true
    return
  }
  rawMode.value = false
  const [kind, name] = value.split(':')
  if (kind === 'front') emit('replace', makeCall(name, props.functions))
  if (kind === 'bool') emit('replace', ident(name))
  if (kind === 'compare') {
    const node = makeCall('integer_eq', props.functions)
    node.args[0] = ident(name)
    emit('replace', node)
  }
}
function changeComparison(event: Event): void {
  if (comparison.value)
    emit('replace', { ...comparison.value, func: (event.target as HTMLSelectElement).value })
}
function changeNumber(event: Event): void {
  if (!comparison.value) return
  const raw = (event.target as HTMLInputElement).value
  if (!raw.trim()) return
  const value = Number(raw)
  if (!Number.isSafeInteger(value)) return
  const number = comparison.value.args[1]
  if (number?.kind !== 'literal') return
  emit('replace', {
    ...comparison.value,
    args: [comparison.value.args[0], { ...number, value }],
  })
}
function restoreNumber(event: FocusEvent): void {
  const input = event.target as HTMLInputElement
  if (!input.value.trim()) input.value = String(comparisonNumber.value)
}
</script>

<template>
  <div class="pico-condition-editor">
    <select :value="mode" aria-label="実行する条件" @change="changeMode">
      <option value="custom">式を自分で作る</option>
      <option value="front:is_rock">前に岩がある</option>
      <option value="front:is_sea">前に海がある</option>
      <option value="front:is_goal">前がゴール</option>
      <template v-for="variable in variables" :key="variable.name">
        <option v-if="variable.type === 'integer'" :value="`compare:${variable.name}`">
          {{ variable.name }} を数と比べる
        </option>
        <option v-else-if="variable.type === 'bool'" :value="`bool:${variable.name}`">
          {{ variable.name }} が true
        </option>
      </template>
    </select>
    <div v-if="comparison" class="pico-condition-comparison">
      <strong>{{ comparisonName }}</strong>
      <select :value="comparison.func" aria-label="比較方法" @change="changeComparison">
        <option v-for="item in comparisons" :key="item.value" :value="item.value">{{ item.label }}</option>
      </select>
      <input
        type="number"
        step="1"
        :value="comparisonNumber"
        aria-label="比べる数"
        @input="changeNumber"
        @blur="restoreNumber"
      />
    </div>
    <PicoExpression
      v-else-if="mode === 'custom'"
      :node="node"
      :functions="functions"
      :variables="variables"
      expected-type="bool"
      default-call="is_rock"
      @replace="emit('replace', $event)"
    />
  </div>
</template>
