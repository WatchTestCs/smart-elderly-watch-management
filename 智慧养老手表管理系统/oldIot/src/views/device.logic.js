import axios from 'axios'

export default {
  data () {
    return {
      deviceList: [], deviceTotal: 0, serverOnlineCount: 0, currentPage: 1, pageSize: 20, filterTimer: null, reloadPending: false,
      deviceDetailVisible: false, detailDevice: null,
      deviceQuery: '', deviceState: 'all', deviceLoading: false, deviceLoadError: '', alertLoadError: '', alertsReady: false, creating: false,
      oldPeople: [],
      autoRefresh: true,
      refreshDisposed: false,
      timer: null,
      createDialogVisible: false,
      bindCodeDialogVisible: false,
      createdDevice: null,
      currentDevice: null,
      bindCodeList: [],
      newBindCode: null,
      bindOldId: '',
      bindExpireDays: '1',
      createForm: {
        alias: '',
        model: 'HarmonyOS-Wear',
        oldId: ''
      },
      // 红点：各设备未处理异常的条数
      alertCounts: {},
      alertTotals: { personTotal: 0, rateTotal: 0, preTotal: 0, total: 0 },
      alertDialog: {
        visible: false,
        loading: false,
        resolving: false,
        hasAlert: 1,
        title: '异常原因',
        kind: 'person',
        deviceId: null,
        deviceNo: '',
        alias: '',
        oldPersonName: '',
        reason: '',
        detail: '',
        value: '',
        threshold: '',
        occurTime: '',
        advice: '',
        recordId: null,
        message: ''
      },
      workbench: {
        visible: false,
        list: []
      }
    }
  },
  computed: {
    visibleDevices () { return this.deviceList },
    onlineCount () { return this.serverOnlineCount }
  },
  created () {
    this.loadDevices()
    this.loadOldPeople()
  },
  mounted () {
    document.addEventListener('visibilitychange', this.handleVisibilityChange)
    this.startTimer()
  },
  beforeUnmount () {
    this.refreshDisposed = true
    clearTimeout(this.filterTimer)
    document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    this.stopTimer()
  },
  watch: {
    deviceQuery () { this.scheduleFilter() },
    deviceState () { this.scheduleFilter() },
    autoRefresh (enabled) { if (enabled) this.startTimer(); else this.stopTimer() }
  },
  methods: {
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
    loadOldPeople () {
      var _this = this
      axios.post('restful/inviteCode/oldPersonList', {}, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        _this.oldPeople = response.data.oldPeople || []
      }).catch(function (error) {
        console.log(error)
      })
    },
    startTimer () {
      this.stopTimer()
      if (!this.autoRefresh || document.hidden || this.refreshDisposed) return
      this.timer = setTimeout(() => {
        this.timer = null
        Promise.resolve(this.loadDevices()).then(() => { this.startTimer() })
      }, 5000)
    },
    handleVisibilityChange () {
      this.stopTimer()
      if (!document.hidden && this.autoRefresh) Promise.resolve(this.loadDevices()).then(() => { this.startTimer() })
    },
    stopTimer () {
      if (this.timer) {
        clearTimeout(this.timer)
        this.timer = null
      }
    },
    createDevice () {
      if (this.creating) return
      this.creating = true
      var _this = this
      var params = {
        alias: this.createForm.alias || '新手表',
        model: this.createForm.model || 'HarmonyOS-Wear',
        oldId: this.createForm.oldId === '' ? '' : this.createForm.oldId
      }
      this.post('create', params).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.createdDevice = response.data
          _this.$message({ type: 'success', message: '设备登记成功' })
          _this.loadDevices()
        } else {
          _this.$message({ type: 'error', message: response.data.message || '登记失败' })
        }
      }).catch(function (error) {
        console.log(error)
        _this.$message({ type: 'error', message: '登记失败' })
      }).then(function () { _this.creating = false })
    },
    closeCreateDialog () {
      this.createDialogVisible = false
      this.createdDevice = null
      this.createForm = { alias: '', model: 'HarmonyOS-Wear', oldId: '' }
    },
    openBindCodeDialog (row) {
      this.currentDevice = row
      this.bindOldId = row.oldId || ''
      this.bindExpireDays = '1'
      this.newBindCode = null
      this.bindCodeDialogVisible = true
      this.loadBindCodes(row.oldId)
    },
    loadBindCodes (oldId) {
      var _this = this
      if (!oldId) {
        this.bindCodeList = []
        return
      }
      this.post('bindCodeList', { oldId: oldId }).then(function (response) {
        _this.bindCodeList = response.data.bindCodes || []
      }).catch(function (error) {
        console.log(error)
      })
    },
    createBindCode () {
      var _this = this
      if (!this.bindOldId) {
        this.$message({ type: 'warning', message: '请先选择要绑定的老人' })
        return
      }
      if (!/^\d+$/.test(this.bindExpireDays)) {
        this.$message({ type: 'warning', message: '有效天数只能填数字' })
        return
      }
      this.post('createBindCode', {
        oldId: this.bindOldId,
        expireDays: this.bindExpireDays,
        createUser: localStorage.getItem('user_name') || '管理员'
      }).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.newBindCode = response.data
          _this.loadBindCodes(_this.bindOldId)
        } else {
          _this.$message({ type: 'error', message: response.data.message || '生成失败' })
        }
      }).catch(function (error) {
        console.log(error)
      })
    },
    deleteBindCode (row) {
      var _this = this
      this.post('deleteBindCode', { id: row.id }).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.$message({ type: 'success', message: '已作废' })
          _this.loadBindCodes(_this.bindOldId)
        } else {
          _this.$message({ type: 'error', message: response.data.message || '作废失败' })
        }
      }).catch(function (error) {
        console.log(error)
      })
    },
    resetSecret (row) {
      var _this = this
      this.$confirm('重置后手表端需要用新密钥重新登录，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('resetSecret', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$alert('新密钥：' + response.data.secret, '密钥已重置', { confirmButtonText: '知道了' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '重置失败' })
          }
        })
      }).catch(function () {})
    },
    unbind (row) {
      var _this = this
      this.$confirm('解绑后该手表将不再上报到这位老人，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('unbind', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已解绑' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '解绑失败' })
          }
        })
      }).catch(function () {})
    },
    deleteDevice (row) {
      var _this = this
      this.$confirm('删除后需要重新登记设备，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('delete', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已删除' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '删除失败' })
          }
        })
      }).catch(function () {})
    },
    copy (text) {
      var input = document.createElement('input')
      input.value = text
      document.body.appendChild(input)
      input.select()
      try {
        document.execCommand('copy')
        this.$message({ type: 'success', message: '已复制：' + text })
      } catch (e) {
        this.$message({ type: 'warning', message: '复制失败，请手动记录：' + text })
      }
      document.body.removeChild(input)
    }
  }
}
