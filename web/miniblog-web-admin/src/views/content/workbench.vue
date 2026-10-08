<template>
  <div class="workbench">
    <aside class="catalog">
      <div class="catalog-title"><h3>内容目录</h3><el-button link :loading="loading" @click="loadCatalog(true)">刷新</el-button></div>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <el-tree v-loading="loading" :data="catalog.tree.value" node-key="key" :props="{ label: 'label', children: 'children' }" default-expand-all highlight-current :current-node-key="selectedNode?.key" @node-click="selectNode" empty-text="暂无目录，请先创建模块和章节" />
      <div class="catalog-actions"><el-button size="small" :disabled="!selectedNode || orderBusy || !canMove(-1)" @click="moveNode(-1)">目录上移</el-button><el-button size="small" :disabled="!selectedNode || orderBusy || !canMove(1)" @click="moveNode(1)">目录下移</el-button></div>
      <div class="catalog-links"><router-link to="/module/list">模块管理</router-link><router-link to="/section/list">章节管理</router-link><router-link to="/subsection/list">子章节管理</router-link></div>
    </aside>
    <main><el-alert v-if="!contentRegistrationEnabled" title="收录功能暂未启用，目录和已有文章仍可管理。" type="info" :closable="false" /><ArticleListPanel ref="panel" :context="selected" :title="catalog.label(selected) || '全部文章'" /></main>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import ArticleListPanel from '@/components/content/ArticleListPanel.vue';
import { useCatalog, type DirectoryNode } from '@/composables/useCatalog';
import useWorkspace from '@/store/modules/contentWorkspace';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { errorMessage } from '@/utils/api-error';
import type { DirectoryContext } from '@/types/content';
const route = useRoute(); const catalog = useCatalog(); const workspace = useWorkspace();
const selected = ref<DirectoryContext>({ module_code: typeof route.query.module_code === 'string' ? route.query.module_code : '', section_code: typeof route.query.section_code === 'string' ? route.query.section_code : '', subsection_code: typeof route.query.subsection_code === 'string' ? route.query.subsection_code : '' });
const loading = ref(false); const error = ref(''); const orderBusy = ref(false); const panel = ref<InstanceType<typeof ArticleListPanel>>();
const nodes = computed(() => catalog.tree.value.flatMap(module => [module, ...(module.children || []).flatMap(section => [section, ...(section.children || [])])]));
const selectedNode = computed(() => nodes.value.find(node => node.module_code === selected.value.module_code && node.section_code === selected.value.section_code && node.subsection_code === selected.value.subsection_code));
function selectNode(node: DirectoryNode) { selected.value = { module_code: node.module_code, section_code: node.section_code, subsection_code: node.subsection_code }; if (catalog.valid(selected.value)) workspace.remember(selected.value); }
async function loadCatalog(force = false) { loading.value = true; error.value = ''; try { await catalog.load(force); } catch (cause) { error.value = errorMessage(cause); } finally { loading.value = false; } }
function canMove(direction: -1 | 1) { const node = selectedNode.value; if (!node) return false; const siblings = catalog.siblings(node); const target = siblings.findIndex(item => item.code === node.code) + direction; return target >= 0 && target < siblings.length; }
async function moveNode(direction: -1 | 1) { const node = selectedNode.value; if (!node) return; orderBusy.value = true; try { await catalog.move(node, direction); } catch (cause) { ElMessage.error(errorMessage(cause)); } finally { orderBusy.value = false; } }
onMounted(async () => {
  await loadCatalog();
  if (!selected.value.module_code && !selected.value.section_code) { const recent = workspace.recentDirectories.find(catalog.valid); if (recent) selected.value = { ...recent }; }
  if (route.query.collect === '1' && contentRegistrationEnabled) { await nextTick(); panel.value?.openCollect(); }
});
</script>
<style scoped>.workbench { display:grid; grid-template-columns:260px minmax(0,1fr); gap:24px; padding:20px; } .catalog { border-right:1px solid var(--el-border-color); padding-right:18px; } .catalog-title { display:flex; align-items:center; justify-content:space-between; } .catalog-actions { display:flex; flex-wrap:wrap; gap:8px; margin-top:18px; } .catalog-links { display:flex; flex-wrap:wrap; gap:12px; margin-top:18px; font-size:13px; } @media(max-width:900px) { .workbench { grid-template-columns:1fr; } .catalog { border-right:0; border-bottom:1px solid var(--el-border-color); padding-bottom:16px; } }</style>
