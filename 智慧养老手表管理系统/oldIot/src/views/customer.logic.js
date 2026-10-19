import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'

export default {
  components: { RecordCards },
  data () {
		var checkPhone = (rule, value, callback) => {
		  const phoneReg = /^1[3|4|5|7|8][0-9]{9}$/
		  if (!value) {
		    return callback(new Error('电话号码不能为空'))
		  }
		  setTimeout(() => {
		
		    if (!Number.isInteger(+value)) {
		      callback(new Error('请输入数字值'))
		    } else {
		      if (phoneReg.test(value)) {
		        callback()
		      } else {
		        callback(new Error('电话号码格式不正确'))
		      }
		    }
		  }, 100)
		}
       var checkAge = (rule, value, callback) => {
        if (!value) {
          return callback(new Error('年龄不能为空'));
        }
        setTimeout(() => {
          if (!Number.isInteger(+value)) {
            callback(new Error('请输入数字值'));
          } else {
            if (+value < 0) {
              callback(new Error('必须大于零'));
            } else if(+value > 150){
							callback(new Error('必须小于150'));
						} else{
              callback();
            }
          }
        }, 1000);
      }
    return {
      searchText: '', // 角色查询内容
      customerDialogVisible: false,
      oldEditDialogVisible: false,
      customerForm: {
        customerName: '',
        customerAge: '',
				customerSex: '',
				customerPhone: '',
				customerAddress: ''
      },
      oldEditForm: {
        customerName: '',
        customerAge: '',
        customerSex: '',
        customerPhone: '',
        customerAddress: '',
				customerId: ''
      },
      
      isLoading: false,
      loadError: '',
      total: 0,
      pageSize: 10,
      currentPage: 1,
      customerList: [],

			rules: {
						customerPhone: [
						  { validator: checkPhone, trigger: 'blur'}
						],
						customerAge: [
            { validator: checkAge, trigger: 'blur' }
          ]
					  }
    }
  },
  created () {
    this.searchCustomers()
  },
  methods: {
    searchCustomers (keepPage = false) {
      if (this.isLoading) return
      this.isLoading = true
      this.loadError = ''
      if (keepPage !== true) this.currentPage = 1
				var _this = this
				var customerName = _this.searchText
       
       return axios.post('restful/customer/select',
         { page: this.currentPage, pageSize: this.pageSize,
           'customerName': customerName === '' ? '' : _this.searchText
         },
         {
           headers: {
             'content-type': 'application/json'
           },
           withCredentials: true
         }).then(function (response) {
         if (!Array.isArray(response.data.customers)) throw new Error('invalid records')
         _this.customerList = response.data.customers
         _this.total = response.data.total ?? response.data.customers.length

       })
         .catch(function () {
           _this.loadError = '监护人信息暂时无法加载，请检查服务连接后重试。'
           _this.customerList = []
           _this.total = 0
         }).finally(function () { _this.isLoading = false })
      
    },
    addRoles () {
      var _this = this
      axios.post('restful/customer/add',
        _this.customerForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        if (response.data.returnCode === '1111') {
          _this.customerDialogVisible = false
          _this.searchCustomers()
          _this.$message({
            type: 'success',
            message: '添加监护人成功'
          })
        } else {
          _this.$message({
            type: 'error',
            message: '添加监护人失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    closeRoleAddDialog () {
      this.$refs.addcustomerFormRef.resetFields()
    },
    editRole () {
      var _this = this
      axios.post('restful/customer/update', _this.oldEditForm,
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
            message: '编辑监护人信息成功'
          })
          _this.searchCustomers()
        } else {
          _this.$message({
            type: 'error',
            message: '编辑监护人信息失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    async deleteRole (customerId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        axios.post('restful/customer/delete', {'customerId': customerId},
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.$message({
              type: 'success',
              message: '删除信息成功'
            })
          }
          _this.searchCustomers()
        })
          .catch(function (error) {
            console.log(error)
          })
      } catch (err) {
        this.$message({
          type: 'info',
          message: '已取消删除'
        })
      }
    },
    handleEditRole (index, row) {
      this.oldEditDialogVisible = true
      this.oldEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchCustomers(true)
    },

    changePage (val) { this.currentPage = val; this.searchCustomers(true)
    },


  }
}
