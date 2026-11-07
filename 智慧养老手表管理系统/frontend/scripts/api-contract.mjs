import { execFileSync } from 'node:child_process'
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

export const generatedPath = fileURLToPath(new URL('../src/api/generated-api.d.ts', import.meta.url))
const healthPath = '/api/v1/health/latest'

export function exportDocument() {
  return JSON.parse(execFileSync('go', ['run', './cmd/openapi'], {
    cwd: fileURLToPath(new URL('../../backend/', import.meta.url)),
    encoding: 'utf8', maxBuffer: 8 * 1024 * 1024
  }))
}

function schemaType(schema, document) {
  if (!schema || typeof schema !== 'object') return 'unknown'
  if (schema.$ref) {
    const path = schema.$ref.replace(/^#\//, '').split('/')
    let value = document
    for (const part of path) value = value?.[part.replace(/~1/g, '/').replace(/~0/g, '~')]
    if (!value) throw new Error('Unresolved OpenAPI reference: ' + schema.$ref)
    return schemaType(value, document)
  }
  if (Object.hasOwn(schema, 'const')) return JSON.stringify(schema.const)
  if (Array.isArray(schema.enum)) return schema.enum.map(x => JSON.stringify(x)).join(' | ') || 'never'
  if (schema.anyOf || schema.oneOf) {
    return (schema.anyOf || schema.oneOf).map(x => schemaType(x, document)).join(' | ')
  }
  if (Array.isArray(schema.type)) {
    return schema.type.map(t => schemaType({ ...schema, type: t }, document)).join(' | ')
  }
  switch (schema.type) {
    case 'string': return 'string'
    case 'integer':
    case 'number': return 'number'
    case 'boolean': return 'boolean'
    case 'null': return 'null'
    case 'array': return '(' + schemaType(schema.items, document) + ')[]'
    case 'object':
    case undefined: {
      const entries = Object.entries(schema.properties || {})
      const required = new Set(schema.required || [])
      const fields = entries.map(([name, value]) =>
        JSON.stringify(name) + (required.has(name) ? ': ' : '?: ') + schemaType(value, document) + ';')
      if (schema.additionalProperties === true) fields.push('[key: string]: unknown;')
      else if (schema.additionalProperties && typeof schema.additionalProperties === 'object') {
        fields.push('[key: string]: ' + schemaType(schema.additionalProperties, document) + ';')
      }
      if (!fields.length) return schema.additionalProperties === false ? 'Record<string, never>' : 'Record<string, unknown>'
      return '{ ' + fields.join(' ') + ' }'
    }
    default: throw new Error('Unsupported OpenAPI schema type: ' + schema.type)
  }
}

export function generateTypes(document) {
  const op = document.paths?.[healthPath]?.post
  if (!op) return '// Generated from the native Go OpenAPI. Do not edit.\nexport interface paths {}\n'
  const request = op.requestBody?.content?.['application/json']?.schema
  const reply = op.responses?.['200']?.content?.['application/json']?.schema
  if (!request || !reply) throw new Error('Missing native Go health contract schema')
  return '// Generated from the native Go OpenAPI. Do not edit.\n'
    + 'export interface paths {\n'
    + '  ' + JSON.stringify(healthPath) + ': {\n'
    + '    post: {\n'
    + '      requestBody: { content: { "application/json": ' + schemaType(request, document) + ' } };\n'
    + '      responses: { 200: { content: { "application/json": ' + schemaType(reply, document) + ' } } };\n'
    + '    };\n'
    + '  };\n'
    + '}\n'
}

export async function generate(check) {
  const document = exportDocument()
  const text = generateTypes(document)
  if (check) {
    if (readFileSync(generatedPath, 'utf8') !== text) {
      throw new Error('Generated API types differ from the published Go contract; run npm run api:generate')
    }
  } else writeFileSync(generatedPath, text)
  console.log(`Published OpenAPI → TypeScript: ${Object.keys(document.paths).length} paths${check ? ' checked' : ' generated'}`)
}
