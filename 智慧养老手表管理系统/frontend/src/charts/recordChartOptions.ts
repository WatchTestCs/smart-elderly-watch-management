import type { EChartsType } from 'echarts/core'

export interface ChartOptions {
  title: string
  subtitle: string
  labels: unknown[]
  values: unknown[]
  unit: string
  type?: 'bar'
}
export function applyRecordChartOptions(chart: EChartsType, options: ChartOptions): void {
  const bar = options.type === 'bar'
  const color = bar ? '#806025' : '#235442'
  const fontFamily = getComputedStyle(document.body).fontFamily
  chart.setOption({
    animation: !window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    animationDuration: 240,
    animationDurationUpdate: 140,
    aria: { enabled: true },
    color: [color],
    textStyle: { fontFamily, fontSize: 16, color: '#53645c' },
    title: { text: options.title, subtext: options.subtitle, textStyle: { fontSize: 20, fontWeight: 500, color: '#213c32' }, subtextStyle: { fontSize: 14, color: '#53645c' } },
    grid: { left: 20, right: 20, top: 92, bottom: 20, containLabel: true },
    tooltip: { trigger: 'axis', backgroundColor: '#183e33', padding: [10, 14], confine: true, textStyle: { fontFamily }, axisPointer: { type: bar ? 'shadow' : 'line', lineStyle: { color: '#81917b', type: 'dashed' }, shadowStyle: { color: 'rgba(35,84,66,.05)' } } },
    xAxis: { type: 'category', data: options.labels, boundaryGap: bar, axisLine: { lineStyle: { color: '#dce4da' } }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16, hideOverlap: true } },
    yAxis: { type: 'value', name: options.unit, nameTextStyle: { color: '#53645c', fontSize: 16 }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16 }, splitLine: { lineStyle: { color: '#e8ece4', type: 'dashed' } } },
    series: [{ name: options.title, type: bar ? 'bar' : 'line', data: options.values, barMaxWidth: 32, symbol: 'circle', symbolSize: 6, lineStyle: { width: 2 }, itemStyle: { color, borderRadius: 3, barBorderRadius: [3, 3, 0, 0] } }]
  }, true)
}
