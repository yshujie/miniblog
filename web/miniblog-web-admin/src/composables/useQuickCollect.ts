import { computed, onScopeDispose, reactive, ref } from 'vue';
import { previewSource, registerArticle } from '@/api/content';
import type { ArticleInfo, CollectRequest, CollectResult, DirectoryContext, SourcePreview } from '@/types/content';
import { ApiError, errorMessage, isUncertain } from '@/utils/api-error';
export function useQuickCollect() {
  const form = reactive({ external_link: '', title: '', module_code: '', section_code: '', subsection_code: '', author: '', tags: [] as string[] });
  const titleTouched = ref(false); const preview = ref<SourcePreview>(); const duplicate = ref<ArticleInfo>();
  const previewLoading = ref(false); const submitting = ref(false); const error = ref('');
  const frozen = ref<{ payload: CollectRequest; key: string; context: DirectoryContext; continueAdding: boolean }>();
  const requestLocked = computed(() => submitting.value || Boolean(frozen.value));
  let linkVersion = 0; let timer: ReturnType<typeof setTimeout> | undefined;
  const context = (): DirectoryContext => ({ module_code: form.module_code, section_code: form.section_code, subsection_code: form.subsection_code });
  function setTitle(value: string) { form.title = value; titleTouched.value = true; }
  function setDirectory(value: DirectoryContext) { Object.assign(form, value); }
  function reset(value: DirectoryContext, author: string) {
    if (timer) clearTimeout(timer);
    linkVersion += 1;
    Object.assign(form, value, { external_link: '', title: '', author, tags: [] });
    titleTouched.value = false; preview.value = undefined; duplicate.value = undefined; frozen.value = undefined; error.value = ''; previewLoading.value = false;
  }
  async function suggestTitle() {
    const url = form.external_link.trim(); const version = ++linkVersion;
    if (!url) return;
    previewLoading.value = true;
    try {
      const result = await previewSource(url);
      if (version !== linkVersion || url !== form.external_link.trim()) return;
      preview.value = result; duplicate.value = result.existing_article;
      if (!titleTouched.value && result.title) form.title = result.title;
    } catch (cause) {
      if (version === linkVersion) preview.value = { provider: 'external', canonical_url: url, title: '', metadata_status: 'manual_required', reason: errorMessage(cause, '无法获取标题，请手动填写') };
    } finally { if (version === linkVersion) previewLoading.value = false; }
  }
  function setLink(value: string) {
    if (value === form.external_link) return;
    form.external_link = value; linkVersion += 1; previewLoading.value = false; preview.value = undefined; duplicate.value = undefined;
    if (!titleTouched.value) form.title = '';
    if (timer) clearTimeout(timer);
    if (value.trim()) timer = setTimeout(() => { void suggestTitle(); }, 500);
  }
  function validation(): string {
    try { const url = new URL(form.external_link.trim()); if (!['https:', 'http:'].includes(url.protocol)) return '请输入 http(s) 文档链接'; } catch { return '请输入有效文档链接'; }
    if (!form.title.trim()) return '请填写标题';
    if (!form.section_code) return '请选择所属章节';
    return '';
  }
  async function submit(publish = true, continueAdding = false): Promise<CollectResult | undefined> {
    if (submitting.value) return;
    error.value = '';
    if (!frozen.value) {
      const invalid = validation(); if (invalid) { error.value = invalid; return; }
      frozen.value = {
        payload: { external_link: form.external_link.trim(), title: form.title.trim(), section_code: form.section_code, subsection_code: form.subsection_code || undefined, author: form.author.trim(), tags: [...form.tags], publish },
        key: crypto.randomUUID(), context: context(), continueAdding
      };
    }
    const attempt = frozen.value;
    submitting.value = true;
    // Pending preview must never rewrite the title after this payload is frozen.
    linkVersion += 1; previewLoading.value = false; if (timer) clearTimeout(timer);
    try {
      const result = await registerArticle(attempt.payload, attempt.key);
      frozen.value = undefined;
      if (result.outcome === 'already_registered') { duplicate.value = result.article; error.value = '该文档已收录，未修改现有文章。'; } else if (attempt.continueAdding) reset(attempt.context, attempt.payload.author);
      return result;
    } catch (cause) {
      error.value = errorMessage(cause, '收录失败，请重试');
      if (cause instanceof ApiError && cause.code === 'ContentRegistrationUnavailable') {
        error.value = '收录暂未开放，请待来源迁移与功能切换完成后重试；本次输入已保留。';
        frozen.value = undefined;
      } else if (!isUncertain(cause)) frozen.value = undefined;
      else error.value += '。结果尚未确认，可原样重试。';
      return undefined;
    } finally { submitting.value = false; }
  }
  function editAgain() { frozen.value = undefined; error.value = ''; }
  onScopeDispose(() => { linkVersion += 1; if (timer) clearTimeout(timer); });
  return { form, titleTouched, preview, duplicate, previewLoading, submitting, error, frozen, requestLocked, context, setTitle, setLink, setDirectory, reset, suggestTitle, submit, editAgain, validation };
}
