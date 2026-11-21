import { describe, it, expect } from 'vitest'
import { scriptBlocks } from '../scripts/sfc-scripts.mjs'

describe('Vue SFC script extraction', () => {
  it('reads whitespace-terminated tags and excludes template decoys', () => {
    const blocks = scriptBlocks(`<template><div>script tag</div></template>
<script lang="ts">import value from './value'</script >
<script setup lang="ts">const result = value</script\t>`, 'fixture.vue')
    expect(blocks.map(block => block.content)).toEqual([
      "import value from './value'", 'const result = value'
    ])
  })

  it('fails closed on malformed SFC scripts', () => {
    expect(() => scriptBlocks('<script setup>const x=1</script><script setup>const y=2</script>', 'bad.vue')).toThrow()
  })
})