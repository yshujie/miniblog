# CI/CD 构建问题修复记录

## 问题描述

CI/CD 在 `setup-buildx-action` 步骤失败，日志显示：
```
##[group]Download buildx from GitHub Releases
##[endgroup]
Post job cleanup.
```

该步骤没有任何输出就直接进入 cleanup，说明下载 buildx 时遇到了网络或代理问题。

## 根本原因

1. **Docker 配置了代理**：
   - HTTP Proxy: http.docker.internal:3128
   - HTTPS Proxy: http.docker.internal:3128

2. **GitHub Actions 尝试下载 buildx 失败**：
   - `docker/setup-buildx-action` 无法通过代理访问 GitHub Releases
   - 但实际上 Mac mini runner 已经安装了 buildx v0.29.1

## 解决方案

### 重构前（使用 GitHub Actions）
```yaml
- uses: docker/setup-qemu-action@v3
- uses: docker/setup-buildx-action@v3
- uses: docker/build-push-action@v6
```

### 重构后（使用原生 Docker 命令）
```yaml
- name: Login to GitHub Container Registry
  run: docker login ghcr.io ...

- name: Setup Docker Buildx
  run: |
    docker buildx version
    docker buildx create --name miniblog-builder --use || docker buildx use miniblog-builder
    docker buildx inspect --bootstrap

- name: Build & Push
  run: docker buildx build --platform linux/amd64 --push ...
```

## 优势

1. ✅ **避免网络依赖**：不再需要从 GitHub Releases 下载
2. ✅ **绕过代理问题**：直接使用本地已安装的工具
3. ✅ **更快速**：减少了下载和缓存步骤
4. ✅ **更可控**：使用原生 Docker 命令，更容易调试
5. ✅ **简化配置**：移除了 QEMU（不需要交叉编译）

## 改动文件

- `.github/workflows/cicd.yml` - 完全重构 build-and-push job
- `scripts/install-buildx.sh` - 添加手动安装脚本（备用）

## 测试建议

1. 提交更改后观察 CI/CD 日志
2. 检查 buildx builder 是否成功创建
3. 验证镜像是否成功推送到 GHCR
4. 确认镜像标签正确（TAG 和 latest）

## 后续优化

考虑添加：
- 构建缓存优化（使用 Docker layer cache）
- 并行构建多个镜像
- 构建时间统计
- 镜像大小报告

---
修复日期：2025-10-24
修复人员：GitHub Copilot
