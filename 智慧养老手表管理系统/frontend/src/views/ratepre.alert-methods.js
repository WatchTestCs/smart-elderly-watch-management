import axios from 'axios'

// Alert queries and resolution share the page's dialog and list state.
export const healthAlertMethods = {
    loadAlertCount () {
      var _this = this
      axios.post('api/v1/ratepre/alertCount', {}, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        _this.rateCount = response.data.rateTotal
        _this.preCount = response.data.preTotal
      }).catch(function (error) {
        console.log(error)
      })
    },
    // 点表格里的"查看原因"
    showAlertRow (row) {
      this.loadAlertDetail(null, row.name, row.oldId)
    },
    // 点顶部的"心率不合格/血压不合格"：找该类型最新一条未处理异常
    showAlertByKind (kind) {
      var _this = this
      var target = null
      this.oldsList.forEach(function (row) {
        var bad = kind === 'rate' ? row.rateState == 1 : row.preState == 1
        if (bad && !target) {
          target = row
        }
      })
      if (!target) {
        this.$message({ type: 'success', message: '当前没有未处理的' + (kind === 'rate' ? '心率' : '血压') + '异常' })
        return
      }
      this.loadAlertDetail(null, target.name, target.oldId, kind)
    },
    // 读取异常详情：优先用 recordId，其次按老人找最新未处理异常
    loadAlertDetail (recordId, name, oldId, kind) {
      var _this = this
      this.alertDialog = {
        visible: true, loading: true, resolving: false, hasAlert: 1,
        title: '异常原因与处理', recordId: recordId, oldPersonName: name || '',
        hr: '', bp: '', rateBad: 0, preBad: 0, reason: '', detail: '',
        occurTime: '', advice: '', message: ''
      }
      var params = recordId ? { recordId: recordId } : { oldId: oldId, kind: kind || '' }
      axios.post('api/v1/ratepre/alertDetail', params, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        var d = response.data
        _this.alertDialog.loading = false
        if (d.returnCode !== '1111') {
          _this.alertDialog.hasAlert = 0
          _this.alertDialog.message = d.message || '没有找到待处理的异常'
          return
        }
        _this.alertDialog.hasAlert = d.hasAlert === undefined ? 1 : d.hasAlert
        _this.alertDialog.message = d.message || ''
        _this.alertDialog.recordId = d.recordId === undefined ? null : d.recordId
        _this.alertDialog.oldPersonName = d.oldPersonName || _this.alertDialog.oldPersonName
        _this.alertDialog.hr = d.hr
        _this.alertDialog.bp = d.bp
        _this.alertDialog.rateBad = d.rateBad || 0
        _this.alertDialog.preBad = d.preBad || 0
        _this.alertDialog.reason = d.reason || ''
        _this.alertDialog.detail = d.detail || ''
        _this.alertDialog.occurTime = d.occurTime || ''
        _this.alertDialog.advice = d.advice || ''
      }).catch(function (error) {
        console.log(error)
        _this.alertDialog.loading = false
        _this.alertDialog.hasAlert = 0
        _this.alertDialog.message = '读取失败，请稍后重试'
      })
    },
    // 处置：消除红点
    resolveAlert (kind) {
      var _this = this
      this.alertDialog.resolving = true
      axios.post('api/v1/ratepre/resolveAlert',
        { recordId: this.alertDialog.recordId, kind: kind },
        {
          headers: { 'content-type': 'application/json' },
          withCredentials: true
        }).then(function (response) {
          _this.alertDialog.resolving = false
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已处理，红点已消除' })
            _this.alertDialog.visible = false
            _this.searchOlds()
            _this.getCount()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '处理失败' })
          }
        }).catch(function (error) {
          console.log(error)
          _this.alertDialog.resolving = false
          _this.$message({ type: 'error', message: '处理失败' })
        })
    },
    // 表格里直接点"标记已处理"
    resolveRow (row) {
      var _this = this
      var tasks = []
      if (row.rateState == 1) {
        tasks.push(axios.post('api/v1/ratepre/resolveAlert', { recordId: row.recordId, kind: 'rate' }, _this.postConfig()))
      }
      if (row.preState == 1) {
        tasks.push(axios.post('api/v1/ratepre/resolveAlert', { recordId: row.recordId, kind: 'pre' }, _this.postConfig()))
      }
      Promise.all(tasks).then(function () {
        _this.$message({ type: 'success', message: '已处理，红点已消除' })
        _this.searchOlds()
        _this.getCount()
      }).catch(function (error) {
        console.log(error)
        _this.$message({ type: 'error', message: '处理失败' })
      })
    },
    postConfig () {
      return {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }
    },
}
