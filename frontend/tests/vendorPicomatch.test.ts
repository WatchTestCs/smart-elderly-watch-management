import { describe, it, expect } from 'vitest'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { removeBackslashes } = require('../vendor/secured-picomatch/lib/utils.js') as { removeBackslashes: (input: string) => string }
const legacy = (input: string) => input.replace(/(?:\[.*?[^\\]\]|\\(?=.))/g, match => match === '\\' ? '' : match)

describe('vendored picomatch escape scanner', () => {
  it('preserves the previous glob escaping semantics', () => {
    const cases = [
      '', '\\', 'a\\b', '\\[abc]', '\\[a\\]b]', '\\[', '\\[]',
      '\\[x\n]', '\\[\n]', '\\[\n\n]', '\\\\[x]', '\\[abc]\\d',
      '\\[\\[x]', 'abc\\\nxyz', '\\[x\u2028]', '\\[\u2028]'
    ]
    for (const input of cases) expect(removeBackslashes(input)).toBe(legacy(input))
    let seed = 123456789
    const chars = ['\\', '[', ']', 'a', '/', '\n', 'x']
    for (let n = 0; n < 120; n++) {
      let input = ''
      for (let i = 0; i < 40; i++) {
        seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
        input += chars[seed % chars.length]
      }
      expect(removeBackslashes(input)).toBe(legacy(input))
    }
  })

  it('handles long unmatched escaped brackets without backtracking', () => {
    const input = '\\['.repeat(16384)
    expect(removeBackslashes(input)).toBe('['.repeat(16384))
  })
})