<template>
  <el-card>
    <h3>评论区</h3>
    <el-form @submit.prevent="submit">
      <el-form-item>
        <el-input v-model="input" placeholder="说点什么..." />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="submit">发表评论</el-button>
      </el-form-item>
    </el-form>
    <el-divider />
    <div v-for="(c, idx) in comments" :key="idx" style="margin-bottom:10px;">
      <el-avatar :size="28">{{ c.author[0].toUpperCase() }}</el-avatar>
      <span style="margin-left:8px;font-weight:bold;">{{ c.author }}</span>：{{ c.content }}
      <span style="float:right;color:#aaa;">{{ c.time }}</span>
    </div>
  </el-card>
</template>
<script setup lang="ts">
import { ref } from 'vue'
const comments = ref([
  { author: 'koala', content: '欢迎留言！', time: '2025-05-01 10:00' }
])
const input = ref('')
function submit() {
  if (input.value.trim()) {
    comments.value.unshift({
      author: '访客',
      content: input.value,
      time: new Date().toLocaleString()
    })
    input.value = ''
  }
}
</script>
