export type ArticleStatus = 'Draft' | 'Published' | 'Unpublished' | 'Deleted';
export interface ModuleItem { id?: string; code: string; title: string; status?: number; sort?: number }
export interface SectionItem { code: string; title: string; module_code: string; sort?: number; status?: number }
export interface SubsectionItem { code: string; title: string; section_code: string; sort?: number; status?: number }
export interface DirectoryContext { module_code: string; section_code: string; subsection_code: string }
export interface ArticleInfo {
  id: string; title: string; author: string; tags: string[]; external_link: string; content: string;
  module: ModuleItem; section: SectionItem; subsection?: SubsectionItem; pos: number; status: ArticleStatus;
  created_at?: string; updated_at?: string;
}
export interface ArticleFilters { module_code?: string; section_code?: string; subsection_code?: string; direct_only?: boolean; title?: string; status?: ArticleStatus | ''; page: number; limit: number }
export interface ArticleList { articles: ArticleInfo[]; total: number }
export interface CollectRequest { external_link: string; title: string; section_code: string; subsection_code?: string; author: string; tags: string[]; publish: boolean }
export interface CollectResult { outcome: 'created' | 'already_registered'; article: ArticleInfo }
export interface SourcePreview { provider: string; canonical_url: string; title: string; metadata_status: 'resolved' | 'manual_required'; reason?: string; existing_article?: ArticleInfo }
export interface UpdateArticleRequest { id: string; title: string; author: string; tags: string[]; module_code: string; section_code: string; subsection_code?: string; external_link: string; content?: string }
export interface CatalogOrder { kind: 'module' | 'section' | 'subsection'; parent_code?: string; codes: string[] }
export const statusLabels: Record<ArticleStatus, string> = { Draft: '草稿', Published: '已发布', Unpublished: '已下架', Deleted: '已归档' };
