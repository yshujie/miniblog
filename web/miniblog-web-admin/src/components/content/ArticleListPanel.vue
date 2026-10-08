<template>
  <div class="article-panel">
    <div class="toolbar"><h3>{{ title || '文章管理' }}</h3><div><el-button @click="list.search()">刷新</el-button><el-button type="primary" :disabled="!contentRegistrationEnabled" @click="openCollect">收录文章</el-button></div></div>
    <el-alert v-if="catalogError" :title="catalogError" type="error" :closable="false" />
    <el-form inline class="filters" @submit.prevent="list.search(true)">
      <el-form-item v-if="directoryFilters" label="目录"><DirectoryPicker :model-value="filterContext" @update:model-value="changeDirectory" /></el-form-item>
      <el-form-item label="标题"><el-input v-model="list.filters.title" placeholder="搜索标题" clearable @keyup.enter="list.search(true)" @clear="list.search(true)" /></el-form-item>
      <el-form-item label="状态"><el-select v-model="list.filters.status" placeholder="全部状态" clearable @change="list.search(true)"><el-option v-for="(label, value) in statusLabels" :key="value" :label="label" :value="value" /></el-select></el-form-item>
      <el-form-item v-if="list.filters.section_code && !list.filters.subsection_code"><el-checkbox v-model="includeChildren" @change="list.search(true)">包含子章节</el-checkbox></el-form-item>
      <el-form-item><el-button type="primary" @click="list.search(true)">查询</el-button><el-button @click="clearFilters">重置筛选</el-button></el-form-item>
    </el-form>
    <el-alert v-if="list.error.value" :title="list.error.value" type="error" :closable="false" show-icon><template #default><el-button link @click="list.search()">重试</el-button></template></el-alert>
    <el-table :data="list.articles.value" v-loading="list.loading.value" border empty-text="当前条件下没有文章">
      <el-table-column label="标题" min-width="240"><template #default="{ row }"><router-link :to="`/article/edit/${row.id}`">{{ row.title }}</router-link><a v-if="sourceURL(row)" class="source-link" :href="sourceURL(row)" target="_blank" rel="noopener noreferrer">原文 ↗</a></template></el-table-column>
      <el-table-column label="目录" min-width="190"><template #default="{ row }">{{ row.module?.title }} / {{ row.section?.title }} / {{ row.subsection?.title || '直属章节' }}</template></el-table-column>
      <el-table-column prop="author" label="作者" width="120" />
      <el-table-column label="标签" min-width="140"><template #default="{ row }"><el-tag v-for="tag in row.tags" :key="tag" class="tag">{{ tag }}</el-tag></template></el-table-column>
      <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag :type="row.status === 'Published' ? 'success' : row.status === 'Deleted' ? 'danger' : 'info'">{{ statusLabels[row.status as ArticleStatus] }}</el-tag><el-tag v-if="isManagedArticle(row)">Notion 同步</el-tag><el-tag v-if="row.publication_hold?.held" type="danger">本地紧急下架</el-tag></template></el-table-column>
      <el-table-column label="操作" width="390" fixed="right"><template #default="{ row }">
        <el-button link type="primary" @click="router.push(`/article/edit/${row.id}`)">{{ isManagedArticle(row) ? '查看 / 本地信息' : '编辑' }}</el-button>
        <el-button v-if="canArticleAction(row, 'move')" link @click="openMove(row)">移动</el-button>
        <el-button v-if="canArticleAction(row, 'publish')" link type="success" :disabled="!!busy" @click="changeStatus(row, 'publish')">发布</el-button>
        <el-button v-if="canArticleAction(row, 'unpublish')" link type="warning" :disabled="!!busy" @click="changeStatus(row, 'unpublish')">下架</el-button>
        <el-button v-if="canArticleAction(row, 'archive')" link type="danger" :disabled="!!busy" @click="changeStatus(row, 'archive')">归档</el-button>
        <el-button v-if="canArticleAction(row, 'restore')" link type="success" :disabled="!!busy" @click="changeStatus(row, 'restore')">恢复草稿</el-button>
        <el-button v-if="canArticleAction(row, 'hold')" link type="danger" :disabled="!!busy" @click="changeHold(row, true)">紧急下架</el-button>
        <el-button v-if="canArticleAction(row, 'release_hold')" link :disabled="!!busy" @click="changeHold(row, false)">解除本地下架</el-button>
        <el-button v-if="canReorder && canArticleAction(row, 'reorder')" link :disabled="!!busy" @click="shiftArticle(row, -1)">上移</el-button>
        <el-button v-if="canReorder && canArticleAction(row, 'reorder')" link :disabled="!!busy" @click="shiftArticle(row, 1)">下移</el-button>
      </template></el-table-column>
    </el-table>
    <p v-if="!canReorder" class="hint">选择章节直属位置或子章节，并清空标题与状态筛选后可调整文章顺序。</p>
    <p class="hint">排序只调整未归档文章。</p>
    <el-pagination v-if="list.total.value" class="pagination" v-model:current-page="list.filters.page" v-model:page-size="list.filters.limit" :total="list.total.value" :page-sizes="[10,20,50,100]" layout="total, sizes, prev, pager, next" @current-change="list.search()" @size-change="list.search(true)" />
    <el-dialog v-model="moveOpen" title="移动文章" width="min(680px, 95%)"><p>{{ movingArticle?.title }}</p><DirectoryPicker active-only v-model="moveTarget" /><template #footer><el-button @click="moveOpen = false">取消</el-button><el-button type="primary" :loading="!!busy" :disabled="!catalog.valid(moveTarget)" @click="submitMove">确定移动</el-button></template></el-dialog>
    <QuickCollectDrawer v-model="collectOpen" :context="filterContext" />
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import DirectoryPicker from './DirectoryPicker.vue';
import QuickCollectDrawer from './QuickCollectDrawer.vue';
import { useCatalog } from '@/composables/useCatalog';
import { useArticleList } from '@/composables/useArticleList';
import useWorkspace from '@/store/modules/contentWorkspace';
import { changeArticleStatus, fetchPositionArticles, moveArticle, reorderArticles, setPublicationHold } from '@/api/content';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { canArticleAction, isManagedArticle, sourceURL } from '@/utils/article-actions';
import { errorMessage } from '@/utils/api-error';
import { statusLabels, type ArticleInfo, type ArticleStatus, type DirectoryContext } from '@/types/content';
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
const openMove = (row: ArticleInfo) => { if (!canArticleAction(row, 'move')) return; movingArticle.value = row; moveTarget.value = { module_code: row.module.code, section_code: row.section.code, subsection_code: row.subsection?.code || '' }; moveOpen.value = true; };
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
</script>
<style scoped>.toolbar { display:flex; align-items:center; justify-content:space-between; gap:12px; } h3 { margin: 4px 0 16px; } .filters .el-select { min-width:140px; } .source-link { display:block; font-size:12px; color:var(--el-text-color-secondary); margin-top:6px; } .tag { margin:2px; } .hint { font-size:12px; color:var(--el-text-color-secondary); } .pagination { margin-top:18px; }</style>
