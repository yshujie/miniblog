# CORS 跨域问题修复说明

## 🐛 问题描述

### 错误信息

```
Access to XMLHttpRequest at 'https://api.yangshujie.com/v1/blog/modules' 
from origin 'https://www.yangshujie.com' has been blocked by CORS policy: 
No 'Access-Control-Allow-Origin' header is present on the requested resource.
```

### 问题原因

**CORS (Cross-Origin Resource Sharing)** - 跨域资源共享

当前端（`www.yangshujie.com`）尝试访问后端 API（`api.yangshujie.com`）时：

- 浏览器检测到**跨域请求**（不同域名）
- 后端没有返回 `Access-Control-Allow-Origin` 响应头
- 浏览器**拦截响应**，前端无法获取数据

## ✅ 解决方案

### 在 Nginx 添加 CORS 响应头

修改 `configs/nginx/conf.d/api.yangshujie.com.conf`，添加以下配置：

```nginx
# CORS 配置 - 动态允许多个域名跨域访问
set $cors_origin "";
if ($http_origin ~* "^https://(www|admin)\.yangshujie\.com$") {
    set $cors_origin $http_origin;
}

# 添加 CORS 响应头
add_header Access-Control-Allow-Origin $cors_origin always;
add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
add_header Access-Control-Allow-Credentials "true" always;
add_header Access-Control-Max-Age "3600" always;

# 处理 OPTIONS 预检请求
if ($request_method = OPTIONS) {
    add_header Access-Control-Allow-Origin $cors_origin always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
    add_header Access-Control-Allow-Credentials "true" always;
    add_header Access-Control-Max-Age "3600" always;
    add_header Content-Length 0;
    add_header Content-Type text/plain;
    return 204;
}
```

## 📝 配置说明

### 1. 动态 Origin 匹配

```nginx
set $cors_origin "";
if ($http_origin ~* "^https://(www|admin)\.yangshujie\.com$") {
    set $cors_origin $http_origin;
}
```

**作用**：

- 检查请求的 `Origin` 头
- 如果来自 `www.yangshujie.com` 或 `admin.yangshujie.com`
- 将其设置为 `$cors_origin` 变量
- 否则 `$cors_origin` 为空（拒绝跨域）

**为什么不直接写 `*`**：

- ❌ `Access-Control-Allow-Origin: *` 不能与 `Access-Control-Allow-Credentials: true` 同时使用
- ✅ 必须明确指定允许的域名
- ✅ 支持多个域名（www 和 admin）

### 2. CORS 响应头

| 响应头 | 值 | 说明 |
|--------|---|------|
| `Access-Control-Allow-Origin` | `$cors_origin` | 允许的域名（动态） |
| `Access-Control-Allow-Methods` | `GET, POST, PUT, DELETE, OPTIONS` | 允许的 HTTP 方法 |
| `Access-Control-Allow-Headers` | `Content-Type, Authorization, X-Requested-With` | 允许的请求头 |
| `Access-Control-Allow-Credentials` | `true` | 允许携带 Cookie/认证信息 |
| `Access-Control-Max-Age` | `3600` | 预检请求缓存时间（1小时） |

### 3. OPTIONS 预检请求处理

**什么是 OPTIONS 请求**：

- 浏览器在发送跨域请求前，会先发送一个 `OPTIONS` 请求
- 询问服务器是否允许跨域
- 只有得到允许后，才会发送真正的请求（GET/POST等）

**为什么要单独处理**：

- `OPTIONS` 请求不需要转发到后端
- 直接在 Nginx 返回 `204 No Content`
- 加快响应速度，减少后端负担

## 🔄 部署步骤

### 方法 1：通过 Jenkins 自动部署（推荐）

1. 提交代码到 Git
2. Jenkins 自动构建
3. Nginx 配置文件自动更新到服务器
4. 重启 Nginx

### 方法 2：手动部署

```bash
# 1. 登录服务器
ssh root@47.94.204.124

# 2. 备份旧配置
cp /etc/nginx/conf.d/api.yangshujie.com.conf /etc/nginx/conf.d/api.yangshujie.com.conf.bak

# 3. 上传新配置文件
# 将本地 configs/nginx/conf.d/api.yangshujie.com.conf 复制到服务器

# 4. 测试配置
nginx -t

# 5. 重新加载 Nginx
nginx -s reload

# 或者重启
systemctl restart nginx
```

## 🧪 验证方法

### 1. 浏览器控制台

访问 `https://www.yangshujie.com`，打开开发者工具：

**Network 标签**：

