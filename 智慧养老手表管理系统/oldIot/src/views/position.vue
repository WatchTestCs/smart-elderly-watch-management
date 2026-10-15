<template>
  <div class="roles">
    
    <div class="sys-research">
      <el-row class="row-search">
		  
        <el-col :span="8">
          
          <label class="table-search-label" for="search-position">长者姓名</label>
          <el-input id="search-position"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              @click="searchposition"></el-button></template>
          </el-input>
        </el-col>
        
				<el-badge :value="this.num" class="item">
				  <el-button size="small">摔倒</el-button>
				</el-badge>
				
      </el-row>
    </div>
    
    <el-dialog
      title="角色添加"
      v-model="roleDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="roleForm"  ref="addRoleFormRef">
        <el-form-item label="角色名称" prop="roleName">
          <el-input v-model="roleForm.roleName"></el-input>
        </el-form-item>
        <el-form-item label="角色描述" prop="roleDesc">
          <el-input v-model="roleForm.roleDesc"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="roleDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addRoles('addRoleFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-dialog
      title="角色编辑"
      v-model="roleEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :model="roleEditForm">
        <el-form-item label="角色名称" prop="roleName">
          <el-input v-model="roleEditForm.roleName"></el-input>
        </el-form-item>
        <el-form-item label="角色描述" prop="roleDesc">
          <el-input v-model="roleEditForm.roleDesc"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="roleEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editRole">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>
    <el-table class="adaptive-record-table" v-loading="isLoading" :data="positionList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
    style="width: 100%" border>
    <template #empty><div  class="position-empty">{{ loadError ? '位置记录暂时无法加载' : '暂无位置记录' }}</div></template>
    
      
      <el-table-column  width="100" type="index" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="140" align="center"></el-table-column>
      <el-table-column prop="sex" label="性别" width="100" align="center"></el-table-column>
	  <el-table-column prop="status" label="是否摔倒" width="150" align="center">
		  <template #default="scope" >
		    <el-tag v-if="scope.row.status == 1" type="danger" size="small">摔倒告警</el-tag>
		    <el-tag v-else-if="scope.row.status == 0" type="success" size="small">未发现摔倒告警</el-tag>
        <span v-else class="unknown-state">状态未知</span>
		  </template>
	  </el-table-column>
	  <el-table-column prop="lng" label="经度" width="160" align="center"></el-table-column>
	  <el-table-column prop="lat" label="纬度" width="160" align="center"></el-table-column>
	  <el-table-column prop="beTime" label="时间" width="auto" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
      	<template #default="scope" >
      		<el-button v-if="scope.row.status==1" size="small" plain type="danger" icon="el-icon-warning"
      		           @click="showFallReason(scope.row)">摔倒原因与处理</el-button>
      		<el-button v-if="scope.row.status==1" size="small" plain type="primary" icon="el-icon-check"
      		           @click="handleFall(scope.row)">标记已处理</el-button>
      		<el-button size="small" plain type="success" icon="el-icon-check" @click="showRoleDialog(scope.row)">
      			详细情况
      		</el-button>
      	</template>
      </el-table-column>
    </el-table>
    <record-cards :records="positionList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"sex","label":"性别"},{"key":"status","label":"是否摔倒","state":"fall"},{"key":"lng","label":"经度"},{"key":"lat","label":"纬度"}]' label="位置与告警记录" title-key="name" key-field="oldId" time-key="beTime">
      <template #actions="scope"  ><el-button v-if="scope.row.status==1" size="small" plain type="danger" icon="el-icon-warning"
      		           @click="showFallReason(scope.row)">摔倒原因与处理</el-button>
<el-button v-if="scope.row.status==1" size="small" plain type="primary" icon="el-icon-check"
      		           @click="handleFall(scope.row)">标记已处理</el-button>
