import { apiErrorMessage } from '../api/httpContract'
import { ref, reactive, getCurrentScope, onScopeDispose } from 'vue'
import { accepted, type Envelope, type PostClient } from '../api/client'
import { createLatestRead } from './latestRead'

export interface Customer {
  customerId?: number | string
  customerName: string
  customerAge: number | string
  customerSex: string
  customerPhone: string
  customerAddress: string
}
export const emptyCustomer = (): Customer => ({ customerName: '', customerAge: '', customerSex: '', customerPhone: '', customerAddress: '' })
export const validPhone = (value: unknown) => /^1\d{10}$/.test(String(value ?? ''))
export const validAge = (value: unknown) => {
  if (typeof value !== 'number' && typeof value !== 'string') return false
  if (typeof value === 'string' && !/^\d+$/.test(value.trim())) return false
  const age = Number(value)
  return Number.isInteger(age) && age >= 0 && age <= 150
}

export function useCustomers(client: PostClient) {
  const searchText = ref(''), customerDialogVisible = ref(false), oldEditDialogVisible = ref(false)
  const customerForm = reactive(emptyCustomer()), oldEditForm = reactive(emptyCustomer())
  const isLoading = ref(false), saving = ref(false), loadError = ref(''), mutationError = ref('')
  const total = ref(0), pageSize = ref(10), currentPage = ref(1), customerList = ref<Customer[]>([])
  type Query = { page: number; pageSize: number; customerName: string }
  const reads = createLatestRead(
    (query: Query, signal: AbortSignal) => client.post<{ customers: Customer[]; total: number }>('api/v1/guardians/list', query, { signal }),
    ({ data }) => {
      if (!Array.isArray(data.customers) || !Number.isInteger(data.total) || data.total < 0) throw new Error('invalid list')
      customerList.value = data.customers; total.value = data.total; loadError.value = ''
    },
    error => {
      customerList.value = []; total.value = 0; loadError.value = apiErrorMessage(error, '监护人信息暂时无法加载，请检查服务连接后重试。')
    },
    value => { isLoading.value = value }
  )
  if (getCurrentScope()) onScopeDispose(reads.dispose)
  function searchCustomers(keepPage = false) {
    if (keepPage !== true) currentPage.value = 1
    loadError.value = ''
    return reads.run({ page: currentPage.value, pageSize: pageSize.value, customerName: searchText.value })
  }
  async function save(edit = false) {
    if (saving.value) return false
    const form = edit ? oldEditForm : customerForm
    if (!validPhone(form.customerPhone) || !validAge(form.customerAge) || !form.customerName.trim()) { mutationError.value = '请填写姓名、有效手机号与 0–150 岁的整数年龄'; return false }
    saving.value = true; mutationError.value = ''
    try {
      const { data } = await client.post<Envelope>(edit ? 'api/v1/guardians/update' : 'api/v1/guardians/create', { ...form })
      accepted(data)
      if (edit) oldEditDialogVisible.value = false
      else { customerDialogVisible.value = false; Object.assign(customerForm, emptyCustomer()) }
      await searchCustomers(true); return true
    } catch (error) { mutationError.value = apiErrorMessage(error, '未收到保存确认，请核对记录'); return false }
    finally { saving.value = false }
  }
  async function remove(id: number | string) {
    if (saving.value) return false
    saving.value = true; mutationError.value = ''
    try { const { data } = await client.post<Envelope>('api/v1/guardians/delete', { customerId: id }); accepted(data); await searchCustomers(true); return true }
    catch (error) { mutationError.value = apiErrorMessage(error, '未收到删除确认，请刷新记录'); return false }
    finally { saving.value = false }
  }
  function handleEditRole(_index: unknown, row: Customer) { Object.assign(oldEditForm, emptyCustomer(), row); oldEditDialogVisible.value = true; mutationError.value = '' }
  function closeRoleAddDialog() { Object.assign(customerForm, emptyCustomer()); mutationError.value = '' }
  function handleSizeChange(value: number) { pageSize.value = value; currentPage.value = 1; return searchCustomers(true) }
  function changePage(value: number) { currentPage.value = value; return searchCustomers(true) }
  return { searchText, customerDialogVisible, oldEditDialogVisible, customerForm, oldEditForm, isLoading, saving, loadError, mutationError, total, pageSize, currentPage, customerList, searchCustomers, save, remove, handleEditRole, closeRoleAddDialog, handleSizeChange, changePage }
}
