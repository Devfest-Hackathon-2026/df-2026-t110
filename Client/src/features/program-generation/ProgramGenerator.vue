<script setup lang="ts">
import { onBeforeUnmount, ref, useId } from 'vue'
import type { PicoTypedProgram } from '../picolang/language/picolang.ts'
import { generateProgram } from './programGenerationApi.ts'

const props = withDefaults(defineProps<{ fieldId?: string; isBattle?: boolean }>(), {
  fieldId: undefined,
  isBattle: false,
})
const emit = defineEmits<{ started: []; generated: [program: PicoTypedProgram] }>()
const prompt = ref('')
const promptId = useId()
const pending = ref(false)
const error = ref('')
const generated = ref('')
let controller: AbortController | null = null

async function submit() {
  const instruction = prompt.value.trim()
  if (!instruction || pending.value) return
  controller = new AbortController()
  const current = controller
  pending.value = true
  error.value = ''
  generated.value = ''
  emit('started')
  try {
    const program = await generateProgram(
      {
        prompt: instruction,
        ...(props.fieldId ? { fieldId: props.fieldId } : {}),
        isBattle: props.isBattle,
      },
      current.signal,
    )
    generated.value = JSON.stringify(program, null, 2)
    emit('generated', program)
  } catch (cause) {
    if (!current.signal.aborted) error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    if (controller === current) pending.value = false
  }
}

onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <form class="program-generator" @submit.prevent="submit">
    <label :for="promptId">作りたい動きを文章で書く</label>
    <textarea :id="promptId" v-model="prompt" required rows="3" maxlength="12000" />
    <button type="submit" :disabled="pending || !prompt.trim()">
      {{ pending ? '生成中…' : 'プログラムを生成' }}
    </button>
    <p v-if="error" role="alert">{{ error }}</p>
    <details v-if="generated">
      <summary>生成されたPicolang JSON</summary>
      <pre>{{ generated }}</pre>
    </details>
  </form>
</template>

<style scoped>
.program-generator {
  display: grid;
  gap: 8px;
}
.program-generator textarea {
  width: 100%;
  padding: 8px;
  font: inherit;
}
.program-generator button {
  justify-self: start;
  padding: 8px 14px;
  cursor: pointer;
}
.program-generator button:disabled {
  cursor: default;
  opacity: 0.55;
}
.program-generator [role='alert'] {
  color: #a74432;
}
.program-generator pre {
  overflow: auto;
  max-height: 320px;
  padding: 12px;
  background: #fff;
  font-size: 0.8rem;
}
</style>
