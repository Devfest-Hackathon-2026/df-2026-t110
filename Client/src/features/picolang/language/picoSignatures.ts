import type { PicoPrimitive } from './picolang.ts'

/** Signature templates can contain parameters; execution JSON types cannot. */
export type PicoSignatureType =
  | PicoPrimitive
  | { readonly kind: 'stack' | 'option'; readonly elementType: PicoSignatureType }
  | { readonly kind: 'type_parameter'; readonly name: string }

export interface PicoBuiltinSignature {
  readonly typeParameters?: readonly string[]
  readonly params: readonly PicoSignatureType[]
  readonly returnType: PicoSignatureType
}

/** Instantiate a template once per call, sharing named parameters within it. */
export function instantiateSignature<T>(
  signature: PicoBuiltinSignature,
  fresh: (name: string) => T,
  primitive: (name: PicoPrimitive) => T,
  compound: (kind: 'stack' | 'option', element: T) => T,
): { params: T[]; result: T } {
  const parameters = new Map<string, T>()
  function instantiate(type: PicoSignatureType): T {
    if (typeof type === 'string') return primitive(type)
    if (type.kind !== 'type_parameter') return compound(type.kind, instantiate(type.elementType))
    if (!parameters.has(type.name)) parameters.set(type.name, fresh(type.name))
    return parameters.get(type.name)!
  }
  return { params: signature.params.map(instantiate), result: instantiate(signature.returnType) }
}

export function hasTypeParameter(type: PicoSignatureType): boolean {
  return typeof type !== 'string' && (type.kind === 'type_parameter' || hasTypeParameter(type.elementType))
}

/** Editing accepts unresolved parameters; semantic inference checks their relationships. */
export function compatibleSignatureTypes(
  left: PicoSignatureType | undefined,
  right: PicoSignatureType | undefined,
): boolean {
  if (!left || !right) return true
  if (typeof left !== 'string' && left.kind === 'type_parameter') return true
  if (typeof right !== 'string' && right.kind === 'type_parameter') return true
  if (typeof left === 'string' || typeof right === 'string') return left === right
  return left.kind === right.kind && compatibleSignatureTypes(left.elementType, right.elementType)
}
