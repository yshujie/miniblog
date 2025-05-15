<template>
  <el-container>
    <el-header style="height:60px;background:#1a1a1a;color:white;">
      <div style="font-size:24px;font-weight:bold;">GoodBlog Vue3</div>
    </el-header>
    <el-main>
      <router-view />
    </el-main>
    <el-footer style="text-align:center;color:#888;">
      &copy; 2025 GoodBlog. Powered by Vue3 + TypeScript.
    </el-footer>
  </el-container>
</template>
