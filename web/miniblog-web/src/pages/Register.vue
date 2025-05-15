<template>
  <el-row justify="center">
    <el-col :span="8">
      <el-card>
        <h2>注册</h2>
        <el-form @submit.prevent="submit">
          <el-form-item>
            <el-input v-model="username" placeholder="用户名" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="password" type="password" placeholder="密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submit">注册</el-button>
          </el-form-item>
        </el-form>
        <div>已有账号？<router-link to="/login">登录</router-link></div>
        <el-alert v-if="msg" :title="msg" :type="msgType" show-icon style="margin-top:10px;" />
      </el-card>
    </el-col>
  </el-row>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api/user'

const router = useRouter()
const username = ref('')
const password = ref('')
const msg = ref('')
const msgType = ref<'error' | 'success'>('error')

async function submit() {
  if (await register(username.value, password.value)) {
    msg.value = '注册成功，即将跳转登录'
    msgType.value = 'success'
    setTimeout(() => router.push('/login'), 1000)
  } else {
    msg.value = '用户名已存在'
    msgType.value = 'error'
  }
}
</script>
