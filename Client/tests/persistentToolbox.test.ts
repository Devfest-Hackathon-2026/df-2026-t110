import test, { after, before, type TestContext } from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import { PersistentToolbox } from '../src/features/picolang/editor/PersistentToolbox.ts'

// Reuse the DOM supplied by Blockly's Node entry point; these tests exercise
// selection and real focus callbacks, rather than SVG layout.
const document = Blockly.utils.xml.createElement('div').ownerDocument
const domWindow = document.defaultView! as Window & typeof globalThis
const globals = {
  document,
  window: domWindow,
  Element: domWindow.Element,
  HTMLElement: domWindow.HTMLElement,
  HTMLInputElement: domWindow.HTMLInputElement,
  HTMLTextAreaElement: domWindow.HTMLTextAreaElement,
  SVGElement: domWindow.SVGElement,
  Node: domWindow.Node,
  PointerEvent: domWindow.PointerEvent,
  FocusEvent: domWindow.FocusEvent,
}
const originalGlobals = Object.fromEntries(
  Object.keys(globals).map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)]),
)

before(() => {
  Object.assign(globalThis, globals)
  Object.defineProperty(domWindow.Element.prototype, 'scrollIntoView', { value: () => {} })
  Object.defineProperty(domWindow.SVGElement.prototype, 'getBBox', {
    value: () => ({ x: 0, y: 0, width: 80, height: 20 }),
  })
  Object.defineProperty(domWindow.HTMLCanvasElement.prototype, 'getContext', {
    value: () => ({ measureText: (text: string) => ({ width: text.length * 8 }) }),
  })
})

after(() => {
  for (const [key, descriptor] of Object.entries(originalGlobals)) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor)
    else Reflect.deleteProperty(globalThis, key)
  }
})

function fixture(t: TestContext) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const workspace = Blockly.inject(host, {
    toolbox: {
      kind: 'categoryToolbox',
      contents: ['動く', '制御'].map((name) => ({
        kind: 'category',
        name,
        contents: [{ kind: 'label', text: name }],
      })),
    },
    plugins: { toolbox: PersistentToolbox },
    sounds: false,
  })
  const toolbox = workspace.getToolbox()!
  const [first, second] = toolbox.getToolboxItems()
  const pointer = (item: Blockly.IToolboxItem, type: string, pointerType = 'touch') =>
    item.getClickTarget()!.dispatchEvent(
      new domWindow.PointerEvent(type, {
        bubbles: true,
        cancelable: true,
        pointerType,
        pointerId: pointerType === 'touch' ? 2 : 1,
        isPrimary: true,
      }),
    )
  const click = (item: Blockly.IToolboxItem) =>
    item
      .getClickTarget()!
      .dispatchEvent(new domWindow.MouseEvent('click', { bubbles: true, cancelable: true }))
  const tap = (item: Blockly.IToolboxItem, focus: 'before' | 'after' | 'none' = 'before') => {
    pointer(item, 'pointerdown')
    pointer(item, 'pointerup')
    if (focus === 'before') item.getFocusableElement().focus()
    click(item)
    if (focus === 'after') item.getFocusableElement().focus()
  }
  t.after(() => {
    workspace.dispose()
    host.remove()
  })
  return { workspace, toolbox, first, second, pointer, click, tap }
}

for (const focus of ['before', 'after', 'none'] as const) {
  test(`カテゴリの再タップで閉じる（ネイティブフォーカス: ${focus}）`, (t) => {
    const { workspace, toolbox, first, tap } = fixture(t)
    tap(first, focus)
    assert.equal(toolbox.getSelectedItem(), first)
    assert.equal(workspace.getFlyout()!.isVisible(), true)

    tap(first, focus)
    assert.equal(toolbox.getSelectedItem(), null)
    assert.equal(workspace.getFlyout()!.isVisible(), false)

    tap(first, focus)
    assert.equal(toolbox.getSelectedItem(), first)
    assert.equal(workspace.getFlyout()!.isVisible(), true)
  })
}

test('別カテゴリへのタップはフォーカスで先に選択されても閉じない', (t) => {
  const { workspace, toolbox, first, second, tap } = fixture(t)
  tap(first)
  tap(second)
  assert.equal(toolbox.getSelectedItem(), second)
  assert.equal(workspace.getFlyout()!.isVisible(), true)
  tap(second)
  assert.equal(toolbox.getSelectedItem(), null)
})

test('キャンセルされたタッチや別カテゴリへのクリックでは開閉しない', (t) => {
  const { toolbox, first, second, pointer, click, tap } = fixture(t)
  tap(first)
  pointer(first, 'pointerdown')
  pointer(first, 'pointercancel')
  click(first)
  assert.equal(toolbox.getSelectedItem(), first)

  pointer(first, 'pointerdown')
  pointer(first, 'pointerup')
  click(second)
  assert.equal(toolbox.getSelectedItem(), first)
})

test('マウスは押下時に開閉し、後続のクリックで二重に切り替わらない', (t) => {
  const { toolbox, first, pointer, click } = fixture(t)
  pointer(first, 'pointerdown', 'mouse')
  assert.equal(toolbox.getSelectedItem(), first)
  pointer(first, 'pointerup', 'mouse')
  click(first)
  assert.equal(toolbox.getSelectedItem(), first)
  pointer(first, 'pointerdown', 'mouse')
  assert.equal(toolbox.getSelectedItem(), null)
  pointer(first, 'pointerup', 'mouse')
  click(first)
  assert.equal(toolbox.getSelectedItem(), null)
})

test('タッチで閉じた後もフォーカスでカテゴリを選択でき、キー入力は保留中のタップを取り消す', (t) => {
  const { workspace, toolbox, first, second, pointer, click, tap } = fixture(t)
  tap(first)
  tap(first)
  second.getFocusableElement().focus()
  assert.equal(toolbox.getSelectedItem(), second)
  assert.equal(workspace.getFlyout()!.isVisible(), true)

  pointer(second, 'pointerdown')
  pointer(second, 'pointerup')
  second
    .getFocusableElement()
    .dispatchEvent(
      new domWindow.KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true, cancelable: true }),
    )
  click(second)
  assert.equal(toolbox.getSelectedItem(), second)
})
