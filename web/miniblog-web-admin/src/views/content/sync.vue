<template>
  <div class="sync-page">
    <header class="page-heading">
      <div><h1>Notion 自动同步</h1><p>在 Notion 维护资料，在这里核对同步、公开情况与目录绑定。</p></div>
      <div class="actions">
        <el-button :loading="sync.loading.value" @click="refreshAll">刷新状态</el-button>
        <el-button type="primary" data-test="run-preview" :disabled="previewDisabled" @click="start('dry_run')">运行预览</el-button>
      </div>
    </header>
    <el-alert v-if="sync.error.value" class="page-error" :title="reason(sync.error.value)" type="error" :closable="false"><template #default><el-button link @click="sync.refresh">重新读取</el-button></template></el-alert>
    <el-tabs v-model="activeTab" class="sync-tabs" data-test="sync-tabs" @tab-change="changeTab">
      <el-tab-pane label="同步概览" name="overview">
        <div v-if="activeTab === 'overview'" data-test="sync-overview">
          <section v-if="sync.status.value" class="panel health">
            <div class="section-heading"><h2><span :class="['health-dot', { 'is-healthy': isHealthy(sync.status.value.health), 'is-warning': ['degraded', 'error', 'stale'].includes(sync.status.value.health) }]" aria-hidden="true" />{{ sync.status.value.enabled ? '自动同步已开启' : '自动同步未开启' }}</h2><el-tag type="info">服务配置控制</el-tag></div>
            <div class="status-tags"><el-tag :type="isHealthy(sync.status.value.health) ? 'success' : 'warning'">{{ label(sync.status.value.health) }}</el-tag><el-tag v-if="sync.status.value.paused" type="warning">同步维护暂停</el-tag><el-tag v-if="sync.status.value.source_writes_paused" type="warning">来源回填暂停</el-tag></div>
            <dl class="health-facts"><div><dt>最近完整扫描</dt><dd>{{ time(sync.status.value.last_complete_scan_at) }}</dd></div><div><dt>最近成功</dt><dd>{{ time(sync.status.value.last_success_at) }}</dd></div><div><dt>当前运行</dt><dd><el-button v-if="sync.status.value.current_run_id" link @click="openRun(sync.status.value.current_run_id!)">{{ sync.status.value.current_run_id }}</el-button><span v-else>没有进行中的运行</span></dd></div></dl>
            <div class="health-actions"><el-button data-test="run-sync" :disabled="syncDisabled" @click="start('sync')">手动同步</el-button><details class="maintenance"><summary>维护控制</summary><div><p class="hint">维护暂停与来源回填暂停各自独立，不改变服务总开关。</p><el-button :disabled="sync.actionBusy.value" @click="sync.control({ paused: !sync.status.value!.paused })">{{ sync.status.value.paused ? '解除维护暂停' : '暂停同步' }}</el-button><el-button :disabled="sync.actionBusy.value" @click="sync.control({ source_writes_paused: !sync.status.value!.source_writes_paused })">{{ sync.status.value.source_writes_paused ? '允许来源回填' : '暂停来源回填' }}</el-button></div></details></div>
            <p class="hint">预览只读取当前来源并记录结果；手动同步重新扫描来源，不沿用旧预览结果。正文仍从外部文档阅读。</p>
            <ul v-if="syncGateReasons.length" class="gate-reasons" data-test="sync-gates"><li v-for="item in syncGateReasons" :key="item">{{ item }}</li></ul>
          </section>
          <section v-else class="panel unavailable" :aria-busy="sync.loading.value"><el-skeleton v-if="sync.loading.value" :rows="3" animated /><template v-else><h2>同步状态暂未读取</h2><p class="hint">读取成功后才能判断同步是否可执行。</p><el-button @click="refreshAll">重新读取</el-button></template></section>
          <div class="overview-grid">
            <section class="panel"><div class="section-heading"><h2>同步来源</h2><el-button link @click="changeTab('sources')">配置与绑定</el-button></div><el-empty v-if="!sync.loading.value && !overviewSources.length" description="暂无已配置来源" :image-size="64" /><div v-for="source in overviewSources" :key="source.source_id" class="source-summary"><div class="topic-icon">{{ moduleTitle(source.module_code).slice(0, 2) }}</div><div><strong>{{ source.label || source.source_id }}</strong><p class="hint">{{ source.enabled ? '绑定到 ' + moduleTitle(source.module_code) : '已有文章保持，不接收新变更' }}</p><p v-if="source.last_error" class="source-error">{{ reason(source.last_error) }}</p></div><el-tag :type="source.enabled ? 'success' : 'info'">{{ source.enabled ? '来源启用' : '来源停用' }}</el-tag></div></section>
            <section class="panel"><div class="section-heading"><h2>需要核对</h2><el-button link @click="changeTab('pages')">页面状态</el-button></div><template v-if="sync.status.value"><p class="attention-item"><el-tag type="warning">待核对</el-tag><span>历史待核对 {{ sync.status.value.pending_count }}</span></p><p class="attention-item"><el-tag type="warning">公开受限</el-tag><span>公开受限 {{ sync.status.value.blocked_count ?? '—' }}</span></p><p class="attention-item"><el-tag type="danger">失败</el-tag><span>读取 / 执行失败 {{ sync.status.value.error_count }}</span></p></template><p v-else class="hint">同步状态尚未读取，计数暂不可用。</p><p class="hint">各项可能重叠。来源停用、同步维护暂停和本地紧急下架分别判断。</p></section>
          </div>
          <section class="panel"><div class="section-heading"><div><h2>{{ sync.runQuery.page === 1 ? '最近运行' : '当前页运行记录' }}</h2><p class="hint">预览与实际同步分别记录。</p></div><el-button link @click="changeTab('runs')">查看记录</el-button></div><el-empty v-if="!sync.loading.value && !sync.runs.value.items.length" description="暂无运行记录" :image-size="64" /><button v-for="run in sync.runs.value.items.slice(0, 3)" :key="run.run_id" type="button" class="run-summary" @click="openRun(run.run_id)"><el-tag :type="runTag(run.status)">{{ label(run.status) }}</el-tag><span><strong>{{ label(run.mode) }}</strong><small>{{ time(run.started_at) }} · 发现 {{ run.counts?.seen ?? '—' }} · {{ isPreview(run.mode) ? '拟更新' : '更新' }} {{ run.counts?.updated ?? '—' }} · 失败 {{ run.counts?.failed ?? '—' }}</small></span><span class="run-arrow" aria-hidden="true">›</span></button></section>
        </div>
      </el-tab-pane>
      <el-tab-pane label="来源与绑定" name="sources">
        <div v-if="activeTab === 'sources'" data-test="sync-sources">
          <p class="tab-description">核对 Notion 主题对应的本站章节，处理冲突后重新预览。来源停用保持既有资料，不等于本地紧急下架。</p>
          <el-empty v-if="!sync.loading.value && !sync.sources.value.items.length" description="暂无已配置来源" />
          <section v-for="source in sync.sources.value.items" :key="source.source_id" class="panel source-card">
            <div class="section-heading source-heading"><div class="source-name"><span class="topic-icon">{{ moduleTitle(source.module_code).slice(0, 2) }}</span><div><h2>{{ source.label || source.source_id }}</h2><p class="hint">绑定到 {{ moduleTitle(source.module_code) }}</p></div></div><div class="actions"><el-tag :type="source.enabled ? 'success' : 'info'">{{ source.enabled ? '来源同步启用' : '来源同步停用' }}</el-tag><el-tag :type="isHealthy(source.health) ? 'success' : 'warning'">{{ label(source.health) }}</el-tag><el-button :disabled="sync.actionBusy.value" @click="editSource(source)">配置</el-button></div></div>
            <p v-if="!source.enabled" class="hint">此来源保持冻结，预览仍可读取；不会同步新变更。停用来源不等于本地紧急下架。</p>
            <dl class="source-facts"><div><dt>最近尝试</dt><dd>{{ time(source.last_attempt_at) }}</dd></div><div><dt>完整扫描</dt><dd>{{ time(source.last_complete_scan_at) }}</dd></div><div><dt>最近成功</dt><dd>{{ time(source.last_success_at) }}</dd></div></dl>
            <el-alert v-if="source.last_error" :title="reason(source.last_error)" type="error" :closable="false" />
            <div class="table-scroll"><el-table :data="source.catalog_bindings || []" empty-text="扫描后显示主题绑定" class="binding-table">
              <el-table-column prop="option_name" label="Notion 主题" min-width="160" /><el-table-column label="本站章节" min-width="160"><template #default="{ row }">{{ sectionTitle(source.module_code, row.section_code) }}</template></el-table-column>
              <el-table-column label="状态 / 原因" min-width="200"><template #default="{ row }"><el-tag :type="row.status === 'bound' ? 'success' : 'warning'">{{ label(row.status) }}</el-tag><p v-if="row.reason" class="hint">{{ reason(row.reason) }}</p></template></el-table-column>
              <el-table-column label="操作" width="120"><template #default="{ row }"><el-button link :disabled="sync.actionBusy.value" @click="editBinding(source, row)">调整绑定</el-button></template></el-table-column>
            </el-table></div>
            <details class="source-identifiers"><summary>来源标识与配置版本</summary><p>来源 {{ source.source_id }} · 模块 {{ source.module_code }} · 配置版本 {{ source.config_revision }}</p></details>
          </section>
          <el-pagination v-if="sync.sources.value.total > sync.sourceQuery.limit" v-model:current-page="sync.sourceQuery.page" :page-size="sync.sourceQuery.limit" :total="sync.sources.value.total" layout="prev, pager, next" @current-change="sync.refresh" />
          <p class="hint">全新主题自动创建章节。失效或同名绑定需选择同主题的正确章节；修改绑定会要求受影响文章重新核验，可能暂时影响前台阅读。历史文章通过受控命令审核与回填，网页不执行接管写入。</p>
        </div>
      </el-tab-pane>
      <el-tab-pane label="页面状态" name="pages">
        <div v-if="activeTab === 'pages'" data-test="sync-pages">
          <p class="tab-description">来源期望、本站状态与实际公开分别显示。最近失败不替代当前公开判断。</p>
          <section class="panel page-list"><el-form class="page-filters" label-position="top" @submit.prevent="searchPages"><el-form-item label="标题"><el-input v-model="sync.pageFilterDraft.title" placeholder="搜索来源页面标题" clearable @keyup.enter="searchPages" /></el-form-item><el-form-item label="来源"><el-select v-model="sync.pageFilterDraft.source_id" clearable placeholder="全部来源"><el-option v-for="source in sync.status.value?.sources || sync.sources.value.items" :key="source.source_id" :label="source.label || source.source_id" :value="source.source_id" /></el-select></el-form-item><el-form-item label="管理状态"><el-select v-model="sync.pageFilterDraft.management_state" clearable placeholder="全部"><el-option v-for="state in ['baseline_pending', 'managed', 'detached']" :key="state" :label="label(state)" :value="state" /></el-select></el-form-item><el-button native-type="submit">查询</el-button></el-form>
            <SyncPageStates :items="sync.pages.value.items" :loading="sync.loading.value" />
            <el-pagination v-model:current-page="sync.pageFilters.page" :page-size="sync.pageFilters.limit" :total="sync.pages.value.total" layout="total, prev, pager, next" @current-change="sync.loadPages" />
          </section>
        </div>
      </el-tab-pane>
      <el-tab-pane label="运行记录" name="runs">
        <div v-if="activeTab === 'runs'" data-test="sync-runs">
          <p class="tab-description">每次预览与同步保留独立记录。打开旧记录只读取明细，不再次执行。</p>
          <section class="panel run-list"><div class="table-scroll"><el-table :data="sync.runs.value.items" empty-text="暂无运行记录" class="runs-table">
            <el-table-column label="运行" min-width="200"><template #default="{ row }"><el-button link class="identifier-link" @click="openRun(row.run_id)">{{ row.run_id }}</el-button></template></el-table-column><el-table-column label="模式" min-width="110"><template #default="{ row }">{{ label(row.mode) }}</template></el-table-column><el-table-column label="状态" min-width="160"><template #default="{ row }"><el-tag :type="runTag(row.status)">{{ label(row.status) }}</el-tag><p class="hint">{{ label(row.phase) }}</p></template></el-table-column><el-table-column label="开始时间" min-width="170"><template #default="{ row }">{{ time(row.started_at) }}</template></el-table-column><el-table-column label="结果" min-width="250"><template #default="{ row }">发现 {{ row.counts?.seen ?? '—' }} · {{ isPreview(row.mode) ? '拟新增' : '新增' }} {{ row.counts?.created ?? '—' }} · {{ isPreview(row.mode) ? '拟更新' : '更新' }} {{ row.counts?.updated ?? '—' }} · 未变 {{ row.counts?.unchanged ?? '—' }} · 待处理 {{ row.counts?.pending ?? '—' }} · 冻结 {{ row.counts?.frozen ?? '—' }} · 阻止 {{ row.counts?.blocked ?? '—' }} · 失败 {{ row.counts?.failed ?? '—' }}</template></el-table-column>
          </el-table></div><div class="run-cards"><article v-for="run in sync.runs.value.items" :key="run.run_id"><div class="section-heading"><strong>{{ label(run.mode) }}</strong><el-tag :type="runTag(run.status)">{{ label(run.status) }}</el-tag></div><p class="hint">{{ time(run.started_at) }} · {{ label(run.phase) }}</p><p>发现 {{ run.counts?.seen ?? '—' }} · {{ isPreview(run.mode) ? '拟新增' : '新增' }} {{ run.counts?.created ?? '—' }} · {{ isPreview(run.mode) ? '拟更新' : '更新' }} {{ run.counts?.updated ?? '—' }} · 未变 {{ run.counts?.unchanged ?? '—' }} · 待处理 {{ run.counts?.pending ?? '—' }} · 冻结 {{ run.counts?.frozen ?? '—' }} · 阻止 {{ run.counts?.blocked ?? '—' }} · 失败 {{ run.counts?.failed ?? '—' }}</p><el-button link class="identifier-link" @click="openRun(run.run_id)">{{ run.run_id }}</el-button></article><el-empty v-if="!sync.loading.value && !sync.runs.value.items.length" description="暂无运行记录" /></div><el-pagination v-model:current-page="sync.runQuery.page" :page-size="sync.runQuery.limit" :total="sync.runs.value.total" layout="total, prev, pager, next" @current-change="sync.loadRuns" /></section>
        </div>
      </el-tab-pane>
    </el-tabs>
    <el-drawer v-model="runOpen" title="同步运行详情" :size="panelSize" class="sync-drawer" data-test="run-panel" @closed="sync.closeRun">
      <p class="hint identifier">运行 {{ runID }}</p>
      <el-alert v-if="sync.detailError.value" :title="reason(sync.detailError.value)" type="error" :closable="false"><template #default><el-button link @click="retryRun">重试读取</el-button></template></el-alert>
      <div v-loading="sync.detailLoading.value" class="run-detail">
        <template v-if="sync.selectedRun.value"><div class="section-heading"><h2>{{ label(sync.selectedRun.value.mode) }}</h2><el-tag :type="runTag(sync.selectedRun.value.status)">{{ label(sync.selectedRun.value.status) }}</el-tag></div><p class="hint">{{ label(sync.selectedRun.value.phase) }} · {{ sync.selectedRun.value.mode === 'dry_run' ? '预览不会写入文章或目录' : label(sync.selectedRun.value.mode) }}</p><dl class="run-facts"><div><dt>开始</dt><dd>{{ time(sync.selectedRun.value.started_at) }}</dd></div><div><dt>结束</dt><dd>{{ time(sync.selectedRun.value.finished_at) }}</dd></div></dl></template>
        <el-alert v-if="sync.selectedRun.value?.error" :title="reason(sync.selectedRun.value.error)" type="error" :closable="false" />
        <BootstrapPreviewList v-if="sync.selectedRun.value?.mode === 'bootstrap_preview'" :items="sync.items.value.items" :source-labels="sourceLabels" :directory-labels="directoryLabels" :run="sync.selectedRun.value" />
        <div v-else class="run-items"><el-empty v-if="!sync.detailLoading.value && !sync.items.value.items.length" description="暂无逐项结果" /><article v-for="item in sync.items.value.items" :key="item.item_id || item.page_id" class="run-item"><div class="section-heading"><strong class="identifier">{{ item.page_id || '主题目录' }}</strong><el-tag>{{ label(item.outcome) }}</el-tag></div><p>{{ reason(item.reason || item.error) }}</p><details v-if="item.before || item.after"><summary>查看前后资料</summary><pre>{{ JSON.stringify({ before: item.before, after: item.after }, null, 2) }}</pre></details></article></div>
        <el-pagination v-model:current-page="sync.itemQuery.page" :page-size="sync.itemQuery.limit" :total="sync.items.value.total" layout="total, prev, pager, next" @current-change="retryRun" />
      </div>
      <template #footer><el-button @click="runOpen = false">关闭</el-button></template>
    </el-drawer>
    <el-drawer v-model="sourceOpen" title="来源配置" :size="panelSize" :close-on-click-modal="false" class="sync-drawer" data-test="source-panel">
      <p class="hint">填写 Notion 属性 ID 和状态选项 ID。修改会校验配置版本；失败时保留输入。读取最新版只显示对照，不会自动改变草稿或保存版本。</p>
      <el-button data-test="source-read-latest" :loading="sync.latestSourceLoading.value" :disabled="sync.actionBusy.value" @click="readLatestSource">读取最新版对照</el-button>
      <section v-if="sourceEditor.latest.value" data-test="source-comparison" class="revision-comparison"><p>编辑依据版本 {{ sourceEditor.source.value?.config_revision }} · 最新版本 {{ sourceEditor.latest.value.config_revision }}</p><div class="comparison-table"><el-table :data="sourceEditor.comparison.value" size="small"><el-table-column prop="label" label="项目" min-width="100" /><el-table-column prop="baseline" label="编辑时资料" /><el-table-column prop="draft" label="本次草稿" /><el-table-column prop="latest" label="最新资料" /></el-table></div><dl v-for="row in sourceEditor.comparison.value" :key="row.label" class="comparison-card"><dt>{{ row.label }}</dt><dd><span>编辑时</span>{{ row.baseline || '未填写' }}</dd><dd><span>本次草稿</span>{{ row.draft || '未填写' }}</dd><dd><span>最新资料</span>{{ row.latest || '未填写' }}</dd></dl><p class="hint">逐项检查差异后可采用最新版本作为保存依据；草稿保持原样，需要的修改仍由你决定。保存时会再次校验版本。</p><el-button v-if="sourceEditor.needsReview.value" data-test="source-accept-latest" @click="sourceEditor.acceptLatest">已核对，以最新版本为保存依据</el-button></section>
      <el-form class="source-form" label-position="top" :disabled="sync.actionBusy.value"><el-form-item label="名称"><el-input v-model="sourceForm.label" data-test="source-label" /></el-form-item><el-form-item label="所属主题"><el-select v-model="sourceForm.module_code"><el-option v-for="module in catalog.modules.modules" :key="module.code" :label="module.title" :value="module.code" /></el-select></el-form-item><el-form-item label="启用来源"><el-switch v-model="sourceForm.enabled" /><p class="hint">停用后保留已有文章资料，不接收新变更；这不是本地紧急下架。</p></el-form-item><h2>字段映射</h2><el-form-item v-for="field in propertyFields" :key="field.key" :label="field.label"><el-input v-model="sourceForm.config[field.key]" :data-test="'source-' + field.key" /></el-form-item><h2>状态选项</h2><el-form-item v-for="state in ['draft', 'published', 'unpublished', 'archived']" :key="state" :label="label(state) + '选项 ID'"><el-input v-model="sourceForm.config.state_option_ids[state]" /></el-form-item></el-form>
      <el-alert v-if="sync.error.value" :title="reason(sync.error.value)" type="error" :closable="false" /><p v-if="sourceEditor.affectsPublication.value" class="publication-impact" data-test="source-publication-impact">模块或字段映射变化将要求已有托管文章重新核验，可能暂时影响前台阅读。</p>
      <template #footer><el-button @click="sourceOpen = false">关闭</el-button><el-button type="primary" :loading="sync.actionBusy.value" :disabled="sourceEditor.needsReview.value || sync.latestSourceLoading.value" data-test="source-save" @click="saveSource">保存配置</el-button></template>
    </el-drawer>
    <el-drawer v-model="bindingOpen" title="调整主题绑定" :size="panelSize" :close-on-click-modal="false" class="sync-drawer" data-test="binding-panel">
      <p><strong>{{ binding?.option_name }}</strong> · {{ reason(binding?.reason) }}</p><p class="hint">选择同主题正常章节。绑定变化后，受影响文章会等待重新核验，可能暂时影响前台阅读。</p><el-button data-test="binding-read-latest" :loading="sync.latestSourceLoading.value" :disabled="sync.actionBusy.value" @click="readLatestBinding">读取最新版对照</el-button>
      <section v-if="latestBindingSource" class="revision-comparison" data-test="binding-comparison"><p>编辑依据版本 {{ bindingSource?.config_revision }} · 最新版本 {{ latestBindingSource.config_revision }}</p><p>编辑时：{{ bindingSource?.module_code }} / {{ binding?.section_code || '未绑定' }} · 本次草稿：{{ bindingCode || '未选择' }}</p><p>最新：{{ latestBindingSource.module_code }} / {{ latestBinding?.section_code || '未绑定' }} · {{ latestBinding ? label(latestBinding.status) : '此主题绑定已不存在，请关闭后重新检查来源' }}</p><el-button v-if="bindingNeedsReview && latestBinding" data-test="binding-accept-latest" @click="acceptLatestBinding">已核对，以最新版本为保存依据</el-button></section>
      <el-form class="binding-form" label-position="top" @submit.prevent="saveBinding"><el-form-item label="本站章节"><el-select v-model="bindingCode" placeholder="请选择同模块的正常章节"><el-option v-for="section in bindingSections" :key="section.code" :label="section.title" :value="section.code" :disabled="section.status !== 1" /></el-select></el-form-item></el-form><el-alert v-if="sync.error.value" :title="reason(sync.error.value)" type="error" :closable="false" />
      <template #footer><el-button @click="bindingOpen = false">关闭</el-button><el-button type="primary" :disabled="!validBindingSelection || bindingNeedsReview || sync.latestSourceLoading.value" data-test="binding-save" :loading="sync.actionBusy.value" @click="saveBinding">保存绑定</el-button></template>
    </el-drawer>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { useNotionSync } from '@/composables/useNotionSync';
