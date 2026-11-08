import { apiErrorMessage } from '../api/httpContract'
import { ref, reactive, getCurrentScope, onScopeDispose } from 'vue'
import { accepted, type Envelope, type PostClient } from '../api/client'
import { createLatestRead } from './latestRead'

export interface Guardian {
  guardianId?: number | string
  guardianName: string
  guardianAge: number | string
  guardianSex: string
  guardianPhone: string
  guardianAddress: string
}
export const emptyGuardian = (): Guardian => ({ guardianName: '', guardianAge: '', guardianSex: '', guardianPhone: '', guardianAddress: '' })
export const validPhone = (value: unknown) => /^1\d{10}$/.test(String(value ?? ''))
export const validAge = (value: unknown) => {
  if (typeof value !== 'number' && typeof value !== 'string') return false
  if (typeof value === 'string' && !/^\d+$/.test(value.trim())) return false
  const age = Number(value)
  return Number.isInteger(age) && age >= 0 && age <= 150
}

export function useGuardians(client: PostClient) {
  const searchText = ref(''), guardianDialogVisible = ref(false), oldEditDialogVisible = ref(false)
  const guardianForm = reactive(emptyGuardian()), oldEditForm = reactive(emptyGuardian())
  const isLoading = ref(false), saving = ref(false), loadError = ref(''), mutationError = ref('')
  const total = ref(0), pageSize = ref(10), currentPage = ref(1), guardianList = ref<Guardian[]>([])
  type Query = { page: number; pageSize: number; guardianName: string }
  const reads = createLatestRead(
    (query: Query, signal: AbortSignal) => client.post<{ guardians: Guardian[]; total: number }>('api/v1/guardians/list', query, { signal }),
    ({ data }) => {
      if (!Array.isArray(data.guardians) || !Number.isInteger(data.total) || data.total < 0) throw new Error('invalid list')
      guardianList.value = data.guardians; total.value = data.total; loadError.value = ''
    },
    error => {
      guardianList.value = []; total.value = 0; loadError.value = apiErrorMessage(error, '监护人信息暂时无法加载，请检查服务连接后重试。')
    },
    value => { isLoading.value = value }
  )
  if (getCurrentScope()) onScopeDispose(reads.dispose)
  function searchGuardians(keepPage = false) {
    if (keepPage !== true) currentPage.value = 1
    loadError.value = ''
    return reads.run({ page: currentPage.value, pageSize: pageSize.value, guardianName: searchText.value })
  }
  async function save(edit = false) {
    if (saving.value) return false
    const form = edit ? oldEditForm : guardianForm
    if (!validPhone(form.guardianPhone) || !validAge(form.guardianAge) || !form.guardianName.trim()) { mutationError.value = '请填写姓名、有效手机号与 0–150 岁的整数年龄'; return false }
    saving.value = true; mutationError.value = ''
    try {
      const { data } = await client.post<Envelope>(edit ? 'api/v1/guardians/update' : 'api/v1/guardians/create', { ...form })
      accepted(data)
      if (edit) oldEditDialogVisible.value = false
      else { guardianDialogVisible.value = false; Object.assign(guardianForm, emptyGuardian()) }
      await searchGuardians(true); return true
    } catch (error) { mutationError.value = apiErrorMessage(error, '未收到保存确认，请核对记录'); return false }
    finally { saving.value = false }
  }
  async function remove(id: number | string) {
    if (saving.value) return false
    saving.value = true; mutationError.value = ''
    try { const { data } = await client.post<Envelope>('api/v1/guardians/delete', { guardianId: id }); accepted(data); await searchGuardians(true); return true }
    catch (error) { mutationError.value = apiErrorMessage(error, '未收到删除确认，请刷新记录'); return false }
    finally { saving.value = false }
  }
  function handleEditRole(_index: unknown, row: Guardian) { Object.assign(oldEditForm, emptyGuardian(), row); oldEditDialogVisible.value = true; mutationError.value = '' }
  function closeRoleAddDialog() { Object.assign(guardianForm, emptyGuardian()); mutationError.value = '' }
  function handleSizeChange(value: number) { pageSize.value = value; currentPage.value = 1; return searchGuardians(true) }
  function changePage(value: number) { currentPage.value = value; return searchGuardians(true) }
  return { searchText, guardianDialogVisible, oldEditDialogVisible, guardianForm, oldEditForm, isLoading, saving, loadError, mutationError, total, pageSize, currentPage, guardianList, searchGuardians, save, remove, handleEditRole, closeRoleAddDialog, handleSizeChange, changePage }
}
