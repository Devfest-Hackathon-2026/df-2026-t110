<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import moveIcon from '@iconify-icons/lucide/move'
import mousePointerIcon from '@iconify-icons/lucide/mouse-pointer-2'
import gitBranchIcon from '@iconify-icons/lucide/git-branch'
import variableIcon from '@iconify-icons/lucide/variable'
import searchIcon from '@iconify-icons/lucide/search'
import bracesIcon from '@iconify-icons/lucide/braces'
import calculatorIcon from '@iconify-icons/lucide/calculator'
import repeatIcon from '@iconify-icons/lucide/repeat-2'
import shuffleIcon from '@iconify-icons/lucide/shuffle'
import listIcon from '@iconify-icons/lucide/list'
import typeIcon from '@iconify-icons/lucide/type'
import squareFunctionIcon from '@iconify-icons/lucide/square-function'
import trashIcon from '@iconify-icons/lucide/trash-2'
import plusIcon from '@iconify-icons/lucide/plus'
import copyIcon from '@iconify-icons/lucide/copy'
import fileJsonIcon from '@iconify-icons/lucide/file-json'
import uploadIcon from '@iconify-icons/lucide/upload'
import xIcon from '@iconify-icons/lucide/x'
import PicoBlock from './PicoBlock.vue'
import PicoTypeInput from './PicoTypeInput.vue'
import { beginPicoDrag, dragVariable, endPicoDrag } from './picoDrag.ts'
import {
  PICO_BUILTINS,
  block,
  defaultExpr,
  id,
  ident,
  isPicoProgram,
  makeStatement,
  newProgram,
  programFunctions,
  type PicoProgram,
  type PicoType,
} from './picolang.ts'

const props = defineProps<{ tutorialId: string }>()
const emit = defineEmits<{ 'update:program': [program: PicoProgram] }>()
const program = ref<PicoProgram>(newProgram())
const activeFunctionName = ref('main')
const category = ref('動き')
const newFunctionName = ref('')
const showNewFunction = ref(false)
const newFunctionInput = ref<HTMLInputElement | null>(null)
const message = ref('')
const showJson = ref(false)
const jsonInput = ref('')
const functions = computed(() => programFunctions(program.value))
const currentFunction = computed(() => functions.value[activeFunctionName.value] ?? functions.value.main)
const categories = [
  '動き',
  '操作',
  '制御',
  '変数',
  '調べる',
  '論理',
  '計算',
  '変換',
  '乱数',
  '配列',
  '表示',
  '関数',
] as const
const categoryIcons = {
  動き: moveIcon,
  操作: mousePointerIcon,
  制御: gitBranchIcon,
  変数: variableIcon,
  調べる: searchIcon,
  論理: bracesIcon,
  計算: calculatorIcon,
  変換: repeatIcon,
  乱数: shuffleIcon,
  配列: listIcon,
  表示: typeIcon,
  関数: squareFunctionIcon,
}
const controls = [
  { name: 'if', label: 'もし' },
  { name: 'if-else', label: 'もし／でなければ' },
  { name: 'while', label: '〜の間くり返す' },
  { name: 'for', label: '数を変えてくり返す' },
  { name: 'declare', label: '変数を作る' },
  { name: 'assign', label: '代入する' },
  { name: 'return', label: '返す' },
  { name: 'break', label: '抜ける' },
  { name: 'continue', label: '次へ' },
]
const palette = computed(() =>
  category.value === '制御' || category.value === '変数'
    ? controls.filter((item) =>
        category.value === '制御'
          ? !['declare', 'assign'].includes(item.name)
          : ['declare', 'assign'].includes(item.name),
      )
    : category.value === '関数'
      ? Object.keys(functions.value)
          .filter((name) => name !== 'main')
          .map((name) => ({ name, label: name }))
      : PICO_BUILTINS.filter((item) => item.category === category.value),
)

