import type { PicoVariableOption } from './picoOptions.ts'
import {
  PICO_BUILTINS,
  ident,
  makeCall,
  samePicoType,
  type PicoExpr,
  type PicoFunction,
  type PicoType,
} from '../language/picolang.ts'

export const PICO_VARIABLE_MIME = 'application/x-picolang-variable'

export function beginPicoDrag(): void {
  document.documentElement.classList.add('pico-dragging')
}

export function endPicoDrag(): void {
  document.documentElement.classList.remove('pico-dragging')
}

export function dragVariable(event: DragEvent, name: string): void {
  if (!event.dataTransfer || !name.trim()) return
  beginPicoDrag()
  event.dataTransfer.setData(PICO_VARIABLE_MIME, name)
  event.dataTransfer.effectAllowed = 'copy'
}

export function expressionFromDrop(
  transfer: DataTransfer | null,
  expectedType: PicoType | undefined,
  functions: Record<string, PicoFunction>,
  variables: PicoVariableOption[],
): PicoExpr | null {
  if (!transfer) return null
  const variableName = transfer.getData(PICO_VARIABLE_MIME)
  if (variableName) {
    const variable = variables.find((item) => item.name === variableName)
    return variable && samePicoType(expectedType, variable.type) ? ident(variableName) : null
  }
  const name = transfer.getData('application/x-picolang-choice')
  if (!name) return null
  const builtin = PICO_BUILTINS.find((item) => item.name === name)
  const returnType = builtin?.returns ?? functions[name]?.returnType
  return returnType &&
    (samePicoType(expectedType, returnType) || (name === 'new' && typeof expectedType === 'object'))
    ? makeCall(name, functions)
    : null
}
