import { mkdir, readFile, writeFile } from 'node:fs/promises'
import openapiTS, { astToString, COMMENT_HEADER } from 'openapi-typescript'
import ts from 'typescript'

const output = new URL('../src/api/schema.d.ts', import.meta.url)
const recursiveAliases = []
const ast = await openapiTS(new URL('../../Schema/openapi.yaml', import.meta.url), {
  postTransform(type, { path }) {
    const schemaName = path?.match(/^#\/components\/schemas\/([^/]+)$/)?.[1]
    if (!schemaName || !ts.isUnionTypeNode(type)) return

    // TypeScript needs a named alias for self-recursive unions such as Value[].
    const aliasName = `RecursiveSchema${recursiveAliases.length}`
    let recursive = false
    const transformed = ts.transform(type, [
      (context) => (root) =>
        ts.visitNode(root, function visit(node) {
          if (
            ts.isIndexedAccessTypeNode(node) &&
            ts.isLiteralTypeNode(node.indexType) &&
            ts.isStringLiteral(node.indexType.literal) &&
            node.indexType.literal.text === schemaName &&
            astToString([node.objectType]).trim() === 'components["schemas"]'
          ) {
            recursive = true
            return ts.factory.createTypeReferenceNode(aliasName)
          }
          return ts.visitEachChild(node, visit, context)
        }),
    ])
    const aliasType = transformed.transformed[0]
    transformed.dispose()
    if (!recursive) return
    recursiveAliases.push(ts.factory.createTypeAliasDeclaration(undefined, aliasName, undefined, aliasType))
    return ts.factory.createTypeReferenceNode(aliasName)
  },
})
ast.push(...recursiveAliases)
// The bundler hoists $defs references to components/schemas, but also emits
// $defs as payload properties. Remove only those metadata properties from the AST.
const result = ts.transform(ast, [
  (context) => (root) =>
    ts.visitNode(root, function visit(node) {
      if (ts.isPropertySignature(node) && node.name.text === '$defs') return undefined
      return ts.visitEachChild(node, visit, context)
    }),
])
const contents = COMMENT_HEADER + astToString(result.transformed)
result.dispose()

if (process.argv.includes('--check')) {
  const existing = await readFile(output, 'utf8').catch((error) => {
    if (error.code === 'ENOENT') return null
    throw error
  })
  if (existing !== contents) {
    console.error('API types are out of date. Run: pnpm -C Client generate:api')
    process.exitCode = 1
  }
} else {
  await mkdir(new URL('.', output), { recursive: true })
  await writeFile(output, contents)
  console.log('Generated Client/src/api/schema.d.ts')
}