watch(
  () => props.tutorialId,
  (tutorialId) => {
    try {
      const saved = localStorage.getItem(`picolang:draft:${tutorialId}`)
      const parsed: unknown = saved ? JSON.parse(saved) : null
      program.value = isPicoProgram(parsed) ? parsed : newProgram()
    } catch {
      program.value = newProgram()
    }
    activeFunctionName.value = 'main'
  },
  { immediate: true },
)

watch(
  program,
  (value) => {
    emit('update:program', value)
    try {
      localStorage.setItem(`picolang:draft:${props.tutorialId}`, JSON.stringify(value))
    } catch {
      /* Storage may be disabled. */
    }
  },
  { deep: true, immediate: true },
)

function addChoice(name: string): void {
  const stmts = currentFunction.value.body.stmts
  const index = stmts.at(-1)?.kind === 'return' ? stmts.length - 1 : stmts.length
  stmts.splice(index, 0, makeStatement(name, functions.value))
  message.value = ''
}
function dragChoice(event: DragEvent, name: string): void {
  beginPicoDrag()
  event.dataTransfer?.setData('application/x-picolang-choice', name)
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copy'
}
function addFunction(): void {
  const name = newFunctionName.value.trim()
  if (!name || name in functions.value || PICO_BUILTINS.some((item) => item.name === name)) {
    message.value = '別の関数名を指定してください。'
    return
  }
  functions.value[name] = {
    kind: 'function',
    hintId: id(),
    params: [],
    returnType: 'unit',
    body: block([makeStatement('return', functions.value)]),
  }
  activeFunctionName.value = name
  closeNewFunction()
  message.value = `${name} を追加しました。`
}
function openNewFunction(): void {
  showNewFunction.value = true
  nextTick(() => newFunctionInput.value?.focus())
}
function closeNewFunction(): void {
  showNewFunction.value = false
  newFunctionName.value = ''
}
function removeFunction(): void {
  if (activeFunctionName.value === 'main') return
  delete functions.value[activeFunctionName.value]
  activeFunctionName.value = 'main'
}
function addParam(): void {
  currentFunction.value.params.push({
    kind: 'param',
    hintId: id(),
    ident: ident(`arg${currentFunction.value.params.length + 1}`),
    type: 'integer',
  })
}
function setReturnType(type: PicoType): void {
  currentFunction.value.returnType = type
  const last = currentFunction.value.body.stmts.at(-1)
  if (last?.kind === 'return') last.value = defaultExpr(type)
}
async function copyJson(): Promise<void> {
  try {
    await navigator.clipboard.writeText(JSON.stringify(program.value, null, 2))
    message.value = 'JSONをコピーしました。'
  } catch {
    message.value = 'コピーできませんでした。JSON欄から選択してください。'
  }
}
function importJson(): void {
  try {
    const parsed: unknown = JSON.parse(jsonInput.value)
    if (!isPicoProgram(parsed)) throw new Error('Picolang v0 の main 関数が必要です。')
    program.value = parsed
    activeFunctionName.value = 'main'
    message.value = 'JSONを読み込みました。'
  } catch (error) {
    message.value = error instanceof Error ? error.message : 'JSONを読み込めませんでした。'
  }
}
</script>

