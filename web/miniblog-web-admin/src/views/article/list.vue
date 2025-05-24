<template>
  <div class="app-container">

    <el-form :model="filters" label-width="120px">
      <el-form-item label="Module Code">
        <el-select v-model="filters.module_code" placeholder="Module Code" @change="initSections">
          <el-option v-for="module in modules" :key="module.code" :label="module.name" :value="module.code" />
        </el-select>

        <el-select v-model="filters.section_code" placeholder="Section Code" @change="section">
          <el-option v-for="section in sections" :key="section.code" :label="section.name" :value="section.code" />
        </el-select>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="search">Search</el-button>
        <el-button @click="resetSearch">Reset</el-button>
      </el-form-item>
    </el-form>

    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column align="center" label="ID" width="80">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column width="180px" align="center" label="Date">
        <template slot-scope="scope">
          <span>{{ scope.row.timestamp | parseTime('{y}-{m}-{d} {h}:{i}') }}</span>
        </template>
      </el-table-column>

      <el-table-column width="120px" align="center" label="Author">
        <template slot-scope="scope">
          <span>{{ scope.row.author }}</span>
        </template>
      </el-table-column>

      <el-table-column width="100px" label="Importance">
        <template slot-scope="scope">
          <svg-icon v-for="n in +scope.row.importance" :key="n" icon-class="star" class="meta-item__icon" />
        </template>
      </el-table-column>

      <el-table-column class-name="status-col" label="Status" width="110">
        <template slot-scope="{row}">
          <el-tag :type="row.status | statusFilter">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column min-width="300px" label="Title">
        <template slot-scope="{row}">
          <router-link :to="'/article/edit/'+row.id" class="link-type">
            <span>{{ row.title }}</span>
          </router-link>
        </template>
      </el-table-column>

      <el-table-column align="center" label="Actions" width="120">
        <template slot-scope="scope">
          <router-link :to="'/article/edit/'+scope.row.id">
            <el-button type="primary" size="small" icon="el-icon-edit">
              Edit
            </el-button>
          </router-link>
        </template>
      </el-table-column>
    </el-table>

    <pagination v-show="total>0" :total="total" :page.sync="filters.page" :limit.sync="filters.limit" @pagination="getList" />
  </div>
</template>

<script>
import { fetchList } from '@/api/article'
import { fetchModules } from '@/api/module'
import { fetchSections } from '@/api/section'
import Pagination from '@/components/Pagination' // Secondary package based on el-pagination

export default {
  name: 'ArticleList',
  components: { Pagination },
  filters: {
    statusFilter(status) {
      const statusMap = {
        published: 'success',
        draft: 'info',
        deleted: 'danger'
      }
      return statusMap[status]
    }
  },
  data() {
    return {
      list: null,
      total: 0,
      listLoading: true,
      modules: [],
      sections: [],
      filters: {
        module_code: '',
        section_code: '',
        page: 1,
        limit: 20
      }
    }
  },
  created() {
    this.init()
  },
  methods: {
    init() {
      this.initModules()
    },

    async initModules() {
      // 清空模块选择
      this.filters.module_code = ''
      this.modules = []

      // 获取模块
      const modulesResp = await fetchModules()
      this.modules = modulesResp.modules

      console.log('modules', this.modules)
    },

    async initSections() {
      // 清空章节选择
      this.filters.section_code = ''
      this.sections = []

      // 获取章节
      const sectionsResp = await fetchSections(this.filters.module_code)
      this.sections = sectionsResp.sections
    },

    async search() {
      this.listLoading = true
      const listResp = await fetchList(this.filters)
      this.list = listResp.items
      this.total = listResp.total
      this.listLoading = false
    },

    resetSearch() {
      this.filters = {
        module_code: '',
        section_code: '',
        page: 1,
        limit: 20
      }
    }
  }
}
</script>

<style scoped>
.edit-input {
  padding-right: 100px;
}
.cancel-btn {
  position: absolute;
  right: 15px;
  top: 10px;
}
</style>
