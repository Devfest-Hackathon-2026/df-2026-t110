<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { traceNodeLabel } from './playback/tracePresentation.ts'
import type { PicolangTrace, TraceNode } from '../language/picolangTrace.ts'

defineProps<{
  trace: PicolangTrace | null
  running: boolean
  ready: boolean
  paused: boolean
  interval: number
}>()
const emit = defineEmits<{
  pause: []
  step: []
  interval: [value: number]
  locate: [node: TraceNode]
}>()
const expanded = ref(true)
let viewport: MediaQueryList | undefined
function updateCompact(): void {
  expanded.value = !viewport?.matches
}
onMounted(() => {
  viewport = window.matchMedia('(max-width: 700px), (max-height: 600px)')
  updateCompact()
  viewport.addEventListener('change', updateCompact)
})
onBeforeUnmount(() => viewport?.removeEventListener('change', updateCompact))
const endLabels: Record<string, string> = {
  completed: 'プログラム終了',
  interrupted: '課題の終了により停止',
  runtime_error: '実行時エラー',
  division_by_zero: '0による除算',
  step_limit: '実行ステップ数の上限',
  memory_limit: 'メモリの上限',
  call_depth_limit: '関数呼び出しの深さの上限',
}
</script>

<template>
  <section class="pico-trace" aria-label="実行の追跡">
    <div class="pico-trace-controls">
      <strong>実行の追跡</strong>
      <span role="status">{{
        running ? (ready ? (paused ? '一時停止中' : '再生中') : '結果を待っています') : '再生終了'
      }}</span>
      <button type="button" :disabled="!running || !ready" @click="emit('pause')">
        {{ paused ? '再開' : '一時停止' }}
      </button>
      <button type="button" :disabled="!running || !ready" @click="emit('step')">1ステップ</button>
      <label
        >ノード間隔
        <select
          :value="interval"
          @change="emit('interval', Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="100">0.1秒</option>
          <option :value="300">0.3秒</option>
          <option :value="600">0.6秒</option>
        </select>
      </label>
    </div>
    <p v-if="trace?.current" class="pico-trace-current">
      現在：<button type="button" @click="emit('locate', trace.current)">
        {{ trace.current.functionName }}：{{ traceNodeLabel(trace.current) }}
      </button>
    </p>
    <details :open="expanded" @toggle="expanded = ($event.target as HTMLDetailsElement).open">
      <summary>変数 {{ trace?.variables.length ?? 0 }} 件・直近のノードの遷移</summary>
      <ol v-if="trace?.trail.length" class="pico-trace-trail" aria-label="直近のノードの遷移">
        <li v-for="(node, index) in trace.trail" :key="index">
          <button type="button" @click="emit('locate', node)">
            {{ node.functionName }}：{{ traceNodeLabel(node) }}
          </button>
        </li>
      </ol>
      <div class="pico-trace-variables">
        <table v-if="trace?.variables.length">
          <caption>
            有効な変数の値
          </caption>
          <thead>
            <tr>
              <th>関数</th>
              <th>変数</th>
              <th>値</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="variable in trace.variables"
              :key="`${variable.hintId}:${variable.depth}`"
              :class="{ 'pico-trace-changed': trace.changedHintId === variable.hintId }"
            >
              <td>{{ variable.functionName }}</td>
              <th scope="row">
                {{ variable.name }}<small v-if="variable.depth > 1">（再帰 {{ variable.depth }}）</small>
              </th>
              <td>
                <code>{{ variable.value }}</code>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else>この時点で有効な変数はありません。</p>
      </div>
    </details>
    <p
      v-if="trace?.finished"
      class="pico-trace-ended"
      :class="{ 'pico-trace-error': !['completed', 'interrupted'].includes(trace.finished.reason) }"
    >
      {{ endLabels[trace.finished.reason]
      }}<span v-if="!['completed', 'interrupted'].includes(trace.finished.reason)"
        >：{{ trace.finished.message }}</span
      >
    </p>
    <p v-else-if="!running && !trace" class="pico-trace-ended">この実行結果には追跡情報がありません。</p>
  </section>
</template>

<style scoped>
.pico-trace {
  flex: none;
  max-height: 38%;
  overflow: auto;
  padding: 8px 12px;
  border-top: 1px solid #b6cbc2;
  background: #f8fbf9;
}
.pico-trace-controls {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.pico-trace-controls > span {
  margin-right: auto;
  font-size: 11px;
}
.pico-trace button {
  border: 1px solid #b6cbc2;
  border-radius: 4px;
  padding: 3px 6px;
  background: white;
  color: #176b62;
}
.pico-trace label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
}
.pico-trace-current {
  margin: 6px 0;
  font-weight: 600;
}
.pico-trace-trail {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 0;
  margin: 5px 0;
  list-style: none;
  font-size: 10px;
}
.pico-trace-trail li:not(:last-child)::after {
  content: ' →';
}
.pico-trace-trail button {
  border: 0;
  padding: 1px 3px;
  background: #e6f0ea;
}
.pico-trace-variables {
  max-height: 150px;
  overflow: auto;
}
.pico-trace-variables p {
  margin: 8px 0;
  font-size: 12px;
}
.pico-trace table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 12px;
}
.pico-trace caption {
  text-align: left;
  font-weight: 600;
  margin: 4px 0;
}
.pico-trace th,
.pico-trace td {
  border-bottom: 1px solid #d9e4de;
  padding: 4px 6px;
  vertical-align: top;
}
.pico-trace code {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
.pico-trace-changed {
  background: #fff1bf;
}
.pico-trace-ended {
  margin: 6px 0 0;
  font-weight: 600;
}
.pico-trace-error {
  color: #9c3525;
}
.pico-trace summary {
  cursor: pointer;
  font-size: 12px;
  margin: 4px 0;
}
@media (max-width: 700px), (max-height: 600px) {
  .pico-trace {
    padding: 5px 8px;
    max-height: 45%;
  }
  .pico-trace-controls {
    gap: 4px;
  }
  .pico-trace-current {
    margin: 3px 0;
  }
}
</style>
