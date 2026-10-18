import axios from 'axios'

export const deviceListMethods = {
    scheduleFilter () {
      clearTimeout(this.filterTimer)
      this.filterTimer = setTimeout(() => { this.currentPage = 1; this.loadDevices() }, 250)
    },
    changeDevicePage (page) { this.currentPage = page; this.loadDevices() },
    handleDeviceCommand (command, row) {
      if (command === 'details') { this.detailDevice = row; this.deviceDetailVisible = true }
      else if (command === 'reset') this.resetSecret(row)
      else if (command === 'unbind') this.unbind(row)
      else if (command === 'delete') this.deleteDevice(row)
    },
    post (path, params) {
      return axios.post('restful/device/' + path, params || {}, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      })
    },
    loadDevices () {
      if (this.deviceLoading) { this.reloadPending = true; return }
      this.deviceLoading = true
      this.deviceLoadError = ''
      var devices = this.post('list', { page: this.currentPage, pageSize: this.pageSize, query: this.deviceQuery, state: this.deviceState }).then((response) => {
        if (!Array.isArray(response.data.devices)) throw new Error('missing devices')
        this.deviceList = response.data.devices
        this.deviceTotal = response.data.total ?? this.deviceList.length
        this.serverOnlineCount = response.data.onlineCount ?? this.deviceList.filter(item => item.online === 1).length
      }).catch(() => { this.deviceLoadError = '设备列表暂时无法更新，请重新加载。' })
      return Promise.all([devices, this.loadAlertCounts()]).then(() => { this.deviceLoading = false; if (this.reloadPending && !this.refreshDisposed) { this.reloadPending = false; this.loadDevices() } })
    },
    loadAlertCounts () {
      this.alertLoadError = ''
      return this.post('alertCounts').then((response) => {
        if (!Array.isArray(response.data.items) || response.data.total == null) throw new Error('missing alerts')
        var map = {}
        response.data.items.forEach(item => { map[item.deviceId] = item })
        this.alertCounts = map
        this.alertTotals = { personTotal: response.data.personTotal || 0, rateTotal: response.data.rateTotal || 0, preTotal: response.data.preTotal || 0, total: response.data.total }
        this.alertsReady = true
      }).catch(() => { this.alertsReady = false; this.alertLoadError = '异常统计暂时无法更新，请重新加载。' })
    },
}
