import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { readHeader, renderHeader } from '../scripts/generate-builtins.mjs'

const schema = new URL('../../Schema/picolang/header_v0/header.schema.yaml', import.meta.url)

test('frontend generator validates headers before emitting code', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'picoheader-'))
  const input = join(directory, 'header.json')
  try {
    const parameter = { kind: 'type_parameter', name: 'A' }
    const valid = {
      identity: {
        typeParameters: ['A', 'B'],
        params: [
          { kind: 'stack', elementType: { kind: 'option', elementType: parameter } },
          { kind: 'type_parameter', name: 'B' },
        ],
        returnType: parameter,
      },
    }
    await writeFile(input, JSON.stringify(valid))
    assert.deepEqual(await readHeader(input, schema), valid)
    for (const invalid of [
      [{ name: 'f', signature: { params: [], returnType: 'bool' } }],
      { f: { params: [], returnType: 'invalid' } },
      { f: { params: [], returnType: 'bool', tags: ['movement'] } },
      { '': { params: [], returnType: 'bool' } },
      { f: { returnType: 'bool' } },
      { f: { typeParameters: ['T', 'T'], params: [], returnType: 'bool' } },
      { f: { params: [], returnType: parameter } },
      {
        f: {
          typeParameters: ['B'],
          params: [{ kind: 'option', elementType: parameter }],
          returnType: 'unit',
        },
      },
    ]) {
      await writeFile(input, JSON.stringify(invalid))
      await assert.rejects(readHeader(input, schema))
    }
    await writeFile(input, '{"f":{"params":[],"returnType":"bool"},"f":{"params":[],"returnType":"integer"}}')
    await assert.rejects(readHeader(input, schema))
  } finally {
    await rm(directory, { recursive: true, force: true })
  }
})

test('frontend generated code is deterministic and current', async () => {
  const input = new URL('../../lang/builtins.picoheader.json', import.meta.url)
  const header = await readHeader(input, schema)
  const reversed = Object.fromEntries(Object.entries(header).reverse())
  const generated = await renderHeader(header)
  assert.equal(await renderHeader(reversed), generated)
  assert.equal(
    await readFile(new URL('../src/features/picolang/language/builtins.gen.ts', import.meta.url), 'utf8'),
    generated,
  )
})
