import { api } from './api/client'
import { apiErrorMessage } from './api/httpContract'

const array = value => Array.isArray(value) ? value : []

// Chart instances stay outside Vue's observation graph and are owned by the view.
export const recordCharts = {
  data () { return { detailsVisible: false, chartLoading: false, chartError: '', chartName: '' } },
  watch: { detailsVisible (visible) { if (!visible) { this.cancelRecordRequest(); this.disposeCharts() } } },
  beforeCreate () { this._chartsUnmounted = false; this._recordCharts = []; this._chartFrame = null; this._chartObserver = null; this._recordRequest = null },
  mounted () { window.addEventListener('resize', this.resizeCharts) },
  beforeUnmount () { this._chartsUnmounted = true; this.cancelRecordRequest(); window.removeEventListener('resize', this.resizeCharts); this.disposeCharts() },
  methods: {
    renderRecordChart (echarts, element, options) {
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
      const fontFamily = getComputedStyle(document.body).fontFamily
      chart.setOption({
        animation: !window.matchMedia('(prefers-reduced-motion: reduce)').matches,
        animationDuration: 240, animationDurationUpdate: 140,
        aria: { enabled: true },
        color: [color],
        textStyle: { fontFamily, fontSize: 16, color: '#53645c' },
        title: { text: options.title, subtext: options.subtitle || '', textStyle: { fontSize: 20, fontWeight: 500, color: '#213c32' }, subtextStyle: { fontSize: 14, color: '#53645c' } },
        grid: { left: 20, right: 20, top: 92, bottom: 20, containLabel: true },
        tooltip: { trigger: 'axis', backgroundColor: '#183e33', padding: [10, 14], confine: true, textStyle: { fontFamily }, axisPointer: { type: bar ? 'shadow' : 'line', lineStyle: { color: '#81917b', type: 'dashed' }, shadowStyle: { color: 'rgba(35,84,66,.05)' } } },
        xAxis: { type: 'category', data: options.labels, boundaryGap: bar, axisLine: { lineStyle: { color: '#dce4da' } }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16, hideOverlap: true } },
        yAxis: { type: 'value', name: options.unit || '', nameTextStyle: { color: '#53645c', fontSize: 16 }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#53645c', fontSize: 16 }, splitLine: { lineStyle: { color: '#e8ece4', type: 'dashed' } } },
        series: [{ name: options.title, type: bar ? 'bar' : 'line', data: options.values, barMaxWidth: 32, symbol: 'circle', symbolSize: 6, lineStyle: { width: 2 }, itemStyle: { color, borderRadius: 3, barBorderRadius: [3, 3, 0, 0] } }]
      }, true)
    },
    async showRecordDetails (kind, oldId, name) {
      this.cancelRecordRequest()
      const request = new AbortController()
      this._recordRequest = request
      this.chartName = name || ''
      this.detailsVisible = this.chartLoading = true
      this.chartError = ''
      const active = () => this._recordRequest === request && this.detailsVisible
      try {
        const [{ data }, { default: echarts }] = await Promise.all([
          api.post(`api/v1/${kind}/chart`, { oldId }, { signal: request.signal }),
          import('./chartEngine')
        ])
        if (!active()) return
        const vital = kind === 'vitals'
        const labels = array(vital ? data.createTimes : data.createTime)
        const values = array(vital ? data.hrs : data.sleep)
        const pressures = array(data.bps)
        if (!values.length && (!vital || !pressures.length)) {
          this.chartError = vital ? '暂无可展示的健康详情。' : '暂无可展示的睡眠详情。'
          return
        }
        await this.$nextTick()
        if (!active()) return
        const subtitle = vital ? this.chartName + ' · ' + (data.time || '') : this.chartName
        this.renderRecordChart(echarts, this.$refs.mychart, vital
          ? { title: '心率记录', subtitle, labels, values, unit: '次/分' }
          : { title: '睡眠时长', subtitle, labels, values, unit: '小时', type: 'bar' })
        if (vital) this.renderRecordChart(echarts, this.$refs.mychartTwo, {
          title: '血压记录', subtitle, labels, values: pressures, unit: 'mmHg'
        })
      } catch (error) {
        if (active()) this.chartError = apiErrorMessage(error, '详情暂时无法加载，请重试。')
      } finally {
        if (this._recordRequest === request) {
          this._recordRequest = null
          this.chartLoading = false
        }
      }
    },
    cancelRecordRequest () {
      this._recordRequest?.abort()
      this._recordRequest = null
      this.chartLoading = false
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
