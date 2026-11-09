import type { EChartsType } from 'echarts/core'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { applyRecordChartOptions } from '@/charts/recordChartOptions'

describe('health and activity chart configuration', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('preserves a readable line chart and reduced-motion preference', () => {
    vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: true })))
    const setOption = vi.fn()
    applyRecordChartOptions({ setOption } as unknown as EChartsType, {
      title: '心率记录', subtitle: '长者', labels: ['08:00'], values: [76], unit: '次/分'
    })
    expect(setOption).toHaveBeenCalledWith(expect.objectContaining({
      animation: false,
      aria: { enabled: true },
      series: [expect.objectContaining({ type: 'line', data: [76] })]
    }), true)
  })

  it('uses bars for sleep duration while keeping numeric observations unchanged', () => {
    vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: false })))
    const setOption = vi.fn()
    applyRecordChartOptions({ setOption } as unknown as EChartsType, {
      title: '睡眠时长', subtitle: '长者', labels: ['周一'], values: [6.5], unit: '小时', type: 'bar'
    })
    expect(setOption).toHaveBeenCalledWith(expect.objectContaining({
      animation: true,
      series: [expect.objectContaining({ type: 'bar', data: [6.5] })]
    }), true)
  })
})
