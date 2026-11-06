<script setup lang="ts">
import { computed } from 'vue'
import type { PicoPrimitive, PicoType } from './picolang.ts'

const props = defineProps<{ modelValue: PicoType }>()
const emit = defineEmits<{ 'update:modelValue': [value: PicoType] }>()
const kind = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : 'vector'))
const primitives: PicoPrimitive[] = ['bool', 'integer', 'unit', 'terrain', 'rotation']
function changeKind(event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  emit(
    'update:modelValue',
    value === 'vector' ? { kind: 'vector', elementType: 'integer' } : (value as PicoPrimitive),
  )
}
</script>

<template>
  <span class="pico-type-input">
    <select :value="kind" aria-label="型" @change="changeKind">
      <option v-for="type in primitives" :key="type" :value="type">{{ type }}</option>
      <option value="vector">vector</option>
    </select>
    <template v-if="typeof modelValue !== 'string'">
      <span aria-hidden="true">&lt;</span>
      <PicoTypeInput
        :model-value="modelValue.elementType"
        @update:model-value="emit('update:modelValue', { kind: 'vector', elementType: $event })"
      />
      <span aria-hidden="true">&gt;</span>
    </template>
  </span>
</template>
