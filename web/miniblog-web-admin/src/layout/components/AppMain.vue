<template>
  <main id="main-content" class="app-main" tabindex="-1">
    <router-view v-slot="{ Component, route }"><component :is="Component" :key="route.path" /></router-view>
  </main>
</template>
<style scoped>
.app-main { min-height:calc(100dvh - 60px); min-width:0; padding:32px; }
@media(max-width:1200px) { .app-main { padding:26px 24px; } }
@media(max-width:780px) { .app-main { padding:22px 16px; } }
@media(max-width:360px) { .app-main { padding:20px 12px; } }
</style>
