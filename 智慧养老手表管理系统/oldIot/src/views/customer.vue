<template>
  <div class="roles">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchCustomers">
        <div class="search-field">
          
          <label class="table-search-label" for="search-customer">监护人姓名</label>
          <el-input id="search-customer"
            placeholder="按监护人姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询监护人姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        <div class="toolbar-actions">
          <el-button plain @click="customerDialogVisible = true">添加监护人</el-button>
        </div>
		
      </form>
    </div>
    
    <el-dialog
      title="监护人添加"
      v-model="customerDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="customerForm" :rules="rules"  ref="addcustomerFormRef">
        <el-form-item label="姓名" prop="customerName">
          <el-input v-model="customerForm.customerName"></el-input>
        </el-form-item>
        <el-form-item label="年龄" prop="customerAge">
          <el-input v-model="customerForm.customerAge"></el-input>
        </el-form-item>
		<el-form-item label="性别" prop="customerSex">
		<el-radio-group v-model="customerForm.customerSex">
		  <el-radio value="男">男</el-radio>
		  <el-radio value="女">女</el-radio>
		</el-radio-group>
		</el-form-item>
		<el-form-item label="电话" prop="customerPhone">
		<el-input v-model="customerForm.customerPhone"></el-input>
		</el-form-item>
		<el-form-item label="地址" prop="customerAddress">
		<el-input v-model="customerForm.customerAddress"></el-input>
		</el-form-item>
		
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="customerDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addRoles('addcustomerFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog
      title="监护人编辑"
      v-model="oldEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :rules="rules" :model="oldEditForm">
        <el-form-item label="姓名" prop="customerName">
        <el-input v-model="oldEditForm.customerName"></el-input>
        </el-form-item>
				<el-form-item label="姓名" prop="customerId" style="display: none;">
				<el-input v-model="oldEditForm.customerId"></el-input>
				</el-form-item>
        <el-form-item label="年龄" prop="customerAge">
        <el-input v-model="oldEditForm.customerAge"></el-input>
        </el-form-item>
        <el-form-item label="性别" prop="customerSex">
        <el-radio-group v-model="oldEditForm.customerSex">
        <el-radio value="男">男</el-radio>
        <el-radio value="女">女</el-radio>
        </el-radio-group>
        </el-form-item>
        <el-form-item label="电话" prop="customerPhone">
        <el-input v-model="oldEditForm.customerPhone"></el-input>
        </el-form-item>
        <el-form-item label="地址" prop="customerAddress">
        <el-input v-model="oldEditForm.customerAddress"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editRole">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-table class="adaptive-record-table" :data="customerList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无监护人信息</strong><span>添加监护人或调整搜索条件后，信息会显示在这里。</span></div></template>

      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="customerName" label="姓名" width="170" align="center"></el-table-column>
      <el-table-column prop="customerAge" label="年龄" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerSex" label="性别" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerPhone" label="电话" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerAddress" label="地址" width="170" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.customerId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="customerList" :fields='[{"key":"customerAge","label":"年龄"},{"key":"customerSex","label":"性别"},{"key":"customerPhone","label":"电话"},{"key":"customerAddress","label":"地址"}]' label="监护人资料" title-key="customerName" key-field="customerId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.customerId)">删除
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
  </div>

</template>

<script src="./customer.logic.js"></script>
