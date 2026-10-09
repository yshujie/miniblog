<template>
  <section class="content-state" :class="{ compact, loading: status === 'loading' }"
    :role="status === 'error' || status === 'not_found' ? 'alert' : 'status'"
    :aria-live="status === 'loading' ? 'polite' : undefined">
    <template v-if="status === 'loading'">
      <h1 class="loading-label">{{ heading }}</h1>
      <div class="skeleton title" aria-hidden="true"></div>
      <div class="skeleton" aria-hidden="true"></div>
      <div class="skeleton short" aria-hidden="true"></div>
      <div class="skeleton block" aria-hidden="true"></div>
    </template>
    <template v-else>
      <div class="state-symbol" aria-hidden="true">↗</div>
      <h1>{{ heading }}</h1><p v-if="message">{{ message }}</p>
      <div v-if="retryLabel || $slots.actions" class="state-actions">
        <button v-if="retryLabel" type="button" class="primary" @click="$emit('retry')">{{ retryLabel }}</button>
        <slot name="actions" />
      </div>
    </template>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue'
const props = withDefaults(defineProps<{ status: string; title?: string; message?: string; retryLabel?: string; compact?: boolean }>(), { compact: false })
defineEmits<{ retry: [] }>()
const heading = computed(() => props.title || ({ loading: '正在加载…', empty: '暂无已发布文章', not_found: '内容不可用', error: '暂时无法加载' }[props.status] || ''))
</script>
<style scoped>
.content-state { flex: 1; min-width: 0; min-height: 0; overflow: auto; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 64px 28px; text-align: center; background: var(--page); }
.content-state h1 { font-size: 26px; line-height: 1.5; overflow-wrap: anywhere; margin-bottom: 14px; }
.content-state p { font-size: 14px; color: var(--secondary); max-width: 440px; line-height: 1.9; margin-bottom: 26px; }
.state-symbol { display: grid; place-items: center; width: 58px; height: 58px; border: 1px solid var(--line); border-radius: 16px; background: var(--zone); color: var(--green); font-size: 26px; margin-bottom: 24px; }
.state-actions { display: flex; justify-content: center; flex-wrap: wrap; gap: 12px; }
.compact { padding: 28px 0; align-items: flex-start; text-align: left; } .compact .state-symbol { display: none; } .compact h1 { font-size: 19px; }
.loading { align-items: stretch; text-align: left; justify-content: flex-start; } .loading .loading-label { font-family: var(--sans); font-size: 14px; color: var(--secondary); margin-bottom: 28px; }
.skeleton { width: 100%; height: 15px; margin-bottom: 16px; border-radius: 5px; background: #eef1f3; } .skeleton.title { width: 58%; height: 28px; margin-bottom: 24px; } .skeleton.short { width: 72%; } .skeleton.block { height: 120px; margin-top: 16px; }
@media (max-width: 650px) { .content-state { padding: 38px 24px; } .content-state h1 { font-size: 23px; } .content-state p { font-size: 13px; } .compact { padding: 28px 0; } }
</style>