import { useCatalog } from '@/composables/useCatalog';
import type { SyncSource, CatalogBinding } from '@/types/notion-sync';
import { errorMessage } from '@/utils/api-error';
import { syncStateLabel as label, syncReasonLabel as reason } from '@/components/content/sync-presentation';
import BootstrapPreviewList from '@/components/content/BootstrapPreviewList.vue';
import SyncPageStates from '@/components/content/sync/SyncPageStates.vue';
import { useSourceConfiguration } from '@/composables/useSourceConfiguration';
const sync = useNotionSync(); const catalog = useCatalog();
const route = useRoute(); const router = useRouter();
const tabs = ['overview', 'sources', 'pages', 'runs'] as const;
type SyncTab = typeof tabs[number];
const readTab = (value: unknown): SyncTab => typeof value === 'string' && tabs.includes(value as SyncTab) ? value as SyncTab : 'overview';
const activeTab = ref<SyncTab>(readTab(route.query.tab));
watch(() => route.query.tab, value => { activeTab.value = readTab(value); });
function changeTab(value: string | number) {
  const tab = readTab(value); activeTab.value = tab;
  if (route.query.tab !== tab) void router.replace({ path: route.path, query: { ...route.query, tab }, hash: route.hash });
}
const smallScreen = ref(window.innerWidth <= 780);
const panelSize = computed(() => smallScreen.value ? '100%' : '620px');
function syncViewport() { smallScreen.value = window.innerWidth <= 780; }
const previewDisabled = computed(() => !sync.status.value || sync.actionBusy.value || Boolean(sync.status.value.current_run_id));
const syncGateReasons = computed(() => {
  const status = sync.status.value; if (!status) return ['同步状态尚未读取，读取成功后才能判断是否可执行。'];
  const reasons: string[] = [];
  if (!status.enabled) reasons.push('总开关由服务配置控制，此处不能开启自动同步；仍可运行只读预览。');
  if (status.paused) reasons.push('同步维护暂停，解除暂停后才能执行同步。');
  if (status.source_writes_paused) reasons.push('来源回填暂停期间仅可预览，不能启动同步。');
  if (status.baseline_frozen !== true) reasons.push('历史基线尚未冻结，完成受控审核后才能执行同步；现在仍可预览。');
  if (status.current_run_id) reasons.push('当前运行尚未完成，请先查看运行详情，避免重复启动。');
  return reasons;
});
const syncDisabled = computed(() => sync.actionBusy.value || syncGateReasons.value.length > 0);
const overviewSources = computed(() => sync.status.value?.sources || sync.sources.value.items);
const moduleTitle = (code: string) => catalog.modules.modules.find(module => module.code === code)?.title || code || '尚未绑定';
const sectionTitle = (code: string, section?: string) => catalog.sections.getSectionsByModule(code).find(item => item.code === section)?.title || section || '尚未绑定';
const isHealthy = (value: string) => ['healthy', 'ok'].includes(value);
const isPreview = (mode: string) => ['dry_run', 'bootstrap_preview'].includes(mode);
const runTag = (status: string): 'success' | 'danger' | 'warning' | 'info' => ['succeeded', 'completed'].includes(status) ? 'success' : ['failed', 'partial_failed', 'completed_with_errors', 'abandoned'].includes(status) ? 'danger' : ['queued', 'running'].includes(status) ? 'warning' : 'info';
async function refreshAll() { await Promise.all([sync.refresh(), catalog.load(true).catch(cause => { sync.error.value = errorMessage(cause, '读取目录失败'); })]); }
const runOpen = ref(false); const runID = ref(''); const sourceOpen = ref(false); const sourceEditor = useSourceConfiguration(); const bindingOpen = ref(false); const binding = ref<CatalogBinding>(); const bindingSource = ref<SyncSource>(); const bindingCode = ref('');
const sourceForm = sourceEditor.form;
const propertyFields: { key: 'title_property_id' | 'state_property_id' | 'topic_property_id' | 'tags_property_id' | 'description_property_id' | 'difficulty_property_id' | 'date_property_id'; label: string }[] = [{ key: 'title_property_id', label: '标题属性 ID' }, { key: 'state_property_id', label: '状态属性 ID' }, { key: 'topic_property_id', label: '主题属性 ID' }, { key: 'tags_property_id', label: '标签属性 ID' }, { key: 'description_property_id', label: '简介属性 ID（可选）' }, { key: 'difficulty_property_id', label: '难度属性 ID（可选）' }, { key: 'date_property_id', label: '日期属性 ID（可选）' }];
const time = (value?: string) => value ? new Date(value).toLocaleString() : '暂无';
const sourceLabels = computed(() => Object.fromEntries((sync.status.value?.sources || sync.sources.value.items).map(item => [item.source_id, item.label || item.source_id])));
const directoryLabels = computed(() => {
  const labels: Record<string, string> = {};
  for (const module of catalog.modules.modules) {
    for (const section of catalog.sections.getSectionsByModule(module.code)) {
      labels[section.code] = `${module.title} / ${section.title}`;
      for (const subsection of catalog.subsections.getSubsectionsBySection(section.code)) labels[subsection.code] = `${labels[section.code]} / ${subsection.title}`;
    }
  }
  return labels;
});
const latestBindingSource = ref<SyncSource>(); const latestBinding = ref<CatalogBinding>();
const bindingSections = computed(() => catalog.sections.getSectionsByModule((latestBindingSource.value || bindingSource.value)?.module_code || ''));
const bindingNeedsReview = computed(() => Boolean(latestBindingSource.value && (!latestBinding.value || latestBindingSource.value.config_revision !== bindingSource.value?.config_revision)));
const validBindingSelection = computed(() => bindingSections.value.some(section => section.code === bindingCode.value.trim() && section.status === 1));
function searchPages() { void sync.searchPages(); }
async function readLatestSource() {
  const editing = sourceEditor.source.value; if (!editing) return;
  const latest = await sync.readSource(editing.source_id);
  if (latest && sourceOpen.value && sourceEditor.source.value === editing) sourceEditor.compareLatest(latest);
}
async function readLatestBinding() {
  const editing = binding.value; const source = bindingSource.value; if (!editing || !source) return;
  const latest = await sync.readSource(source.source_id);
  if (!latest || !bindingOpen.value || binding.value !== editing) return;
  latestBindingSource.value = { ...latest }; latestBinding.value = latest.catalog_bindings.find(row => row.id === editing.id);
  try { await catalog.load(true); } catch (cause) { sync.error.value = errorMessage(cause, '读取最新目录失败，草稿仍保留'); }
}
function acceptLatestBinding() {
  if (!latestBindingSource.value || !latestBinding.value) return;
  bindingSource.value = { ...latestBindingSource.value }; binding.value = { ...latestBinding.value };
}
async function openRun(id: string) { runID.value = id; runOpen.value = true; await sync.selectRun(id); }
const retryRun = () => sync.selectRun(runID.value, false);
async function start(mode: 'dry_run' | 'sync') {
  if (mode === 'sync') { try { await ElMessageBox.confirm('同步会更新来源管理的资料。本次将重新扫描当前来源，预览不会直接重放。请先处理主题冲突；待基线审核的历史文章仍不会自动接管。是否继续？', '手动同步'); } catch { return; } }
  if (await sync.start(mode)) { runID.value = sync.selectedRunID.value; runOpen.value = Boolean(runID.value); }
}
function editSource(value: SyncSource) { sourceEditor.open(value); sourceOpen.value = true; }
async function saveSource() {
  const editing = sourceEditor.source.value; const update = sourceEditor.update.value;
  if (!editing || sourceEditor.needsReview.value || sync.latestSourceLoading.value) return;
  if (!update) { ElMessage.info('配置没有变化，无需保存'); sourceOpen.value = false; return; }
  if (sourceEditor.affectsPublication.value) {
    try { await ElMessageBox.confirm('模块或字段映射发生变化。如该来源已有托管文章，保存后会要求重新核验，文章可能暂时无法在前台阅读。请在保存后重新预览并同步核验。是否继续？', '确认公开影响', { confirmButtonText: '保存并重新核验', cancelButtonText: '返回检查' }); } catch { return; }
  }
  if (await sync.saveSource(editing, update) && sourceEditor.source.value === editing) sourceOpen.value = false;
}
function editBinding(value: SyncSource, row: CatalogBinding) { bindingSource.value = { ...value }; binding.value = { ...row }; bindingCode.value = row.section_code || ''; latestBindingSource.value = undefined; latestBinding.value = undefined; bindingOpen.value = true; }
async function saveBinding() {
  const editing = binding.value; const source = bindingSource.value; const code = bindingCode.value.trim();
  if (!editing || !source || !validBindingSelection.value || bindingNeedsReview.value || sync.latestSourceLoading.value) return;
  if (code === (editing.section_code || '')) { ElMessage.info('主题绑定没有变化，无需保存'); bindingOpen.value = false; return; }
  try { await ElMessageBox.confirm('主题章节绑定发生变化。受影响的托管文章会等待重新核验，可能暂时无法在前台阅读。保存后请重新预览并同步核验。是否继续？', '确认公开影响', { confirmButtonText: '保存并重新核验', cancelButtonText: '返回检查' }); } catch { return; }
  if (await sync.saveBinding(editing, code, source.config_revision) && binding.value === editing) bindingOpen.value = false;
}
watch(() => route.query.run_id, id => {
  if (typeof id === 'string' && id.trim()) void openRun(id);
}, { immediate: true });
onMounted(() => { window.addEventListener('resize', syncViewport); void refreshAll(); });
onBeforeUnmount(() => { window.removeEventListener('resize', syncViewport); });
</script>
<style scoped>
.sync-page { color:var(--admin-ink,#20262e); min-width:0; }
h1 { margin:0 0 8px; font-size:26px; font-weight:650; line-height:1.4; }
h2 { margin:0; font-size:17px; font-weight:650; line-height:1.5; }
p { line-height:1.65; overflow-wrap:anywhere; }
.page-heading { display:flex; justify-content:space-between; align-items:flex-start; gap:20px; margin-bottom:24px; }
.page-heading p { margin:0; color:var(--admin-muted,#59636e); font-size:13px; }
.actions { display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
.actions :deep(.el-button + .el-button) { margin-left:0; }
.page-error { margin-bottom:20px; }
.sync-tabs :deep(.el-tabs__header) { margin:0 0 28px; }
.sync-tabs :deep(.el-tabs__nav-wrap::after) { height:1px; background:var(--admin-line,#e1e6eb); }
.sync-tabs :deep(.el-tabs__item) { height:52px; font-size:13px; color:var(--admin-muted,#59636e); }
.sync-tabs :deep(.el-tabs__item.is-active) { color:var(--admin-green,#086858); font-weight:600; }
.sync-tabs :deep(.el-tabs__content) { overflow:visible; }
.panel { border:1px solid var(--admin-line,#e1e6eb); border-radius:8px; padding:24px; background:#fff; margin-bottom:24px; min-width:0; }
.section-heading { display:flex; gap:16px; align-items:center; justify-content:space-between; margin-bottom:20px; min-width:0; }
.section-heading > div { min-width:0; }
.section-heading .hint { margin:5px 0 0; }
.section-heading :deep(.el-button.is-link) { flex:none; font-size:12px; }
.status-tags { display:flex; flex-wrap:wrap; gap:8px; margin:-4px 0 20px; }
.health-dot { display:inline-block; width:7px; height:7px; margin:0 8px 2px 0; border-radius:50%; background:var(--admin-muted,#59636e); }
.health-dot.is-healthy { background:var(--admin-green,#086858); }
.health-dot.is-warning { background:#a36a14; }
.health-facts,.source-facts,.run-facts { display:flex; flex-wrap:wrap; gap:28px; margin:0 0 24px; }
dt { color:var(--admin-muted,#59636e); font-size:12px; margin-bottom:8px; }
dd { margin:0; font-size:14px; overflow-wrap:anywhere; }
.health-facts > div,.source-facts > div,.run-facts > div { min-width:0; }
.health-facts :deep(.el-button) { white-space:normal; text-align:left; }
.health-actions { display:flex; justify-content:space-between; align-items:flex-start; gap:16px; border-top:1px solid var(--admin-line,#e1e6eb); padding-top:16px; }
.hint,.tab-description { color:var(--admin-muted,#59636e); font-size:12px; }
.tab-description { margin:0 0 20px; font-size:13px; }
.maintenance { max-width:420px; text-align:right; }
.maintenance summary { color:var(--admin-green,#086858); cursor:pointer; min-height:40px; display:flex; align-items:center; justify-content:flex-end; gap:8px; }
.maintenance summary::after { content:'⌄'; }
.maintenance > div { padding:12px 0; text-align:left; }
.maintenance :deep(.el-button) { margin:6px 8px 6px 0; }
.gate-reasons { padding:12px 16px 12px 30px; margin:16px 0 0; background:#faf3e3; color:#805611; border-radius:6px; font-size:12px; line-height:1.8; }
.overview-grid { display:grid; grid-template-columns:1.2fr 1fr; gap:24px; }
.source-summary { display:flex; gap:14px; align-items:center; border-top:1px solid var(--admin-line,#e1e6eb); padding:16px 0; min-width:0; }
.source-summary:last-child { padding-bottom:0; }
.source-summary > div:nth-child(2) { min-width:0; flex:1; }
.source-summary strong { font-size:14px; overflow-wrap:anywhere; }
.source-summary p { margin:4px 0 0; }
.topic-icon { display:flex; align-items:center; justify-content:center; width:40px; height:40px; flex:none; background:var(--admin-pale,#eaf5f1); color:var(--admin-green,#086858); font-size:13px; font-weight:650; border-radius:7px; }
.source-error { font-size:12px; color:var(--el-color-danger); }
.attention-item { display:flex; align-items:center; gap:12px; margin:0 0 12px; font-size:13px; }
.run-summary { display:flex; align-items:center; gap:18px; border:0; border-top:1px solid var(--admin-line,#e1e6eb); padding:18px 0; width:100%; background:#fff; text-align:left; color:inherit; cursor:pointer; min-height:80px; }
.run-summary:first-of-type { border-top:0; }
.run-summary:last-child { padding-bottom:0; }
.run-summary > span:nth-child(2) { flex:1; min-width:0; }
.run-summary strong { display:block; font-size:14px; font-weight:600; }
.run-summary small { display:block; margin-top:5px; font-size:12px; color:var(--admin-muted,#59636e); line-height:1.65; }
.run-arrow { color:var(--admin-muted,#59636e); font-size:22px; }
.source-name { display:flex; align-items:center; gap:14px; }
.source-heading .hint { margin:3px 0 0; }
.source-facts { padding-bottom:20px; border-bottom:1px solid var(--admin-line,#e1e6eb); margin-bottom:16px; }
.table-scroll { width:100%; overflow-x:auto; }
.binding-table { min-width:640px; }
.runs-table { min-width:900px; }
.source-identifiers { margin-top:16px; font-size:12px; color:var(--admin-muted,#59636e); overflow-wrap:anywhere; }
.source-identifiers summary { cursor:pointer; min-height:32px; line-height:32px; }
.page-filters { display:flex; gap:14px; align-items:flex-end; flex-wrap:wrap; margin-bottom:24px; }
.page-filters :deep(.el-form-item) { margin:0; min-width:170px; flex:1; }
.page-filters :deep(.el-form-item:first-child) { min-width:220px; flex:1.4; }
.page-filters :deep(.el-select) { width:100%; }
.el-pagination { margin-top:20px; flex-wrap:wrap; max-width:100%; }
.identifier,.identifier-link { overflow-wrap:anywhere; white-space:normal; }
.identifier-link { text-align:left; line-height:1.5; height:auto; }
.run-cards { display:none; }
.source-form,.binding-form { margin-top:24px; }
.source-form h2 { margin:28px 0 18px; }
.source-form :deep(.el-select),.binding-form :deep(.el-select) { width:100%; }
.source-form :deep(.el-form-item) { margin-bottom:24px; }
.source-form :deep(.el-form-item__label),.binding-form :deep(.el-form-item__label) { color:var(--admin-ink,#20262e); font-size:13px; font-weight:600; }
.source-form .hint { width:100%; margin:6px 0 0; }
.revision-comparison { margin:20px 0; padding:16px; background:var(--admin-canvas,#f6f8fa); border:1px solid var(--admin-line,#e1e6eb); border-radius:6px; overflow-wrap:anywhere; }
.revision-comparison > p:first-child { margin-top:0; }
.revision-comparison :deep(.el-table .cell) { overflow-wrap:anywhere; }
.revision-comparison :deep(.el-button) { white-space:normal; height:auto; min-height:40px; line-height:1.5; }
.comparison-card { display:none; }
.publication-impact { padding:12px 16px; border-radius:6px; background:#faf3e3; color:#805611; font-size:13px; }
.run-detail { min-height:100px; }
.run-items { margin-top:24px; }
.run-item { padding:18px 0; border-top:1px solid var(--admin-line,#e1e6eb); }
.run-item .section-heading { margin-bottom:8px; }
.run-item strong { font-size:13px; min-width:0; }
.run-item p { font-size:13px; }
.run-item details { font-size:12px; }
.run-item summary { cursor:pointer; min-height:32px; line-height:32px; color:var(--admin-green,#086858); }
pre { white-space:pre-wrap; overflow-wrap:anywhere; max-height:360px; overflow:auto; background:var(--admin-canvas,#f6f8fa); padding:12px; border-radius:4px; }
@media (max-width:1024px) { .overview-grid { grid-template-columns:1fr; gap:0; } .source-heading { align-items:flex-start; flex-wrap:wrap; } }
@media (max-width:780px) {
  .page-heading { flex-wrap:wrap; gap:16px; }
  .page-heading .actions { width:100%; }
  .panel { padding:20px; }
  .health-facts,.source-facts { gap:20px; }
  .health-facts > div:last-child { flex-basis:100%; }
  .maintenance summary { min-height:44px; }
  .source-identifiers summary,.run-item summary { min-height:44px; line-height:44px; }
  .identifier-link { min-height:44px; }
  .source-heading .actions { width:100%; gap:8px; }
  .source-heading .actions :deep(.el-button) { margin-left:auto; }
  .page-filters { gap:12px; }
  .page-filters :deep(.el-form-item),.page-filters :deep(.el-form-item:first-child) { min-width:100%; }
  .page-filters > .el-button { width:100%; }
  .comparison-table { display:none; }
  .comparison-card { display:block; border-top:1px solid var(--admin-line,#e1e6eb); padding:12px 0; margin:0; }
  .comparison-card dt { font-weight:600; color:var(--admin-ink,#20262e); }
  .comparison-card dd { display:grid; grid-template-columns:70px minmax(0,1fr); gap:8px; font-size:12px; padding:4px 0; }
  .comparison-card span { color:var(--admin-muted,#59636e); }
}
@media (max-width:600px) {
  h1 { font-size:24px; }
  .panel { padding:18px; margin-bottom:20px; }
  .section-heading { gap:10px; margin-bottom:18px; }
  .section-heading h2 { font-size:16px; }
  .sync-tabs :deep(.el-tabs__item) { padding:0 12px; font-size:12px; height:48px; }
  .sync-tabs :deep(.el-tabs__header) { margin-bottom:24px; }
  .health-actions { flex-wrap:wrap; }
  .maintenance { text-align:left; max-width:100%; }
  .maintenance summary { justify-content:flex-start; }
  .health-facts,.source-facts,.run-facts { gap:20px 24px; }
  .health-facts > div,.source-facts > div,.run-facts > div { flex-basis:calc(50% - 12px); }
  .health-facts > div:last-child { flex-basis:100%; }
  .source-summary { flex-wrap:wrap; gap:10px; }
  .run-summary { align-items:flex-start; gap:10px; flex-wrap:wrap; }
  .run-summary > span:nth-child(2) { flex-basis:calc(100% - 34px); }
  .run-list > .table-scroll { display:none; }
  .run-cards { display:block; }
  .run-cards article { border-bottom:1px solid var(--admin-line,#e1e6eb); padding:18px 0; font-size:13px; }
  .run-cards article:first-child { padding-top:0; }
  .run-cards .section-heading { margin-bottom:6px; }
  .page-list { padding:18px 12px; }
}
@media (max-width:350px) { .panel { padding:16px; } .sync-tabs :deep(.el-tabs__item) { padding:0 8px; } .attention-item { gap:8px; font-size:12px; } .health .section-heading { flex-wrap:wrap; } }
</style>
