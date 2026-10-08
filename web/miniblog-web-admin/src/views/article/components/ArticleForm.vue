<template>
  <div class="article-form">
    <div class="toolbar">
      <el-button type="primary" :loading="editor.loading.value" :disabled="!editor.article.value || editor.article.value.status === 'Deleted'" @click="save">保存资料</el-button>
      <el-button v-if="editor.article.value?.status !== 'Deleted' && editor.article.value?.status !== 'Published'" type="success" :disabled="!editor.article.value" :loading="editor.loading.value" @click="changeStatus('publish')">发布</el-button>
      <el-button v-if="editor.article.value?.status === 'Published'" type="warning" :loading="editor.loading.value" @click="changeStatus('unpublish')">下架</el-button>
      <el-button v-if="editor.article.value?.status === 'Deleted'" type="success" :loading="editor.loading.value" @click="changeStatus('restore')">恢复为草稿</el-button>
      <el-button v-else :disabled="!editor.article.value" :loading="editor.loading.value" @click="changeStatus('archive')">归档</el-button>
      <el-tag v-if="editor.article.value">{{ statusLabels[editor.article.value.status] }}</el-tag>
      <el-button :disabled="editor.loading.value || editor.dirty.value" @click="load">重新读取</el-button>
      <span v-if="editor.dirty.value" class="dirty">有未保存的修改</span>
    </div>
    <p class="hint">保存标题、标签和归属会保留当前发布状态；正文继续在外部文档中维护。</p>
    <el-alert v-if="editor.error.value || catalogError" :title="editor.error.value || catalogError" type="error" :closable="false"><template #default><el-button v-if="!editor.article.value" link @click="load">重试</el-button></template></el-alert>
    <el-form label-width="110px" :disabled="editor.loading.value || !editor.article.value || editor.article.value.status === 'Deleted'">
      <el-form-item label="标题" required><el-input v-model="editor.form.title" maxlength="255" /></el-form-item>
      <el-form-item label="作者（可选）"><el-input v-model="editor.form.author" maxlength="128" /></el-form-item>
      <el-form-item label="归属目录" required><DirectoryPicker active-only v-model="directory" /></el-form-item>
      <el-form-item label="标签（可选）"><el-select v-model="editor.form.tags" multiple filterable allow-create default-first-option><el-option v-for="tag in editor.form.tags" :key="tag" :label="tag" :value="tag" /></el-select></el-form-item>
      <el-form-item label="来源文档"><a :href="editor.form.external_link" target="_blank" rel="noopener noreferrer">{{ editor.form.external_link }}</a></el-form-item>
    </el-form>
    <el-collapse v-if="editor.article.value?.content"><el-collapse-item title="查看历史正文" name="content"><pre class="historic-content">{{ editor.article.value.content }}</pre></el-collapse-item></el-collapse>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { onBeforeRouteLeave, useRoute } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import DirectoryPicker from '@/components/content/DirectoryPicker.vue';
import { useArticleEditor } from '@/composables/useArticleEditor';
import { useCatalog } from '@/composables/useCatalog';
import useWorkspace from '@/store/modules/contentWorkspace';
import { statusLabels, type DirectoryContext } from '@/types/content';
import { errorMessage } from '@/utils/api-error';
defineProps<{ isEdit: boolean }>();
const route = useRoute(); const editor = useArticleEditor(); const catalog = useCatalog(); const workspace = useWorkspace(); const catalogError = ref('');
const directory = computed<DirectoryContext>({ get: () => ({ module_code: editor.form.module_code, section_code: editor.form.section_code, subsection_code: editor.form.subsection_code || '' }), set: value => { Object.assign(editor.form, value); } });
const load = () => editor.load(String(route.params.id || ''));
async function save() { if (!catalog.valid(directory.value)) { editor.error.value = '请选择有效目录'; return; } if (await editor.save()) { workspace.invalidateArticles(); ElMessage.success('文章资料已保存，发布状态保持'); } }
async function changeStatus(command: 'publish' | 'unpublish' | 'archive' | 'restore') {
  if (editor.dirty.value) { try { await ElMessageBox.confirm('有未保存的修改，先保存资料再执行此操作？', '保存修改', { confirmButtonText: '保存并继续', cancelButtonText: '返回修改' }); } catch { return; } }
  if (command === 'archive') { try { await ElMessageBox.confirm('归档后文章移出前台，记录仍保留，可恢复为草稿。', '归档文章'); } catch { return; } }
  if (await editor.changeStatus(command, true)) { workspace.invalidateArticles(); ElMessage.success('状态已更新'); }
}
onBeforeRouteLeave(async () => { if (!editor.dirty.value) return true; try { await ElMessageBox.confirm('修改尚未保存，是否离开？', '未保存修改', { confirmButtonText: '离开', cancelButtonText: '继续编辑' }); return true; } catch { return false; } });
onMounted(async () => { await Promise.all([load(), catalog.load().catch(cause => { catalogError.value = errorMessage(cause, '加载目录失败'); })]); });
</script>
<style scoped>.article-form { padding:24px; max-width:1100px; } .toolbar { display:flex; flex-wrap:wrap; align-items:center; gap:8px; margin-bottom:16px; } .hint { color:var(--el-text-color-secondary); } .dirty { color:var(--el-color-warning); } .el-select { width:100%; } .historic-content { white-space:pre-wrap; word-break:break-word; }</style>
