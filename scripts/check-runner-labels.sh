#!/bin/bash

# 检查 miniblog runner 的标签配置

set -e

echo "=== 检查 miniblog Runner 标签 ==="
echo ""

RUNNER_DIR="/Users/yangshujie/actions-runner"

if [ ! -d "$RUNNER_DIR" ]; then
    echo "❌ Runner 目录不存在: $RUNNER_DIR"
    exit 1
fi

echo "📁 Runner 目录: $RUNNER_DIR"
echo ""

# 检查配置文件
if [ -f "$RUNNER_DIR/.runner" ]; then
    echo "📋 Runner 基本信息:"
    cat "$RUNNER_DIR/.runner" | grep -E "(agentName|gitHubUrl)" | sed 's/^/   /'
    echo ""
fi

# 检查运行状态
if pgrep -f "actions-runner/bin/Runner.Listener" > /dev/null; then
    echo "✅ Runner 进程正在运行"
    PID=$(pgrep -f "actions-runner/bin/Runner.Listener" | head -1)
    echo "   PID: $PID"
else
    echo "❌ Runner 进程未运行"
fi

echo ""
echo "=== 建议的标签配置 ==="
echo ""
echo "当前 CI/CD 使用的标签:"
echo "   - [self-hosted, macmini, prod]          # test & build-and-push"
echo "   - [self-hosted, macOS, ARM64]           # deploy"
echo ""
echo "💡 如何配置标签："
echo "   1. 在 GitHub 仓库页面: Settings > Actions > Runners"
echo "   2. 点击 runner 名称 'yangshujiedeMac-mini'"
echo "   3. 查看当前标签，应该包含："
echo "      - self-hosted (自动)"
echo "      - macOS (自动)"
echo "      - ARM64 (自动)"
echo "      - macmini (需要手动添加)"
echo "      - prod (需要手动添加)"
echo ""
echo "   或者简化 CI/CD 配置，只使用系统自动标签："
echo "      runs-on: [self-hosted, macOS, ARM64]"
echo ""
