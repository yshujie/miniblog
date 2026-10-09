<template>
  <div class="article-actions"><el-button link type="primary" @click="emit('edit')">{{ isManagedArticle(article) ? '查看资料' : '编辑' }}</el-button><el-dropdown v-if="commands.length" trigger="click"><el-button link :disabled="busy" aria-label="更多文章操作">更多 <span aria-hidden="true">⌄</span></el-button><template #dropdown><el-dropdown-menu><el-dropdown-item v-for="command in commands" :key="command.action" :disabled="busy" @click="emit('command', command.action)">{{ command.label }}</el-dropdown-item></el-dropdown-menu></template></el-dropdown></div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { canArticleAction, isManagedArticle } from '@/utils/article-actions';
import type { ArticleInfo } from '@/types/content';
const props = defineProps<{ article: ArticleInfo; busy?: boolean; canReorder?: boolean }>();
const emit = defineEmits<{ edit: []; command: [action:string] }>();
const commands = computed(() => [{ action: 'move', label: '移动目录' }, { action: 'publish', label: '发布' }, { action: 'unpublish', label: '下架' }, { action: 'archive', label: '归档' }, { action: 'restore', label: '恢复草稿' }, { action: 'hold', label: '紧急下架' }, { action: 'release_hold', label: '解除本地下架' }, ...(props.canReorder ? [{ action: 'up', label: '上移' }, { action: 'down', label: '下移' }] : [])].filter(command => canArticleAction(props.article, ['up', 'down'].includes(command.action) ? 'reorder' : command.action)));
</script>
<style scoped>.article-actions { display:flex; align-items:center; gap:12px; } .article-actions :deep(.el-button) { margin:0; min-height:40px; } @media(max-width:780px) { .article-actions :deep(.el-button) { min-height:44px; min-width:44px; } }</style>
