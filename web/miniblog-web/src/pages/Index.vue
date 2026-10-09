<template>
  <div class="home-main">
    <section class="home-hero" aria-labelledby="home-title">
      <div><h1 id="home-title">把技术想清楚，<br>也把过程写下来。</h1>
        <p class="hero-description">对 AI 的探索，对技术的思考，对生活的记录。<span>在这里，留下从问题到理解、从想法到实现的过程。</span></p>
        <div class="hero-actions"><button type="button" class="primary" @click="browseTopics">浏览技术主题<span aria-hidden="true">↗</span></button><button type="button" class="text-button read-btn" :disabled="starting" @click="goToBlog">{{ starting ? '正在打开…' : '开始阅读' }}</button></div>
        <p v-if="entryError" class="entry-error" role="alert">{{ entryError }}</p>
      </div>
      <aside class="author-note" aria-label="作者介绍"><img :src="logo" alt="Shujie 的头像"><p class="author-name">你好，我是 Shujie。</p><blockquote>What I cannot create,<br>I do not understand.</blockquote><p class="signature">在创造中理解，在记录中积累。</p></aside>
    </section>
    <section id="topics" ref="topicsSection" class="topic-shelf" aria-labelledby="topics-title">
      <div class="section-heading"><h2 id="topics-title">从一个主题开始</h2><p>按章节阅读，让知识慢慢连起来。</p></div>
      <ContentState v-if="store.listStatus === 'loading' && !store.modules.length" compact status="loading" title="正在加载阅读主题…" />
      <ContentState v-else-if="store.listStatus === 'empty'" compact status="empty" title="暂无阅读模块" message="主题还在整理中，稍后再来看看。" retry-label="刷新主题" @retry="retryList" />
      <ContentState v-else-if="store.listStatus === 'error' && !store.modules.length" compact status="error" title="主题暂时无法加载" message="请检查网络连接后重试。" retry-label="重试加载" @retry="retryList" />
      <template v-else>
        <p v-if="store.listStatus === 'error'" class="list-warning" role="alert">主题更新失败，正在显示上次读取的目录。<button type="button" class="text-button" @click="retryList">重试更新</button></p>
        <div class="topic-columns">
          <article v-for="topic in store.modules" :key="topic.code" :ref="element => registerTopic(element, topic.code)" class="topic-column">
            <h3><router-link :to="{ name: 'TopicOverview', params: { module: topic.code } }">{{ topic.title }}<span aria-hidden="true">›</span></router-link></h3>
            <p class="topic-description">{{ topicDescription(topic.code) }}</p>
            <template v-if="previews[topic.code]?.status === 'success' && previews[topic.code]?.module">
              <nav class="chapter-preview" :aria-label="topic.title + '章节'"><router-link v-for="section in previews[topic.code].module?.sections" :key="section.id" :to="{ name: 'TopicOverview', params: { module: topic.code }, query: { chapter: section.code } }">{{ section.title }}<span aria-hidden="true">›</span></router-link></nav>
              <p class="topic-count">{{ previews[topic.code].module?.sections.length }} 个章节，{{ articleCount(topic.code) }} 篇文章</p>
              <p v-if="articleCount(topic.code) === 0" class="preview-empty">暂无已发布文章</p>
            </template>
            <div v-else-if="previews[topic.code]?.status === 'error'" class="preview-error" role="status"><p>章节预览暂时无法加载。</p><button type="button" class="text-button" @click="retryTopic(topic.code)">重试章节预览</button></div>
            <div v-else class="chapter-placeholder" :aria-busy="previews[topic.code]?.status === 'loading'"><span v-if="previews[topic.code]?.status === 'loading'" class="sr-only" role="status">正在加载章节预览…</span><div></div><div></div><div></div></div>
            <router-link class="text-button" :to="{ name: 'TopicOverview', params: { module: topic.code } }">浏览全部文章</router-link>
          </article>
        </div>
      </template>
    </section>
    <aside class="home-note"><h3>开发者，从不止于编码。</h3><p>需求分析、领域建模、架构设计、编码实现。把每一步想清楚，也记录一路上的选择。</p></aside>
    <section class="home-principles" aria-label="阅读理念"><div><h3>功不唐捐</h3><p>功不唐捐，玉汝于成。</p></div><div><h3>终身学习</h3><p>在创造中理解，在长期记录中积累。</p></div></section>
  </div>
