import axios from 'axios'

// Resident writes use the page-owned drafts, form refs and refresh callbacks.
export const residentCommandMethods = {
    addOldPersons () {
      var _this = this
      // 提交前执行表单校验（手机号格式等），此前规则定义了但从未被调用
      this.$refs.addoldFormRef.validate(function (valid) {
        if (!valid) return
        var param = _this.oldForm
        axios.post('api/v1/oldPerson/add',
          param,
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.oldDialogVisible = false
            _this.searchOldPersons()
            _this.$message({
              type: 'success',
              message: '添加老人成功'
            })
          } else {
            _this.$message({
              type: 'error',
              message: '添加老人失败'
            })
          }
        })
          .catch(function (error) {
            console.log(error)
          })
      })
    },
    closeRoleAddDialog () {
      this.$refs.addoldFormRef.resetFields()
    },
    editOldPerson () {
      this.addCustomerList()
      var _this = this
      this.$refs.editoldFormRef.validate(function (valid) {
        if (!valid) return
        axios.post('api/v1/oldPerson/update', _this.oldEditForm,
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.oldEditDialogVisible = false
            _this.$message({
              type: 'success',
              message: '编辑老人信息成功'
            })
            _this.searchOldPersons()
          } else {
            _this.$message({
              type: 'error',
              message: '编辑老人信息失败，请检查填写内容后重试'
            })
          }
        })
          .catch(function (error) {
            console.log(error)
          })
      })
    },
    async deleteOldPerson (oldId) {
      try {
        await this.$confirm('此操作将永久删除该长者信息，是否继续？', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
      } catch (err) {
        this.$message({ type: 'info', message: '已取消删除' })
        return
      }

      try {
        const response = await axios.post('api/v1/oldPerson/delete', { oldId }, {
          headers: { 'content-type': 'application/json' },
          withCredentials: true
        })
        if (response.data.returnCode !== '1111') {
          this.$message({ type: 'error', message: '删除失败，请刷新列表后重试' })
          return
        }
        this.$message({ type: 'success', message: '删除信息成功' })
        this.searchOldPersons()
      } catch (error) {
        this.$message({
          type: 'error',
          message: error.response?.status === 409
            ? '请先解除该长者名下设备绑定，再删除长者'
            : '删除失败，请稍后重试'
        })
      }
    },
}
