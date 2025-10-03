<template>
  <div class="login-container">
    <el-form
      ref="formRef"
      :model="loginForm"
      :rules="rules"
      class="login-form"
      label-position="left"
    >
      <h2 class="login-title">管理后台登录</h2>

      <el-form-item prop="username">
        <el-input
          v-model="loginForm.username"
          placeholder="用户名"
          autocomplete="username"
          @keyup.enter="handleLogin"
        />
      </el-form-item>

      <el-form-item prop="password">
        <el-input
          v-model="loginForm.password"
          :type="showPassword ? 'text' : 'password'"
          placeholder="密码"
          autocomplete="current-password"
          @keyup.enter="handleLogin"
        >
          <template #suffix>
            <el-icon class="pwd-toggle" @click="togglePassword">
              <component :is="passwordIcon" />
            </el-icon>
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
  const valid = await formRef.value.validate();
  if (!valid) return;

  loading.value = true;
  try {
    await store.login(loginForm);
    const target = redirect.value || '/';
    await router.replace({ path: target, query: otherQuery.value });
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '登录失败';
    ElMessage.error(message);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped lang="scss">
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #324057 100%);
}

.login-form {
  width: 360px;
  padding: 40px 32px;
  border-radius: 12px;
  background-color: rgba(255, 255, 255, 0.1);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(12px);
}

.login-title {
  margin-bottom: 32px;
  text-align: center;
  color: #fff;
  font-size: 24px;
  font-weight: 600;
}

.el-input__wrapper {
  background: rgba(255, 255, 255, 0.9);
}

.login-button {
  width: 100%;
  margin-top: 12px;
}

.pwd-toggle {
  cursor: pointer;
  color: var(--el-text-color-secondary);
}
</style>
