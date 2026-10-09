<template>
  <section class="context-panel" aria-label="所选目录资料">
    <template v-if="node && item">
      <div class="context-heading"><div><p class="eyebrow">{{ kindLabels[node.kind] }} · {{ path }}</p><h2>{{ item.title }}</h2></div><el-tag :type="item.status === 1 ? 'success' : 'info'">{{ statusLabel(item.status) }}</el-tag></div>
      <div class="context-meta"><span>代码 <code>{{ node.code }}</code></span><span>排序 {{ item.sort ?? '尚未提供' }}</span></div>
      <div class="context-actions"><el-button @click="openEdit">编辑资料</el-button><el-button v-if="node.kind !== 'subsection'" @click="openCreate(node.kind === 'module' ? 'section' : 'subsection')">新增{{ node.kind === 'module' ? '章节' : '子章节' }}</el-button>
        <el-dropdown trigger="click"><el-button :loading="busy">更多操作 <span aria-hidden="true">⌄</span></el-button><template #dropdown><el-dropdown-menu>
          <el-dropdown-item v-if="item.status !== 1" @click="changeVisibility(true)">上架目录</el-dropdown-item><el-dropdown-item v-if="item.status !== 2" @click="changeVisibility(false)">下架目录</el-dropdown-item><el-dropdown-item divided @click="remove">永久删除目录</el-dropdown-item>
        </el-dropdown-menu></template></el-dropdown>
      </div>
    </template>
    <template v-else><p class="eyebrow">目录与收录</p><h2>组织内容，让读者顺着目录阅读</h2><p class="context-description">从左侧选择主题、章节或子章节，管理目录资料与当前范围的文章。</p><el-button @click="openCreate('module')">新增主题</el-button></template>
    <el-alert v-if="actionError" :title="actionError" type="error" :closable="false" class="context-error" />
  </section>
  <el-drawer v-model="formOpen" :title="`${editing ? '编辑' : '新增'}${kindLabels[formKind]}`" size="min(540px, 100%)" :close-on-click-modal="false" :before-close="beforeClose" class="catalog-editor">
    <p v-if="parentLabel" class="parent-note">所属{{ formKind === 'section' ? '主题' : '章节' }}：{{ parentLabel }}</p>
    <p v-if="editing" class="parent-note">目录代码与所属位置保持不变。</p>
    <el-alert v-if="formError" :title="formError" type="error" :closable="false" />
    <el-form label-position="top" :disabled="busy" @submit.prevent="save">
      <el-form-item :label="`${kindLabels[formKind]}名称`" required><el-input v-model="form.title" data-test="catalog-title" maxlength="255" /></el-form-item>
      <el-form-item label="目录代码" required><el-input v-model="form.code" data-test="catalog-code" :disabled="editing" placeholder="稳定且唯一的代码" /><span class="field-note">用于目录地址，创建后不可修改。</span></el-form-item>
      <el-form-item label="排序值"><el-input-number v-model="form.sort" :min="0" :precision="0" /><span class="field-note">也可在目录树中按同级顺序调整。</span></el-form-item>
    </el-form>
    <template #footer><div class="sheet-actions"><el-button :disabled="busy" @click="beforeClose(() => formOpen = false)">取消</el-button><el-button type="primary" :loading="busy" @click="save">{{ editing ? '保存资料' : '创建目录' }}</el-button></div></template>
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import useModuleStore from '@/store/modules/module';
import useSectionStore from '@/store/modules/section';
import useSubsectionStore from '@/store/modules/subsection';
import { useCatalog, type DirectoryNode } from '@/composables/useCatalog';
import { errorMessage } from '@/utils/api-error';
import type { CatalogOrder, DirectoryContext } from '@/types/content';
const props = defineProps<{ node?: DirectoryNode }>();
const emit = defineEmits<{ changed: [context: DirectoryContext, deleted?: boolean] }>();
const modules = useModuleStore(); const sections = useSectionStore(); const subsections = useSubsectionStore(); const catalog = useCatalog();
const kindLabels: Record<CatalogOrder['kind'], string> = { module: '主题', section: '章节', subsection: '子章节' };
const item = computed(() => { const node = props.node; if (!node) return undefined; if (node.kind === 'module') return modules.getModuleByCode(node.code); if (node.kind === 'section') return sections.getSectionsByModule(node.module_code).find(value => value.code === node.code); return subsections.getSubsectionsBySection(node.section_code).find(value => value.code === node.code); });
const path = computed(() => props.node ? catalog.label(props.node) : '');
const statusLabel = (value?: number) => value === 1 ? '正常' : value === 2 ? '未上架' : '状态待确认';
const busy = ref(false); const actionError = ref(''); const formError = ref(''); const formOpen = ref(false); const editing = ref(false); const formKind = ref<CatalogOrder['kind']>('module');
const form = reactive({ code: '', title: '', sort: 0 }); const formContext = ref<DirectoryContext>({ module_code: '', section_code: '', subsection_code: '' }); const baseline = ref('');
const signature = () => JSON.stringify(form); const parentLabel = computed(() => formKind.value === 'section' ? modules.getModuleByCode(formContext.value.module_code)?.title : formKind.value === 'subsection' ? sections.getSectionsByModule(formContext.value.module_code).find(value => value.code === formContext.value.section_code)?.title : '');
function openCreate(kind: CatalogOrder['kind']) {
  if (busy.value || (kind !== 'module' && !props.node)) return;
  editing.value = false; formKind.value = kind; formError.value = ''; formContext.value = props.node ? { module_code: props.node.module_code, section_code: props.node.section_code, subsection_code: '' } : { module_code: '', section_code: '', subsection_code: '' };
  Object.assign(form, { code: '', title: '', sort: 0 }); baseline.value = signature(); formOpen.value = true;
}
function openEdit() { if (!props.node || !item.value || busy.value) return; editing.value = true; formKind.value = props.node.kind; formContext.value = { ...props.node }; Object.assign(form, { code: props.node.code, title: item.value.title, sort: item.value.sort || 0 }); baseline.value = signature(); formError.value = ''; formOpen.value = true; }
async function beforeClose(done: () => void) { if (busy.value) return; if (signature() !== baseline.value) { try { await ElMessageBox.confirm('目录资料尚未保存，关闭并放弃本次修改？', '未保存修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }); } catch { return; } } done(); }
async function save() {
  if (busy.value) return;
  if (!form.title.trim() || !form.code.trim()) { formError.value = '请填写名称与目录代码'; return; }
  busy.value = true; formError.value = '';
  const kind = formKind.value; const context = { ...formContext.value }; const payload = { title: form.title.trim(), sort: form.sort };
  try {
    if (kind === 'module') { if (editing.value) await modules.updateExistingModule(form.code, payload); else await modules.createNewModule({ ...payload, code: form.code.trim() }); context.module_code = form.code.trim(); context.section_code = ''; context.subsection_code = ''; } else if (kind === 'section') { if (editing.value) await sections.updateSection(form.code, payload); else await sections.createSection({ ...payload, code: form.code.trim(), module_code: context.module_code }); context.section_code = form.code.trim(); context.subsection_code = ''; } else { if (editing.value) await subsections.updateSubsection(form.code, payload); else await subsections.createSubsection({ ...payload, code: form.code.trim(), section_code: context.section_code }); context.subsection_code = form.code.trim(); }
    formOpen.value = false; emit('changed', context); ElMessage.success(editing.value ? '目录资料已保存' : '目录已创建');
  } catch (cause) { formError.value = errorMessage(cause, '保存目录失败，本次输入已保留'); } finally { busy.value = false; }
}
async function changeVisibility(published: boolean) {
  const node = props.node; if (!node || busy.value) return;
  try { await ElMessageBox.confirm(published ? '上架后，文章仍按自身与完整目录状态决定是否公开。' : '下架会影响此目录下文章的前台可见性，不删除目录或文章。', `${published ? '上架' : '下架'}${kindLabels[node.kind]}`); } catch { return; }
  busy.value = true; actionError.value = '';
  try { if (node.kind === 'module') await (published ? modules.publishExistingModule(node.code) : modules.unpublishExistingModule(node.code)); else if (node.kind === 'section') await (published ? sections.publishSection(node.code) : sections.unpublishSection(node.code)); else await (published ? subsections.publishSubsection(node.code) : subsections.unpublishSubsection(node.code)); emit('changed', { ...node }); ElMessage.success('目录状态已更新'); } catch (cause) { actionError.value = errorMessage(cause, '目录状态更新失败'); } finally { busy.value = false; }
}
async function remove() {
  const node = props.node; if (!node || busy.value) return;
  try { await ElMessageBox.confirm(`将永久删除“${item.value?.title || node.label}”。若仍有子目录或文章，服务会拒绝删除；不会自动删除关联内容。`, '永久删除目录', { confirmButtonText: '永久删除', cancelButtonText: '取消', type: 'warning' }); } catch { return; }
  busy.value = true; actionError.value = '';
  try { if (node.kind === 'module') await modules.deleteExistingModule(node.code); else if (node.kind === 'section') await sections.deleteSection(node.code); else await subsections.deleteSubsection(node.code); emit('changed', { ...node }, true); ElMessage.success('目录已删除'); } catch (cause) { actionError.value = errorMessage(cause, '删除目录失败'); } finally { busy.value = false; }
}
defineExpose({ openCreate });
</script>
<style scoped>
.context-panel { border:1px solid var(--admin-line); background:white; border-radius:8px; padding:24px; margin-bottom:24px; min-width:0; } .context-heading { display:flex; justify-content:space-between; align-items:flex-start; gap:16px; } h2 { margin:6px 0 14px; font-size:24px; letter-spacing:-.5px; overflow-wrap:anywhere; color:var(--admin-ink); } .eyebrow,.context-description,.parent-note,.field-note { color:var(--admin-muted); } .eyebrow { font-size:12px; margin:0; overflow-wrap:anywhere; } .context-meta { display:flex; flex-wrap:wrap; gap:20px; color:var(--admin-muted); font-size:13px; } code { color:var(--admin-ink); } .context-actions { display:flex; flex-wrap:wrap; gap:8px; margin-top:18px; } .context-actions :deep(.el-button + .el-button) { margin-left:0; } .context-error { margin-top:16px; } .field-note { display:block; font-size:12px; line-height:1.6; margin-top:6px; } .sheet-actions { display:flex; justify-content:flex-end; gap:8px; } @media(max-width:780px) { .context-panel { padding:18px; border-radius:8px; margin-bottom:16px; } h2 { font-size:21px; } .context-heading { flex-wrap:wrap; } }
@media(max-width:780px) { :global(.catalog-editor) { width:100%!important; } }
</style>
