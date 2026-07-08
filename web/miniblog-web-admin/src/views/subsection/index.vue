<template>
  <div class="app-container">
    <div class="toolbar">
      <el-select
        v-model="selectedModuleCode"
        placeholder="选择模块"
        clearable
        class="module-select"
        @change="handleModuleChange"
      >
        <el-option
          v-for="item in moduleOptions"
          :key="item.code"
          :label="item.title"
          :value="item.code"
        />
      </el-select>
      <el-select
        v-model="selectedSectionCode"
        placeholder="选择章节"
        clearable
        class="section-select"
        :disabled="!sectionOptions.length"
        @change="loadSubsections"
      >
        <el-option
          v-for="item in sectionOptions"
          :key="item.code"
          :label="item.title"
          :value="item.code"
        />
      </el-select>
      <el-button type="primary" :disabled="!selectedSectionCode" @click="openCreateDialog">新增子章节</el-button>
      <el-button :loading="loading" @click="() => loadSubsections(true)">刷新</el-button>
    </div>

    <el-empty v-if="!subsections.length && !loading" :description="emptyDescription" />

    <el-table
      v-else
      v-loading="loading"
      :data="subsections"
      border
      style="width: 100%"
    >
      <el-table-column prop="code" label="子章节编码" min-width="160" />
      <el-table-column prop="title" label="子章节标题" min-width="220" />
      <el-table-column prop="sort" label="排序" width="120" align="center" />
      <el-table-column label="状态" width="140" align="center">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" align="center">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openEditDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)" style="margin-left:8px">删除</el-button>
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
        <el-form-item label="所属章节" prop="section_code">
          <el-select v-model="formModel.section_code" placeholder="请选择章节" :disabled="dialogMode === 'edit'">
            <el-option
              v-for="item in sectionOptions"
              :key="item.code"
              :label="item.title"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="子章节编码" prop="code">
          <el-input v-model="formModel.code" placeholder="请输入子章节编码" maxlength="128" :disabled="dialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="子章节标题" prop="title">
          <el-input v-model="formModel.title" placeholder="请输入子章节标题" maxlength="255" />
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
import { ElMessage, ElMessageBox } from 'element-plus';
import 'element-plus/es/components/message-box/style/css';
import type { FormInstance, FormRules } from 'element-plus';
import useModuleStore from '@/store/modules/module';
import type { ModuleItem } from '@/store/modules/module';
import useSectionStore from '@/store/modules/section';
import type { SectionItem } from '@/store/modules/section';
import useSubsectionStore from '@/store/modules/subsection';
import type { SubsectionItem } from '@/store/modules/subsection';

const moduleStore = useModuleStore();
const sectionStore = useSectionStore();
const subsectionStore = useSubsectionStore();

const moduleOptions = computed<ModuleItem[]>(() => moduleStore.modules);
const selectedModuleCode = ref('');
const selectedSectionCode = ref('');
const NORMAL_STATUS = 1;

const sectionOptions = computed<SectionItem[]>(() => sectionStore.getSectionsByModule(selectedModuleCode.value));
const subsections = computed(() => subsectionStore.getSubsectionsBySection(selectedSectionCode.value));
const loading = computed(() => Boolean(selectedSectionCode.value && subsectionStore.loadingSections[selectedSectionCode.value]));

const formDialogVisible = ref(false);
const formSubmitting = ref(false);
const formRef = ref<FormInstance>();
const dialogMode = ref<'create' | 'edit'>('create');
const editingCode = ref('');
const statusChangingCode = ref('');
const statusChangingType = ref<'publish' | 'unpublish' | ''>('');

const formModel = reactive({
  section_code: '',
  code: '',
  title: '',
  sort: 0
});

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新增子章节' : '编辑子章节'));
const emptyDescription = computed(() => (selectedSectionCode.value ? '暂无子章节' : '请选择章节查看子章节'));

const formRules: FormRules = {
  section_code: [{ required: true, message: '请选择章节', trigger: 'change' }],
  code: [
    { required: true, message: '请输入子章节编码', trigger: 'blur' },
    { min: 1, max: 128, message: '编码长度需在 1-128 个字符之间', trigger: 'blur' }
  ],
  title: [
    { required: true, message: '请输入子章节标题', trigger: 'blur' },
    { min: 1, max: 255, message: '标题长度需在 1-255 个字符之间', trigger: 'blur' }
  ]
};

