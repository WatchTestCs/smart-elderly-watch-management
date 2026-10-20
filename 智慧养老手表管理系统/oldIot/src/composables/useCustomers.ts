import { ref, reactive } from 'vue'
import { accepted, type Envelope, type PostClient } from '../api/client'

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
export const validAge = (value: unknown) => value !== '' && value !== null && value !== undefined && Number.isInteger(Number(value)) && Number(value) >= 0 && Number(value) <= 150

export function useCustomers(client: PostClient) {
  const searchText = ref(''), customerDialogVisible = ref(false), oldEditDialogVisible = ref(false)
  const customerForm = reactive(emptyCustomer()), oldEditForm = reactive(emptyCustomer())
  const isLoading = ref(false), saving = ref(false), loadError = ref(''), mutationError = ref('')
  const total = ref(0), pageSize = ref(10), currentPage = ref(1), customerList = ref<Customer[]>([])
  async function searchCustomers(keepPage = false) {
    if (isLoading.value) return
    if (keepPage !== true) currentPage.value = 1
    isLoading.value = true; loadError.value = ''
    try {
      const { data } = await client.post<{ customers: Customer[]; total: number }>('restful/customer/select', { page: currentPage.value, pageSize: pageSize.value, customerName: searchText.value })
      if (!Array.isArray(data.customers) || !Number.isInteger(data.total) || data.total < 0) throw new Error('invalid list')
      customerList.value = data.customers; total.value = data.total
    } catch {
      customerList.value = []; total.value = 0; loadError.value = '监护人信息暂时无法加载，请检查服务连接后重试。'
    } finally { isLoading.value = false }
  }
  async function save(edit = false) {
    if (saving.value) return false
    const form = edit ? oldEditForm : customerForm
    if (!validPhone(form.customerPhone) || !validAge(form.customerAge) || !form.customerName.trim()) { mutationError.value = '请填写姓名、有效手机号与 0–150 岁的整数年龄'; return false }
    saving.value = true; mutationError.value = ''
    try {
      const { data } = await client.post<Envelope>('restful/customer/' + (edit ? 'update' : 'add'), { ...form })
      accepted(data)
      if (edit) oldEditDialogVisible.value = false
      else { customerDialogVisible.value = false; Object.assign(customerForm, emptyCustomer()) }
      await searchCustomers(true); return true
    } catch (error) { mutationError.value = error instanceof Error ? error.message : '操作未确认成功，请重试'; return false }
    finally { saving.value = false }
  }
  async function remove(id: number | string) {
    if (saving.value) return false
    saving.value = true; mutationError.value = ''
    try { const { data } = await client.post<Envelope>('restful/customer/delete', { customerId: id }); accepted(data); await searchCustomers(true); return true }
    catch { mutationError.value = '删除未确认成功，记录仍保留，请重新核实'; return false }
    finally { saving.value = false }
  }
  function handleEditRole(_index: unknown, row: Customer) { Object.assign(oldEditForm, emptyCustomer(), row); oldEditDialogVisible.value = true; mutationError.value = '' }
  function closeRoleAddDialog() { Object.assign(customerForm, emptyCustomer()); mutationError.value = '' }
  function handleSizeChange(value: number) { pageSize.value = value; currentPage.value = 1; return searchCustomers(true) }
  function changePage(value: number) { currentPage.value = value; return searchCustomers(true) }
  return { searchText, customerDialogVisible, oldEditDialogVisible, customerForm, oldEditForm, isLoading, saving, loadError, mutationError, total, pageSize, currentPage, customerList, searchCustomers, save, remove, handleEditRole, closeRoleAddDialog, handleSizeChange, changePage }
}
