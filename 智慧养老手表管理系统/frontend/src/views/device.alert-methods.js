export const deviceAlertMethods = {
    alertOf (deviceId) {
      return this.alertCounts[deviceId] || { personCount: 0, rateCount: 0, preCount: 0 }
    },
    // 点红点：查看问题原因与处理办法
    showAlert (row, kind) {
      var _this = this
      this.alertDialog = {
        visible: true, loading: true, resolving: false, hasAlert: 1,
        title: kind === 'rate' ? '心率不合格 · 原因与处理' : (kind === 'pre' ? '血压不合格 · 原因与处理' : '人员不合格 · 原因与处理'),
        kind: kind, deviceId: row.deviceId, deviceNo: row.deviceNo, alias: row.alias || '',
        oldPersonName: '', reason: '', detail: '', value: '', threshold: '',
        occurTime: '', advice: '', recordId: null, message: ''
      }
      this.post('alertDetail', { deviceId: row.deviceId, kind: kind }).then(function (response) {
        var d = response.data
        _this.alertDialog.loading = false
        _this.alertDialog.hasAlert = d.hasAlert === undefined ? 1 : d.hasAlert
        _this.alertDialog.message = d.message || ''
        _this.alertDialog.oldPersonName = d.oldPersonName || ''
        _this.alertDialog.reason = d.reason || ''
        _this.alertDialog.detail = d.detail || ''
        _this.alertDialog.value = d.value === undefined || d.value === null ? '' : d.value
        _this.alertDialog.threshold = d.threshold || ''
        _this.alertDialog.occurTime = d.occurTime || ''
        _this.alertDialog.advice = d.advice || ''
        _this.alertDialog.recordId = d.recordId === undefined ? null : d.recordId
      }).catch(function (error) {
        console.log(error)
        _this.alertDialog.loading = false
        _this.$message({ type: 'error', message: '读取异常详情失败' })
      })
    },
    // 消除红点
    resolveAlert () {
      var _this = this
      this.alertDialog.resolving = true
      this.post('resolveAlert', {
        deviceId: this.alertDialog.deviceId,
        kind: this.alertDialog.kind,
        recordId: this.alertDialog.recordId === null ? '' : this.alertDialog.recordId
      }).then(function (response) {
        _this.alertDialog.resolving = false
        if (response.data.returnCode === '1111') {
          _this.$message({ type: 'success', message: '已处理：' + (_this.alertDialog.reason || '异常') + '，红点已消除' })
          _this.alertDialog.visible = false
          _this.loadDevices()
        } else {
          _this.$message({ type: 'error', message: response.data.message || '处理失败' })
        }
      }).catch(function (error) {
        console.log(error)
        _this.alertDialog.resolving = false
        _this.$message({ type: 'error', message: '处理失败' })
      })
    },
    // 工作台里的"已处理"
    resolveAlertRow (row) {
      var _this = this
      this.post('resolveAlert', { deviceId: row.deviceId, kind: row.kind, recordId: row.recordId || '' })
        .then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已处理，红点已消除' })
            _this.loadDevices()
            _this.openWorkbench()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '处理失败' })
          }
        }).catch(function (error) {
          console.log(error)
        })
    },
    // 打开"全部待处理异常"
    openWorkbench () {
      var _this = this
      this.workbench.visible = true
      var list = []
      var kinds = ['person', 'rate', 'pre']
      var devices = this.deviceList.slice()
      var tasks = []
      devices.forEach(function (device) {
        kinds.forEach(function (kind) {
          var counts = _this.alertOf(device.deviceId)
          var count = kind === 'person' ? counts.personCount : (kind === 'rate' ? counts.rateCount : counts.preCount)
          if (count > 0) {
            tasks.push(_this.post('alertDetail', { deviceId: device.deviceId, kind: kind }).then(function (response) {
              var d = response.data
              if (d.hasAlert === 1) {
                list.push({
                  deviceId: device.deviceId,
                  deviceNo: d.deviceNo,
                  alias: d.alias,
                  oldPersonName: d.oldPersonName,
                  kind: kind,
                  kindText: d.kindText,
                  reason: d.reason,
                  recordId: d.recordId
                })
              }
            }))
          }
        })
      })
      Promise.all(tasks).then(function () {
        _this.workbench.list = list
      })
    },
    // 跳到相关页面看原始数据
    gotoPage (kind) {
      if (kind === 'person') {
        this.$router.push({ path: '/watch' })
      } else {
        this.$router.push({ path: '/ratepre' })
      }
      this.alertDialog.visible = false
    },
}
