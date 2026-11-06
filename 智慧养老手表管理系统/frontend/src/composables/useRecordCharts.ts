import { nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import type { EChartsType } from 'echarts/core'
import { api } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'

interface ChartOptions {
  title: string
  subtitle: string
  labels: unknown[]
  values: unknown[]
  unit: string
  type?: 'bar'
}
interface ChartData {
  createTimes?: unknown[]
  createTime?: unknown[]
  hrs?: unknown[]
  sleep?: unknown[]
  bps?: unknown[]
  time?: string
}

const array = (value: unknown): unknown[] => Array.isArray(value) ? value : []

export function useRecordCharts() {
  const detailsVisible = ref(false)
  const chartLoading = ref(false)
  const chartError = ref('')
  const chartName = ref('')
  const primaryChartElement = shallowRef<HTMLElement | null>(null)
  const secondaryChartElement = shallowRef<HTMLElement | null>(null)
  const charts: EChartsType[] = []
  let frame: number | null = null
  let observer: ResizeObserver | null = null
  let request: AbortController | null = null
  let unmounted = false

  function disposeCharts() {
    if (frame !== null) cancelAnimationFrame(frame)
    frame = null
    observer?.disconnect()
    observer = null
    for (const chart of charts) if (!chart.isDisposed()) chart.dispose()
    charts.length = 0
  }

  function cancelRecordRequest() {
    request?.abort()
    request = null
    chartLoading.value = false
  }

  function resizeCharts() {
    if (frame !== null || unmounted) return
    frame = requestAnimationFrame(() => {
      frame = null
      for (const chart of charts) if (!chart.isDisposed()) chart.resize()
    })
  }

  async function showRecordDetails(kind: 'vitals' | 'activity', oldId: string | number, name: string) {
    cancelRecordRequest()
    const controller = new AbortController()
    request = controller
    chartName.value = name || ''
    chartError.value = ''
    detailsVisible.value = true
    chartLoading.value = true
    const active = () => request === controller && detailsVisible.value && !unmounted
    try {
      const [{ data }, { default: echarts }] = await Promise.all([
        api.post<ChartData>(`api/v1/${kind}/chart`, { oldId }, { signal: controller.signal }),
        import('../chartEngine')
      ])
      if (!active()) return
      const vital = kind === 'vitals'
      const labels = array(vital ? data.createTimes : data.createTime)
      const values = array(vital ? data.hrs : data.sleep)
      const pressures = array(data.bps)
      if (!values.length && (!vital || !pressures.length)) {
        chartError.value = vital ? '暂无可展示的健康详情。' : '暂无可展示的睡眠详情。'
        return
      }
      await nextTick()
      if (!active()) return
      const subtitle = vital ? chartName.value + ' · ' + (data.time || '') : chartName.value

      function render(element: HTMLElement | null, options: ChartOptions) {
        if (!element || !active()) return
        const chart: EChartsType = echarts.getInstanceByDom(element) || echarts.init(element)
        if (!charts.includes(chart)) {
          charts.push(chart)
          if (typeof ResizeObserver !== 'undefined') {
            if (!observer) observer = new ResizeObserver(resizeCharts)
            observer.observe(element)
          }
        }
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

      render(primaryChartElement.value, vital
        ? { title: '心率记录', subtitle, labels, values, unit: '次/分' }
        : { title: '睡眠时长', subtitle, labels, values, unit: '小时', type: 'bar' })
      if (vital) render(secondaryChartElement.value, { title: '血压记录', subtitle, labels, values: pressures, unit: 'mmHg' })
    } catch (error) {
      if (active()) chartError.value = apiErrorMessage(error, '详情暂时无法加载，请重试。')
    } finally {
      if (request === controller) {
        request = null
        chartLoading.value = false
      }
    }
  }

  watch(detailsVisible, visible => {
    if (!visible) {
      cancelRecordRequest()
      disposeCharts()
    }
  })
  onMounted(() => window.addEventListener('resize', resizeCharts))
  onBeforeUnmount(() => {
    unmounted = true
    cancelRecordRequest()
    window.removeEventListener('resize', resizeCharts)
    disposeCharts()
  })
  return {
    detailsVisible, chartLoading, chartError, chartName,
    primaryChartElement, secondaryChartElement,
    showRecordDetails, cancelRecordRequest, resizeCharts, disposeCharts
  }
}
