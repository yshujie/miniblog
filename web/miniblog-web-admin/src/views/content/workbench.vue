<template>
  <div class="workbench-page">
    <header class="page-heading"><div><p class="eyebrow">内容工作台</p><h1>目录与收录</h1><p>整理阅读路径，在合适的位置收录文章。</p></div><el-button @click="contextPanel?.openCreate('module')">新增主题</el-button></header>
    <SyncHealthBar />
    <el-alert v-if="!contentRegistrationEnabled" title="收录功能暂未启用，目录和已有文章仍可管理。" type="info" :closable="false" />
    <div class="workbench">
      <aside class="catalog desktop-catalog" aria-label="内容目录"><div class="catalog-title"><h2>内容目录</h2><el-button link :loading="loading" @click="loadCatalog(true)">刷新</el-button></div><el-alert v-if="error" :title="error" type="error" :closable="false" />
        <el-button class="all-content" :class="{ selected: !selectedNode }" @click="selectAll">全部文章</el-button>
        <el-tree v-loading="loading" :data="catalog.tree.value" node-key="key" :props="{ label: 'label', children: 'children' }" default-expand-all highlight-current :current-node-key="selectedNode?.key" @node-click="selectNode" empty-text="暂无目录，请先创建主题和章节"><template #default="{ data }"><span class="node-label" :title="data.label">{{ data.label }}</span></template></el-tree>
        <div class="catalog-actions"><el-button :disabled="!selectedNode || orderBusy || !canMove(-1)" @click="moveNode(-1)">上移</el-button><el-button :disabled="!selectedNode || orderBusy || !canMove(1)" @click="moveNode(1)">下移</el-button></div><p class="order-note">按完整同级目录调整，包含未上架目录。</p>
        <details class="legacy-links"><summary>兼容管理入口</summary><router-link to="/module/list">主题管理</router-link><router-link to="/section/list">章节管理</router-link><router-link to="/subsection/list">子章节管理</router-link></details>
      </aside>
      <section class="workspace-content"><el-button class="mobile-directory" @click="directoryOpen = true">选择目录 <span class="current-path">{{ catalog.label(selected) || '全部文章' }}</span></el-button><CatalogContextPanel ref="contextPanel" :node="selectedNode" @changed="catalogChanged" /><ArticleListPanel ref="panel" :context="selected" :title="selectedNode ? '此目录的文章' : '全部文章'" /></section>
    </div>
    <el-drawer v-model="directoryOpen" title="选择内容目录" size="min(360px, 100%)" class="directory-sheet"><el-alert v-if="error" :title="error" type="error" :closable="false" /><el-button class="all-content" @click="selectAll">全部文章</el-button><el-tree :data="catalog.tree.value" node-key="key" :props="{ label:'label',children:'children' }" default-expand-all highlight-current :current-node-key="selectedNode?.key" @node-click="selectNode" empty-text="暂无目录"><template #default="{ data }"><span class="node-label">{{ data.label }}</span></template></el-tree><template #footer><el-button @click="directoryOpen = false">完成</el-button></template></el-drawer>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import SyncHealthBar from '@/components/content/SyncHealthBar.vue';
