<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { PicoDiagnostic } from '../language/analyzePicolang.ts'

const props = defineProps<{ diagnostics: PicoDiagnostic[] }>()
const emit = defineEmits<{ select: [diagnostic: PicoDiagnostic] }>()
const errors = computed(() => props.diagnostics.filter((item) => item.severity === 'error').length)
const warnings = computed(() => props.diagnostics.length - errors.value)
const compact = ref(false)
let viewport: MediaQueryList | undefined
function updateCompact(): void {
  compact.value = viewport?.matches ?? false
}
onMounted(() => {
  viewport = window.matchMedia('(max-width: 700px), (max-height: 600px)')
  updateCompact()
  viewport.addEventListener('change', updateCompact)
})
onBeforeUnmount(() => viewport?.removeEventListener('change', updateCompact))
</script>

<template>
  <details v-if="diagnostics.length" class="pico-diagnostics" :open="!compact">
    <summary aria-live="polite">プログラムの確認：エラー {{ errors }} 件・警告 {{ warnings }} 件</summary>
    <ul>
      <li v-for="(item, index) in diagnostics" :key="index" :class="`pico-diagnostic--${item.severity}`">
        <button type="button" @click="emit('select', item)">
          <strong>{{ item.severity === 'error' ? 'エラー' : '警告' }}</strong>
          <span>{{ item.functionName }}：{{ item.message }}</span>
        </button>
      </li>
    </ul>
  </details>
</template>

<style scoped>
.pico-diagnostics {
  flex: none;
  max-height: 28%;
  overflow: auto;
  padding: 6px 12px;
  border-top: 1px solid #cbdad3;
  background: #f8fbf9;
  color: #173a34;
  font-size: 12px;
}
summary {
  cursor: pointer;
  font-weight: 600;
}
ul {
  margin: 6px 0 0;
  padding: 0;
  list-style: none;
}
button {
  display: flex;
  gap: 8px;
  width: 100%;
  padding: 5px 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}
strong {
  flex: none;
  align-self: flex-start;
  padding: 1px 6px;
  border-radius: 3px;
}
.pico-diagnostic--error strong {
  background: var(--color-error);
  color: #fff;
}
.pico-diagnostic--warning strong {
  background: var(--color-warning);
  color: var(--color-warning-ink);
}
button:hover span {
  text-decoration: underline;
}
:is(button, summary):focus-visible {
  outline: 2px solid #126d88;
  outline-offset: 2px;
}
</style>
