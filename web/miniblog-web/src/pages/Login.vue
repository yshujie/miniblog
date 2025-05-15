<template>
  <el-row justify="center">
    <el-col :span="8">
      <el-card>
        <h2>登录</h2>
        <el-form @submit.prevent="submit">
          <el-form-item>
            <el-input v-model="username" placeholder="用户名" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="password" type="password" placeholder="密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submit">登录</el-button>
          </el-form-item>
        </el-form>
        <div>没有账号？<router-link to="/register">注册</router-link></div>
        <el-alert v-if="msg" :title="msg" type="error" show-icon style="margin-top:10px;" />
      </el-card>
    </el-col>
  </el-row>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store'
import { login } from '../api/user'

const router = useRouter()
const username = ref('')
const password = ref('')
const msg = ref('')
const store = useUserStore()

async function submit() {
  if (await login(username.value, password.value)) {
    store.setUser({ username: username.value })
    router.push('/')
  } else {
    msg.value = '用户名或密码错误'
  }
}
</script>
