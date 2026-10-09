<template>
  <div class="app-shell" :class="{ 'reading-layout': isReadingPage }">
    <SiteHeader />
    <main class="app-main"><router-view /></main>
    <Footer v-if="!isReadingPage" />
  </div>
</template>
<script setup lang="ts">
import { computed, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import SiteHeader from './components/SiteHeader.vue'
import Footer from './components/Footer.vue'
const route = useRoute(), ui = useUiStore()
const isReadingPage = computed(() => route.meta.layout === 'reading')
watch(isReadingPage, reading => document.body.classList.toggle('blog-page-body', reading), { immediate: true })
watch(() => route.fullPath, () => ui.closePanels())
onUnmounted(() => { document.body.classList.remove('blog-page-body'); ui.closePanels() })
</script>
<style scoped>
.app-shell { min-height: 100vh; min-height: 100dvh; display: flex; flex-direction: column; background: var(--page); }
.app-main { flex: 1; min-width: 0; width: 100%; }
.reading-layout { height: 100vh; height: 100dvh; overflow: hidden; } .reading-layout .app-main { min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
</style>
