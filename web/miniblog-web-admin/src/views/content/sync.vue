<template>
  <div class="sync-page">
    <div class="toolbar"><h2>Notion 自动同步</h2><el-button :loading="sync.loading.value" @click="refreshAll">刷新状态</el-button></div>
    <p class="hint">同步标题、主题、标签、链接和发布状态；正文仍从外部文档阅读。作者、历史正文、排序和本地下架由博客保留。</p>
    <el-alert v-if="sync.error.value" :title="sync.error.value" type="error" :closable="false"><template #default><el-button link @click="sync.refresh">重新读取</el-button></template></el-alert>
    <section v-if="sync.status.value" class="health">
      <el-tag>{{ label(sync.status.value.health) }}</el-tag><el-tag v-if="!sync.status.value.enabled" type="info">服务未启用</el-tag><el-tag v-if="sync.status.value.paused" type="warning">同步已暂停</el-tag><el-tag v-if="sync.status.value.source_writes_paused" type="warning">来源回填已暂停</el-tag>
      <p>最近完整扫描：{{ time(sync.status.value.last_complete_scan_at) }} · 最近成功：{{ time(sync.status.value.last_success_at) }} · 待处理 {{ sync.status.value.pending_count }} · 错误 {{ sync.status.value.error_count }}</p>
      <p v-if="sync.status.value.current_run_id">正在运行：<el-button link @click="openRun(sync.status.value.current_run_id!)">{{ sync.status.value.current_run_id }}</el-button></p>
      <div class="actions">
        <el-button :disabled="sync.actionBusy.value || !sync.status.value.enabled || !!sync.status.value.current_run_id" @click="start('dry_run')">运行预览</el-button>
        <el-button type="primary" :disabled="sync.actionBusy.value || !sync.status.value.enabled || sync.status.value.paused || !!sync.status.value.current_run_id" @click="start('sync')">手动同步</el-button>
        <el-button :disabled="sync.actionBusy.value" @click="sync.control({ paused: !sync.status.value.paused })">{{ sync.status.value.paused ? '恢复同步' : '暂停同步' }}</el-button>
        <el-button :disabled="sync.actionBusy.value" @click="sync.control({ source_writes_paused: !sync.status.value.source_writes_paused })">{{ sync.status.value.source_writes_paused ? '允许来源回填' : '暂停来源回填' }}</el-button>
      </div>
      <p v-if="!sync.status.value.enabled" class="hint">总开关由服务配置控制，此处不能开启服务。</p>
    </section>
    <h3>来源与主题绑定</h3>
    <p class="hint">全新主题自动创建章节。遇到同名冲突或失效绑定，请选择正确章节后重新预览。历史文章需通过受控命令审核当前博客状态并回填，网页不会执行接管写入。</p>
    <el-empty v-if="!sync.loading.value && !sync.sources.value.items.length" description="暂无已配置来源" />
    <section v-for="source in sync.sources.value.items" :key="source.source_id" class="source-card">
      <div class="toolbar"><strong>{{ source.label || source.source_id }}</strong><div><el-tag>{{ label(source.health) }}</el-tag> <el-tag v-if="!source.enabled">停用</el-tag> <el-button link @click="editSource(source)">配置</el-button></div></div>
      <p>模块 {{ source.module_code }} · 最近尝试 {{ time(source.last_attempt_at) }} · 完整扫描 {{ time(source.last_complete_scan_at) }} · 成功 {{ time(source.last_success_at) }}</p>
      <el-alert v-if="source.last_error" :title="source.last_error" type="error" :closable="false" />
      <el-table :data="source.catalog_bindings || []" empty-text="扫描后显示主题绑定" size="small">
        <el-table-column prop="option_name" label="Notion 主题" /><el-table-column prop="section_code" label="章节" />
        <el-table-column label="状态"><template #default="{ row }">{{ label(row.status) }}<p class="hint">{{ row.reason }}</p></template></el-table-column>
        <el-table-column label="操作" width="120"><template #default="{ row }"><el-button link @click="editBinding(source, row)">调整绑定</el-button></template></el-table-column>
      </el-table>
    </section>
    <el-pagination v-if="sync.sources.value.total > sync.sourceQuery.limit" v-model:current-page="sync.sourceQuery.page" :page-size="sync.sourceQuery.limit" :total="sync.sources.value.total" layout="prev, pager, next" @current-change="sync.refresh" />
    <h3>页面同步情况</h3>
    <el-form inline @submit.prevent="searchPages">
      <el-form-item label="标题"><el-input v-model="sync.pageFilters.title" clearable @keyup.enter="searchPages" /></el-form-item>
      <el-form-item label="来源"><el-select v-model="sync.pageFilters.source_id" clearable placeholder="全部来源"><el-option v-for="source in sync.status.value?.sources || sync.sources.value.items" :key="source.source_id" :label="source.label || source.source_id" :value="source.source_id" /></el-select></el-form-item>
      <el-form-item label="管理状态"><el-select v-model="sync.pageFilters.management_state" clearable placeholder="全部"><el-option v-for="state in ['baseline_pending', 'managed', 'detached']" :key="state" :label="label(state)" :value="state" /></el-select></el-form-item>
      <el-button @click="searchPages">查询</el-button>
    </el-form>
    <el-table :data="sync.pages.value.items" empty-text="当前条件下没有同步页面">
      <el-table-column label="标题" min-width="220"><template #default="{ row }"><a v-if="safeURL(row.public_url || row.page_url)" :href="safeURL(row.public_url || row.page_url)" target="_blank" rel="noopener noreferrer">{{ row.title || row.page_id }}</a><span v-else>{{ row.title || row.page_id }}</span><router-link v-if="row.article_id" class="source-link" :to="`/article/edit/${row.article_id}`">查看博客资料</router-link></template></el-table-column>
      <el-table-column label="来源状态"><template #default="{ row }">{{ label(row.desired_state) }}</template></el-table-column>
      <el-table-column label="管理状态"><template #default="{ row }">{{ label(row.management_state) }}<el-tag v-if="row.publication_hold" type="danger">本地紧急下架</el-tag></template></el-table-column>
      <el-table-column label="阻止原因 / 最近错误" min-width="240"><template #default="{ row }">{{ row.publish_block_reason || row.last_error || '—' }}<p v-if="row.management_state === 'baseline_pending'" class="hint">等待当前博客状态审核与回填确认</p></template></el-table-column>
    </el-table>
    <el-pagination v-model:current-page="sync.pageFilters.page" :page-size="sync.pageFilters.limit" :total="sync.pages.value.total" layout="total, prev, pager, next" @current-change="sync.loadPages" />
    <h3>运行历史</h3>
    <el-table :data="sync.runs.value.items" empty-text="暂无运行记录">
      <el-table-column label="运行"><template #default="{ row }"><el-button link @click="openRun(row.run_id)">{{ row.run_id }}</el-button></template></el-table-column>
      <el-table-column label="模式"><template #default="{ row }">{{ row.mode === 'dry_run' ? '预览' : '同步' }}</template></el-table-column>
      <el-table-column label="状态"><template #default="{ row }">{{ label(row.status) }} / {{ label(row.phase) }}</template></el-table-column>
      <el-table-column label="开始时间"><template #default="{ row }">{{ time(row.started_at) }}</template></el-table-column>
      <el-table-column label="结果" min-width="240"><template #default="{ row }">发现 {{ row.counts?.seen || 0 }} · 新增 {{ row.counts?.created || 0 }} · 更新 {{ row.counts?.updated || 0 }} · 阻止 {{ row.counts?.blocked || 0 }} · 失败 {{ row.counts?.failed || 0 }}</template></el-table-column>
    </el-table>
    <el-pagination v-model:current-page="sync.runQuery.page" :page-size="sync.runQuery.limit" :total="sync.runs.value.total" layout="total, prev, pager, next" @current-change="sync.loadRuns" />
    <el-drawer v-model="runOpen" title="同步运行详情" size="min(960px, 100%)" @closed="sync.closeRun">
      <el-alert v-if="sync.detailError.value" :title="sync.detailError.value" type="error" :closable="false"><template #default><el-button link @click="retryRun">重试读取</el-button></template></el-alert>
      <div v-loading="sync.detailLoading.value">
        <p v-if="sync.selectedRun.value">{{ label(sync.selectedRun.value.status) }} / {{ label(sync.selectedRun.value.phase) }} · {{ sync.selectedRun.value.mode === 'dry_run' ? '预览不会写入文章或目录' : '同步运行' }}</p>
        <el-alert v-if="sync.selectedRun.value?.error" :title="sync.selectedRun.value.error" type="error" :closable="false" />
        <el-table :data="sync.items.value.items" :row-key="(item: SyncItem) => item.item_id || item.page_id" empty-text="暂无逐项结果">
          <el-table-column label="页面 / 主题" min-width="180"><template #default="{ row }">{{ row.page_id || '主题目录' }}</template></el-table-column><el-table-column label="结果"><template #default="{ row }">{{ label(row.outcome) }}</template></el-table-column>
          <el-table-column label="原因" min-width="180"><template #default="{ row }">{{ row.reason || row.error || '—' }}</template></el-table-column>
          <el-table-column label="变更" min-width="220"><template #default="{ row }"><details v-if="row.before || row.after"><summary>查看前后资料</summary><pre>{{ JSON.stringify({ before: row.before, after: row.after }, null, 2) }}</pre></details></template></el-table-column>
        </el-table>
        <el-pagination v-model:current-page="sync.itemQuery.page" :page-size="sync.itemQuery.limit" :total="sync.items.value.total" layout="total, prev, pager, next" @current-change="retryRun" />
      </div>
    </el-drawer>
    <el-dialog v-model="sourceOpen" title="来源配置" width="min(700px, 95%)" :close-on-click-modal="false">
      <p class="hint">填写 Notion 属性 ID 和状态选项 ID。修改会校验配置版本；失败时保留本次输入，请重新读取后核对。</p>
      <el-form label-width="150px"><el-form-item label="名称"><el-input v-model="sourceForm.label" /></el-form-item><el-form-item label="模块"><el-select v-model="sourceForm.module_code"><el-option v-for="module in catalog.modules.modules" :key="module.code" :label="module.title" :value="module.code" /></el-select></el-form-item><el-form-item label="启用来源"><el-switch v-model="sourceForm.enabled" /></el-form-item>
        <el-form-item v-for="field in propertyFields" :key="field.key" :label="field.label"><el-input v-model="sourceForm.config[field.key]" /></el-form-item>
        <el-form-item v-for="state in ['draft', 'published', 'unpublished', 'archived']" :key="state" :label="label(state) + '选项 ID'"><el-input v-model="sourceForm.config.state_option_ids[state]" /></el-form-item>
      </el-form>
      <el-alert v-if="sync.error.value" :title="sync.error.value" type="error" :closable="false" />
      <template #footer><el-button @click="sourceOpen = false">关闭</el-button><el-button type="primary" :loading="sync.actionBusy.value" @click="saveSource">保存配置</el-button></template>
    </el-dialog>
    <el-dialog v-model="bindingOpen" title="调整主题绑定" width="min(560px, 95%)" :close-on-click-modal="false">
      <p>{{ binding?.option_name }}：{{ binding?.reason }}</p><el-select v-model="bindingCode" placeholder="请选择同模块的正常章节"><el-option v-for="section in bindingSections" :key="section.code" :label="section.title" :value="section.code" :disabled="section.status !== 1" /></el-select>
      <el-alert v-if="sync.error.value" :title="sync.error.value" type="error" :closable="false" />
      <template #footer><el-button @click="bindingOpen = false">关闭</el-button><el-button type="primary" :disabled="!bindingCode" :loading="sync.actionBusy.value" @click="saveBinding">保存绑定</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import { useNotionSync } from '@/composables/useNotionSync';
