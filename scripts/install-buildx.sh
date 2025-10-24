#!/bin/bash

# 在 Mac mini runner 上手动安装 Docker Buildx

set -e

echo "=== 安装 Docker Buildx ==="
echo ""

# 1. 创建插件目录
DOCKER_CLI_PLUGINS_DIR="$HOME/.docker/cli-plugins"
mkdir -p "$DOCKER_CLI_PLUGINS_DIR"

# 2. 获取最新版本
echo "📥 获取最新 buildx 版本信息..."
BUILDX_VERSION=$(curl -s https://api.github.com/repos/docker/buildx/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
echo "   最新版本: $BUILDX_VERSION"
echo ""

# 3. 下载 buildx（macOS ARM64）
echo "📥 下载 buildx..."
BUILDX_URL="https://github.com/docker/buildx/releases/download/${BUILDX_VERSION}/buildx-${BUILDX_VERSION}.darwin-arm64"
echo "   URL: $BUILDX_URL"

# 使用 curl 下载，禁用代理以避免代理问题
curl -L --noproxy '*' "$BUILDX_URL" -o "$DOCKER_CLI_PLUGINS_DIR/docker-buildx"

# 4. 添加执行权限
chmod +x "$DOCKER_CLI_PLUGINS_DIR/docker-buildx"

# 5. 验证安装
echo ""
echo "✅ Buildx 安装完成"
echo ""
echo "=== 验证安装 ==="
docker buildx version

echo ""
echo "=== 创建并设置 builder ==="
# 创建一个新的 builder 实例
docker buildx create --name miniblog-builder --driver docker-container --use || \
docker buildx use miniblog-builder

# 启动 builder
docker buildx inspect --bootstrap

echo ""
echo "=== Builder 列表 ==="
docker buildx ls

echo ""
echo "✅ 所有设置完成！"
