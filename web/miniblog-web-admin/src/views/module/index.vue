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
      <el-table-column label="操作" width="260" align="center">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openEditDialog(row)">编辑</el-button>
          <el-button
            v-if="row.status !== NORMAL_STATUS"
            size="small"
            type="success"
            :loading="statusChangingCode === row.code && statusChangingType === 'publish'"
            :disabled="statusChangingCode === row.code"
            @click="handlePublish(row)"
          >上架</el-button>
          <el-button
            v-else
            size="small"
            type="warning"
            :loading="statusChangingCode === row.code && statusChangingType === 'unpublish'"
            :disabled="statusChangingCode === row.code"
            @click="handleUnpublish(row)"
          >下架</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="formDialogVisible"
      :title="dialogTitle"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="编码" prop="code">
          <el-input
            v-model="createForm.code"
            placeholder="请输入模块编码"
            maxlength="128"
            :disabled="dialogMode === 'edit'"
          />
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="createForm.title" placeholder="请输入模块标题" maxlength="255" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleFormCancel">取消</el-button>
        <el-button type="primary" :loading="formSubmitting" @click="handleFormSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { fetchModules, createModule, updateModule, publishModule, unpublishModule } from '@/api/module';

interface ModuleItem {
  id?: number | string;
  code: string;
  title: string;
  status?: number;
}

interface FetchModulesResponse {
  modules?: ModuleItem[];
}

const NORMAL_STATUS = 1;

const moduleList = ref<ModuleItem[]>([]);
const loading = ref(false);

const formDialogVisible = ref(false);
const formSubmitting = ref(false);
const createFormRef = ref<FormInstance>();
const dialogMode = ref<'create' | 'edit'>('create');
const editingCode = ref('');

const statusChangingCode = ref('');
const statusChangingType = ref<'publish' | 'unpublish' | ''>('');

const createForm = reactive({
  code: '',
  title: ''
});

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新增模块' : '编辑模块'));

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
    case NORMAL_STATUS:
      return '正常';
    case 2:
      return '未上架';
    default:
      return '未知';
  }
};

const statusType = (status?: number) => {
  switch (status) {
    case NORMAL_STATUS:
      return 'success';
    case 2:
      return 'info';
    default:
      return 'warning';
  }
};

const openCreateDialog = () => {
  dialogMode.value = 'create';
  editingCode.value = '';
  createForm.code = '';
  createForm.title = '';
  formDialogVisible.value = true;
  nextTick(() => {
    createFormRef.value?.clearValidate();
  });
};

const openEditDialog = (moduleItem: ModuleItem) => {
  dialogMode.value = 'edit';
  editingCode.value = moduleItem.code;
  createForm.code = moduleItem.code;
  createForm.title = moduleItem.title;
  formDialogVisible.value = true;
  nextTick(() => {
    createFormRef.value?.clearValidate();
  });
};

const handleFormCancel = () => {
  formDialogVisible.value = false;
  nextTick(() => {
    createFormRef.value?.clearValidate();
  });
};

const handleFormSubmit = async () => {
  if (!createFormRef.value) return;
  try {
    await createFormRef.value.validate();
  } catch {
    return;
  }

  formSubmitting.value = true;
  try {
    if (dialogMode.value === 'create') {
      await createModule({
        code: createForm.code,
        title: createForm.title
      });
      ElMessage.success('新增模块成功');
    } else {
      await updateModule(editingCode.value, {
        title: createForm.title
      });
      ElMessage.success('更新模块成功');
    }
    formDialogVisible.value = false;
    await loadModules();
  } catch (error: unknown) {
    const defaultMessage = dialogMode.value === 'create' ? '新增模块失败' : '更新模块失败';
    const message = error instanceof Error ? error.message : defaultMessage;
    ElMessage.error(message);
  } finally {
    formSubmitting.value = false;
  }
};

const changeModuleStatus = async (moduleItem: ModuleItem, action: 'publish' | 'unpublish') => {
  if (!moduleItem?.code) {
    return;
  }
  statusChangingCode.value = moduleItem.code;
  statusChangingType.value = action;

  try {
    if (action === 'publish') {
      await publishModule(moduleItem.code);
      ElMessage.success('模块已上架');
    } else {
      await unpublishModule(moduleItem.code);
      ElMessage.success('模块已下架');
    }
    await loadModules();
  } catch (error: unknown) {
    const defaultMessage = action === 'publish' ? '上架模块失败' : '下架模块失败';
    const message = error instanceof Error ? error.message : defaultMessage;
    ElMessage.error(message);
  } finally {
    statusChangingCode.value = '';
    statusChangingType.value = '';
  }
};

const handlePublish = (moduleItem: ModuleItem) => changeModuleStatus(moduleItem, 'publish');
const handleUnpublish = (moduleItem: ModuleItem) => changeModuleStatus(moduleItem, 'unpublish');

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
