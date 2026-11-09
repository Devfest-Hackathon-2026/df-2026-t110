import test from 'node:test'
import assert from 'node:assert/strict'
import * as Blockly from 'blockly'
import { callType, picoToolbox, registerPicoBlocks } from '../src/features/picolang/editor/blocklyPico.ts'
import { PICO_BUILTIN_OPTIONS } from '../src/features/picolang/editor/picoOptions.ts'
import {
  PICO_BUILTIN_SIGNATURES,
  newProgram,
  programFunctions,
} from '../src/features/picolang/language/picolang.ts'
import type { PicoSignatureType } from '../src/features/picolang/language/picoSignatures.ts'

// Blockly checks container kinds for generic elements, leaving their equality
// constraints to semantic inference. Concrete nested types retain their key.
function connectionCheck(type: PicoSignatureType): string[] | null {
  if (typeof type === 'string') return [type]
  if (type.kind === 'type_parameter') return null
  const element = connectionCheck(type.elementType)
  return !element || (element.length === 1 && typeof type.elementType !== 'string')
    ? [type.kind]
    : [`${type.kind}:${element[0]}`, type.kind]
}

test('every builtin block has the header inputs and output, including toolbox defaults', () => {
  const functions = programFunctions(newProgram())
  registerPicoBlocks(functions)
  const toolbox = picoToolbox(functions)
  const workspace = new Blockly.Workspace()
  try {
    for (const option of PICO_BUILTIN_OPTIONS) {
      const signature = PICO_BUILTIN_SIGNATURES[option.name]
      for (const statement of [false, true]) {
        const block = workspace.newBlock(callType(option.name, statement))
        assert.equal(
          block.inputList.filter((input) => input.name.startsWith('ARG')).length,
          signature.params.length,
          option.name,
        )
        signature.params.forEach((param, index) => {
          assert.deepEqual(
            block.getInput(`ARG${index}`)!.connection!.getCheck(),
            connectionCheck(param),
            `${option.name} argument ${index}`,
          )
        })
        if (!statement)
          assert.deepEqual(
            block.outputConnection!.getCheck(),
            connectionCheck(signature.returnType),
            `${option.name} output`,
          )
      }
      const category = toolbox.contents.find(
        (item) => item.kind === 'category' && 'name' in item && item.name === option.category,
      ) as Blockly.utils.toolbox.StaticCategoryInfo
      const type = callType(option.name, ['動く', 'オブジェクト', '発言'].includes(option.category))
      const entry = category.contents.find(
        (item) => item.kind === 'block' && 'type' in item && item.type === type,
      ) as Blockly.utils.toolbox.BlockInfo
      assert.ok(entry, `${option.name} toolbox entry`)
      const block = Blockly.serialization.blocks.append({ ...entry, type }, workspace)
      signature.params.forEach((_, index) =>
        assert.ok(block.getInputTargetBlock(`ARG${index}`), `${option.name} default ${index}`),
      )
    }
    const distance = workspace.newBlock(callType('distance', false))
    assert.deepEqual(distance.getInput('ARG0')!.connection!.getCheck(), ['distance_target'])
    for (const name of ['is_own_territory', 'is_enemy_territory', 'is_luggage']) {
      const block = workspace.newBlock(callType(name, false))
      assert.deepEqual(block.getInput('ARG0')!.connection!.getCheck(), ['terrain'])
      assert.deepEqual(block.outputConnection!.getCheck(), ['bool'])
    }
  } finally {
    workspace.dispose()
  }
})
