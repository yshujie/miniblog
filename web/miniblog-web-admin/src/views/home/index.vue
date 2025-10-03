<template>
  <div class="home">
    <el-row :gutter="20">
      <el-col :span="16" :xs="24">
        <el-card class="welcome-card">
          <h2 class="welcome-title">欢迎回来，{{ userName }}</h2>
          <p class="welcome-subtitle">当前角色：{{ rolesDisplay }}</p>
          <div class="quick-actions">
            <el-button type="primary" icon="Edit" @click="goCreate">新增文章</el-button>
            <el-button type="default" icon="List" @click="goList">文章列表</el-button>
          </div>
        </el-card>

        <el-card class="todo-card">
          <template #header>
            <span>操作指南</span>
          </template>
          <el-timeline>
            <el-timeline-item timestamp="1" placement="top">
              登录后可在左侧导航进入文章管理模块。
            </el-timeline-item>
            <el-timeline-item timestamp="2" placement="top">
              在“文章列表”中查询、筛选已有文章。
            </el-timeline-item>
            <el-timeline-item timestamp="3" placement="top">
              点击“新增文章”编写内容，保存后可继续编辑或发布。
            </el-timeline-item>
          </el-timeline>
        </el-card>
      </el-col>

      <el-col :span="8" :xs="24">
        <el-card>
          <template #header>
            <span>我的信息</span>
          </template>
          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item label="昵称">{{ userName }}</el-descriptions-item>
            <el-descriptions-item label="角色">{{ rolesDisplay }}</el-descriptions-item>
            <el-descriptions-item label="简介">{{ introduction || '暂无简介' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import userStore from '@/store/modules/user';

const router = useRouter();
const store = userStore();

const userName = computed(() => store.name || '管理员');
const rolesDisplay = computed(() => store.roles?.join('、') || '未分配');
const introduction = computed(() => store.introduction);

const goCreate = () => {
  router.push({ path: '/article/create' });
};

const goList = () => {
  router.push({ path: '/article/list' });
};
</script>

<style scoped>
.home {
  padding: 20px;
}

.welcome-card {
  margin-bottom: 20px;
}

.welcome-title {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 600;
}

.welcome-subtitle {
  margin: 0 0 16px;
  color: var(--el-text-color-secondary);
}

.quick-actions {
  display: flex;
  gap: 12px;
}

.todo-card {
  margin-top: 20px;
}
</style>
