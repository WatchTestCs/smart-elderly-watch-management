import { deleteRecord, writeRecord } from '../composables/recordActions'

const residentFields = ['name', 'bothdate', 'sex', 'phone', 'room', 'illness', 'warn', 'foodLike', 'foodHate', 'drag', 'customerId']
const editable = draft => Object.fromEntries(residentFields.map(key => [key, draft[key]]))

export const residentCommandMethods = {
  addOldPersons () {
    this.$refs.addoldFormRef.validate(valid => {
      if (valid) return writeRecord(this, 'api/v1/residents/create', editable(this.oldForm), '添加老人成功', () => {
        this.oldDialogVisible = false
        this.searchOldPersons()
      })
    })
  },
  closeRoleAddDialog () { this.$refs.addoldFormRef.resetFields() },
  editOldPerson () {
    this.$refs.editoldFormRef.validate(valid => {
      if (!valid) return
      const draft = this.oldEditForm
      return writeRecord(this, 'api/v1/residents/update', { ...editable(draft), oldId: draft.oldId }, '编辑老人信息成功', () => {
        this.oldEditDialogVisible = false
        this.searchOldPersons(true)
      })
    })
  },
  deleteOldPerson (oldId) {
    return deleteRecord(this, 'api/v1/residents/delete', { oldId }, '此操作将永久删除该长者信息，是否继续？', () => this.searchOldPersons(true))
  }
}
