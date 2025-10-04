<template>
  <div class="app-container">
    <div class="toolbar">
      <el-select
        v-model="selectedModuleCode"
        placeholder="选择模块"
        clearable
        class="module-select"
        @change="loadSections"
      >
        <el-option
          v-for="item in moduleOptions"
          :key="item.code"
          :label="item.title"
          :value="item.code"
        />
      </el-select>
      <el-button type="primary" :disabled="!moduleOptions.length" @click="openCreateDialog">新增章节</el-button>
      <el-button :loading="loading" @click="loadSections">刷新</el-button>
    </div>

    <el-empty v-if="!sections.length && !loading" :description="emptyDescription" />

    <el-table
      v-else
      v-loading="loading"
      :data="sections"
      border
      style="width: 100%"
    >
      <el-table-column prop="code" label="章节编码" min-width="160" />
      <el-table-column prop="title" label="章节标题" min-width="220" />
      <el-table-column label="模块" min-width="200">
        <template #default>
          {{ moduleLabel(selectedModuleCode) }}
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="createDialogVisible"
      title="新增章节"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="所属模块" prop="module_code">
          <el-select v-model="createForm.module_code" placeholder="请选择模块">
            <el-option
              v-for="item in moduleOptions"
              :key="item.code"
              :label="item.title"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="章节编码" prop="code">
          <el-input v-model="createForm.code" placeholder="请输入章节编码" maxlength="128" />
        </el-form-item>
        <el-form-item label="章节标题" prop="title">
          <el-input v-model="createForm.title" placeholder="请输入章节标题" maxlength="255" />
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
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { fetchModules } from '@/api/module';
import { fetchSections, createSection } from '@/api/section';

interface ModuleItem {
  code: string;
  title: string;
}

interface SectionItem {
  code: string;
  title: string;
}

interface FetchModulesResponse {
  modules?: ModuleItem[];
}

interface FetchSectionsResponse {
  sections?: SectionItem[];
}

const moduleOptions = ref<ModuleItem[]>([]);
const selectedModuleCode = ref('');
const sections = ref<SectionItem[]>([]);
const loading = ref(false);

const createDialogVisible = ref(false);
const createSubmitting = ref(false);
const createFormRef = ref<FormInstance>();

const createForm = reactive({
  module_code: '',
  code: '',
  title: ''
});

const createRules: FormRules = {
  module_code: [{ required: true, message: '请选择模块', trigger: 'change' }],
  code: [
    { required: true, message: '请输入章节编码', trigger: 'blur' },
    { min: 1, max: 128, message: '编码长度需在 1-128 个字符之间', trigger: 'blur' }
  ],
  title: [
    { required: true, message: '请输入章节标题', trigger: 'blur' },
    { min: 1, max: 255, message: '标题长度需在 1-255 个字符之间', trigger: 'blur' }
  ]
};

const moduleLabel = (moduleCode: string) => {
  const target = moduleOptions.value.find((item) => item.code === moduleCode);
  return target?.title ?? moduleCode ?? '';
};

const emptyDescription = computed(() => (selectedModuleCode.value ? '暂无章节' : '请选择模块查看章节'));

const openCreateDialog = () => {
  if (!moduleOptions.value.length) {
    ElMessage.warning('请先创建模块');
    return;
  }
  if (!selectedModuleCode.value) {
    createForm.module_code = moduleOptions.value[0]?.code ?? '';
  } else {
    createForm.module_code = selectedModuleCode.value;
  }
  createForm.code = '';
  createForm.title = '';
  createFormRef.value?.clearValidate();
  createDialogVisible.value = true;
};

const handleCreateCancel = () => {
  createDialogVisible.value = false;
  createFormRef.value?.clearValidate();
};

const handleCreateSubmit = async () => {
  if (!createFormRef.value) {
    return;
  }
  await createFormRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    try {
      createSubmitting.value = true;
      await createSection({
        module_code: createForm.module_code,
        code: createForm.code,
        title: createForm.title
      });
      ElMessage.success('新增章节成功');
      createDialogVisible.value = false;
      if (selectedModuleCode.value === createForm.module_code) {
        await loadSections();
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '新增章节失败';
      ElMessage.error(message);
    } finally {
      createSubmitting.value = false;
    }
  });
};

const loadModules = async () => {
  try {
    const res = await fetchModules() as FetchModulesResponse;
    moduleOptions.value = res.modules ?? [];
    if (!selectedModuleCode.value && moduleOptions.value.length) {
      selectedModuleCode.value = moduleOptions.value[0].code;
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载模块失败';
    ElMessage.error(message);
  }
};

const loadSections = async () => {
  if (!selectedModuleCode.value) {
    sections.value = [];
    return;
  }
  loading.value = true;
  try {
    const res = await fetchSections(selectedModuleCode.value) as FetchSectionsResponse;
    sections.value = res.sections ?? [];
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载章节失败';
    ElMessage.error(message);
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  await loadModules();
  await loadSections();
});
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.module-select {
  width: 240px;
}
</style>