<template>
  <div class="pico-editor">
    <nav class="pico-categories" aria-label="命令の種類">
      <button
        v-for="name in categories"
        :key="name"
        type="button"
        :aria-pressed="category === name"
        @click="category = name"
      >
        <Icon :icon="categoryIcons[name]" aria-hidden="true" />
        <span>{{ name }}</span>
      </button>
    </nav>
    <aside class="pico-palette" aria-label="命令パレット">
      <h3>{{ category }}</h3>
      <p>タップで命令を追加。ドラッグすると命令間や式の入力欄に置けます。</p>
      <button
        v-for="item in palette"
        :key="item.name"
        type="button"
        draggable="true"
        :class="`pico-palette-item--${category}`"
        @click="addChoice(item.name)"
        @dragstart="dragChoice($event, item.name)"
        @dragend="endPicoDrag"
      >
        <Icon :icon="plusIcon" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
    </aside>
    <div class="pico-workspace">
      <div class="pico-functions" aria-label="関数を選択">
        <button
          v-for="name in Object.keys(functions)"
          :key="name"
          type="button"
          :aria-pressed="activeFunctionName === name"
          @click="activeFunctionName = name"
        >
          {{ name }}
        </button>
        <button
          v-if="!showNewFunction"
          type="button"
          class="pico-function-add-button"
          aria-label="関数を追加"
          title="関数を追加"
          @click="openNewFunction"
        >
          <Icon :icon="plusIcon" aria-hidden="true" />
        </button>
        <form v-else class="pico-new-function" @submit.prevent="addFunction">
          <input
            ref="newFunctionInput"
            v-model="newFunctionName"
            aria-label="新しい関数名"
            placeholder="関数名"
            @keydown.esc="closeNewFunction"
          />
          <button type="submit" aria-label="関数を作成" title="関数を作成">
            <Icon :icon="plusIcon" aria-hidden="true" />追加
          </button>
          <button type="button" aria-label="キャンセル" title="キャンセル" @click="closeNewFunction">
            <Icon :icon="xIcon" aria-hidden="true" />
          </button>
        </form>
      </div>
      <div class="pico-function-card">
        <div class="pico-function-header">
          <strong>関数 {{ activeFunctionName }}</strong>
          <span>→</span>
          <PicoTypeInput
            v-if="activeFunctionName !== 'main'"
            :model-value="currentFunction.returnType"
            @update:model-value="setReturnType"
          />
          <span v-else>unit</span>
          <button v-if="activeFunctionName !== 'main'" type="button" @click="removeFunction">
            <Icon :icon="trashIcon" aria-hidden="true" />関数を削除
          </button>
        </div>
        <div v-if="activeFunctionName !== 'main'" class="pico-params">
          <span>引数</span>
          <div v-for="(param, index) in currentFunction.params" :key="param.hintId">
            <input v-model="param.ident.name" aria-label="引数名" class="pico-name-input" />
            <PicoTypeInput v-model="param.type" />
            <button
              type="button"
              class="pico-variable-token"
              draggable="true"
              :aria-label="`引数 ${param.ident.name} をドラッグ`"
              @dragstart="dragVariable($event, param.ident.name)"
              @dragend="endPicoDrag"
            >
              {{ param.ident.name }}
            </button>
            <button
              type="button"
              :aria-label="`引数${index + 1}を削除`"
              @click="currentFunction.params.splice(index, 1)"
            >
              <Icon :icon="xIcon" aria-hidden="true" />
            </button>
          </div>
          <button type="button" @click="addParam"><Icon :icon="plusIcon" aria-hidden="true" />引数</button>
        </div>
        <PicoBlock
          :block="currentFunction.body"
          :functions="functions"
          :variables="currentFunction.params.map((param) => ({ name: param.ident.name, type: param.type }))"
          :return-type="currentFunction.returnType"
          terminal-return
        />
      </div>
      <details
        class="pico-json"
        :open="showJson"
        @toggle="showJson = ($event.target as HTMLDetailsElement).open"
      >
        <summary><Icon :icon="fileJsonIcon" aria-hidden="true" />Schema形式のJSON</summary>
        <button type="button" @click="copyJson">
          <Icon :icon="copyIcon" aria-hidden="true" />JSONをコピー
        </button>
        <pre>{{ JSON.stringify(program, null, 2) }}</pre>
        <label for="pico-json-input">JSONを読み込む</label>
        <textarea
          id="pico-json-input"
          v-model="jsonInput"
          spellcheck="false"
          placeholder="Picolang v0 のJSONを貼り付け"
        ></textarea>
        <button type="button" @click="importJson">
          <Icon :icon="uploadIcon" aria-hidden="true" />読み込む
        </button>
      </details>
    </div>
  </div>
</template>
