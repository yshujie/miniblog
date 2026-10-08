import { computed, reactive, ref } from 'vue';
import { getArticle, saveArticle, changeArticleStatus, saveArticleLocalFields, setPublicationHold } from '@/api/content';
import type { ArticleInfo, ArticleStatus, UpdateArticleRequest } from '@/types/content';
import { canArticleAction, isManagedArticle } from '@/utils/article-actions';
import { ApiError, errorMessage } from '@/utils/api-error';
export function useArticleEditor() {
  const article = ref<ArticleInfo>(); const loading = ref(false); const error = ref(''); const baseline = ref('');
  const form = reactive<UpdateArticleRequest>({ id: '', title: '', author: '', tags: [], module_code: '', section_code: '', subsection_code: '', external_link: '' });
  const signature = () => JSON.stringify(isManagedArticle(article.value) ? { author: form.author } : form);
  const conflictDraft = ref<UpdateArticleRequest>();
  let version = 0;
  const dirty = computed(() => Boolean(article.value && signature() !== baseline.value));
  function accept(value: ArticleInfo, preserveAuthor = false) {
    const author = form.author; const preserve = preserveAuthor && dirty.value;
    article.value = value;
    Object.assign(form, { id: value.id, title: value.title, author: value.author, tags: [...value.tags], module_code: value.module?.code || '', section_code: value.section?.code || '', subsection_code: value.subsection?.code || '', external_link: value.external_link });
    baseline.value = signature();
    if (preserve) form.author = author;
  }
  async function load(id: string) {
    const current = ++version; loading.value = true; error.value = '';
    try { const result = await getArticle(id); if (current === version) accept(result.article); } catch (cause) { if (current === version) error.value = errorMessage(cause, '读取文章失败'); } finally { if (current === version) loading.value = false; }
  }
  async function save(): Promise<boolean> {
    if (!article.value || loading.value) return false;
    error.value = '';
    const managed = isManagedArticle(article.value);
    if (!canArticleAction(article.value, managed ? 'edit_local_fields' : 'edit')) { error.value = '这些资料由来源管理，不能在此修改'; return false; }
    if (!managed && (!form.title.trim() || !form.section_code)) { error.value = '请填写标题及所属章节'; return false; }
    loading.value = true;
    try { accept((await (managed ? saveArticleLocalFields(article.value.id, { author: form.author }) : saveArticle({ ...form, tags: [...form.tags] }))).article); return true; } catch (cause) {
      error.value = errorMessage(cause, '保存失败');
      if (cause instanceof ApiError && cause.code === 'SourceManaged') {
        conflictDraft.value = { ...form, tags: [...form.tags] };
        try {
          const current = (await getArticle(form.id)).article;
          if (isManagedArticle(current)) { accept(current, true); error.value += '。已读取来源管理资料；本地作者和接管前输入已保留。'; }
        } catch { error.value += '。读取最新资料失败，所有输入已保留。'; }
      }
      return false;
    } finally { loading.value = false; }
  }
  async function changeStatus(command: 'publish' | 'unpublish' | 'archive' | 'restore', saveDirty = false): Promise<boolean> {
    if (!article.value || loading.value) return false;
    if (isManagedArticle(article.value) && !canArticleAction(article.value, command)) { error.value = '发布状态由来源管理，请在来源修改；紧急下架请使用本地下架'; return false; }
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
  async function changeHold(held: boolean, reason?: string): Promise<boolean> {
    if (!article.value || loading.value || !canArticleAction(article.value, held ? 'hold' : 'release_hold')) return false;
    loading.value = true; error.value = '';
    try { accept((await setPublicationHold(article.value.id, { held, reason })).article, true); return true; } catch (cause) { error.value = errorMessage(cause, '本地下架状态更新失败'); return false; } finally { loading.value = false; }
  }
  return { form, article, loading, error, dirty, conflictDraft, load, save, changeStatus, changeHold };
}