<el-button size="small" plain type="success" icon="el-icon-check" @click="showRoleDialog(scope.row)">
      			详细情况
      		</el-button></template>
    </record-cards>
    
    <div style="margin-top: 10px;text-align: center">
      <el-pagination
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total">
      </el-pagination>
    </div>
    
    
    <el-dialog title="角色授权" v-model="isShowRoleDialog">
      
      
	  
	  
	  <div id="myChart" ref="mychart" :style="{width: '300px', height: '300px'}"></div>
		
      <template #footer><div  class="dialog-footer">
        <el-button @click="isShowRoleDialog = false">取 消</el-button>
        <el-button type="primary" @click="assignRights">确 定</el-button>
      </div></template>
    </el-dialog>
	
	<section class="map-panel">
    <div class="map-heading"><div><h2>位置地图</h2><p>选择记录中的“详细情况”查看对应坐标。</p></div></div>
    <div v-if="mapAvailable" id="allmap"></div>
    <div v-else class="map-unavailable"><i class="el-icon-location-outline"></i><strong>地图暂不可用</strong><span>请检查地图 SDK 配置与网络连接；原始经纬度仍可在上方记录中查看。</span></div>
	</section>
  </div>
	
</template>

<script>
// 引入
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'


export default {
  components: { RecordCards },
  data () {
    return {
      searchText: '', // 角色查询内容
      roleDialogVisible: false,
      roleEditDialogVisible: false,
      roleForm: {
        roleName: '',
        roleDesc: ''
      },
			value6: '',
      roleEditForm: {
        roleId: '',
        roleName: '',
        roleDesc: ''
      },
      popoverRole: true,
      total: 0,
      pageSize: 10,
      currentPage: 1,
			positionList: [],
			isLoading: false,
			loadError: '',
			mapAvailable: false,
			mapInstance: null,
      // 控制分配权限对话框的展示和隐藏
      isShowRoleDialog: false,

      // 树形控件的数据
      rightsTree: [],

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      defaultProps: {
        // children: 'children',
        label: 'authName'
      },
      // 当前被分配权限的角色id
      roleId: -1,
			lng: '',
			lat: '',
			num:''
    }
  },
  created () {
	  
	  // let myChart = this.$echarts.init(document.getElementById('myChart'))
    this.searchposition()
		
	// this.drawLine()
  },
  updated() {
	  
	  // this.drawLine()
     // this.getAttendanceData();
  },
  mounted(){
	  this.ready()
		// this.theLocation()
    // this.drawLine()
  },
  methods: {
    // 查询
    searchposition () {
      var _this = this
      _this.isLoading = true
      _this.loadError = ''
      var name = _this.searchText

      axios.post('restful/position/select',
        {
          'name': name === '' ? '' : _this.searchText
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
				_this.positionList = response.data && response.data.positionDtos ? response.data.positionDtos : []
				var count=0;
        var len=_this.positionList.length
				for(var i=0;i<len;i++){
					if(_this.positionList[i].status=='1'){
						console.log('摔倒')
						count++;
					}
				}
				console.log(count)
				_this.num=count
        _this.total = _this.positionList.length
      })
        .catch(function (error) {
          console.log(error)
          _this.positionList = []
          _this.total = 0
          _this.num = ''
          _this.loadError = '请检查服务连接后重试'
        }).then(function () {
          _this.isLoading = false
        })
    },
    // 摔倒原因与处理办法（红点点进来就是这个）
    showFallReason (row) {
      this.$alert(
        '老人：' + row.name + '\n'
        + '发生时间：' + row.beTime + '\n'
        + '位置：经度 ' + row.lng + '，纬度 ' + row.lat + '\n\n'
        + '怎么处理：\n'
        + '① 立即电话联系老人或家属确认是否受伤；\n'
        + '② 无人应答时按位置上门查看，必要时拨打 120；\n'
        + '③ 确认处理后点「标记已处理」，右上角摔倒红点数量会减少。',
        '摔倒告警 · 原因与处理',
        { confirmButtonText: '知道了', callback: function () {} }
      )
    },
    // 处置：消除摔倒红点
    handleFall (row) {
      var _this = this
      this.$confirm('确认该摔倒告警已经处理（联系过老人/家属）？处理后会从红点统计里移除。', '提示', {
        confirmButtonText: '确定已处理',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        axios.post('restful/device/handleAlarm', {'oldId': row.oldId},
          {
            headers: {'content-type': 'application/json'},
            withCredentials: true
          }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({type: 'success', message: '已处理，摔倒红点已消除'})
          } else {
            _this.$message({type: 'error', message: response.data.message || '处理失败'})
          }
          _this.searchposition()
        }).catch(function (error) {
          console.log(error)
          _this.$message({type: 'error', message: '处理失败'})
        })
      }).catch(function () {})
    },
    // 添加角色
    addRoles () {
      var _this = this
      axios.post('restful/roles/save',
        {
          'roleName': _this.roleForm.roleName
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.roleDialogVisible = false
          _this.searchposition()
          _this.$message({
            type: 'success',
            message: '添加角色成功'
          })
        } else {
          _this.$message({
            type: 'error',
            message: '添加角色失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    // 关闭添加角色对话框
    closeRoleAddDialog () {
      // 重置表单
      this.$refs.addRoleFormRef.resetFields()
    },
    // 编辑
    editRole () {
      var _this = this
      axios.post('restful/roles/update', _this.roleEditForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.roleEditDialogVisible = false
          _this.$message({
            type: 'success',
            message: '编辑角色信息成功'
          })
          _this.searchposition()
        } else {
          _this.$message({
            type: 'error',
            message: '编辑角色信息失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
	// 用经纬度设置地图中心点
	ready(){
		this.mapAvailable = typeof window.BMap !== 'undefined'
	},
	theLocation(valuef,valuel){
		var BMap = window.BMap
		var longitude = Number(valuef)
		var latitude = Number(valuel)
		if (!this.mapAvailable || !BMap || !isFinite(longitude) || !isFinite(latitude) || longitude === 0 || latitude === 0) {
			this.$message({ type: 'warning', message: '当前记录没有可用坐标或地图服务未就绪' })
			return
		}
		var map = this.mapInstance
		if (!map) {
			map = new BMap.Map('allmap')
			this.mapInstance = map
			map.enableScrollWheelZoom(true)
		}
		map.centerAndZoom(new BMap.Point(longitude, latitude), 16)
		map.clearOverlays()
			var new_point = new BMap.Point(longitude,latitude)
			var marker = new BMap.Marker(new_point);  // 创建标注
			map.addOverlay(marker);              // 将标注添加到地图中
			map.panTo(new_point);      
	},
    // 删除
    async deleteRole (roleName) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        console.log(roleName)
        axios.post('restful/roles/delete', {'roleName': roleName},
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
          _this.searchposition()
        })
          .catch(function (error) {
            console.log(error)
          })
      } catch (err) {
        // 取消删除
        this.$message({
          type: 'info',
          message: '已取消删除'
        })
      }
    },
    // 编辑信息
    handleEditRole (index, row) {
      this.roleEditDialogVisible = true
      this.roleEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
      // console.log(row)
    },
    handleSizeChange (val) {
      console.log(`每页 ${val} 条`)
      this.pageSize = val
    },
    handleCurrentChange (val) {
      console.log(`当前页: ${val}`)
      this.currentPage = val
    },
    changePage (val) {
      console.log(`当前页: ${val}`)
      this.currentPage = val
    },
    // 展示分配权限对话框
    showRoleDialog (curRole) {
      // curRole 表示当前角色的数据，children属性是当前角色拥有的权限
      console.log('showRoleDialog', curRole)
			var _this=this
      // 展示对话框
      // _this.isShowRoleDialog = true
      _this.lng=curRole.lng
			_this.lat=curRole.lat
			console.log(curRole.lng)
			_this.theLocation(_this.lng,_this.lat)
    },

    // 给角色分配权限
    async assignRights () {
    }
	

  }
}

</script>



<style lang="less" scoped>
  .map-panel { margin-top: 24px; padding: 24px; background: #fff; border: 1px solid #e7ecef; border-radius: 12px; }
  .map-heading { margin-bottom: 16px; }
  .map-heading h2 { margin: 0; font-size: 18px; color: #243746; }
  .map-heading p { margin: 8px 0 0; color: #71808c; font-size: 13px; }
  .map-unavailable { min-height: 180px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; color: #71808c; text-align: center; }
  .map-unavailable i { font-size: 28px; color: #91a4ad; }
  .map-unavailable strong { color: #344955; }
  .map-unavailable span { max-width: 520px; line-height: 1.6; }
  .position-empty, .unknown-state { color: #71808c; }
  .sys-research {
    height: 60px;
    padding: 20px 20px 0;
  }

  .level1 {
    border-bottom: 1px dotted #ccc;
    padding: 10px 0;

    &:last-child {
      border: 0;
    }
  }

  .level2 {
    padding: 5px 0;
  }

  .level3 {
    margin: 3px;
  }
  #allmap { height: 420px; width: 100%; border-radius: 8px; }

</style>
