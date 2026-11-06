import { defineComponent, h, type App } from 'vue'
import {
  ElAlert, ElBadge, ElButton, ElConfigProvider, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu,
  ElForm, ElFormItem, ElInput, ElInputNumber, ElLoading, ElMenu, ElMenuItem, ElOption,
  ElPagination, ElRadio, ElRadioGroup, ElSelect, ElSubMenu, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload
} from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

const components = [
  ElAlert, ElBadge, ElButton, ElDatePicker, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElForm, ElFormItem,
  ElInput, ElInputNumber, ElMenu, ElMenuItem, ElOption, ElPagination, ElRadio, ElRadioGroup, ElSelect, ElSubMenu,
  ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElUpload
]

// 页面模板中以全局组件名使用的图标占位（样式见 assets/css/icons.css）。
const iconNames = [
  'el-icon-arrow-left', 'el-icon-back', 'el-icon-check', 'el-icon-delete',
  'el-icon-edit', 'el-icon-plus', 'el-icon-search', 'el-icon-warning'
]

const CareConfig = defineComponent({
  setup (_, { slots }) { return () => h(ElConfigProvider, { locale: zhCn }, slots) }
})

export default function installElement (app: App) {
  components.forEach(component => app.use(component))
  app.directive('loading', ElLoading.directive)
  app.component('CareConfig', CareConfig)
  iconNames.forEach(name => app.component(name, { render: () => h('i', { class: name, 'aria-hidden': 'true' }) }))
}
