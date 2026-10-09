<template>
  <div class="admin-sidebar">
    <router-link class="admin-brand" to="/home" @click="$emit('navigate')"><img :src="logo" alt="" width="34" height="34"><div><strong>Shujie's Blog</strong><span>内容管理</span></div></router-link>
    <nav aria-label="管理端主导航">
      <template v-for="group in navigation" :key="group.title"><p v-if="group.title" class="nav-group">{{ group.title }}</p><router-link v-for="item in group.items" :key="item.path" :to="item.path" :class="{ current: activePath === item.path }" :aria-current="activePath === item.path ? 'page' : undefined" @click="$emit('navigate')"><svg-icon :icon-class="item.icon" /><span>{{ item.title }}</span></router-link></template>
    </nav>
    <button class="account-trigger" type="button" aria-label="查看我的信息" @click="$emit('account')"><span class="account-initial">{{ user.name?.slice(0, 1) || '我' }}</span><span><strong>{{ user.name || '当前用户' }}</strong><small>{{ user.roles.join('、') || '账户信息' }}</small></span><span aria-hidden="true">⌄</span></button>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import usePermission from '@/store/modules/permission';
import useUser from '@/store/modules/user';
import { adminNavigation } from '@/utils/admin-navigation';
import logo from '@/assets/logo.jpeg';
defineEmits<{ navigate: []; account: [] }>();
const route = useRoute(); const permission = usePermission(); const user = useUser();
const navigation = computed(() => adminNavigation(permission.routes));
const activePath = computed(() => {
  if (/^\/(module|section|subsection)\//.test(route.path)) return '/content/workbench';
  return String(route.meta.activeMenu || route.path);
});
</script>
<style scoped>
.admin-sidebar { height:100%; min-height:0; display:flex; flex-direction:column; background:white; padding:0; margin:0; }
.admin-brand { display:flex; align-items:center; gap:10px; padding:26px 20px 34px; }.admin-brand img { border-radius:8px; object-fit:cover; }.admin-brand strong { font-family:Georgia,serif; font-size:20px; white-space:nowrap; }.admin-brand span { display:block; font-size:12px; color:var(--admin-muted); margin-top:4px; }
nav { padding:0 12px; overflow-y:auto; flex:1; min-height:0; }nav a { display:flex; gap:12px; align-items:center; padding:12px 14px; min-height:44px; color:var(--admin-muted); border-radius:6px; font-size:14px; margin-bottom:4px; }nav a.current { color:var(--admin-green); background:var(--admin-pale); font-weight:600; }nav a:hover { background:var(--admin-canvas); }nav a.current:hover { background:var(--admin-pale); }.nav-group { margin:28px 12px 12px; font-size:12px; color:var(--admin-muted); }
.account-trigger { display:flex; width:100%; align-items:center; gap:12px; padding:20px 24px; text-align:left; background:white; border:0; border-top:1px solid var(--admin-line); color:var(--admin-muted); cursor:pointer; }.account-trigger strong { font-weight:500; font-size:14px; }.account-trigger small { display:block; margin-top:3px; font-size:11px; }.account-trigger > span:last-child { margin-left:auto; }.account-initial { width:30px; height:30px; border-radius:50%; background:var(--admin-pale); color:var(--admin-green); display:grid; place-items:center; }
</style>
