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
		drag: '',
		deviceId: ''
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
      if (keepPage !== true) this.currentPage = 1
      var _this = this
      var name = _this.searchText
      
      axios.post('restful/oldPerson/selectDetail',
        { page: this.currentPage, pageSize: this.pageSize,
          'name': name === '' ? '' : _this.searchText
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.personDetailList = response.data.oldPersonDetails
        _this.total = response.data.total ?? response.data.oldPersonDetails.length

      })
        .catch(function (error) {
          console.log(error)
        })
      
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
            message: '编辑老人信息失败，手表ID重复或没有手表设备'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
		addCustomerList(){
			var _this = this
			  
			  axios.post('restful/customer/select',
			    {
			      'customerName': ""
			    },
			    {
			      headers: {
			        'content-type': 'application/json'
			      },
			      withCredentials: true
			    }).then(function (response) {
			    _this.options = response.data.customers
			  })
			    .catch(function (error) {
			      console.log(error)
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
