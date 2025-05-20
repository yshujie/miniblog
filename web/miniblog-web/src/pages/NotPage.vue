<template>
  <div class="not-page">
    <div class="content">
      <h1 class="error-code">404</h1>
      <div class="error-message">
        <h2>页面未找到</h2>
        <p>抱歉，您访问的页面不存在或已被移除</p>
      </div>
      <div class="actions">
        <router-link to="/" class="home-button">
          <i class="fas fa-home"></i>
          返回首页
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 无需额外的逻辑
</script>

<style scoped>
.not-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  padding: 20px;
}

.content {
  text-align: center;
  background: white;
  padding: 40px;
  border-radius: 20px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
  max-width: 500px;
  width: 100%;
  animation: fadeIn 0.5s ease-out;
}

.error-code {
  font-size: 120px;
  font-weight: bold;
  color: #2c3e50;
  margin: 0;
  line-height: 1;
  background: linear-gradient(45deg, #2c3e50, #3498db);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: bounce 2s infinite;
}

.error-message {
  margin: 30px 0;
}

.error-message h2 {
  font-size: 28px;
  color: #2c3e50;
  margin-bottom: 10px;
}

.error-message p {
  color: #7f8c8d;
  font-size: 16px;
  line-height: 1.6;
}

.actions {
  margin-top: 30px;
}

.home-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: #3498db;
  color: white;
  text-decoration: none;
  border-radius: 8px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.home-button:hover {
  background: #2980b9;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(52, 152, 219, 0.3);
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes bounce {
  0%, 20%, 50%, 80%, 100% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(-20px);
  }
  60% {
    transform: translateY(-10px);
  }
}

@media (max-width: 480px) {
  .content {
    padding: 20px;
  }
  
  .error-code {
    font-size: 80px;
  }
  
  .error-message h2 {
    font-size: 24px;
  }
}
</style> 