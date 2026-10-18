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
      customerDialogVisible: false,
      oldEditDialogVisible: false,
      familyForm: {
        familyName: '',
        familyRelation: '',
		familySex: '',
		familyPhone: '',
		familyAddress: '',
		oldPhone: ''
      },
      familyEditForm: {},
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      familyList: [],

	  rules: {
			familyPhone: [
			  { validator: checkPhone, trigger: 'blur'}
			],
			oldPhone: [
			  { validator: checkPhone, trigger: 'blur'}
			]
		  }
    }
  },
  created () {
    this.searchfamily()
  },
  methods: {
    searchfamily (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
     var _this = this
       var familyName = _this.searchText
       
       axios.post('restful/family/select',
         { page: this.currentPage, pageSize: this.pageSize,
           'familyName': familyName === '' ? '' : _this.searchText
         },
         {
           headers: {
             'content-type': 'application/json'
           },
           withCredentials: true
         }).then(function (response) {
         _this.familyList = response.data.familyDtos
         _this.total = response.data.total ?? response.data.familyDtos.length

       })
         .catch(function (error) {
           console.log(error)
         })
      
    },
    addFamily () {
      var _this = this
      axios.post('restful/family/add',
        _this.familyForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        if (response.data.returnCode === '1111') {
          _this.customerDialogVisible = false
          _this.searchfamily()
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
      this.$refs.addfamilyFormRef.resetFields()
    },
    editFamily () {
      var _this = this
      axios.post('restful/family/updateFamily', _this.familyEditForm,
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
          _this.searchfamily()
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
    async deleteFamily (familyId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        axios.post('restful/family/deleteFamily', {'familyId': familyId},
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
          _this.searchfamily()
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
    handleeditFamily (index, row) {
      this.oldEditDialogVisible = true
      this.familyEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchfamily(true)
    },

    changePage (val) { this.currentPage = val; this.searchfamily(true)
    },


  }
}
