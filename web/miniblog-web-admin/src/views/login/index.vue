<template>
  <div class="login-container">
    <el-form
      ref="formRef"
      :model="loginForm"
      :rules="rules"
      class="login-form"
      label-position="top"
    >
      <router-link class="login-brand" to="/home"><img :src="logo" alt="" width="36" height="36"><strong>Shujie's Blog</strong></router-link>
      <h1 class="login-title">登录内容工作台</h1><p class="login-description">同步外部文档，整理目录与文章。</p>

      <el-form-item prop="username" label="用户名">
        <el-input
          v-model="loginForm.username"
          placeholder="用户名"
          autocomplete="username"
          @keyup.enter="handleLogin"
        />
      </el-form-item>

      <el-form-item prop="password" label="密码">
        <el-input
          v-model="loginForm.password"
          :type="showPassword ? 'text' : 'password'"
          placeholder="密码"
          autocomplete="current-password"
          @keyup.enter="handleLogin"
        >
          <template #suffix>
            <button class="pwd-toggle" type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword" @click="togglePassword"><el-icon><component :is="passwordIcon" /></el-icon></button>
          </template>
        </el-input>
      </el-form-item>

      <el-button
        type="primary"
        :loading="loading"
        class="login-button"
        @click="handleLogin"
      >
        登录
      </el-button>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { FormInstance, FormRules } from 'element-plus';
import { ElMessage } from 'element-plus';
import userStore from '@/store/modules/user';
import { View, Hide } from '@element-plus/icons-vue';
import logo from '@/assets/logo.jpeg';
import { loginDestination } from '@/utils/login-redirect';

const formRef = ref<FormInstance>();
const loading = ref(false);
const showPassword = ref(false);
const passwordIcon = computed(() => (showPassword.value ? View : Hide));

const loginForm = reactive({
  username: '',
  password: ''
});

const rules: FormRules = {
  username: [
    {
      required: true,
      message: '请输入用户名',
      trigger: 'blur'
    }
  ],
  password: [
    {
      required: true,
      message: '请输入密码',
      trigger: 'blur'
    },
    {
      min: 6,
      message: '密码长度至少 6 位',
      trigger: 'blur'
    }
  ]
};

const route = useRoute();
const router = useRouter();
const store = userStore();
const redirect = ref<string | undefined>();
const otherQuery = ref<Record<string, string>>({});

watch(
  () => route.query,
  (query) => {
    redirect.value = query.redirect as string | undefined;
    const newQuery: Record<string, string> = {};
    Object.keys(query).forEach((key) => {
      if (key !== 'redirect') {
        newQuery[key] = query[key] as string;
      }
    });
    otherQuery.value = newQuery;
  },
  { immediate: true }
);

const togglePassword = () => {
  showPassword.value = !showPassword.value;
};

const handleLogin = async () => {
  if (!formRef.value) return;
  if (loading.value) return;
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  loading.value = true;
  try {
    await store.login(loginForm);
    await router.replace(loginDestination(redirect.value, otherQuery.value));
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '登录失败';
    ElMessage.error(message);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-container { min-height:100dvh; display:grid; place-items:center; padding:28px 16px; background:var(--admin-canvas); }
.login-form { width:min(100%,420px); padding:36px; border:1px solid var(--admin-line); border-radius:8px; background:white; }
.login-brand { display:flex; gap:12px; align-items:center; }.login-brand img { border-radius:8px; }.login-brand strong { font-size:23px; font-family:Georgia,serif; }
.login-title { margin:32px 0 8px; font-size:24px; font-weight:650; }.login-description { color:var(--admin-muted); font-size:14px; line-height:1.7; margin:0 0 28px; }.login-button { width:100%; margin-top:12px; }.pwd-toggle { display:grid; place-items:center; width:36px; height:36px; border:0; background:none; color:var(--admin-muted); cursor:pointer; }
@media(max-width:780px) { .login-form { padding:28px 24px; }.pwd-toggle { width:44px; height:44px; } }
</style>
