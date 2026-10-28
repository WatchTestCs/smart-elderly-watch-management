export const deviceAdminMethods = {
    createDevice () {
      if (this.creating) return
      this.creating = true
      var _this = this
      var params = {
        alias: this.createForm.alias || '新手表',
        model: this.createForm.model || 'HarmonyOS-Wear',
        oldId: this.createForm.oldId === '' ? '' : this.createForm.oldId
      }
      this.post('create', params).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.createdDevice = response.data
          _this.$message({ type: 'success', message: '设备登记成功' })
          _this.loadDevices()
        } else {
          _this.$message({ type: 'error', message: response.data.message || '登记失败' })
        }
      }).catch(function (error) {
        console.log(error)
        _this.$message({ type: 'error', message: '登记失败' })
      }).then(function () { _this.creating = false })
    },
    closeCreateDialog () {
      this.createDialogVisible = false
      this.createdDevice = null
      this.createForm = { alias: '', model: 'HarmonyOS-Wear', oldId: '' }
    },
    openBindCodeDialog (row) {
      this.currentDevice = row
      this.bindOldId = row.oldId || ''
      this.bindExpireDays = '1'
      this.newBindCode = null
      this.bindCodeDialogVisible = true
      this.loadBindCodes(row.oldId)
    },
    loadBindCodes (oldId) {
      var _this = this
      if (!oldId) {
        this.bindCodeList = []
        return
      }
      this.post('bindCodeList', { oldId: oldId }).then(function (response) {
        _this.bindCodeList = response.data.bindCodes || []
      }).catch(function (error) {
        console.log(error)
      })
    },
    createBindCode () {
      var _this = this
      if (!this.bindOldId) {
        this.$message({ type: 'warning', message: '请先选择要绑定的老人' })
        return
      }
      if (!/^\d+$/.test(this.bindExpireDays)) {
        this.$message({ type: 'warning', message: '有效天数只能填数字' })
        return
      }
      this.post('createBindCode', {
        oldId: this.bindOldId,
        expireDays: this.bindExpireDays,
        createUser: localStorage.getItem('user_name') || '管理员'
      }).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.newBindCode = response.data
          _this.loadBindCodes(_this.bindOldId)
        } else {
          _this.$message({ type: 'error', message: response.data.message || '生成失败' })
        }
      }).catch(function (error) {
        console.log(error)
      })
    },
    deleteBindCode (row) {
      var _this = this
      this.post('deleteBindCode', { id: row.id }).then(function (response) {
        if (response.data.returnCode === '1111') {
          _this.$message({ type: 'success', message: '已作废' })
          _this.loadBindCodes(_this.bindOldId)
        } else {
          _this.$message({ type: 'error', message: response.data.message || '作废失败' })
        }
      }).catch(function (error) {
        console.log(error)
      })
    },
    resetSecret (row) {
      var _this = this
      this.$confirm('重置后手表端需要用新密钥重新登录，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('resetSecret', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$alert('新密钥：' + response.data.secret, '密钥已重置', { confirmButtonText: '知道了' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '重置失败' })
          }
        })
      }).catch(function () {})
    },
    unbind (row) {
      var _this = this
      this.$confirm('解绑后该手表将不再上报到这位老人，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('unbind', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已解绑' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '解绑失败' })
          }
        })
      }).catch(function () {})
    },
    deleteDevice (row) {
      var _this = this
      this.$confirm('删除后需要重新登记设备，是否继续？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        _this.post('delete', { deviceId: row.deviceId }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已删除' })
            _this.loadDevices()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '删除失败' })
          }
        })
      }).catch(function () {})
    },
    copy (text) {
      var input = document.createElement('input')
      input.value = text
      document.body.appendChild(input)
      input.select()
      try {
        document.execCommand('copy')
        this.$message({ type: 'success', message: '已复制：' + text })
      } catch (e) {
        this.$message({ type: 'warning', message: '复制失败，请手动记录：' + text })
      }
      document.body.removeChild(input)
    }
}