const statusText = (status?: number) => (status === NORMAL_STATUS ? '正常' : status === 2 ? '未上架' : '未知');
const statusType = (status?: number) => (status === NORMAL_STATUS ? 'success' : status === 2 ? 'info' : 'warning');

const openCreateDialog = () => {
  if (!selectedSectionCode.value) {
    ElMessage.warning('请先选择章节');
    return;
  }
  dialogMode.value = 'create';
  editingCode.value = '';
  formModel.section_code = selectedSectionCode.value;
  formModel.code = '';
  formModel.title = '';
  formModel.sort = 0;
  formDialogVisible.value = true;
  nextTick(() => formRef.value?.clearValidate());
};

const openEditDialog = (subsection: SubsectionItem) => {
  dialogMode.value = 'edit';
  editingCode.value = subsection.code;
  formModel.section_code = subsection.section_code;
  formModel.code = subsection.code;
  formModel.title = subsection.title;
  formModel.sort = subsection.sort ?? 0;
  formDialogVisible.value = true;
  nextTick(() => formRef.value?.clearValidate());
};

const handleFormCancel = () => {
  formDialogVisible.value = false;
  nextTick(() => formRef.value?.clearValidate());
};

const handleFormSubmit = async () => {
  if (!formRef.value) return;
  try {
    await formRef.value.validate();
  } catch {
    return;
  }

  formSubmitting.value = true;
  try {
    if (dialogMode.value === 'create') {
      await subsectionStore.createSubsection({
        section_code: formModel.section_code,
        code: formModel.code,
        title: formModel.title
      });
      ElMessage.success('新增子章节成功');
    } else {
      await subsectionStore.updateSubsection(editingCode.value, {
        title: formModel.title,
        sort: formModel.sort
      });
      ElMessage.success('更新子章节成功');
    }
    formDialogVisible.value = false;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '操作失败';
    ElMessage.error(message);
  } finally {
    formSubmitting.value = false;
  }
};

const changeSubsectionStatus = async (subsection: SubsectionItem, action: 'publish' | 'unpublish') => {
  statusChangingCode.value = subsection.code;
  statusChangingType.value = action;
  try {
    if (action === 'publish') {
      await subsectionStore.publishSubsection(subsection.code);
      ElMessage.success('子章节已上架');
    } else {
      await subsectionStore.unpublishSubsection(subsection.code);
      ElMessage.success('子章节已下架');
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '操作失败';
    ElMessage.error(message);
  } finally {
    statusChangingCode.value = '';
    statusChangingType.value = '';
  }
};

const handlePublish = (subsection: SubsectionItem) => changeSubsectionStatus(subsection, 'publish');
const handleUnpublish = (subsection: SubsectionItem) => changeSubsectionStatus(subsection, 'unpublish');

const loadModules = async () => {
  await moduleStore.ensureLoaded();
  if (!selectedModuleCode.value && moduleOptions.value.length) {
    selectedModuleCode.value = moduleOptions.value[0].code;
  }
};

const loadSections = async () => {
  if (!selectedModuleCode.value) return;
  await sectionStore.fetchSections(selectedModuleCode.value);
  if (!selectedSectionCode.value && sectionOptions.value.length) {
    selectedSectionCode.value = sectionOptions.value[0].code;
  }
};

const loadSubsections = async (force = false) => {
  if (!selectedSectionCode.value) return;
  try {
    await subsectionStore.fetchSubsections(selectedSectionCode.value, force);
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '加载子章节失败';
    ElMessage.error(message);
  }
};

const handleModuleChange = async () => {
  selectedSectionCode.value = '';
  await loadSections();
  await loadSubsections();
};

const handleDelete = async (subsection: SubsectionItem) => {
  try {
    await ElMessageBox.confirm('确认删除该子章节？该操作不可恢复', '确认删除', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    });
    await subsectionStore.deleteSubsection(subsection.code);
    ElMessage.success('删除子章节成功');
    await loadSubsections(true);
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return;
    const message = error instanceof Error ? error.message : '删除子章节失败';
    ElMessage.error(message);
  }
};

onMounted(async () => {
  await loadModules();
  await loadSections();
  await loadSubsections();
});
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.module-select,
.section-select {
  width: 220px;
}
</style>