- 找到 API 请求（如 `/v1/blog/modules`）
- 查看 **Response Headers**
- 应该能看到：

  ```
  access-control-allow-origin: https://www.yangshujie.com
  access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
  access-control-allow-credentials: true
  ```

### 2. 命令行测试

```bash
# 测试 OPTIONS 预检请求
curl -X OPTIONS https://api.yangshujie.com/v1/blog/modules \
  -H "Origin: https://www.yangshujie.com" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v

# 应该返回 204，并包含 CORS 响应头
```

```bash
# 测试实际 GET 请求
curl https://api.yangshujie.com/v1/blog/modules \
  -H "Origin: https://www.yangshujie.com" \
  -v

# 应该返回数据，并包含 CORS 响应头
```

### 3. 前端测试

访问网站，查看是否还有 CORS 错误：

- ✅ 成功：数据正常加载，控制台无错误
- ❌ 失败：仍然看到 CORS 错误

## 📚 CORS 工作流程

### 简单请求（Simple Request）

```
浏览器 → API 服务器
  GET /v1/blog/modules
  Origin: https://www.yangshujie.com

API 服务器 → 浏览器
  HTTP/1.1 200 OK
  Access-Control-Allow-Origin: https://www.yangshujie.com
  { "data": [...] }

浏览器：检查 Origin 匹配 → 允许前端访问响应数据
```

### 预检请求（Preflight Request）

```
1. 预检阶段
浏览器 → API 服务器
  OPTIONS /v1/blog/modules
  Origin: https://www.yangshujie.com
  Access-Control-Request-Method: POST
  Access-Control-Request-Headers: Content-Type

API 服务器 → 浏览器
  HTTP/1.1 204 No Content
  Access-Control-Allow-Origin: https://www.yangshujie.com
  Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
  Access-Control-Allow-Headers: Content-Type, Authorization

浏览器：检查通过 → 发送实际请求

2. 实际请求阶段
浏览器 → API 服务器
  POST /v1/blog/modules
  Origin: https://www.yangshujie.com
  Content-Type: application/json

API 服务器 → 浏览器
  HTTP/1.1 200 OK
  Access-Control-Allow-Origin: https://www.yangshujie.com
  { "success": true }

浏览器：允许前端访问响应数据
```

## ⚠️ 注意事项

### 1. always 标志

```nginx
add_header Access-Control-Allow-Origin $cors_origin always;
```

- `always` 确保在**所有响应**中添加该头（包括错误响应）
- 如果不加 `always`，错误响应（4xx, 5xx）不会包含 CORS 头
- 导致浏览器无法显示真正的错误信息

### 2. if 语句的限制

Nginx 的 `if` 在 `server` 块中有一些限制：

- ✅ 可以设置变量
- ✅ 可以 `return`
- ⚠️ 不能在 `if` 块内使用 `add_header`（需要在外面）

所以我们的配置：

```nginx
if ($http_origin ~* "^https://(www|admin)\.yangshujie\.com$") {
    set $cors_origin $http_origin;  # ✅ 设置变量
}
add_header Access-Control-Allow-Origin $cors_origin always;  # ✅ 在 if 外面
```

### 3. 安全性

- ✅ 使用白名单机制（只允许特定域名）
- ✅ 不使用 `*` 通配符
- ✅ 正则表达式严格匹配域名
- ⚠️ 如果后续添加新域名，需要更新正则表达式

## 🔧 扩展配置

### 允许更多域名

如果需要允许其他域名（如测试环境）：

```nginx
if ($http_origin ~* "^https://(www|admin|test)\.yangshujie\.com$") {
    set $cors_origin $http_origin;
}
```

### 允许更多请求头

如果前端需要发送自定义头：

```nginx
add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With, X-Custom-Header" always;
```

### 调试模式

临时允许所有域名（仅用于调试，不要用于生产）：

```nginx
# ⚠️ 仅用于调试！
add_header Access-Control-Allow-Origin "*" always;
# 注意：使用 * 时不能启用 credentials
# add_header Access-Control-Allow-Credentials "true" always;  # 这行要注释掉
```

## 🎯 总结

- ✅ 问题：前端无法访问后端 API（CORS 错误）
- ✅ 原因：后端没有返回 CORS 响应头
- ✅ 解决：在 Nginx 添加 CORS 配置
- ✅ 支持：www.yangshujie.com 和 admin.yangshujie.com
- ✅ 安全：使用白名单，不允许任意域名访问
- ✅ 性能：OPTIONS 预检请求直接在 Nginx 返回，缓存 1 小时

现在前端应该可以正常访问 API 了！🚀
