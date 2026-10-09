<template>
  <div class="page-states">
    <el-empty v-if="!loading && !items.length" description="当前条件下没有同步页面" :image-size="72" />
    <template v-else>
      <div class="table-scroll"><el-table :data="items" empty-text="正在读取同步页面" class="pages-table">
        <el-table-column label="标题" min-width="230"><template #default="{ row }"><a v-if="safeURL(row.public_url || row.page_url)" :href="safeURL(row.public_url || row.page_url)" target="_blank" rel="noopener noreferrer">{{ row.title || row.page_id }}</a><span v-else>{{ row.title || row.page_id }}</span><router-link v-if="row.article_id" class="source-link" :to="articleURL(row.article_id)">查看博客资料</router-link></template></el-table-column>
        <el-table-column label="来源期望" min-width="110"><template #default="{ row }">{{ row.desired_state ? label(row.desired_state) : '未填写或无法识别' }}</template></el-table-column>
        <el-table-column label="本站状态" min-width="110"><template #default="{ row }">{{ localState(row) }}</template></el-table-column>
        <el-table-column label="实际公开" min-width="140"><template #default="{ row }"><el-tag :type="row.effective_visibility === true ? 'success' : 'info'">{{ visibility(row) }}</el-tag><p v-if="row.needs_revalidation" class="hint">等待重新核验</p></template></el-table-column>
        <el-table-column label="管理状态" min-width="140"><template #default="{ row }">{{ label(row.management_state) }}<el-tag v-if="row.publication_hold" type="danger">本地紧急下架</el-tag></template></el-table-column>
        <el-table-column label="公开条件 / 最近失败" min-width="240"><template #default="{ row }"><p>{{ reason(row.visibility_reason ?? row.publish_block_reason) }}</p><p v-if="row.last_error" class="hint">最近失败：{{ reason(row.last_error) }}</p><p v-if="row.management_state === 'baseline_pending'" class="hint">等待当前博客状态审核与回填确认</p></template></el-table-column>
      </el-table></div>
      <div class="page-cards"><article v-for="row in items" :key="row.page_id"><h3><a v-if="safeURL(row.public_url || row.page_url)" :href="safeURL(row.public_url || row.page_url)" target="_blank" rel="noopener noreferrer">{{ row.title || row.page_id }}</a><span v-else>{{ row.title || row.page_id }}</span></h3><dl><dt>来源期望</dt><dd>{{ row.desired_state ? label(row.desired_state) : '未填写或无法识别' }}</dd><dt>本站状态</dt><dd>{{ localState(row) }}</dd><dt>实际公开</dt><dd><el-tag :type="row.effective_visibility === true ? 'success' : 'info'">{{ visibility(row) }}</el-tag><p v-if="row.needs_revalidation" class="hint">等待重新核验</p></dd><dt>管理状态</dt><dd>{{ label(row.management_state) }}<el-tag v-if="row.publication_hold" type="danger">本地紧急下架</el-tag></dd></dl><p class="visibility-reason">{{ reason(row.visibility_reason ?? row.publish_block_reason) }}</p><p v-if="row.last_error" class="hint">最近失败：{{ reason(row.last_error) }}</p><p v-if="row.management_state === 'baseline_pending'" class="hint">等待当前博客状态审核与回填确认</p><router-link v-if="row.article_id" class="source-link" :to="articleURL(row.article_id)">查看博客资料</router-link></article></div>
    </template>
  </div>
</template>
<script setup lang="ts">
import type { SyncPage } from '@/types/notion-sync';
import { syncStateLabel as label, syncReasonLabel as reason } from '@/components/content/sync-presentation';
defineProps<{ items: SyncPage[]; loading: boolean }>();
function safeURL(value?: string) { try { const url = new URL(value || ''); return ['http:', 'https:'].includes(url.protocol) ? url.href : ''; } catch { return ''; } }
const articleURL = (id: string) => `/article/edit/${encodeURIComponent(id)}`;
const localState = (row: SyncPage) => row.local_state === null ? '尚无本站文章' : row.local_state ? label(row.local_state) : '尚未提供';
const visibility = (row: SyncPage) => row.effective_visibility === true ? '前台可见' : row.effective_visibility === false ? '前台不可见' : '尚未核验';
</script>
<style scoped>
.page-states { min-width:0; }
.table-scroll { max-width:100%; overflow-x:auto; }
.pages-table { min-width:970px; }
a { color:var(--admin-green,#086858); text-decoration:none; overflow-wrap:anywhere; }
a:hover { text-decoration:underline; }
p { margin:6px 0; line-height:1.65; overflow-wrap:anywhere; }
.hint { font-size:12px; color:var(--admin-muted,#59636e); }
.source-link { display:block; margin-top:8px; font-size:12px; }
.page-cards { display:none; }
@media(max-width:600px) {
  .table-scroll { display:none; }
  .page-cards { display:block; }
  article { border-top:1px solid var(--admin-line,#e1e6eb); padding:20px 4px; }
  article:first-child { padding-top:0; border:0; }
  article:last-child { padding-bottom:0; }
  h3 { font-size:14px; font-weight:600; line-height:1.65; margin:0 0 16px; overflow-wrap:anywhere; }
  dl { display:grid; grid-template-columns:70px minmax(0,1fr); gap:12px 10px; font-size:12px; }
  dt { color:var(--admin-muted,#59636e); }
  dd { margin:0; overflow-wrap:anywhere; }
  dd :deep(.el-tag) { margin:0 5px 5px 0; }
  dd .hint { margin:0; }
  .visibility-reason { border-top:1px solid var(--admin-line,#e1e6eb); margin-top:18px; padding-top:12px; font-size:12px; }
  .source-link { min-height:44px; display:flex; align-items:center; }
}
</style>
