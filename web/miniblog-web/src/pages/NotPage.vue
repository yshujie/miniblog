<template>
  <section class="not-page" aria-labelledby="not-page-title">
    <span class="error-code" aria-hidden="true">404</span><h1 id="not-page-title">这页没有找到</h1>
    <p>地址可能有误，或页面已经移动。回到首页，重新选择一个技术主题。</p>
    <div class="not-page-actions"><router-link to="/" class="primary">返回首页</router-link><router-link :to="{ path: '/', hash: '#topics' }" class="secondary-button">浏览技术主题</router-link></div>
  </section>
</template>
<style scoped>
.not-page { min-height: 60vh; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; padding: 64px 24px; }
.error-code { font-size: 72px; font-family: var(--display); color: var(--green); line-height: 1.2; margin-bottom: 22px; } .not-page h1 { font-family: var(--display); font-size: 34px; margin-bottom: 14px; }
.not-page p { font-size: 14px; color: var(--secondary); max-width: 440px; line-height: 1.9; margin-bottom: 26px; } .not-page-actions { display: flex; flex-wrap: wrap; justify-content: center; gap: 12px; }
@media (max-width: 650px) { .not-page { padding: 38px 24px; } .not-page h1 { font-size: 28px; } }
</style>