import { useCatalog } from '@/composables/useCatalog';
import type { SyncConfig, SyncSource, SyncItem, CatalogBinding } from '@/types/notion-sync';
import { errorMessage } from '@/utils/api-error';
const sync = useNotionSync(); const catalog = useCatalog();
async function refreshAll() { await Promise.all([sync.refresh(), catalog.load(true).catch(cause => { sync.error.value = errorMessage(cause, '读取目录失败'); })]); }
const runOpen = ref(false); const runID = ref(''); const sourceOpen = ref(false); const source = ref<SyncSource>(); const bindingOpen = ref(false); const binding = ref<CatalogBinding>(); const bindingSource = ref<SyncSource>(); const bindingCode = ref('');
const blankConfig = (): SyncConfig => ({ title_property_id: '', state_property_id: '', topic_property_id: '', tags_property_id: '', state_option_ids: { draft: '', published: '', unpublished: '', archived: '' }});
const sourceForm = reactive({ label: '', module_code: '', enabled: false, config: blankConfig() });
const propertyFields: { key: 'title_property_id' | 'state_property_id' | 'topic_property_id' | 'tags_property_id' | 'description_property_id' | 'difficulty_property_id' | 'date_property_id'; label: string }[] = [{ key: 'title_property_id', label: '标题属性 ID' }, { key: 'state_property_id', label: '状态属性 ID' }, { key: 'topic_property_id', label: '主题属性 ID' }, { key: 'tags_property_id', label: '标签属性 ID' }, { key: 'description_property_id', label: '简介属性 ID（可选）' }, { key: 'difficulty_property_id', label: '难度属性 ID（可选）' }, { key: 'date_property_id', label: '日期属性 ID（可选）' }];
const labels: Record<string, string> = { healthy: '正常', ok: '正常', degraded: '异常', disabled: '未启用', error: '失败', pending: '等待', baseline_pending: '待基线审核', managed: '同步管理', detached: '脱离管理', draft: '草稿', published: '已发布', unpublished: '已下架', archived: '已归档', running: '运行中', queued: '排队', discovering: '扫描中', scanning: '扫描中', planned: '预览完成', applying: '应用中', completed: '完成', succeeded: '成功', failed: '失败', partial_failed: '部分失败', unchanged: '未变化', created: '新增', updated: '更新', blocked: '已阻止', bound: '已绑定', conflict: '绑定冲突', invalid: '失效绑定' };
const label = (value?: string) => value ? labels[value] || value : '—';
const time = (value?: string) => value ? new Date(value).toLocaleString() : '暂无';
function safeURL(value?: string) { try { const url = new URL(value || ''); return ['http:', 'https:'].includes(url.protocol) ? url.href : ''; } catch { return ''; } }
const bindingSections = computed(() => catalog.sections.getSectionsByModule(bindingSource.value?.module_code || ''));
function searchPages() { sync.pageFilters.page = 1; void sync.loadPages(); }
async function openRun(id: string) { runID.value = id; runOpen.value = true; await sync.selectRun(id); }
const retryRun = () => sync.selectRun(runID.value, false);
async function start(mode: 'dry_run' | 'sync') {
  if (mode === 'sync') { try { await ElMessageBox.confirm('同步会更新来源管理的资料。本次将重新扫描当前来源，预览不会直接重放。请先处理主题冲突；待基线审核的历史文章仍不会自动接管。是否继续？', '手动同步'); } catch { return; } }
  if (await sync.start(mode)) { runID.value = sync.selectedRunID.value; runOpen.value = Boolean(runID.value); }
}
function editSource(value: SyncSource) { source.value = value; Object.assign(sourceForm, { label: value.label, module_code: value.module_code, enabled: value.enabled, config: { ...blankConfig(), ...value.config, state_option_ids: { ...blankConfig().state_option_ids, ...value.config?.state_option_ids }}}); sourceOpen.value = true; }
async function saveSource() { if (!source.value) return; if (await sync.saveSource(source.value, { ...sourceForm, config: { ...sourceForm.config, state_option_ids: { ...sourceForm.config.state_option_ids }}, expected_config_revision: source.value.config_revision })) sourceOpen.value = false; }
function editBinding(value: SyncSource, row: CatalogBinding) { bindingSource.value = value; binding.value = row; bindingCode.value = row.section_code || ''; bindingOpen.value = true; }
async function saveBinding() { if (binding.value && bindingSource.value && await sync.saveBinding(binding.value, bindingCode.value, bindingSource.value.config_revision)) bindingOpen.value = false; }
onMounted(refreshAll);
</script>
<style scoped>.sync-page { padding:24px; } .toolbar,.actions { display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; } .actions { justify-content:flex-start; } .health,.source-card { border:1px solid var(--el-border-color); border-radius:8px; padding:16px; margin:16px 0; } .hint,.source-card p { color:var(--el-text-color-secondary); font-size:13px; } .source-link { display:block; margin-top:6px; } .el-select { min-width:180px; } .el-pagination { margin-top:12px; } pre { white-space:pre-wrap; overflow-wrap:anywhere; max-height:360px; overflow:auto; } h3 { margin-top:28px; }</style>
