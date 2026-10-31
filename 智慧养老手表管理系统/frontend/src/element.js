import { h } from 'vue'
import { ElAlert, ElBadge, ElButton, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElForm, ElFormItem, ElInput, ElInputNumber, ElMenu, ElMenuItem, ElOption, ElPagination, ElRadio, ElRadioGroup, ElSelect, ElSubMenu, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload, ElLoading, ElMessage, ElMessageBox } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { ElConfigProvider } from 'element-plus'

export default function installElement (app) {
  [ElAlert, ElBadge, ElButton, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElForm, ElFormItem, ElInput, ElInputNumber, ElMenu, ElMenuItem, ElOption, ElPagination, ElRadio, ElRadioGroup, ElSelect, ElSubMenu, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload].forEach(component => app.use(component))
  app.directive('loading', ElLoading.directive)
  app.component('CareConfig', { setup (_, { slots }) { return () => h(ElConfigProvider, { locale: zhCn }, slots) } });
  ["el-icon-arrow-left","el-icon-back","el-icon-check","el-icon-delete","el-icon-edit","el-icon-plus","el-icon-search","el-icon-warning"].forEach(name => app.component(name, { render: () => h('i', { class: name, 'aria-hidden': 'true' }) }))
  app.config.globalProperties.$message = ElMessage
  app.config.globalProperties.$alert = ElMessageBox.alert
  app.config.globalProperties.$confirm = ElMessageBox.confirm
}
