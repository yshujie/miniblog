<template>
  <div class="directory-picker">
    <el-select :model-value="modelValue.module_code" placeholder="模块" :disabled="disabled" @change="changeModule">
      <el-option v-for="item in catalog.modules.modules" :key="item.code" :value="item.code" :label="item.title" :disabled="activeOnly && item.status !== 1" />
    </el-select>
    <el-select :model-value="modelValue.section_code" placeholder="所属章节" :disabled="disabled || !modelValue.module_code" @change="changeSection">
      <el-option v-for="item in catalog.sections.getSectionsByModule(modelValue.module_code)" :key="item.code" :value="item.code" :label="item.title" :disabled="activeOnly && item.status !== 1" />
    </el-select>
    <el-select :model-value="modelValue.subsection_code" placeholder="直属章节（可选子章节）" clearable :disabled="disabled || !modelValue.section_code" @change="changeSubsection">
      <el-option v-for="item in catalog.subsections.getSubsectionsBySection(modelValue.section_code)" :key="item.code" :value="item.code" :label="item.title" :disabled="activeOnly && item.status !== 1" />
    </el-select>
  </div>
  <el-alert v-if="error" :title="error" type="error" :closable="false" />
</template>
<script setup lang="ts">
import { ref } from 'vue';
import { errorMessage } from '@/utils/api-error';
import { useCatalog } from '@/composables/useCatalog';
import type { DirectoryContext } from '@/types/content';
const props = defineProps<{ modelValue: DirectoryContext; disabled?: boolean; activeOnly?: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [DirectoryContext] }>();
const catalog = useCatalog(); const error = ref('');
const changeModule = async (module_code: string) => { emit('update:modelValue', { module_code, section_code: '', subsection_code: '' }); error.value = ''; try { await catalog.sections.fetchSections(module_code); } catch (cause) { error.value = errorMessage(cause, '加载章节失败，请重新选择模块'); } };
const changeSection = async (section_code: string) => { emit('update:modelValue', { ...props.modelValue, section_code, subsection_code: '' }); error.value = ''; try { await catalog.subsections.fetchSubsections(section_code); } catch (cause) { error.value = errorMessage(cause, '加载子章节失败，请重新选择章节'); } };
const changeSubsection = (subsection_code: string) => emit('update:modelValue', { ...props.modelValue, subsection_code: subsection_code || '' });
</script>
<style scoped>.directory-picker { display: flex; flex-wrap: wrap; gap: 8px; } .el-select { min-width: 180px; flex: 1; }</style>
