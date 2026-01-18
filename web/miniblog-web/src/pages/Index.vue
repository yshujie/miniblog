<template>
  <div class="home-root">
   
    <!-- 中心区 -->
    <div class="main-info">
      <el-avatar class="avatar" :src="logo" size="large">
        <img :src="logo" alt="logo" />
      </el-avatar>
      <h1>Shujie's Blog</h1>
      <p class="desc">
        《费曼物理讲义》中讲：What I cannot create, I do not understand.
        <br />
        这里便是我创造的：对 AI 的探索、对技术的思考、对生活的记录。
      </p>
      <el-button type="success" size="large" class="read-btn" @click="goToBlog">开始阅读</el-button>
    </div>

    <!-- 分割线 -->
    <div class="divider"></div>
    
    <!-- 三栏介绍 -->
    <div class="columns">
      <div class="column">
        <div class="col-title">功不唐捐</div>
        <div class="col-desc">
          功不唐捐，玉汝于成
        </div>
      </div>
      <div class="column">
        <div class="col-title">面向未来的开发者</div>
        <div class="col-desc">
          需求分析、领域建模、架构设计、编码实现，开发者从不止于编码
        </div>
      </div>
      <div class="column">
        <div class="col-title">终身学习</div>
        <div class="col-desc">
          积少成多，如果放到五到十年这个长度上，人和人之间的距离要大到地球到月球上了
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import logo from '@/assets/logo.jpeg'
import { useModuleStore } from '@/stores/module'

const moduleStore = useModuleStore()

const goToBlog = async () => {
  const modules = moduleStore.modules
  if (modules.length === 0) {
    await moduleStore.loadModules()
  }
  if (modules.length > 0) {
    window.location.href = `/blog/${modules[0].code}`
  }
}
</script>

<style lang="less" scoped>
.home-root {
  max-width: 1100px;
  margin: 48px auto 0 auto;
  padding: 24px 12px;
}
.main-info {
  text-align: center;
  margin-bottom: 48px;

  .avatar {
    width: 96px;
    height: 96px;
    margin-bottom: 16px;
  }
  h1 {
    font-size: 36px;
    font-weight: bold;
    margin: 16px 0 8px 0;
    color: #333;
  }
  .desc {
    color: #888;
    font-size: 18px;
    margin-bottom: 22px;
    line-height: 1.8;
  }
  .read-btn {
    padding: 10px 32px;
    font-size: 18px;
    font-weight: 500;
  }
}

.divider {
  height: 1px;
  background: #eee;
  margin: 48px 0;
}

.columns {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  margin-top: 40px;

  .column {
    flex: 1 1 0;
    padding: 18px;
    background: #fff;
    border-radius: 10px;
    text-align: center;
    min-width: 250px;
  }
  .col-title {
    font-size: 18px;
    font-weight: bold;
    margin-bottom: 12px;
    color: #222;
  }
  .col-desc {
    color: #888;
    font-size: 15px;
    line-height: 1.8;
  }
}
</style>
