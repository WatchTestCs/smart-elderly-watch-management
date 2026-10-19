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
    return {
      searchText: '', // 角色查询内容
      oldDialogVisible: false,
      oldEditDialogVisible: false,
      oldForm: {
        name: '',
        bothdate: '',
		sex: '',
		phone: '',
		room: '',

		beTime: '',
		customerId: '',
		illness: '',
		warn: '',
		foodLike: '',
		foodHate: '',
		drag: ''
      },
      oldEditForm: {
       		name: '',
       		bothdate: '',
			oldId: '',
       sex: '',
       phone: '',
       room: '',

       beTime: '',
       customerId: '',
			 customerName: '',
			 illness: '',
			 warn: '',
			 foodLike: '',
			 foodHate: '',
			 drag: '',
			 deviceId: ''
      },
			options:[],
      customerLoading: false,
      customerError: '',
      
      isLoading: false,
      loadError: '',
      total: 0,
      pageSize: 10,
      currentPage: 1,
      personDetailList: [],

	  rules: {
	  			phone: [
	  			  { validator: checkPhone, trigger: 'blur'}
	  			]
	  		  }
    }
  },
  created () {
    this.searchOldPersons()
		this.addCustomerList()
  },
  methods: {
    searchOldPersons (keepPage = false) {
      if (this.isLoading) return
      this.isLoading = true
      this.loadError = ''
      if (keepPage !== true) this.currentPage = 1
      var _this = this
      var name = _this.searchText
      
      return axios.post('restful/oldPerson/selectDetail',
        { page: this.currentPage, pageSize: this.pageSize,
          'name': name === '' ? '' : _this.searchText
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        if (!Array.isArray(response.data.oldPersonDetails)) throw new Error('invalid records')
        _this.personDetailList = response.data.oldPersonDetails
        _this.total = response.data.total ?? response.data.oldPersonDetails.length

      })
        .catch(function () {
          _this.loadError = '老人档案暂时无法加载，请检查服务连接后重试。'
          _this.personDetailList = []
          _this.total = 0
        }).finally(function () { _this.isLoading = false })
      
    },
    addOldPersons () {
      var _this = this
	  var param = _this.oldForm
      axios.post('restful/oldPerson/add',
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
    },
    closeRoleAddDialog () {
      this.$refs.addoldFormRef.resetFields()
    },
    editOldPerson () {
			this.addCustomerList()
      var _this = this
      axios.post('restful/oldPerson/update', _this.oldEditForm,
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
    },
    async addCustomerList () {
      if (this.customerLoading) return
      this.customerLoading = true
      this.customerError = ''
      try {
        var options = [], page = 1, total = 0
        do {
          var response = await axios.post('restful/customer/select', { customerName: '', page, pageSize: 100 }, { withCredentials: true })
          var items = response.data.customers
          if (!Array.isArray(items)) throw new Error('invalid guardians')
          total = response.data.total ?? items.length
          if (!Number.isInteger(total) || total < 0 || (!items.length && options.length < total)) throw new Error('incomplete guardians')
          options.push(...items)
          page += 1
        } while (options.length < total)
        this.options = options
      } catch (_) {
        this.options = []
        this.customerError = '监护人选项暂时无法加载，请重试。'
      } finally { this.customerLoading = false }
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
        const response = await axios.post('restful/oldPerson/delete', { oldId }, {
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
    handleeditOldPerson (index, row) {
      this.oldEditDialogVisible = true
      this.oldEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchOldPersons(true)
    },

    changePage (val) { this.currentPage = val; this.searchOldPersons(true)
    },


  }
}
