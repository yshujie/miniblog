import { defineStore } from 'pinia';
import type { DirectoryContext } from '@/types/content';
const key = 'miniblog.recentDirectories.v1';
function readRecent(): DirectoryContext[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(key) || '[]');
    return Array.isArray(value) ? value.filter(item => item && typeof item.module_code === 'string' && typeof item.section_code === 'string' && typeof item.subsection_code === 'string').slice(0, 8) : [];
  } catch { return []; }
}
export default defineStore('contentWorkspace', {
  state: () => ({ recentDirectories: readRecent(), lastAuthor: '', articleRevision: 0 }),
  actions: {
    remember(context: DirectoryContext, author?: string) {
      if (!context.section_code) return;
      this.recentDirectories = [context, ...this.recentDirectories.filter(item => item.section_code !== context.section_code || item.subsection_code !== context.subsection_code)].slice(0, 8);
      if (author !== undefined) this.lastAuthor = author;
      try { localStorage.setItem(key, JSON.stringify(this.recentDirectories)); } catch { /* The working context remains available in this session. */ }
    },
    invalidateArticles() { this.articleRevision += 1; }
  }
});
