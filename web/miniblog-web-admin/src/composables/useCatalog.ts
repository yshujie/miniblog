import { computed } from 'vue';
import useModuleStore from '@/store/modules/module';
import useSectionStore from '@/store/modules/section';
import useSubsectionStore from '@/store/modules/subsection';
import type { DirectoryContext, CatalogOrder } from '@/types/content';
import { reorderCatalog } from '@/api/content';
export interface DirectoryNode extends DirectoryContext { key: string; label: string; kind: CatalogOrder['kind']; code: string; children?: DirectoryNode[] }
const nodeLabel = (item: { title: string; status?: number }) => item.status === 1 ? item.title : `${item.title}（${item.status === 2 ? '未上架' : '状态待确认'}）`;
export function useCatalog() {
  const modules = useModuleStore(); const sections = useSectionStore(); const subsections = useSubsectionStore();
  async function load(force = false) {
    await modules.ensureLoaded(force);
    await Promise.all(modules.modules.map(async module => {
      await sections.fetchSections(module.code, force);
      await Promise.all(sections.getSectionsByModule(module.code).map(section => subsections.fetchSubsections(section.code, force)));
    }));
  }
  const tree = computed<DirectoryNode[]>(() => modules.modules.map(module => ({
    key: `module:${module.code}`, code: module.code, label: nodeLabel(module), kind: 'module', module_code: module.code, section_code: '', subsection_code: '',
    children: sections.getSectionsByModule(module.code).map(section => ({
      key: `section:${section.code}`, code: section.code, label: nodeLabel(section), kind: 'section', module_code: module.code, section_code: section.code, subsection_code: '',
      children: subsections.getSubsectionsBySection(section.code).map(subsection => ({ key: `subsection:${subsection.code}`, code: subsection.code, label: nodeLabel(subsection), kind: 'subsection', module_code: module.code, section_code: section.code, subsection_code: subsection.code }))
    }))
  })));
  function valid(context?: DirectoryContext): context is DirectoryContext {
    if (!context || !modules.modules.some(item => item.code === context.module_code && item.status === 1)) return false;
    if (!sections.getSectionsByModule(context.module_code).some(item => item.code === context.section_code && item.status === 1)) return false;
    return !context.subsection_code || subsections.getSubsectionsBySection(context.section_code).some(item => item.code === context.subsection_code && item.status === 1);
  }
  function label(context: DirectoryContext) {
    return [modules.getModuleByCode(context.module_code)?.title, sections.getSectionsByModule(context.module_code).find(item => item.code === context.section_code)?.title, subsections.getSubsectionsBySection(context.section_code).find(item => item.code === context.subsection_code)?.title].filter(Boolean).join(' / ');
  }
  function siblings(node: DirectoryNode) {
    if (node.kind === 'module') return modules.modules;
    if (node.kind === 'section') return sections.getSectionsByModule(node.module_code);
    return subsections.getSubsectionsBySection(node.section_code);
  }
  async function move(node: DirectoryNode, direction: -1 | 1) {
    const items = siblings(node).map(item => item.code); const index = items.indexOf(node.code); const target = index + direction;
    if (index < 0 || target < 0 || target >= items.length) return;
    [items[index], items[target]] = [items[target], items[index]];
    await reorderCatalog({ kind: node.kind, parent_code: node.kind === 'section' ? node.module_code : node.kind === 'subsection' ? node.section_code : undefined, codes: items });
    if (node.kind === 'module') modules.invalidate();
    else if (node.kind === 'section') sections.invalidate(node.module_code);
    else subsections.invalidate(node.section_code);
    await load(true);
  }
  return { modules, sections, subsections, tree, load, valid, label, siblings, move };
}
