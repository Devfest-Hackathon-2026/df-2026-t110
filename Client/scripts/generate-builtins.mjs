import { readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'
import { parseArgs } from 'node:util'
import Ajv2020 from 'ajv/dist/2020.js'
import yaml from 'js-yaml'
import { format } from 'prettier'

export async function readHeader(input, schemaURL) {
  const source = await readFile(input, 'utf8')
  const header = JSON.parse(source)
  // YAML's JSON-compatible reader also rejects duplicate mapping keys.
  yaml.load(source)
  const loadSchema = async (uri) => yaml.load(await readFile(fileURLToPath(uri), 'utf8'))
  const schema = await loadSchema(schemaURL.href)
  const ajv = new Ajv2020({ strict: false, loadSchema })
  const validate = await ajv.compileAsync({ ...schema, $id: schemaURL.href })
  if (!validate(header)) throw new Error(ajv.errorsText(validate.errors))
  for (const [name, signature] of Object.entries(header)) {
    const declared = new Set(signature.typeParameters ?? [])
    function visit(type) {
      if (typeof type === 'string') return
      if (type.kind === 'type_parameter') {
        if (!declared.has(type.name)) throw new Error(`${name}: undeclared type parameter ${type.name}`)
      } else visit(type.elementType)
    }
    signature.params.forEach(visit)
    visit(signature.returnType)
  }
  return header
}

export async function renderHeader(header) {
  const entries = Object.keys(header)
    .sort()
    .map((name) => `[${JSON.stringify(name)}]: ${JSON.stringify(header[name])}`)
  return format(
    `// Code generated from lang/builtins.picoheader.json; DO NOT EDIT.
import type { PicoBuiltinSignature } from './picoSignatures.ts'

export const PICO_BUILTIN_SIGNATURES = {${entries.join(',')}} as const satisfies Record<string, PicoBuiltinSignature>
export type PicoBuiltinName = keyof typeof PICO_BUILTIN_SIGNATURES
`,
    { parser: 'typescript', semi: false, singleQuote: true, printWidth: 110 },
  )
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const { values } = parseArgs({
    options: {
      input: { type: 'string' },
      output: { type: 'string' },
      check: { type: 'boolean', default: false },
    },
  })
  const input = values.input ?? new URL('../../lang/builtins.picoheader.json', import.meta.url)
  const output =
    values.output ?? new URL('../src/features/picolang/language/builtins.gen.ts', import.meta.url)
  const schema = new URL('../../Schema/picolang/header_v0/header.schema.yaml', import.meta.url)
  const content = await renderHeader(await readHeader(input, schema))
  if (values.check) {
    if ((await readFile(output, 'utf8')) !== content)
      throw new Error('Builtin types are out of date; run mise run generate')
  } else await writeFile(output, content)
}
