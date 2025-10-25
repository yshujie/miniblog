#!/bin/bash

# 手动修改 Docker Desktop 配置文件（如果 GUI 修改不生效）

set -e

echo "=== 手动修改 Docker Desktop 配置 ==="
echo ""

# Docker Desktop 配置文件路径
SETTINGS_FILE="$HOME/Library/Group Containers/group.com.docker/settings.json"

if [ ! -f "$SETTINGS_FILE" ]; then
    echo "❌ 配置文件不存在: $SETTINGS_FILE"
    exit 1
fi

echo "📁 配置文件位置: $SETTINGS_FILE"
echo ""

# 备份原始配置
BACKUP_FILE="${SETTINGS_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$SETTINGS_FILE" "$BACKUP_FILE"
echo "✅ 已备份配置到: $BACKUP_FILE"
echo ""

# 检查当前配置
echo "📋 当前代理配置:"
if command -v jq &> /dev/null; then
    cat "$SETTINGS_FILE" | jq '.proxies' 2>/dev/null || echo "无法解析 JSON"
else
    grep -A 10 '"proxies"' "$SETTINGS_FILE" || echo "未找到 proxies 配置"
fi
echo ""

echo "⚠️  警告：自动修改配置文件有风险！"
echo ""
echo "建议操作："
echo "  1. 先尝试在 Docker Desktop GUI 中点击 'Apply' 按钮"
echo "  2. 如果 GUI 不生效，再使用此脚本"
echo ""
echo "如果确定要继续，请按 Ctrl+C 取消，或按 Enter 继续..."
read

# 使用 jq 修改配置（如果安装了 jq）
if command -v jq &> /dev/null; then
    echo "📝 使用 jq 修改配置..."
    
    # 读取当前配置并修改
    jq '.proxies.httpProxy = "http://docker.internal:3128" |
        .proxies.httpsProxy = "http://docker.internal:3128" |
        .proxies.exclude = "hubproxy.docker.internal,hubproxy.docker.internal:5555,ghcr.io,*.ghcr.io,github.com,*.github.com,raw.githubusercontent.com,*.pkg.github.com"' \
        "$SETTINGS_FILE" > "${SETTINGS_FILE}.tmp"
    
    mv "${SETTINGS_FILE}.tmp" "$SETTINGS_FILE"
    echo "✅ 配置已更新"
else
    echo "❌ 未安装 jq 命令，无法自动修改"
    echo ""
    echo "请手动安装 jq:"
    echo "  brew install jq"
    echo ""
    echo "或者手动编辑配置文件:"
    echo "  open -a TextEdit '$SETTINGS_FILE'"
    exit 1
fi

echo ""
echo "📋 新的代理配置:"
cat "$SETTINGS_FILE" | jq '.proxies'

echo ""
echo "✅ 配置已更新！"
echo ""
echo "下一步："
echo "  1. 重启 Docker Desktop"
echo "  2. 运行验证脚本确认配置生效"
