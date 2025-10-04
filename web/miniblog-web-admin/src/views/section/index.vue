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
      <el-button :loading="loading" @click="() => loadSections(true)">刷新</el-button>
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
      <el-table-column prop="sort" label="排序" width="120" align="center" />
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
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="formModel" :rules="formRules" label-width="110px">
        <el-form-item label="所属模块" prop="module_code">
          <el-select v-model="formModel.module_code" placeholder="请选择模块" :disabled="dialogMode === 'edit'">
            <el-option
              v-for="item in moduleOptions"
              :key="item.code"
              :label="item.title"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="章节编码" prop="code">
          <el-input v-model="formModel.code" placeholder="请输入章节编码" maxlength="128" :disabled="dialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="章节标题" prop="title">
          <el-input v-model="formModel.title" placeholder="请输入章节标题" maxlength="255" />
        </el-form-item>
        <el-form-item label="排序值" prop="sort">
          <el-input-number v-model="formModel.sort" :min="0" :max="9999" :controls="false" placeholder="请输入排序值" />
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
import useModuleStore from '@/store/modules/module';
import type { ModuleItem } from '@/store/modules/module';
import useSectionStore from '@/store/modules/section';
import type { SectionItem } from '@/store/modules/section';

const moduleStore = useModuleStore();
const sectionStore = useSectionStore();
const moduleOptions = computed<ModuleItem[]>(() => moduleStore.modules);
const selectedModuleCode = ref('');
const NORMAL_STATUS = 1;

const sections = computed(() => sectionStore.getSectionsByModule(selectedModuleCode.value));
const loading = computed(() => Boolean(selectedModuleCode.value && sectionStore.loadingModules[selectedModuleCode.value]));

const formDialogVisible = ref(false);
const formSubmitting = ref(false);
const formRef = ref<FormInstance>();
const dialogMode = ref<'create' | 'edit'>('create');
const editingCode = ref('');

const statusChangingCode = ref('');
const statusChangingType = ref<'publish' | 'unpublish' | ''>('');

const formModel = reactive({
  module_code: '',
  code: '',
  title: '',
  sort: 0
});

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新增章节' : '编辑章节'));

const formRules: FormRules = {
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
  if (!moduleOptions.value.length) {
    ElMessage.warning('请先创建模块');
    return;
  }
  dialogMode.value = 'create';
  editingCode.value = '';
  formModel.module_code = selectedModuleCode.value || moduleOptions.value[0]?.code || '';
  formModel.code = '';
  formModel.title = '';
  formModel.sort = 0;
  formDialogVisible.value = true;
  nextTick(() => {
    formRef.value?.clearValidate();
  });
};

const openEditDialog = (section: SectionItem) => {
  dialogMode.value = 'edit';
  editingCode.value = section.code;
  formModel.module_code = selectedModuleCode.value || section.module_code || '';
  formModel.code = section.code;
  formModel.title = section.title;
  formModel.sort = section.sort ?? 0;
  formDialogVisible.value = true;
  nextTick(() => {
    formRef.value?.clearValidate();
  });
};

const handleFormCancel = () => {
  formDialogVisible.value = false;
  nextTick(() => {
    formRef.value?.clearValidate();
  });
};

const handleFormSubmit = async () => {
  if (!formRef.value) {
    return;
  }
  try {
    await formRef.value.validate();
  } catch {
    return;
  }

  formSubmitting.value = true;
  try {
    if (dialogMode.value === 'create') {
      await sectionStore.createSection({
        module_code: formModel.module_code,
        code: formModel.code,
        title: formModel.title
      });
      ElMessage.success('新增章节成功');
    } else {
      await sectionStore.updateSection(editingCode.value, {
        title: formModel.title,
        sort: formModel.sort
      });
      ElMessage.success('更新章节成功');
    }
    formDialogVisible.value = false;
  } catch (error: unknown) {
    const defaultMessage = dialogMode.value === 'create' ? '新增章节失败' : '更新章节失败';
    const message = error instanceof Error ? error.message : defaultMessage;
    ElMessage.error(message);
  } finally {
    formSubmitting.value = false;
  }
};

const changeSectionStatus = async (section: SectionItem, action: 'publish' | 'unpublish') => {
  if (!section.code) {
    return;
  }
  statusChangingCode.value = section.code;
  statusChangingType.value = action;

  try {
    if (action === 'publish') {
      await sectionStore.publishSection(section.code);
      ElMessage.success('章节已上架');
    } else {
      await sectionStore.unpublishSection(section.code);
      ElMessage.success('章节已下架');
    }
  } catch (error: unknown) {
    const defaultMessage = action === 'publish' ? '上架章节失败' : '下架章节失败';
    const message = error instanceof Error ? error.message : defaultMessage;
    ElMessage.error(message);
  } finally {
    statusChangingCode.value = '';
    statusChangingType.value = '';
  }
};

const handlePublish = (section: SectionItem) => changeSectionStatus(section, 'publish');
const handleUnpublish = (section: SectionItem) => changeSectionStatus(section, 'unpublish');

const loadModules = async () => {
  try {
    await moduleStore.ensureLoaded();
    if (!selectedModuleCode.value && moduleOptions.value.length) {
      selectedModuleCode.value = moduleOptions.value[0].code;
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载模块失败';
    ElMessage.error(message);
  }
};

const loadSections = async (force = false) => {
  if (!selectedModuleCode.value) {
    return;
  }
  try {
    await sectionStore.fetchSections(selectedModuleCode.value, force);
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载章节失败';
    ElMessage.error(message);
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
