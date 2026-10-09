<template>
  <div class="admin-shell">
    <Sidebar class="sidebar-container admin-rail" @account="openAccount" />
    <div class="admin-main">
      <Navbar :navigation-open="navigationOpen" @navigation="navigationOpen = true" />
      <AppMain />
    </div>
    <el-drawer v-model="navigationOpen" title="内容管理导航" direction="ltr" size="264px" class="admin-navigation" :destroy-on-close="true">
      <Sidebar @navigate="navigationOpen = false" @account="openAccount" />
    </el-drawer>
    <el-drawer v-model="accountOpen" title="我的信息" :size="mobile ? '100%' : '440px'" class="admin-account">
      <div class="account-profile"><span class="account-avatar">{{ user.name?.slice(0, 1) || '我' }}</span><h2>{{ user.name || '当前用户' }}</h2><p>账户资料</p></div>
      <el-descriptions :column="1" border><el-descriptions-item label="昵称">{{ user.name || '—' }}</el-descriptions-item><el-descriptions-item label="角色">{{ user.roles.join('、') || '尚未提供' }}</el-descriptions-item><el-descriptions-item label="简介">{{ user.introduction || '暂无简介' }}</el-descriptions-item></el-descriptions>
      <el-alert v-if="logoutError" :title="logoutError" type="error" :closable="false" role="alert" />
      <template #footer><el-button @click="accountOpen = false">关闭</el-button><el-button type="danger" plain :loading="loggingOut" @click="logout">退出登录</el-button></template>
    </el-drawer>
  </div>
</template>
<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import Sidebar from './components/Sidebar/index.vue';
import Navbar from './components/Navbar.vue';
import AppMain from './components/AppMain.vue';
import useUser from '@/store/modules/user';
import { useAdminMobile } from '@/composables/useAdminMobile';
import { errorMessage } from '@/utils/api-error';
const route = useRoute(); const router = useRouter(); const user = useUser(); const mobile = useAdminMobile();
const navigationOpen = ref(false); const accountOpen = ref(false); const loggingOut = ref(false); const logoutError = ref('');
watch(() => route.fullPath, () => { navigationOpen.value = false; });
watch(mobile, () => { navigationOpen.value = false; });
async function openAccount() { navigationOpen.value = false; await nextTick(); logoutError.value = ''; accountOpen.value = true; }
async function logout() {
  if (loggingOut.value) return;
  loggingOut.value = true; logoutError.value = '';
  const redirect = route.fullPath;
  try { await user.logout(); accountOpen.value = false; await router.replace({ path: '/login', query: { redirect }}); } catch (cause) { logoutError.value = errorMessage(cause, '退出失败，请重试'); } finally { loggingOut.value = false; }
}
</script>
<style scoped>
.admin-shell { min-height:100dvh; }
.admin-rail { position:fixed; inset:0 auto 0 0; width:224px; border-right:1px solid var(--admin-line); background:white; z-index:20; }
.admin-main { margin-left:224px; min-width:0; }
.account-profile { text-align:center; padding:20px 0 28px; }.account-profile h2 { margin:12px 0 4px; }.account-profile p { color:var(--admin-muted); }.account-avatar { display:inline-grid; place-items:center; width:56px; height:56px; border-radius:50%; background:var(--admin-pale); color:var(--admin-green); font-size:22px; }
@media(max-width:780px) { .admin-rail { display:none; }.admin-main { margin-left:0; } }
</style>
