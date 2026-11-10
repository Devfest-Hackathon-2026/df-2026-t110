<script setup lang="ts">
import { onBeforeUnmount, ref, useId } from 'vue'
import type { PicoTypedProgram } from '../picolang/language/picolang.ts'
import { generateProgram, ProgramGenerationApiError } from './programGenerationApi.ts'

const props = withDefaults(
  defineProps<{ fieldId?: string; isBattle?: boolean; generate?: typeof generateProgram }>(),
  {
    fieldId: undefined,
    isBattle: false,
  },
)
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
    const program = await (props.generate ?? generateProgram)(
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
    if (!current.signal.aborted) {
      console.error('プログラム生成に失敗しました', cause)
      error.value =
        cause instanceof ProgramGenerationApiError
          ? cause.message
          : 'プログラムを生成できませんでした。接続を確認して、もう一度お試しください。'
    }
  } finally {
    if (controller === current) pending.value = false
  }
}

onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <form class="program-generator" :aria-busy="pending" @submit.prevent="submit">
    <label :for="promptId">作りたい動きを文章で書く</label>
    <textarea :id="promptId" v-model="prompt" required rows="3" maxlength="12000" />
    <div class="program-generator-action">
      <div
        v-if="pending"
        class="program-generator-progress"
        role="progressbar"
        aria-label="プログラムを生成中"
      >
        <span aria-hidden="true" />
      </div>
      <button v-else type="submit" :disabled="!prompt.trim()">プログラムを生成</button>
    </div>
    <p v-if="error" role="alert">{{ error }}</p>
    <details v-if="generated">
      <summary>生成したプログラムのJSON</summary>
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
  padding: 8px 14px;
  cursor: pointer;
}
.program-generator-action {
  display: flex;
  justify-content: flex-end;
  min-height: 40px;
}
.program-generator-progress {
  position: relative;
  align-self: center;
  width: min(100%, 260px);
  height: 8px;
  overflow: hidden;
  border-radius: 4px;
  background: var(--color-edge);
}
.program-generator-progress span {
  position: absolute;
  inset-block: 0;
  left: 0;
  width: 38%;
  border-radius: inherit;
  background: var(--color-accent-deep);
  animation: program-generator-loading 1.5s ease-in-out infinite alternate;
}
@keyframes program-generator-loading {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(165%);
  }
}
@media (prefers-reduced-motion: reduce) {
  .program-generator-progress span {
    width: 100%;
    animation: none;
  }
}
.program-generator button:disabled {
  cursor: default;
  opacity: 0.55;
}
.program-generator [role='alert'] {
  color: var(--color-error);
}
.program-generator pre {
  overflow: auto;
  max-height: 320px;
  padding: 12px;
  background: #fff;
  font-size: 0.8rem;
}
</style>
