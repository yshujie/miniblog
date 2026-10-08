<template>
  <div class="article-form">
    <div class="toolbar">
      <el-button v-if="canSave" type="primary" :loading="editor.loading.value" :disabled="!editor.article.value || editor.article.value.status === 'Deleted'" @click="save">{{ managed ? '保存本地信息' : '保存资料' }}</el-button>
      <el-button v-if="can('publish')" type="success" :disabled="!editor.article.value" :loading="editor.loading.value" @click="changeStatus('publish')">发布</el-button>
      <el-button v-if="can('unpublish')" type="warning" :loading="editor.loading.value" @click="changeStatus('unpublish')">下架</el-button>
      <el-button v-if="can('restore')" type="success" :loading="editor.loading.value" @click="changeStatus('restore')">恢复为草稿</el-button>
      <el-button v-if="can('archive')" :disabled="!editor.article.value" :loading="editor.loading.value" @click="changeStatus('archive')">归档</el-button>
      <el-button v-if="can('hold')" type="danger" :loading="editor.loading.value" @click="changeHold(true)">紧急下架</el-button>
      <el-button v-if="can('release_hold')" :loading="editor.loading.value" @click="changeHold(false)">解除本地下架</el-button>
      <el-tag v-if="managed">Notion 同步管理</el-tag><el-tag v-if="editor.article.value?.publication_hold?.held" type="danger">本地紧急下架</el-tag>
      <el-tag v-if="managed && editor.article.value?.effective_visibility === false && !editor.article.value?.publication_hold?.held" type="warning">前台暂不可用</el-tag>
      <router-link v-if="managed" to="/content/sync">查看同步状态</router-link>
      <el-tag v-if="editor.article.value">{{ statusLabels[editor.article.value.status] }}</el-tag>
      <el-button :disabled="editor.loading.value || editor.dirty.value" @click="load">重新读取</el-button>
      <span v-if="editor.dirty.value" class="dirty">有未保存的修改</span>
    </div>
    <p class="hint">{{ managed ? '标题、目录、标签、链接和发布状态由 Notion 管理；这里仅保存本地作者信息。排序仍在工作台调整。' : '保存标题、标签和归属会保留当前发布状态；正文继续在外部文档中维护。' }}</p>
    <p v-if="editor.article.value?.publication_hold?.held" class="hint">下架原因：{{ editor.article.value.publication_hold.reason || '未填写' }}。解除后需重新同步核验来源，再按来源和目录状态决定是否公开。</p>
    <el-alert v-if="editor.error.value || catalogError" :title="editor.error.value || catalogError" type="error" :closable="false"><template #default><el-button v-if="!editor.article.value" link @click="load">重试</el-button></template></el-alert>
    <el-form label-width="110px" :disabled="editor.loading.value || !editor.article.value || editor.article.value.status === 'Deleted'">
      <el-form-item label="标题" required><el-input :disabled="managed || !can('edit')" v-model="editor.form.title" maxlength="255" /></el-form-item>
      <el-form-item label="作者（可选）"><el-input :disabled="!canSave" v-model="editor.form.author" maxlength="128" /></el-form-item>
      <el-form-item label="归属目录" required><DirectoryPicker :disabled="managed || !can('edit')" active-only v-model="directory" /></el-form-item>
      <el-form-item label="标签（可选）"><el-select :disabled="managed || !can('edit')" v-model="editor.form.tags" multiple filterable allow-create default-first-option><el-option v-for="tag in editor.form.tags" :key="tag" :label="tag" :value="tag" /></el-select></el-form-item>
      <el-form-item label="来源文档"><a :href="editor.article.value ? sourceURL(editor.article.value) : ''" target="_blank" rel="noopener noreferrer">{{ editor.article.value ? sourceURL(editor.article.value) : editor.form.external_link }}</a></el-form-item>
      <el-form-item v-if="managed && editor.article.value && sourceURL(editor.article.value) !== editor.form.external_link" label="原始收录链接"><span>{{ editor.form.external_link }}</span></el-form-item>
    </el-form>
    <el-collapse v-if="editor.conflictDraft.value"><el-collapse-item title="查看接管前未保存的资料" name="conflict">
      <p>这些输入不会写入来源管理字段，可保留作为核对资料。</p>
      <dl><dt>标题</dt><dd>{{ editor.conflictDraft.value.title }}</dd><dt>作者</dt><dd>{{ editor.conflictDraft.value.author }}</dd><dt>标签</dt><dd>{{ editor.conflictDraft.value.tags.join('、') }}</dd><dt>目录</dt><dd>{{ catalog.label(editor.conflictDraft.value as DirectoryContext) || [editor.conflictDraft.value.module_code, editor.conflictDraft.value.section_code, editor.conflictDraft.value.subsection_code].filter(Boolean).join(' / ') }}</dd><dt>链接</dt><dd>{{ editor.conflictDraft.value.external_link }}</dd></dl>
      <el-button @click="copyConflictDraft">复制旧输入</el-button><el-button @click="editor.conflictDraft.value = undefined">清除这份旧输入</el-button>
    </el-collapse-item></el-collapse>
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
import { canArticleAction, isManagedArticle, sourceURL } from '@/utils/article-actions';
import { errorMessage } from '@/utils/api-error';
defineProps<{ isEdit: boolean }>();
const route = useRoute(); const editor = useArticleEditor(); const catalog = useCatalog(); const workspace = useWorkspace(); const catalogError = ref('');
const managed = computed(() => isManagedArticle(editor.article.value));
const can = (action: string) => canArticleAction(editor.article.value, action);
const canSave = computed(() => can(managed.value ? 'edit_local_fields' : 'edit'));
const directory = computed<DirectoryContext>({ get: () => ({ module_code: editor.form.module_code, section_code: editor.form.section_code, subsection_code: editor.form.subsection_code || '' }), set: value => { Object.assign(editor.form, value); } });
const load = () => editor.load(String(route.params.id || ''));
async function save() { if (!managed.value && !catalog.valid(directory.value)) { editor.error.value = '请选择有效目录'; return; } if (await editor.save()) { workspace.invalidateArticles(); ElMessage.success(managed.value ? '本地作者信息已保存' : '文章资料已保存，发布状态保持'); } }
async function changeStatus(command: 'publish' | 'unpublish' | 'archive' | 'restore') {
  if (editor.dirty.value) { try { await ElMessageBox.confirm('有未保存的修改，先保存资料再执行此操作？', '保存修改', { confirmButtonText: '保存并继续', cancelButtonText: '返回修改' }); } catch { return; } }
  if (command === 'archive') { try { await ElMessageBox.confirm('归档后文章移出前台，记录仍保留，可恢复为草稿。', '归档文章'); } catch { return; } }
  if (await editor.changeStatus(command, true)) { workspace.invalidateArticles(); ElMessage.success('状态已更新'); }
}
async function copyConflictDraft() {
  const draft = editor.conflictDraft.value; if (!draft) return;
  const text = `标题：${draft.title}\n作者：${draft.author}\n标签：${draft.tags.join('、')}\n目录：${[draft.module_code, draft.section_code, draft.subsection_code].filter(Boolean).join(' / ')}\n链接：${draft.external_link}`;
  try { await navigator.clipboard.writeText(text); ElMessage.success('旧输入已复制'); } catch { editor.error.value = '无法自动复制，请从上方资料中选择并复制。'; }
}
async function changeHold(held: boolean) {
  let reason: string | undefined;
  try {
    if (held) reason = (await ElMessageBox.prompt('只阻止 miniblog 阅读，不撤回外部原文权限。可填写原因。', '紧急下架', { inputPlaceholder: '可选原因', confirmButtonText: '确认下架' })).value;
    else await ElMessageBox.confirm('解除后需重新同步核验来源，再按来源及目录状态决定是否公开。是否继续？', '解除本地下架');
  } catch { return; }
  if (await editor.changeHold(held, reason)) { workspace.invalidateArticles(); ElMessage.success('本地下架状态已更新'); }
}
onBeforeRouteLeave(async () => { if (!editor.dirty.value) return true; try { await ElMessageBox.confirm('修改尚未保存，是否离开？', '未保存修改', { confirmButtonText: '离开', cancelButtonText: '继续编辑' }); return true; } catch { return false; } });
onMounted(async () => { await Promise.all([load(), catalog.load().catch(cause => { catalogError.value = errorMessage(cause, '加载目录失败'); })]); });
</script>
<style scoped>.article-form { padding:24px; max-width:1100px; } .toolbar { display:flex; flex-wrap:wrap; align-items:center; gap:8px; margin-bottom:16px; } .hint { color:var(--el-text-color-secondary); } .dirty { color:var(--el-color-warning); } .el-select { width:100%; } .historic-content { white-space:pre-wrap; word-break:break-word; }</style>
