import { deviceListMethods } from './device.list-methods.js'
import { deviceAlertMethods } from './device.alert-methods.js'
import { deviceRefreshMethods } from './device.refresh-methods.js'
import { deviceAdminMethods } from './device.admin-methods.js'

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
    ...deviceListMethods,
    ...deviceAlertMethods,
    ...deviceRefreshMethods,
    ...deviceAdminMethods
  }
}
