export interface PageResult<T> { items: T[]; total: number; page: number; limit: number }
export interface SyncConfig {
  title_property_id: string; state_property_id: string; topic_property_id: string; tags_property_id: string;
  state_option_ids: Record<string, string>; description_property_id?: string; difficulty_property_id?: string; date_property_id?: string;
}
export interface CatalogBinding { id: string; source_id: string; option_id: string; option_name: string; section_code?: string; status: string; reason?: string }
export interface SyncSource {
  source_id: string; label: string; module_code: string; enabled: boolean; config_revision: number; config: SyncConfig;
  health: string; last_attempt_at?: string; last_complete_scan_at?: string; last_success_at?: string; last_error?: string; catalog_bindings: CatalogBinding[];
}
export interface SyncStatus {
  enabled: boolean; paused: boolean; source_writes_paused?: boolean; baseline_frozen?: boolean; current_run_id?: string;
  health: string; pending_count: number; blocked_count?: number; error_count: number; last_complete_scan_at?: string; last_success_at?: string; sources: SyncSource[];
}
export type SyncArticleState = 'draft' | 'published' | 'unpublished' | 'archived';
export interface SyncPage {
  page_id: string; article_id?: string; source_id: string; title: string; desired_state: string; status: string; management_state: string;
  local_state?: SyncArticleState | null; needs_revalidation?: boolean; effective_visibility?: boolean; visibility_reason?: string;
  publish_block_reason?: string; last_error?: string; page_url: string; public_url?: string; publication_hold: boolean; publication_hold_reason?: string; bootstrap_state?: string;
}
export interface RunCounts { seen: number; created: number; updated: number; unchanged: number; blocked: number; failed: number; pending?: number; frozen?: number }
export interface SyncRun {
  run_id: string; mode: string; status: string; phase: string; started_at?: string; finished_at?: string;
  counts: RunCounts; error?: string;
}
export interface SyncItem { item_id?: string; page_id: string; article_id?: string; outcome: string; reason?: string; before?: unknown; after?: unknown; error?: string }
export interface SourceUpdate { label?: string; module_code?: string; enabled?: boolean; config?: SyncConfig; expected_config_revision?: number }
export interface PageFilters { page: number; limit: number; source_id?: string; management_state?: string; status?: string; title?: string }
export interface BootstrapCandidate {
  page_id: string; source_id: string; title: string; topic: string; notion_state: string;
  new_page: boolean; match_method: string; requires_legacy_alias: boolean; public_condition: string;
  proposed_section_code?: string; proposed_section_title: string; placement_change: string;
  candidate_article_ids: string[]; title_hint_article_ids: string[]; article_id?: string;
  local_state?: string; local_title?: string; local_section_code?: string; local_subsection_code?: string;
  expected_fingerprint?: string; publish_block_reason?: string; reason?: string;
}
export interface BootstrapPreviewAfter { bootstrap_preview: BootstrapCandidate }
