/// <reference types="vite/client" />

import type { ElMessage, ElMessageBox } from 'element-plus'

declare module 'vue' {
  interface ComponentCustomProperties {
    $message: typeof ElMessage
    $alert: typeof ElMessageBox.alert
    $confirm: typeof ElMessageBox.confirm
  }
}
