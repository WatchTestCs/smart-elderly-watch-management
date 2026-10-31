<template>
  <div class="roles" :aria-busy="String(isLoading)">
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>
    <el-button v-if="loadError" link @click="searchOldPersonLife">重新加载生活资料</el-button>
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchOldPersonLife">
		  
        <div class="search-field">
          
          <label class="table-search-label" for="search-life">长者姓名</label>
          <el-input id="search-life"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>

      </form>
    </div>
    
    <el-table class="adaptive-record-table" :data="OldPersonList"
    style="width: 100%" border v-loading="isLoading">
    <template #empty><div  class="care-empty-state"><strong>暂无生活信息</strong><span>设备上报的生活信息会显示在这里。</span></div></template>

      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="140" align="center"></el-table-column>
      <el-table-column prop="sex" label="性别" width="140" align="center"></el-table-column>
	  <el-table-column prop="illness" label="病史" width="140" align="center"></el-table-column>
	  <el-table-column prop="warn" label="过敏药物" width="140" align="center"></el-table-column>
	  <el-table-column prop="drag" label="常用药物" width="140" align="center"></el-table-column>
      <el-table-column prop="foodLike" label="喜欢吃的食物" width="auto" align="center"></el-table-column>
	  <el-table-column prop="foodHate" label="忌口" width="auto" align="center"></el-table-column>
    </el-table>
    <record-cards :records="OldPersonList" :fields='[{"key":"sex","label":"性别"},{"key":"illness","label":"病史"},{"key":"warn","label":"过敏药物"},{"key":"drag","label":"常用药物"},{"key":"foodLike","label":"喜欢吃的食物"},{"key":"foodHate","label":"忌口"}]' label="生活资料" title-key="name" key-field="oldId"></record-cards>
    
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

<script>
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'

export default {
  components: { RecordCards },
  data () { return { searchText: '', OldPersonList: [], pageSize: 10, currentPage: 1, total: 0, isLoading: false, loadError: '' } },
  created () { this.searchOldPersonLife() },
  methods: {
    handleSizeChange (value) { this.pageSize = value; this.currentPage = 1; this.searchOldPersonLife(true) },
    changePage (value) { this.currentPage = value; this.searchOldPersonLife(true) },
    searchOldPersonLife (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
      if (this.isLoading) return
      this.isLoading = true
      this.loadError = ''
      return axios.post('api/v1/oldPerson/select', { page: this.currentPage, pageSize: this.pageSize,  name: this.searchText.trim() }, { withCredentials: true }).then(response => {
        if (!Array.isArray(response.data.oldPeople)) throw new Error('missing life records')
        this.OldPersonList = response.data.oldPeople
        this.total = response.data.total ?? this.OldPersonList.length
      }).catch(() => { this.loadError = '生活资料暂时无法加载，请重新加载。'; this.OldPersonList = [] }).then(() => { this.isLoading = false })
    }
  }
}
</script>
