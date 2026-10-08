import { fetchArticles, getArticle, createLegacyArticle, saveArticle, changeArticleStatus } from './content';
export const fetchList = fetchArticles;
export const fetchArticle = (id) => getArticle(String(id));
export const createArticle = createLegacyArticle;
export const updateArticle = saveArticle;
export const publishArticle = (data) => changeArticleStatus(String(data.id), 'publish');
export const unpublishArticle = (data) => changeArticleStatus(String(data.id), 'unpublish');
