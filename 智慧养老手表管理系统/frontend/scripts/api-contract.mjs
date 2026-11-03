import { execFileSync } from 'node:child_process'
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import openapiTS, { astToString } from 'openapi-typescript'

export const generatedPath = fileURLToPath(new URL('../src/api/generated-api.d.ts', import.meta.url))

export function exportDocument() {
  return JSON.parse(execFileSync('go', ['run', './cmd/openapi'], {
    cwd: fileURLToPath(new URL('../../backend/', import.meta.url)),
    encoding: 'utf8', maxBuffer: 8 * 1024 * 1024
  }))
}

export async function generateTypes(document) {
  const ast = await openapiTS(document)
  return '// Generated from the published Go OpenAPI by npm run api:generate. Do not edit.\n' + astToString(ast)
}

export async function generate(check) {
  const document = exportDocument()
  const text = await generateTypes(document)
  if (check) {
    if (readFileSync(generatedPath, 'utf8') !== text) {
      throw new Error('Generated API types differ from the published Go contract; run npm run api:generate')
    }
  } else writeFileSync(generatedPath, text)
  console.log(`Published OpenAPI → TypeScript: ${Object.keys(document.paths).length} paths${check ? ' checked' : ' generated'}`)
}
