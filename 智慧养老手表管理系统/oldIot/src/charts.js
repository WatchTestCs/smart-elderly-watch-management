import * as echarts from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, TitleComponent, AriaComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([LineChart, BarChart, GridComponent, TooltipComponent, TitleComponent, AriaComponent, CanvasRenderer])

// Chart instances stay outside Vue's observation graph and are owned by the view.
export const recordCharts = {
  beforeCreate () { this._chartsUnmounted = false; this._recordCharts = []; this._chartFrame = null; this._chartObserver = null },
  mounted () { window.addEventListener('resize', this.resizeCharts) },
  beforeUnmount () { this._chartsUnmounted = true; window.removeEventListener('resize', this.resizeCharts); this.disposeCharts() },
  methods: {
    renderRecordChart (element, options) {
      if (!element || this._chartsUnmounted) return
      var chart = echarts.getInstanceByDom(element) || echarts.init(element)
      if (this._recordCharts.indexOf(chart) === -1) {
        this._recordCharts.push(chart)
        if (typeof ResizeObserver !== 'undefined') {
          if (!this._chartObserver) this._chartObserver = new ResizeObserver(this.resizeCharts)
          this._chartObserver.observe(element)
        }
      }
      var bar = options.type === 'bar'
      var color = bar ? '#806025' : '#235442'
      chart.setOption({
        animation: !window.matchMedia('(prefers-reduced-motion: reduce)').matches,
        animationDuration: 240, animationDurationUpdate: 140,
        aria: { enabled: true },
        color: [color],
        textStyle: { fontFamily: getComputedStyle(document.body).fontFamily, fontSize: 16, color: '#53645c' },
        title: { text: options.title, subtext: options.subtitle || '', textStyle: { fontSize: 20, fontWeight: 500, color: '#213c32' }, subtextStyle: { fontSize: 14, color: '#53645c' } },
        grid: { left: 20, right: 20, top: 92, bottom: 20, containLabel: true },
        tooltip: { trigger: 'axis', backgroundColor: '#183e33', padding: [10, 14], confine: true, textStyle: { fontFamily: getComputedStyle(document.body).fontFamily }, axisPointer: { type: bar ? 'shadow' : 'line', lineStyle: { color: '#81917b', type: 'dashed' }, shadowStyle: { color: 'rgba(35,84,66,.05)' } } },
        xAxis: { type: 'category', data: options.labels, boundaryGap: bar, axisLine: { lineStyle: { color: '#dce4da' } }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16, hideOverlap: true } },
        yAxis: { type: 'value', name: options.unit || '', nameTextStyle: { color: '#53645c', fontSize: 16 }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16 }, splitLine: { lineStyle: { color: '#e8ece4', type: 'dashed' } } },
        series: [{ name: options.title, type: bar ? 'bar' : 'line', data: options.values, barMaxWidth: 32, symbol: 'circle', symbolSize: 6, lineStyle: { width: 2 }, itemStyle: { color, borderRadius: 3, barBorderRadius: [3, 3, 0, 0] } }]
      }, true)
    },
    resizeCharts () {
      if (this._chartFrame !== null) return
      this._chartFrame = requestAnimationFrame(() => {
        this._chartFrame = null
        this._recordCharts.forEach(chart => { if (!chart.isDisposed()) chart.resize() })
      })
    },
    disposeCharts () {
      if (this._chartFrame !== null) cancelAnimationFrame(this._chartFrame)
      this._chartFrame = null
      if (this._chartObserver) this._chartObserver.disconnect()
      this._chartObserver = null
      this._recordCharts.forEach(chart => { if (!chart.isDisposed()) chart.dispose() })
      this._recordCharts = []
    }
  }
}
