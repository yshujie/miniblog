# SSL 证书配置说明

## 证书文件

- yangshujie.com.crt
- yangshujie.com.key

## 获取证书

1. 登录 Cloudflare 控制台
2. 进入 SSL/TLS -> Origin Server
3. 创建 Origin Certificate
4. 下载证书和私钥

## 部署步骤

1. 将证书文件放在此目录
2. 设置正确的文件权限：

   ```bash
   chmod 600 yangshujie.com.key
   chmod 644 yangshujie.com.crt
   ```
