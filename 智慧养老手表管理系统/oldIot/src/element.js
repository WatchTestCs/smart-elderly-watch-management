import { h } from 'vue'
import { ElAlert, ElBadge, ElButton, ElCol, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElForm, ElFormItem, ElInput, ElInputNumber, ElMenu, ElMenuItem, ElOption, ElPagination, ElRadio, ElRadioGroup, ElRow, ElSelect, ElSubMenu, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload, ElLoading, ElMessage, ElMessageBox } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { ElConfigProvider } from 'element-plus'

export default function installElement (app) {
  [ElAlert, ElBadge, ElButton, ElCol, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElForm, ElFormItem, ElInput, ElInputNumber, ElMenu, ElMenuItem, ElOption, ElPagination, ElRadio, ElRadioGroup, ElRow, ElSelect, ElSubMenu, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload].forEach(component => app.use(component))
  app.directive('loading', ElLoading.directive)
  app.component('CareConfig', { setup (_, { slots }) { return () => h(ElConfigProvider, { locale: zhCn }, slots) } });
  ["el-icon-menu","el-icon-document","el-icon-time","el-icon-location-outline","el-icon-plus","el-icon-arrow-up","el-icon-arrow-down","el-icon-service","el-icon-message","el-icon-search","el-icon-edit","el-icon-delete","el-icon-success","el-icon-arrow-right","el-icon-refresh","el-icon-info","el-icon-mobile-phone","el-icon-setting","el-icon-view","el-icon-warning","el-icon-check","el-icon-back","el-icon-arrow-left","el-icon-upload"].forEach(name => app.component(name, { render: () => h('i', { class: name, 'aria-hidden': 'true' }) }))
  app.config.globalProperties.$message = ElMessage
  app.config.globalProperties.$alert = ElMessageBox.alert
  app.config.globalProperties.$confirm = ElMessageBox.confirm
}
