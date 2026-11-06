<script setup lang="ts">
import { computed, ref } from 'vue'
import { Icon } from '@iconify/vue'
import arrowUpIcon from '@iconify-icons/lucide/arrow-up'
import arrowDownIcon from '@iconify-icons/lucide/arrow-down'
import trashIcon from '@iconify-icons/lucide/trash-2'
import plusIcon from '@iconify-icons/lucide/plus'
import PicoExpression from './PicoExpression.vue'
import PicoTypeInput from './PicoTypeInput.vue'
import {
  PICO_BUILTINS,
  block as makeBlock,
  makeStatement,
  type PicoBlock,
  type PicoFunction,
  type PicoStatement,
} from './picolang.ts'

defineOptions({ name: 'PicoBlock' })
const props = defineProps<{
  block: PicoBlock
  functions: Record<string, PicoFunction>
  terminalReturn?: boolean
}>()
const statements = computed(() => props.block.stmts)
const choice = ref('forward')
const controlChoices = [
  { value: 'if', label: 'もし' },
  { value: 'if-else', label: 'もし／でなければ' },
  { value: 'while', label: '〜の間くり返す' },
  { value: 'for', label: '数を変えてくり返す' },
  { value: 'declare', label: '変数を作る' },
  { value: 'assign', label: '変数へ代入' },
  { value: 'return', label: '返す' },
  { value: 'break', label: 'くり返しを抜ける' },
  { value: 'continue', label: '次のくり返しへ' },
]
function append(value = choice.value): void {
  const last = statements.value.at(-1)
  if (props.terminalReturn && last?.kind === 'return')
    statements.value.splice(statements.value.length - 1, 0, makeStatement(value, props.functions))
  else statements.value.push(makeStatement(value, props.functions))
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
      PICO_BUILTINS.some((item) => item.name === value) ||
      value in props.functions)
  )
    append(value)
}
function dropInto(target: PicoBlock, event: DragEvent): void {
  event.preventDefault()
  event.stopPropagation()
  const value = event.dataTransfer?.getData('application/x-picolang-choice')
  if (
    value &&
    (controlChoices.some((item) => item.value === value) ||
      PICO_BUILTINS.some((item) => item.name === value) ||
      value in props.functions)
  )
    target.stmts.push(makeStatement(value, props.functions))
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
    <p v-if="!statements.length" class="pico-drop-hint">ここへ命令を置く</p>
    <div
      v-for="(stmt, index) in statements"
      :key="stmt.hintId"
      class="pico-statement"
      :class="`pico-statement--${stmt.kind}`"
    >
      <div class="pico-statement-toolbar">
        <span class="pico-statement-order">{{ index + 1 }}</span>
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
            <span class="pico-control-field-label">この条件が成り立つとき</span>
            <PicoExpression
              :node="stmt.condition"
              :functions="functions"
              @replace="stmt.condition = $event"
            />
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="もしの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.then, $event)"
          >
            <span class="pico-scope-label">条件が真のとき</span>
            <PicoBlock :block="stmt.then" :functions="functions" />
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
              @drop="dropInto(stmt.else, $event)"
            >
              <span class="pico-scope-label">条件が偽のとき</span>
              <PicoBlock :block="stmt.else" :functions="functions" />
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
            <PicoExpression
              :node="stmt.condition"
              :functions="functions"
              @replace="stmt.condition = $event"
            />
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="whileの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.body, $event)"
          >
            <span class="pico-scope-label">くり返す処理</span>
            <PicoBlock :block="stmt.body" :functions="functions" />
          </div>
          <div class="pico-control-tail" aria-hidden="true">}</div>
        </div>
        <div v-else-if="stmt.kind === 'for'" class="pico-control-shell">
          <div class="pico-control-head">
            <strong>数を変えてくり返す</strong>
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
              />
            </div>
            <div class="pico-control-field" role="group" aria-label="最初の値">
              <span class="pico-control-field-label">最初の値</span>
              <PicoExpression :node="stmt.start" :functions="functions" @replace="stmt.start = $event" />
            </div>
            <div class="pico-control-field" role="group" aria-label="終わりの値。この値は含まない">
              <span class="pico-control-field-label">終わりの値 <small>含まない</small></span>
              <PicoExpression :node="stmt.end" :functions="functions" @replace="stmt.end = $event" />
            </div>
            <p class="pico-control-help">変数を最初の値から1ずつ増やし、終わりの値の手前で止めます。</p>
          </div>
          <div
            class="pico-control-scope"
            role="group"
            aria-label="forの中の処理"
            @dragover.stop.prevent
            @drop="dropInto(stmt.body, $event)"
          >
            <span class="pico-scope-label">くり返す処理</span>
            <PicoBlock :block="stmt.body" :functions="functions" />
          </div>
          <div class="pico-control-tail" aria-hidden="true">}</div>
        </div>
        <PicoExpression
          v-else-if="stmt.kind === 'call' || stmt.kind === 'literal' || stmt.kind === 'ident'"
          :node="stmt"
          :functions="functions"
          :statement="stmt.kind === 'call'"
          @replace="replace(index, $event)"
        />
        <template v-else-if="stmt.kind === 'declare'">
          <span>変数を作る</span>
          <PicoTypeInput v-model="stmt.type" />
          <input v-model="stmt.ident.name" class="pico-name-input" aria-label="変数名" />
          <span>＝</span
          ><PicoExpression :node="stmt.value" :functions="functions" @replace="stmt.value = $event" />
        </template>
        <template v-else-if="stmt.kind === 'assign'">
          <span>変数へ代入</span
          ><input v-model="stmt.ident.name" class="pico-name-input" aria-label="変数名" /> <span>＝</span
          ><PicoExpression :node="stmt.value" :functions="functions" @replace="stmt.value = $event" />
        </template>
        <template v-else-if="stmt.kind === 'return'">
          <span>返す</span
          ><PicoExpression :node="stmt.value" :functions="functions" @replace="stmt.value = $event" />
        </template>
        <span v-else>{{ stmt.kind === 'break' ? 'くり返しを抜ける' : '次のくり返しへ' }}</span>
      </div>
    </div>
    <div class="pico-add-row">
      <select v-model="choice" aria-label="追加する命令">
        <optgroup label="動きと操作">
          <option
            v-for="item in PICO_BUILTINS.filter((item) => ['動き', '操作'].includes(item.category))"
            :key="item.name"
            :value="item.name"
          >
            {{ item.label }}
          </option>
        </optgroup>
        <optgroup label="制御と変数">
          <option v-for="item in controlChoices" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </optgroup>
        <optgroup label="組み込み関数">
          <option
            v-for="item in PICO_BUILTINS.filter((item) => !['動き', '操作'].includes(item.category))"
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
