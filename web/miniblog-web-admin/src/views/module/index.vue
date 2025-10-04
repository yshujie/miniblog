<template>
  <div class="app-container">
    <div class="toolbar">
      <el-button type="primary" @click="openCreateDialog">新增模块</el-button>
      <el-button :loading="loading" @click="loadModules">刷新</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="moduleList"
      border
      style="width: 100%"
    >
      <el-table-column prop="code" label="编码" min-width="160" />
      <el-table-column prop="title" label="标题" min-width="220" />
      <el-table-column label="状态" width="140" align="center">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="createDialogVisible"
      title="新增模块"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="编码" prop="code">
          <el-input v-model="createForm.code" placeholder="请输入模块编码" maxlength="128" />
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="createForm.title" placeholder="请输入模块标题" maxlength="255" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleCreateCancel">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="handleCreateSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { fetchModules, createModule } from '@/api/module';

interface ModuleItem {
  id?: number | string;
  code: string;
  title: string;
  status?: number;
}

interface FetchModulesResponse {
  modules?: ModuleItem[];
}

const moduleList = ref<ModuleItem[]>([]);
const loading = ref(false);

const createDialogVisible = ref(false);
const createSubmitting = ref(false);
const createFormRef = ref<FormInstance>();

const createForm = reactive({
  code: '',
  title: ''
});

const createRules: FormRules = {
  code: [
    { required: true, message: '请输入模块编码', trigger: 'blur' },
    { min: 1, max: 128, message: '编码长度需在 1-128 个字符之间', trigger: 'blur' }
  ],
  title: [
    { required: true, message: '请输入模块标题', trigger: 'blur' },
    { min: 1, max: 255, message: '标题长度需在 1-255 个字符之间', trigger: 'blur' }
  ]
};

const statusText = (status?: number) => {
  switch (status) {
    case 1:
      return '正常';
    case 2:
      return '已下架';
    default:
      return '未知';
  }
};

const statusType = (status?: number) => {
  switch (status) {
    case 1:
      return 'success';
    case 2:
      return 'info';
    default:
      return 'warning';
  }
};

const resetCreateForm = () => {
  createForm.code = '';
  createForm.title = '';
  createFormRef.value?.clearValidate();
};

const openCreateDialog = () => {
  resetCreateForm();
  createDialogVisible.value = true;
};

const handleCreateCancel = () => {
  createDialogVisible.value = false;
};

const handleCreateSubmit = async () => {
  if (!createFormRef.value) return;
  await createFormRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    try {
      createSubmitting.value = true;
      await createModule({
        code: createForm.code,
        title: createForm.title
      });
      ElMessage.success('新增模块成功');
      createDialogVisible.value = false;
      await loadModules();
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '新增模块失败';
      ElMessage.error(message);
    } finally {
      createSubmitting.value = false;
    }
  });
};

const loadModules = async () => {
  loading.value = true;
  try {
    const res = await fetchModules() as FetchModulesResponse;
    moduleList.value = res.modules ?? [];
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载模块列表失败';
    ElMessage.error(message);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadModules();
});
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
}

.toolbar :deep(.el-button + .el-button) {
  margin-left: 12px;
}
</style>
