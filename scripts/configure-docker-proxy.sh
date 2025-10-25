#!/bin/bash

# 配置 Docker Desktop 代理例外，允许直接访问 ghcr.io
# 这样可以保留代理加速功能，同时避免 GHCR 认证问题

set -e

echo "=== Docker 代理配置最佳实践 ==="
echo ""

echo "🎯 目标："
echo "   ✅ 保留代理 - 加速 Docker Hub、Quay.io 等镜像拉取"
echo "   ✅ 例外配置 - GHCR 和 GitHub 直接连接（避免认证问题）"
echo ""

echo "⚠️  重要：Docker Desktop 的代理配置需要通过 GUI 修改"
echo ""
echo "📖 配置步骤："
echo ""
echo "   1. 打开 Docker Desktop"
echo "   2. 点击右上角 ⚙️  Settings (设置)"
echo "   3. 进入 Resources > Proxies"
echo "   4. 确保启用代理："
echo "      ☑ Manual proxy configuration"
echo "      Web Server (HTTP): http.docker.internal:3128"
echo "      Secure Web Server (HTTPS): http.docker.internal:3128"
echo ""
echo "   5. ⭐ 关键：在 'Bypass proxy settings for these hosts & domains' 中添加："
echo ""
echo "      ghcr.io,*.ghcr.io,github.com,*.github.com,raw.githubusercontent.com,*.pkg.github.com"
echo ""
echo "   6. 点击 'Apply & Restart'"
echo ""

echo "💡 这样配置的好处："
echo "   • Docker Hub 镜像通过代理 → 更快"
echo "   • GHCR 直接连接 → 认证成功"
echo "   • GitHub Actions 下载通过代理 → 更稳定"
echo "   • GHCR 推送直接连接 → 无干扰"
echo ""

echo "=== 验证配置 ==="
echo ""

echo "1️⃣  测试 Docker info（检查代理配置）:"
docker info | grep -i proxy || echo "   未找到代理配置"

echo ""
echo "2️⃣  测试 GHCR 连接:"
if curl -s -I https://ghcr.io/v2/ | head -1 | grep -q "200\|401"; then
    echo "   ✅ GHCR 连接正常"
else
    echo "   ❌ GHCR 连接失败"
fi

echo ""
echo "3️⃣  测试 GitHub 连接:"
if curl -s -I https://api.github.com | head -1 | grep -q "200"; then
    echo "   ✅ GitHub API 连接正常"
else
    echo "   ❌ GitHub API 连接失败"
fi

echo ""
echo "=== 配置完成后，请重新运行 CI/CD ==="
