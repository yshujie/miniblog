import { computed, reactive, ref } from 'vue';
import { getArticle, saveArticle, changeArticleStatus } from '@/api/content';
import type { ArticleInfo, ArticleStatus, UpdateArticleRequest } from '@/types/content';
import { errorMessage } from '@/utils/api-error';
export function useArticleEditor() {
  const article = ref<ArticleInfo>(); const loading = ref(false); const error = ref(''); const baseline = ref('');
  const form = reactive<UpdateArticleRequest>({ id: '', title: '', author: '', tags: [], module_code: '', section_code: '', subsection_code: '', external_link: '' });
  const signature = () => JSON.stringify(form);
  const dirty = computed(() => Boolean(article.value && signature() !== baseline.value));
  function accept(value: ArticleInfo) {
    article.value = value;
    Object.assign(form, { id: value.id, title: value.title, author: value.author, tags: [...value.tags], module_code: value.module?.code || '', section_code: value.section?.code || '', subsection_code: value.subsection?.code || '', external_link: value.external_link });
    baseline.value = signature();
  }
  async function load(id: string) {
    loading.value = true; error.value = '';
    try { accept((await getArticle(id)).article); } catch (cause) { error.value = errorMessage(cause, '读取文章失败'); } finally { loading.value = false; }
  }
  async function save(): Promise<boolean> {
    if (!article.value || loading.value) return false;
    error.value = '';
    if (!form.title.trim() || !form.section_code) { error.value = '请填写标题及所属章节'; return false; }
    loading.value = true;
    try { accept((await saveArticle({ ...form, tags: [...form.tags] })).article); return true; } catch (cause) { error.value = errorMessage(cause, '保存失败'); return false; } finally { loading.value = false; }
  }
  async function changeStatus(command: 'publish' | 'unpublish' | 'archive' | 'restore', saveDirty = false): Promise<boolean> {
    if (!article.value || loading.value) return false;
    if (dirty.value) {
      if (!saveDirty) { error.value = '有未保存的修改，请先保存后再操作'; return false; }
      if (!await save()) return false;
    }
    loading.value = true; error.value = '';
    try {
      const result = await changeArticleStatus(form.id, command);
      if (result?.article) accept(result.article);
      else {
        const statuses: Record<typeof command, ArticleStatus> = { publish: 'Published', unpublish: 'Unpublished', archive: 'Deleted', restore: 'Draft' };
        article.value = { ...article.value, status: statuses[command] };
      }
      // A confirmed command stays confirmed even if the follow-up read is unavailable.
      try { accept((await getArticle(form.id)).article); } catch (cause) { error.value = `状态已更新，但读取最新文章失败：${errorMessage(cause)}。可重新读取，无需重复状态操作。`; }
      return true;
    } catch (cause) { error.value = errorMessage(cause, '状态变更失败'); return false; } finally { loading.value = false; }
  }
  return { form, article, loading, error, dirty, load, save, changeStatus };
}
