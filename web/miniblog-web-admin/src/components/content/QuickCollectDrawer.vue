<template>
  <el-drawer :model-value="modelValue" title="收录外部文章" size="min(620px, 100%)" :close-on-click-modal="false" :before-close="beforeClose" @update:model-value="emit('update:modelValue', $event)">
    <p class="intro">粘贴 Notion 或飞书文档链接，在当前目录发布。</p>
    <el-alert v-if="catalogError" :title="catalogError" type="error" :closable="false" />
    <el-form label-position="top" v-loading="catalogLoading">
      <el-form-item v-if="recent.length" label="最近目录">
        <el-select placeholder="选择最近使用的目录" :disabled="quick.requestLocked.value" @change="selectRecent">
          <el-option v-for="(item, index) in recent" :key="`${item.section_code}:${item.subsection_code}`" :label="catalog.label(item)" :value="index" />
        </el-select>
      </el-form-item>
      <el-form-item label="所属目录" required>
        <DirectoryPicker active-only :model-value="quick.context()" :disabled="quick.requestLocked.value" @update:model-value="quick.setDirectory" />
      </el-form-item>
      <el-form-item label="文档链接" required>
        <el-input data-test="collect-link" :model-value="quick.form.external_link" :disabled="quick.requestLocked.value" placeholder="https://…" @update:model-value="quick.setLink" />
      </el-form-item>
      <el-form-item label="标题" required>
        <el-input data-test="collect-title" :model-value="quick.form.title" :disabled="quick.requestLocked.value" placeholder="自动建议，也可手动填写" maxlength="255" @update:model-value="quick.setTitle" />
        <div class="preview-note">
          <span v-if="quick.previewLoading.value">正在获取 Notion 标题…</span>
          <span v-else-if="quick.preview.value?.metadata_status === 'resolved'">已取得标题建议</span>
          <span v-else>飞书可直接手填；Notion 标题不可用时也可手填。</span>
          <el-button link :disabled="!quick.form.external_link || quick.requestLocked.value" @click="quick.suggestTitle">重新获取</el-button>
        </div>
        <p v-if="quick.preview.value?.reason" class="preview-note">{{ quick.preview.value.reason }}</p>
      </el-form-item>
      <el-form-item label="作者（可选）"><el-input v-model="quick.form.author" data-test="collect-author" :disabled="quick.requestLocked.value" maxlength="128" /></el-form-item>
      <el-form-item label="标签（可选）">
        <el-select v-model="quick.form.tags" multiple filterable allow-create default-first-option :disabled="quick.requestLocked.value" placeholder="输入标签后按 Enter"><el-option v-for="tag in quick.form.tags" :key="tag" :label="tag" :value="tag" /></el-select>
      </el-form-item>
    </el-form>
    <el-alert v-if="quick.error.value" data-test="collect-error" :title="quick.error.value" type="error" :closable="false" show-icon />
    <div v-if="quick.duplicate.value" class="duplicate" data-test="collect-duplicate">
      <p>现有文章：{{ quick.duplicate.value.title }}（{{ statusLabels[quick.duplicate.value.status] }}）</p>
      <p>归属：{{ quick.duplicate.value.module?.title }} / {{ quick.duplicate.value.section?.title }} / {{ quick.duplicate.value.subsection?.title || '直属章节' }}</p>
      <p v-if="quick.duplicate.value.status === 'Deleted'">归档文章需先恢复为草稿，再移动目录。</p>
      <el-button :disabled="quick.requestLocked.value || duplicateBusy" @click="openExisting">打开现有文章</el-button>
      <el-button :disabled="quick.requestLocked.value || quick.duplicate.value.status === 'Deleted' || !catalog.valid(quick.context())" :loading="duplicateBusy" @click="moveExisting">移动到所选目录</el-button>
      <el-button v-if="quick.duplicate.value.status === 'Deleted'" :disabled="quick.requestLocked.value" :loading="duplicateBusy" @click="restoreExisting">恢复为草稿</el-button>
    </div>
    <template #footer>
      <template v-if="quick.frozen.value && !quick.submitting.value">
        <el-button @click="quick.editAgain">返回修改</el-button><el-button type="primary" @click="submit()">原样重试</el-button>
      </template>
      <template v-else>
        <el-button :disabled="quick.submitting.value" @click="close">关闭</el-button>
        <el-button data-test="collect-publish" type="primary" :loading="quick.submitting.value" :disabled="catalogLoading || !!catalogError" @click="submit(false)">发布</el-button>
        <el-button data-test="collect-continue" type="success" :loading="quick.submitting.value" :disabled="catalogLoading || !!catalogError" @click="submit(true)">发布并继续</el-button>
      </template>
    </template>
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import DirectoryPicker from './DirectoryPicker.vue';
import { useQuickCollect } from '@/composables/useQuickCollect';
import { useCatalog } from '@/composables/useCatalog';
import useWorkspace from '@/store/modules/contentWorkspace';
import useUserStore from '@/store/modules/user';
import { moveArticle, changeArticleStatus } from '@/api/content';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { errorMessage } from '@/utils/api-error';
import { statusLabels, type DirectoryContext, type ArticleInfo } from '@/types/content';
const props = defineProps<{ modelValue: boolean; context?: DirectoryContext }>();
const emit = defineEmits<{ 'update:modelValue': [boolean]; collected: [] }>();
const router = useRouter(); const quick = useQuickCollect(); const catalog = useCatalog(); const workspace = useWorkspace(); const user = useUserStore();
const catalogLoading = ref(false); const catalogError = ref(''); const duplicateBusy = ref(false);
const recent = computed(() => workspace.recentDirectories.filter(catalog.valid));
let openVersion = 0;
watch(() => props.modelValue, async open => {
  const version = ++openVersion; if (!open) return;
  if (!contentRegistrationEnabled) { catalogError.value = '收录功能暂未启用，现有文章仍可管理。'; return; }
  catalogLoading.value = true; catalogError.value = '';
  try {
    await catalog.load(); if (version !== openVersion) return;
    const context = catalog.valid(props.context) ? props.context : recent.value.find(item => !props.context?.module_code || item.module_code === props.context.module_code) || recent.value[0] || props.context || { module_code: '', section_code: '', subsection_code: '' };
    quick.reset(context, workspace.lastAuthor || user.name || '');
  } catch (cause) { catalogError.value = errorMessage(cause, '加载目录失败'); } finally { if (version === openVersion) catalogLoading.value = false; }
}, { immediate: true });
const selectRecent = (index: number) => { if (recent.value[index]) quick.setDirectory(recent.value[index]); };
async function submit(continueAdding = false) {
  if (!contentRegistrationEnabled) { quick.error.value = '收录功能暂未启用'; return; }
  const before = quick.context(); const author = quick.form.author;
  if (!catalog.valid(before)) { quick.error.value = '请选择有效的章节或子章节'; return; }
  const result = await quick.submit(true, continueAdding);
  if (result?.outcome === 'created') {
    workspace.remember(before, author); workspace.invalidateArticles(); emit('collected'); ElMessage.success('文章已发布');
    if (!continueAdding && !quick.form.external_link) return;
    if (!continueAdding) emit('update:modelValue', false);
  }
}
const openExisting = () => { if (quick.duplicate.value) { emit('update:modelValue', false); router.push(`/article/edit/${quick.duplicate.value.id}`); } };
async function duplicateAction(action: () => Promise<{ article?: ArticleInfo }>) {
  duplicateBusy.value = true;
  try { const result = await action(); if (result?.article) quick.duplicate.value = result.article; workspace.invalidateArticles(); emit('collected'); await quick.suggestTitle(); ElMessage.success('文章已更新'); } catch (cause) { quick.error.value = errorMessage(cause); } finally { duplicateBusy.value = false; }
}
const moveExisting = () => {
  const article = quick.duplicate.value; if (!article) return;
  return duplicateAction(() => moveArticle(article.id, { section_code: quick.form.section_code, subsection_code: quick.form.subsection_code || undefined }));
};
const restoreExisting = () => { const article = quick.duplicate.value; if (article) return duplicateAction(() => changeArticleStatus(article.id, 'restore')); };
async function beforeClose(done: () => void) {
  if (quick.submitting.value) return;
  if (quick.form.external_link || quick.form.title) {
    try { await ElMessageBox.confirm('关闭后将清空本次尚未收录的输入，是否关闭？', '关闭收录'); } catch { return; }
  }
  done();
}
const close = () => beforeClose(() => emit('update:modelValue', false));
</script>
<style scoped>.intro,.preview-note { color: var(--el-text-color-secondary); } .preview-note { font-size: 13px; margin-top: 6px; } .duplicate { margin-top: 16px; padding: 12px; background: var(--el-fill-color-light); } .el-select { width: 100%; }</style>
