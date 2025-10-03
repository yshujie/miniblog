<template>
  <div class="app-container">
    <el-form :model="filters" label-width="110px" class="filter-form">
      <el-form-item label="模块">
        <el-select
          v-model="filters.module_code"
          placeholder="选择模块"
          class="module-select"
          clearable
          @change="handleModuleChange"
        >
          <el-option
            v-for="moduleItem in modules"
            :key="moduleItem.code"
            :label="moduleItem.title"
            :value="moduleItem.code"
          />
        </el-select>
        <el-select
          v-model="filters.section_code"
          placeholder="选择章节"
          class="section-select"
          clearable
          :disabled="!sections.length"
          @change="search"
        >
          <el-option
            v-for="section in sections"
            :key="section.code"
            :label="section.title"
            :value="section.code"
          />
        </el-select>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="search">查询</el-button>
        <el-button @click="resetFilters">重置</el-button>
      </el-form-item>
    </el-form>

    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%"
    >
      <el-table-column label="ID" align="center" width="80">
        <template #default="{ row }">
          <span>{{ row.id }}</span>
        </template>
      </el-table-column>
      <el-table-column label="模块/章节" min-width="180" align="center">
        <template #default="{ row }">
          <span>{{ row.module?.title }} - {{ row.section?.title }}</span>
        </template>
      </el-table-column>
      <el-table-column label="作者" width="140" align="center">
        <template #default="{ row }">
          <span>{{ row.author }}</span>
        </template>
      </el-table-column>
      <el-table-column label="标题" min-width="260">
        <template #default="{ row }">
          <router-link :to="`/article/edit/${row.id}`" class="link-type">
            {{ row.title }}
          </router-link>
        </template>
      </el-table-column>
      <el-table-column label="标签" min-width="220">
        <template #default="{ row }">
          <el-tag
            v-for="tag in row.tags || []"
            :key="`${row.id}-${tag}`"
            type="primary"
            class="tag-item"
          >
            {{ tag }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="140" align="center">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" align="center">
        <template #default="{ row }">
          <router-link :to="`/article/edit/${row.id}`">
            <el-button size="small" type="primary" icon="Edit">编辑</el-button>
          </router-link>
        </template>
      </el-table-column>
    </el-table>

    <pagination
      v-show="total > 0"
      :total="total"
      v-model:page="filters.page"
      v-model:limit="filters.limit"
      @pagination="search"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import Pagination from '@/components/Pagination/index.vue';
import { fetchList } from '@/api/article';
import { fetchModules } from '@/api/module';
import { fetchSections } from '@/api/section';

interface ModuleItem {
  code: string;
  title: string;
}

interface SectionItem {
  code: string;
  title: string;
}

interface ArticleFilters {
  module_code: string;
  section_code: string;
  page: number;
  limit: number;
}

interface ArticleItem {
  id: string | number;
  module?: ModuleItem | { title?: string };
  section?: SectionItem | { title?: string };
  author?: string;
  title?: string;
  tags?: string[];
  status?: string;
}

interface FetchModulesResponse {
  modules?: ModuleItem[];
}

interface FetchSectionsResponse {
  sections?: SectionItem[];
}

interface FetchArticlesResponse {
  articles?: ArticleItem[];
  total?: number;
}

const list = ref<ArticleItem[]>([]);
const total = ref(0);
const listLoading = ref(false);
const modules = ref<ModuleItem[]>([]);
const sections = ref<SectionItem[]>([]);

const filters = reactive<ArticleFilters>({
  module_code: '',
  section_code: '',
  page: 1,
  limit: 20
});

const statusType = (status?: string) => {
  switch (status) {
    case 'Published':
      return 'success';
    case 'Draft':
      return 'info';
    case 'Unpublished':
      return 'danger';
    default:
      return 'warning';
  }
};

const statusText = (status?: string) => {
  if (!status) return '未知';
  const map: Record<string, string> = {
    Published: '已发布',
    Draft: '草稿',
    Unpublished: '未发布'
  };
  return map[status] || status;
};

const loadModules = async () => {
  try {
    const response = await fetchModules() as FetchModulesResponse;
    modules.value = response.modules ?? [];
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载模块失败';
    ElMessage.error(message);
  }
};

const loadSections = async () => {
  if (!filters.module_code) {
    sections.value = [];
    filters.section_code = '';
    return;
  }
  try {
    const response = await fetchSections(filters.module_code) as FetchSectionsResponse;
    sections.value = response.sections ?? [];
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载章节失败';
    ElMessage.error(message);
  }
};

const search = async () => {
  listLoading.value = true;
  try {
    const response = await fetchList({ ...filters }) as FetchArticlesResponse;
    list.value = response.articles ?? [];
    total.value = response.total ?? 0;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载文章列表失败';
    ElMessage.error(message);
  } finally {
    listLoading.value = false;
  }
};

const resetFilters = () => {
  filters.module_code = '';
  filters.section_code = '';
  filters.page = 1;
  filters.limit = 20;
  sections.value = [];
  search();
};

const handleModuleChange = async () => {
  filters.section_code = '';
  await loadSections();
  search();
};

onMounted(async () => {
  await loadModules();
  await search();
});
</script>

<style scoped>
.filter-form {
  margin-bottom: 20px;
}

.module-select {
  width: 200px;
  margin-right: 12px;
}

.section-select {
  width: 220px;
}

.tag-item {
  margin-right: 6px;
}

.link-type {
  color: var(--el-color-primary);
}
</style>
