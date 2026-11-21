import { parse } from '@vue/compiler-sfc'

// Vue's SFC parser accepts valid end tags with whitespace and avoids treating
// arbitrary HTML or comments as script content.
export function scriptBlocks(source, filename) {
  const { descriptor, errors } = parse(source, { filename })
  if (errors.length) {
    throw new Error(`${filename}: ${errors.map(error => String(error)).join('; ')}`)
  }
  return [descriptor.script, descriptor.scriptSetup].filter(Boolean)
}