<template>
  <section class="bootstrap-preview">
    <p class="notice">这是已保存的历史基线审核清单，仅供核对；此页面不会写入 Notion 或直接接管文章。确认关联与当前博客状态后，再通过受控审核流程执行。</p>
    <p v-if="!items.length" class="notice">本页暂无审核候选。</p>
    <article v-for="(item, index) in items" :key="item.item_id || item.page_id || index" class="candidate">
      <template v-if="candidate(item)">
        <h3>{{ candidate(item)!.title || candidate(item)!.page_id }}</h3>
        <p class="notice">{{ sourceLabels[candidate(item)!.source_id] || candidate(item)!.source_id }} · 页面 {{ candidate(item)!.page_id }}</p>
        <div class="comparison">
          <section>
            <h4>Notion 来源资料</h4>
            <dl><dt>标题</dt><dd>{{ candidate(item)!.title || '未填写' }}</dd><dt>主题</dt><dd>{{ candidate(item)!.topic || '未填写' }}</dd><dt>来源期望状态</dt><dd>{{ candidate(item)!.notion_state ? syncStateLabel(candidate(item)!.notion_state) : '未填写或无法识别' }}</dd></dl>
          </section>
          <section>
            <h4>当前博客资料</h4>
            <template v-if="candidate(item)!.article_id">
              <dl><dt>文章</dt><dd><router-link :to="articleURL(candidate(item)!.article_id!)">{{ candidate(item)!.local_title || candidate(item)!.article_id }}</router-link><span class="identifier">文章 ID {{ candidate(item)!.article_id }}</span></dd><dt>本站状态</dt><dd>{{ candidate(item)!.local_state ? syncStateLabel(candidate(item)!.local_state) : '尚未提供' }}</dd><dt>当前位置</dt><dd>{{ directory(candidate(item)!.local_section_code, candidate(item)!.local_subsection_code) }}</dd></dl>
            </template>
            <p v-else>尚未确认本站文章关联；{{ candidate(item)!.new_page ? '可能是全新页面，仍需审核是否有历史文章' : '请先核对历史文章' }}。</p>
          </section>
        </div>
        <dl class="proposal"><dt>关联依据</dt><dd>{{ matchLabel(candidate(item)!.match_method) }}</dd><dt>建议主题位置</dt><dd>{{ directoryLabels[candidate(item)!.proposed_section_code || ''] || candidate(item)!.proposed_section_title || candidate(item)!.proposed_section_code || '尚未确定' }}</dd><dt>位置处理</dt><dd>{{ candidate(item)!.placement_change || '尚未确定' }}</dd><dt>公开阅读条件</dt><dd>{{ publicLabel(candidate(item)!.public_condition) }}</dd></dl>
        <p v-if="candidate(item)!.requires_legacy_alias" class="warning">需要额外确认旧阅读链接与此 Notion 页面确为同一文档，审核流程才能登记历史别名。</p>
        <p v-if="candidate(item)!.publish_block_reason" class="warning">公开限制：{{ syncReasonLabel(candidate(item)!.publish_block_reason) }}</p>
        <p v-if="candidate(item)!.reason || item.reason || item.error" class="warning">待核对原因：{{ syncReasonLabel(candidate(item)!.reason || item.reason || item.error) }}</p>
        <div v-if="candidate(item)!.candidate_article_ids?.length" class="matches"><strong>候选历史文章</strong><router-link v-for="id in candidate(item)!.candidate_article_ids" :key="id" :to="articleURL(id)">文章 {{ id }}</router-link></div>
        <div v-if="candidate(item)!.title_hint_article_ids?.length" class="matches"><strong>仅同名参考，不作为自动关联依据</strong><router-link v-for="id in candidate(item)!.title_hint_article_ids" :key="id" :to="articleURL(id)">文章 {{ id }}</router-link></div>
      </template>
      <template v-else>
        <h3>{{ item.page_id || '审核记录' }}</h3>
        <p>这条旧审核记录没有完整对照清单，请重新生成审核预览后核对。</p>
        <p v-if="item.reason || item.error" class="warning">{{ syncReasonLabel(item.reason || item.error) }}</p>
      </template>
    </article>
  </section>
</template>
<script setup lang="ts">
import type { BootstrapCandidate, SyncItem } from '@/types/notion-sync';
import { syncStateLabel, syncReasonLabel } from './sync-presentation';
const props = withDefaults(defineProps<{ items: SyncItem[]; sourceLabels?: Record<string, string>; directoryLabels?: Record<string, string> }>(), { sourceLabels: () => ({}), directoryLabels: () => ({}) });
function candidate(item: SyncItem): BootstrapCandidate | undefined {
  if (!item.after || typeof item.after !== 'object' || !('bootstrap_preview' in item.after)) return undefined;
  const value = item.after.bootstrap_preview;
  return value && typeof value === 'object' && 'page_id' in value && typeof value.page_id === 'string' && 'source_id' in value && typeof value.source_id === 'string' ? value as BootstrapCandidate : undefined;
}
const articleURL = (id: string) => `/article/edit/${encodeURIComponent(id)}`;
function directory(section?: string, subsection?: string) {
  // The parent loads current catalog names; absent historical codes remain inspectable.
  return props.directoryLabels[subsection || section || ''] || [section, subsection].filter(Boolean).join(' / ') || '尚未提供';
}
function matchLabel(value: string) {
  const labels: Record<string, string> = { trusted_page_id: '可信页面 ID 一致', known_reading_url: '已知阅读链接一致', explicit_alias_review: '人工指定关联，需审核旧链接别名', unmatched: '尚未找到可信历史关联', ambiguous: '多个候选或身份冲突，需要人工核对' };
  return labels[value] || value || '尚未提供';
}
function publicLabel(value: string) {
  const labels: Record<string, string> = { public_url_available: '来源返回公开阅读链接，仍需核验本站状态与公开条件', public_url_missing: '来源没有公开阅读链接，暂不能公开', native_archived_or_in_trash: '来源已归档或移入回收站，不能公开' };
  return labels[value] || value || '尚未提供';
}
</script>
<style scoped>
.notice { color:var(--el-text-color-secondary); font-size:13px; overflow-wrap:anywhere; }
.candidate { margin:16px 0; padding:16px; border:1px solid var(--el-border-color); border-radius:8px; }
h3,h4 { margin:0 0 12px; } .comparison { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:16px; }
dl { display:grid; grid-template-columns:100px minmax(0,1fr); gap:8px; } dt { color:var(--el-text-color-secondary); } dd { margin:0; overflow-wrap:anywhere; }
.identifier { display:block; font-size:12px; color:var(--el-text-color-secondary); } .warning { color:var(--el-color-warning-dark-2); }
.matches { display:flex; flex-wrap:wrap; gap:10px; margin-top:12px; } .matches strong { flex-basis:100%; }
@media (max-width:640px) { .comparison { grid-template-columns:1fr; } dl { grid-template-columns:85px minmax(0,1fr); } .candidate { padding:12px; } }
</style>