import ArticleListPanel from '@/components/content/ArticleListPanel.vue';
import CatalogContextPanel from '@/components/content/CatalogContextPanel.vue';
import { useCatalog, type DirectoryNode } from '@/composables/useCatalog';
import useWorkspace from '@/store/modules/contentWorkspace';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { errorMessage } from '@/utils/api-error';
import type { DirectoryContext } from '@/types/content';
const route = useRoute(); const catalog = useCatalog(); const workspace = useWorkspace();
const queryContext = (): DirectoryContext => ({ module_code: typeof route.query.module_code === 'string' ? route.query.module_code : '', section_code: typeof route.query.section_code === 'string' ? route.query.section_code : '', subsection_code: typeof route.query.subsection_code === 'string' ? route.query.subsection_code : '' });
const selected = ref<DirectoryContext>(queryContext()); const loading = ref(false); const error = ref(''); const orderBusy = ref(false); const directoryOpen = ref(false); const panel = ref<InstanceType<typeof ArticleListPanel>>(); const contextPanel = ref<InstanceType<typeof CatalogContextPanel>>(); let ready = false;
const nodes = computed(() => catalog.tree.value.flatMap(module => [module, ...(module.children || []).flatMap(section => [section, ...(section.children || [])])]));
const selectedNode = computed(() => nodes.value.find(node => node.module_code === selected.value.module_code && node.section_code === selected.value.section_code && node.subsection_code === selected.value.subsection_code));
function selectNode(node: DirectoryNode) { selected.value = { module_code: node.module_code, section_code: node.section_code, subsection_code: node.subsection_code }; if (catalog.valid(selected.value)) workspace.remember(selected.value); directoryOpen.value = false; }
function selectAll() { selected.value = { module_code: '', section_code: '', subsection_code: '' }; directoryOpen.value = false; }
async function loadCatalog(force = false) { loading.value = true; error.value = ''; try { await catalog.load(force); } catch (cause) { error.value = errorMessage(cause); } finally { loading.value = false; } }
function canMove(direction: -1 | 1) { const node = selectedNode.value; if (!node) return false; const siblings = catalog.siblings(node); const target = siblings.findIndex(item => item.code === node.code) + direction; return target >= 0 && target < siblings.length; }
async function moveNode(direction: -1 | 1) { const node = selectedNode.value; if (!node || orderBusy.value) return; orderBusy.value = true; try { await catalog.move(node, direction); } catch (cause) { ElMessage.error(errorMessage(cause)); } finally { orderBusy.value = false; } }
async function catalogChanged(context: DirectoryContext, deleted = false) { selected.value = deleted ? (context.subsection_code ? { ...context, subsection_code: '' } : context.section_code ? { ...context, section_code: '', subsection_code: '' } : { module_code: '', section_code: '', subsection_code: '' }) : { module_code: context.module_code, section_code: context.section_code, subsection_code: context.subsection_code }; workspace.invalidateArticles(); await loadCatalog(true); }
watch(() => [route.query.module_code, route.query.section_code, route.query.subsection_code, route.query.collect], async (value, previous) => {
  if (value.slice(0, 3).some((item, index) => item !== previous[index])) selected.value = queryContext();
  if (ready && route.query.collect === '1' && contentRegistrationEnabled) { await nextTick(); panel.value?.openCollect(); }
});
onMounted(async () => { await loadCatalog(); ready = true; if (!selected.value.module_code && !selected.value.section_code) { const recent = workspace.recentDirectories.find(catalog.valid); if (recent) selected.value = { ...recent }; } if (route.query.collect === '1' && contentRegistrationEnabled) { await nextTick(); panel.value?.openCollect(); } });
</script>
<style scoped>
.page-heading { display:flex; justify-content:space-between; gap:20px; align-items:center; margin-bottom:24px; } .eyebrow { font-size:12px; color:var(--admin-muted); margin:0 0 6px; } h1 { color:var(--admin-ink); font-size:26px; letter-spacing:-1px; margin:0; } .page-heading p:last-child { color:var(--admin-muted); margin:10px 0 0; } .workbench { display:grid; grid-template-columns:240px minmax(0,1fr); gap:24px; margin-top:24px; } .catalog { background:white; border:1px solid var(--admin-line); border-radius:8px; padding:18px 12px; align-self:start; min-width:0; } .catalog-title { display:flex; align-items:center; justify-content:space-between; padding:0 6px; } .catalog-title h2 { font-size:15px; margin:0; } .all-content { width:100%; justify-content:flex-start; margin:12px 0 8px; border:0; background:var(--admin-canvas); } .all-content.selected { color:var(--admin-green); background:var(--admin-pale); } .node-label { white-space:normal; overflow-wrap:anywhere; font-size:13px; line-height:1.6; padding:9px 4px 9px 0; } .catalog :deep(.el-tree-node__content),.directory-sheet :deep(.el-tree-node__content) { min-height:44px; height:auto; align-items:flex-start; } :deep(.el-tree-node__expand-icon) { margin-top:8px; } :deep(.el-tree--highlight-current .el-tree-node.is-current > .el-tree-node__content) { background:var(--admin-pale); color:var(--admin-green); border-radius:6px; } .catalog-actions { display:flex; gap:8px; margin-top:20px; } .catalog-actions :deep(.el-button + .el-button) { margin-left:0; } .order-note,.legacy-links { font-size:12px; color:var(--admin-muted); line-height:1.7; } .legacy-links { margin-top:22px; } .legacy-links summary { cursor:pointer; } .legacy-links a { display:block; padding:6px 0; } .workspace-content { min-width:0; } .mobile-directory { display:none; } @media(max-width:1200px) { .workbench { display:block; } .desktop-catalog { display:none; } .mobile-directory { display:flex; width:100%; height:auto; min-height:44px; justify-content:space-between; white-space:normal; margin-bottom:16px; text-align:left; gap:12px; } .current-path { overflow-wrap:anywhere; font-size:12px; color:var(--admin-muted); } } @media(max-width:780px) { .page-heading { align-items:flex-start; gap:12px; margin-bottom:18px; } h1 { font-size:24px; } .page-heading p:last-child { font-size:13px; } .workbench { display:block; margin-top:16px; } .desktop-catalog { display:none; } .mobile-directory { display:flex; width:100%; height:auto; min-height:44px; justify-content:space-between; white-space:normal; margin-bottom:16px; text-align:left; gap:12px; } .current-path { overflow-wrap:anywhere; font-size:12px; color:var(--admin-muted); } }
@media(max-width:780px) { :global(.directory-sheet) { width:100%!important; } }
</style>