</template>
<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import logo from '@/assets/logo.jpeg'
import { useReadingNavigation } from '@/composables/useReadingNavigation'
import { useHomeTopics } from '@/composables/useHomeTopics'
import { topicDescription } from '@/util/topic'
import { flattenCatalog } from '@/util/catalog'
import ContentState from '@/components/ContentState.vue'
const { openFirstModule } = useReadingNavigation()
const { store, previews, registerTopic, retryTopic, retryList } = useHomeTopics()
const route = useRoute()
const topicsSection = ref<HTMLElement>(), starting = ref(false), entryError = ref('')
const articleCount = (code: string) => { const module = previews[code]?.module; return module ? flattenCatalog(module).length : 0 }
function browseTopics() {
  const reduced = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  topicsSection.value?.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'start' })
}
onMounted(() => { if (route.hash === '#topics') void nextTick(browseTopics) })
async function goToBlog() {
  if (starting.value) return
  starting.value = true; entryError.value = ''
  try { await openFirstModule() } catch { entryError.value = '主题加载失败，请再次点击开始阅读重试。' } finally { starting.value = false }
}
</script>
<style scoped>
.home-main { max-width: var(--content-width); margin: auto; padding: 72px 32px 40px; }
.home-hero { display: grid; grid-template-columns: minmax(0, 1fr) 276px; gap: 72px; align-items: center; padding-bottom: 64px; }
.home-hero h1 { font-family: var(--display); font-size: 48px; line-height: 1.42; letter-spacing: -1.8px; }
.hero-description { font-size: 16px; color: var(--secondary); line-height: 1.9; margin: 22px 0 28px; max-width: 590px; } .hero-description span { display: block; }
.hero-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 24px; } .entry-error { color: var(--warning); margin-top: 16px; font-size: 14px; }
.author-note { border-left: 1px solid var(--line); padding-left: 36px; align-self: stretch; display: flex; flex-direction: column; justify-content: center; }
.author-note img { width: 86px; height: 86px; border-radius: 24px; object-fit: cover; margin-bottom: 18px; } .author-name { font-size: 15px; font-weight: 600; margin-bottom: 10px; }
.author-note blockquote { font-family: var(--display); margin: 0; color: var(--secondary); font-size: 18px; line-height: 1.65; } .signature { margin-top: 13px; color: var(--secondary); font-size: 12px; }
.topic-shelf { border-top: 1px solid var(--line); padding-top: 34px; scroll-margin-top: 24px; } .section-heading { display: flex; justify-content: space-between; align-items: baseline; gap: 20px; margin-bottom: 30px; } .section-heading h2 { font-size: 24px; letter-spacing: -.4px; } .section-heading p { font-size: 13px; color: var(--secondary); }
.topic-columns { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 36px; } .topic-column { padding-top: 18px; border-top: 3px solid var(--green); min-width: 0; }
.topic-column h3 a { display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 24px; letter-spacing: -.5px; min-height: 44px; overflow-wrap: anywhere; } .topic-column h3 span { flex: none; }
.topic-description { font-size: 14px; color: var(--secondary); margin: 12px 0 20px; min-height: 50px; }
.chapter-preview { display: flex; flex-direction: column; margin-bottom: 16px; } .chapter-preview a { font-size: 14px; min-height: 44px; padding: 8px 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; border-bottom: 1px solid var(--line); overflow-wrap: anywhere; }
.chapter-preview span { flex: none; color: var(--secondary); } .topic-count { font-size: 12px; color: var(--secondary); margin: 10px 0 14px; } .topic-column a:hover { color: var(--green); }
.chapter-placeholder { min-height: 132px; margin-bottom: 16px; } .chapter-placeholder div { height: 12px; border-radius: 4px; background: var(--zone); margin: 22px 0; } .chapter-placeholder div:nth-child(2) { width: 80%; }
.preview-error { min-height: 132px; font-size: 13px; color: var(--secondary); } .preview-empty { color: var(--secondary); font-size: 13px; margin-bottom: 12px; } .list-warning { font-size: 13px; margin-bottom: 20px; color: var(--warning); }
.home-note { display: flex; gap: 26px; align-items: baseline; background: var(--zone); padding: 24px 28px; margin-top: 44px; border-radius: 8px; } .home-note h3 { font-family: var(--display); font-size: 18px; white-space: nowrap; } .home-note p { font-size: 13px; color: var(--secondary); }
.home-principles { display: flex; gap: 48px; padding-top: 28px; color: var(--secondary); } .home-principles h3 { font-size: 14px; margin-bottom: 7px; } .home-principles p { font-size: 12px; }
@media (max-width: 900px) { .home-main { padding-top: 54px; } .home-hero { grid-template-columns: minmax(0, 1fr) 200px; gap: 28px; } .home-hero h1 { font-size: 38px; letter-spacing: -1px; } .author-note { padding-left: 24px; } .topic-columns { gap: 24px; } .topic-column h3 a { font-size: 21px; } .topic-description { min-height: 76px; } }
@media (max-width: 650px) { .home-main { padding: 38px 24px 24px; } .home-hero { display: block; padding-bottom: 38px; } .home-hero h1 { font-size: 34px; line-height: 1.5; letter-spacing: -1px; } .hero-description { font-size: 14px; margin: 18px 0 24px; } .hero-description span { display: inline; } .hero-actions { gap: 20px; } .hero-actions .primary { padding: 0 19px; min-height: 46px; } .read-btn { font-size: 13px; } .author-note { display: none; } .section-heading { flex-direction: column; align-items: flex-start; gap: 9px; margin-bottom: 24px; } .section-heading h2 { font-size: 22px; } .section-heading p { font-size: 12px; } .topic-shelf { padding-top: 28px; } .topic-columns { grid-template-columns: minmax(0, 1fr); gap: 32px; } .topic-column { padding-top: 14px; } .topic-column h3 a { font-size: 25px; } .topic-description { min-height: 0; margin: 10px 0 16px; font-size: 13px; } .home-note { margin-top: 32px; display: block; padding: 22px; } .home-note h3 { font-size: 18px; margin-bottom: 10px; white-space: normal; } .home-note p { font-size: 12px; line-height: 1.9; } .home-principles { gap: 24px; flex-wrap: wrap; } }
@media (max-width: 360px) { .home-main { padding-left: 20px; padding-right: 20px; } .home-hero h1 { font-size: 30px; } .hero-actions { gap: 12px; } .hero-actions .primary { padding: 0 14px; } .read-btn { font-size: 12px; } }
</style>
