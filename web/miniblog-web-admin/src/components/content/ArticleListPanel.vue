<template>
  <section class="article-panel" aria-label="文章列表">
    <header class="toolbar"><div><h2>{{ title || '文章库' }} <span class="article-count">{{ list.hasResult.value && !list.loading.value && !list.error.value ? list.total.value : '—' }}</span></h2><p>查看文章资料、发布状态和前台实际可见性。</p></div><div class="toolbar-actions"><el-button :loading="list.loading.value" @click="list.search()">刷新</el-button><el-button type="primary" :disabled="!contentRegistrationEnabled" @click="openCollect">收录文章</el-button></div></header>
    <el-alert v-if="catalogError" :title="catalogError" type="error" :closable="false" />
    <el-form label-position="top" class="filters" @submit.prevent="list.search(true)">
      <el-form-item v-if="directoryFilters" label="目录范围" class="directory-filter"><DirectoryPicker :model-value="filterContext" @update:model-value="changeDirectory" /></el-form-item>
      <el-form-item label="文章标题" class="title-filter"><el-input v-model="list.filters.title" aria-label="搜索文章标题" placeholder="搜索文章标题" clearable @keyup.enter="list.search(true)" @clear="list.search(true)" /></el-form-item>
      <el-form-item label="发布状态" class="status-filter"><el-select v-model="list.filters.status" aria-label="筛选发布状态" placeholder="全部状态" clearable @change="list.search(true)"><el-option v-for="(label, value) in statusLabels" :key="value" :label="label" :value="value" /></el-select></el-form-item>
      <div class="filter-actions"><el-button @click="list.search(true)">查询</el-button><el-button link @click="clearFilters">重置筛选</el-button></div>
      <div v-if="list.filters.section_code && !list.filters.subsection_code" class="include-children"><el-checkbox v-model="includeChildren" @change="list.search(true)">包含子章节</el-checkbox></div>
    </el-form>
    <el-alert v-if="list.error.value" :title="list.error.value" type="error" :closable="false" show-icon><template #default><el-button link @click="list.search()">重试</el-button></template></el-alert>
    <div class="desktop-list"><el-table :data="list.articles.value" v-loading="list.loading.value" empty-text="当前条件下没有文章" class="article-table">
      <el-table-column label="文章" min-width="230"><template #default="{ row }"><router-link class="article-title" :to="`/article/edit/${row.id}`">{{ row.title }}</router-link><div class="article-subline"><span>{{ isManagedArticle(row) ? 'Notion 自动同步' : '手工收录' }}</span><span v-if="row.author">{{ row.author }}</span><a v-if="sourceURL(row)" class="source-link" :href="sourceURL(row)" target="_blank" rel="noopener noreferrer">打开原文 ↗</a></div></template></el-table-column>
      <el-table-column label="所属目录" min-width="150"><template #default="{ row }"><span class="directory-label">{{ directoryLabel(row) }}</span></template></el-table-column>
      <el-table-column label="发布状态" width="110"><template #default="{ row }"><el-tag :type="row.status === 'Published' ? 'success' : row.status === 'Deleted' ? 'danger' : 'info'">{{ statusLabels[row.status] || '状态待确认' }}</el-tag><p v-if="row.publication_hold?.held" class="hold-label">本地紧急下架</p></template></el-table-column>
      <el-table-column label="前台可见性" width="130"><template #default="{ row }"><span :class="['visibility', { visible: row.effective_visibility === true }]">{{ visibilityLabel(row) }}</span></template></el-table-column>
      <el-table-column label="操作" width="135"><template #default="{ row }"><ArticleActions :article="row" :busy="!!busy" :can-reorder="canReorder" @edit="router.push(`/article/edit/${row.id}`)" @command="handleCommand(row, $event)" /></template></el-table-column>
    </el-table></div>
    <div class="mobile-list" v-loading="list.loading.value"><article v-for="row in list.articles.value" :key="row.id" class="article-card"><div class="card-heading"><el-tag :type="row.status === 'Published' ? 'success' : 'info'">{{ statusLabels[row.status] || '状态待确认' }}</el-tag><span class="source-badge">{{ isManagedArticle(row) ? 'Notion 自动同步' : '手工收录' }}</span></div><router-link class="article-title" :to="`/article/edit/${row.id}`">{{ row.title }}</router-link><p class="directory-label">{{ directoryLabel(row) }}</p><div class="card-meta"><span>{{ visibilityLabel(row) }}</span><span v-if="row.author">{{ row.author }}</span><span v-if="row.publication_hold?.held" class="hold-label">本地紧急下架</span></div><div class="card-footer"><ArticleActions :article="row" :busy="!!busy" :can-reorder="canReorder" @edit="router.push(`/article/edit/${row.id}`)" @command="handleCommand(row, $event)" /><a v-if="sourceURL(row)" class="source-link" :href="sourceURL(row)" target="_blank" rel="noopener noreferrer">打开原文 ↗</a></div></article><p v-if="!list.loading.value && !list.error.value && !list.articles.value.length" class="empty-list">当前条件下没有文章。调整筛选，或在此目录收录一篇。</p></div>
    <p v-if="!canReorder" class="hint">选择章节直属位置或子章节，并清空标题与状态筛选后可调整文章顺序。</p><p class="hint">排序仅调整当前完整位置中的未归档文章。前台可见性缺失时，请以实际阅读结果为准。</p>
    <div v-if="list.total.value" class="pagination"><label class="page-size">每页<el-select v-model="list.filters.limit" aria-label="每页文章数" @change="list.search(true)"><el-option v-for="size in [10,20,50,100]" :key="size" :value="size" :label="`${size} 篇`" /></el-select></label><el-pagination v-model:current-page="list.filters.page" :page-size="list.filters.limit" :total="list.total.value" :pager-count="5" layout="prev, pager, next" @current-change="list.search()" /></div>
    <el-drawer class="article-move-sheet" v-model="moveOpen" title="移动文章" size="min(580px, 100%)" :close-on-click-modal="false" :before-close="done => { if (!busy) done(); }"><p class="moving-title">{{ movingArticle?.title }}</p><p class="hint">移动只改变本地目录归属，保留文章资料和发布状态。</p><DirectoryPicker active-only v-model="moveTarget" /><template #footer><el-button :disabled="!!busy" @click="moveOpen = false">取消</el-button><el-button type="primary" :loading="!!busy" :disabled="!catalog.valid(moveTarget)" @click="submitMove">确定移动</el-button></template></el-drawer>
    <QuickCollectDrawer v-model="collectOpen" :context="filterContext" />
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import DirectoryPicker from './DirectoryPicker.vue';
import QuickCollectDrawer from './QuickCollectDrawer.vue';
import ArticleActions from './ArticleActions.vue';
import { useCatalog } from '@/composables/useCatalog';
import { useArticleList } from '@/composables/useArticleList';
import useWorkspace from '@/store/modules/contentWorkspace';
import { changeArticleStatus, fetchPositionArticles, moveArticle, reorderArticles, setPublicationHold } from '@/api/content';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { canArticleAction, isManagedArticle, sourceURL } from '@/utils/article-actions';
import { errorMessage } from '@/utils/api-error';
import { statusLabels, type ArticleInfo, type DirectoryContext } from '@/types/content';
const props = defineProps<{ context?: DirectoryContext; title?: string; directoryFilters?: boolean }>();
const router = useRouter(); const catalog = useCatalog(); const workspace = useWorkspace(); const list = useArticleList();
const collectOpen = ref(false); const catalogError = ref(''); const busy = ref(''); const moveOpen = ref(false); const movingArticle = ref<ArticleInfo>();
const moveTarget = ref<DirectoryContext>({ module_code: '', section_code: '', subsection_code: '' });
const filterContext = computed<DirectoryContext>(() => ({ module_code: list.filters.module_code || '', section_code: list.filters.section_code || '', subsection_code: list.filters.subsection_code || '' }));
const includeChildren = computed({ get: () => !list.filters.direct_only, set: value => { list.filters.direct_only = !value; } });
const canReorder = computed(() => Boolean(list.filters.section_code && (list.filters.subsection_code || list.filters.direct_only) && !list.filters.title && !list.filters.status));
watch(() => props.context, value => { if (value) Object.assign(list.filters, value, { direct_only: Boolean(value.section_code && !value.subsection_code) }); void list.search(true); }, { immediate: true, deep: true });
watch(() => workspace.articleRevision, () => { void list.search(); });
onMounted(async () => { try { await catalog.load(); } catch (cause) { catalogError.value = errorMessage(cause, '加载目录失败'); } });
function changeDirectory(value: DirectoryContext) { Object.assign(list.filters, value); list.filters.direct_only = Boolean(value.section_code && !value.subsection_code); void list.search(true); }
function clearFilters() { list.filters.title = ''; list.filters.status = ''; if (props.directoryFilters) Object.assign(list.filters, { module_code: '', section_code: '', subsection_code: '', direct_only: false }); void list.search(true); }
const openCollect = () => { collectOpen.value = true; };
defineExpose({ openCollect });
async function action(id: string, run: () => Promise<unknown>) {
  if (busy.value) return;
  busy.value = id;
  try { await run(); workspace.invalidateArticles(); ElMessage.success('操作成功'); } catch (cause) { ElMessage.error(errorMessage(cause)); } finally { busy.value = ''; }
}
async function changeStatus(row: ArticleInfo, command: 'publish' | 'unpublish' | 'archive' | 'restore') {
  if (!canArticleAction(row, command)) return;
  if (command === 'archive') { try { await ElMessageBox.confirm('归档后文章将移出前台，记录仍保留，可恢复为草稿。', '归档文章'); } catch { return; } }
  await action(row.id, () => changeArticleStatus(row.id, command));
}
const openMove = (row: ArticleInfo) => { if (!canArticleAction(row, 'move')) return; movingArticle.value = row; moveTarget.value = { module_code: row.module?.code || '', section_code: row.section?.code || '', subsection_code: row.subsection?.code || '' }; moveOpen.value = true; };
async function submitMove() {
  const article = movingArticle.value; if (!article || !canArticleAction(article, 'move') || !catalog.valid(moveTarget.value)) return;
  await action(article.id, async () => { await moveArticle(article.id, { section_code: moveTarget.value.section_code, subsection_code: moveTarget.value.subsection_code || undefined }); moveOpen.value = false; });
}
async function changeHold(row: ArticleInfo, held: boolean) {
  if (!canArticleAction(row, held ? 'hold' : 'release_hold')) return;
  let reason: string | undefined;
  try { if (held) reason = (await ElMessageBox.prompt('只阻止 miniblog 阅读，不撤回外部原文权限。可填写原因。', '紧急下架')).value; else await ElMessageBox.confirm('解除后需重新同步核验来源，再按来源和目录状态决定是否公开。', '解除本地下架'); } catch { return; }
  await action(row.id, () => setPublicationHold(row.id, { held, reason }));
}
async function shiftArticle(row: ArticleInfo, direction: -1 | 1) {
  const section = list.filters.section_code; if (!section || !canReorder.value || !canArticleAction(row, 'reorder')) return;
  const subsection = list.filters.subsection_code || '';
  await action(row.id, async () => {
    const items = await fetchPositionArticles(section, subsection); const ids = items.map(item => item.id); const index = ids.indexOf(row.id); const target = index + direction;
    if (index < 0 || target < 0 || target >= ids.length) return;
    [ids[index], ids[target]] = [ids[target], ids[index]];
    await reorderArticles({ section_code: section, subsection_code: subsection || undefined, article_ids: ids });
  });
}
const directoryLabel = (row: ArticleInfo) => [row.module?.title, row.section?.title, row.subsection?.title || (row.section ? '直属章节' : '')].filter(Boolean).join(' / ') || '目录尚未提供';
const visibilityLabel = (row: ArticleInfo) => row.effective_visibility === true ? '前台可见' : row.effective_visibility === false ? '前台不可见' : '尚未提供';
function handleCommand(row: ArticleInfo, command: string) {
  if (command === 'move') openMove(row);
  else if (['publish', 'unpublish', 'archive', 'restore'].includes(command)) void changeStatus(row, command as 'publish'|'unpublish'|'archive'|'restore');
  else if (command === 'hold' || command === 'release_hold') void changeHold(row, command === 'hold');
  else if (command === 'up' || command === 'down') void shiftArticle(row, command === 'up' ? -1 : 1);
}
</script>
<style scoped>
.article-panel { min-width:0; } .toolbar { display:flex; justify-content:space-between; align-items:center; gap:16px; margin-bottom:20px; } h2 { font-size:20px; color:var(--admin-ink); margin:0; } .article-count { display:inline-flex; font-size:12px; color:var(--admin-muted); background:var(--admin-pale); padding:3px 8px; border-radius:20px; vertical-align:middle; margin-left:8px; } .toolbar p { color:var(--admin-muted); font-size:13px; margin:8px 0 0; } .toolbar-actions { display:flex; gap:8px; flex-shrink:0; } .toolbar-actions :deep(.el-button + .el-button) { margin-left:0; } .filters { display:flex; flex-wrap:wrap; align-items:flex-end; gap:12px; padding:18px; border:1px solid var(--admin-line); border-radius:8px; background:white; margin-bottom:18px; } .filters :deep(.el-form-item) { margin:0; min-width:0; } .title-filter { flex:1; min-width:180px!important; } .status-filter { width:150px; } .directory-filter { flex-basis:100%; width:100%; } .directory-filter :deep(.directory-picker) { width:100%; } .filter-actions { display:flex; gap:8px; } .include-children { width:100%; } .article-table { border:1px solid var(--admin-line); border-radius:8px; } .article-table :deep(.el-table__cell) { padding:16px 0; } .article-table :deep(th.el-table__cell) { background:var(--admin-canvas); color:var(--admin-muted); font-size:12px; font-weight:500; } .article-title { color:var(--admin-ink); font-weight:600; line-height:1.65; overflow-wrap:anywhere; } .article-title:hover { color:var(--admin-green); } .article-subline { display:flex; flex-wrap:wrap; gap:10px; color:var(--admin-muted); font-size:12px; margin-top:6px; } .source-link { font-size:12px; color:var(--admin-muted); text-decoration:none; } .directory-label { color:var(--admin-muted); font-size:12px; overflow-wrap:anywhere; line-height:1.7; } .visibility { color:var(--admin-muted); font-size:12px; } .visibility.visible { color:var(--admin-green); } .hold-label { color:#a34f39; font-size:12px; margin:6px 0 0; } .hint { color:var(--admin-muted); font-size:12px; line-height:1.7; margin:12px 0 0; } .pagination { display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:12px; margin-top:20px; } .page-size { display:flex; align-items:center; gap:8px; font-size:12px; color:var(--admin-muted); } .page-size .el-select { width:95px; } .mobile-list { display:none; } .moving-title { overflow-wrap:anywhere; font-weight:600; } @media(max-width:1000px) { .toolbar { align-items:flex-start; flex-wrap:wrap; } } @media(max-width:780px) { .filters { padding:14px; gap:10px; } .toolbar { gap:12px; } .toolbar-actions { width:100%; justify-content:flex-end; } .title-filter { flex-basis:100%; } .status-filter { flex:1; } .filter-actions { gap:4px; } .pagination :deep(.el-pagination button),.pagination :deep(.el-pager li) { min-height:44px; min-width:40px; } } @media(max-width:600px) { .desktop-list { display:none; } .mobile-list { display:block; } .article-card { padding:16px; background:white; border:1px solid var(--admin-line); border-radius:8px; margin-bottom:12px; } .card-heading { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:12px; } .source-badge,.card-meta { font-size:12px; color:var(--admin-muted); } .card-meta { display:flex; flex-wrap:wrap; gap:12px; margin-bottom:12px; } .card-meta .hold-label { margin:0; } .card-footer { display:flex; align-items:center; justify-content:space-between; gap:12px; padding-top:8px; border-top:1px solid var(--admin-line); } .card-footer .source-link { display:flex; align-items:center; min-height:44px; flex-shrink:0; } .empty-list { color:var(--admin-muted); padding:32px 16px; text-align:center; line-height:1.8; } .pagination { justify-content:center; } }
@media(max-width:780px) { :global(.article-move-sheet) { width:100%!important; } }
</style>
