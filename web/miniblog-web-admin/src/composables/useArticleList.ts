import { reactive, ref } from 'vue';
import { fetchArticles } from '@/api/content';
import type { ArticleFilters, ArticleInfo } from '@/types/content';
import { errorMessage } from '@/utils/api-error';
export function useArticleList() {
  const filters = reactive<ArticleFilters>({ module_code: '', section_code: '', subsection_code: '', title: '', status: '', direct_only: false, page: 1, limit: 20 });
  const articles = ref<ArticleInfo[]>([]); const total = ref(0); const hasResult = ref(false); const loading = ref(false); const error = ref('');
  let version = 0;
  async function search(resetPage = false) {
    if (resetPage) filters.page = 1;
    const requestVersion = ++version; loading.value = true; error.value = '';
    try {
      const result = await fetchArticles({ ...filters });
      if (requestVersion !== version) return;
      articles.value = result.articles; total.value = result.total; hasResult.value = true;
      if (!result.articles.length && result.total > 0 && filters.page > 1) { filters.page = Math.ceil(result.total / filters.limit); await search(); }
    } catch (cause) { if (requestVersion === version) { error.value = errorMessage(cause, '加载文章失败'); articles.value = []; total.value = 0; hasResult.value = false; } } finally { if (requestVersion === version) loading.value = false; }
  }
  return { filters, articles, total, hasResult, loading, error, search };
}
