#!/bin/bash

# 检查 GitHub Actions Runner 和 Docker 状态

set -e

echo "=== GitHub Actions Runner 状态检查 ==="
echo ""

# 1. 检查 Docker 状态
echo "1. 检查 Docker 状态..."
if command -v docker &> /dev/null; then
    echo "   ✅ Docker 命令已安装"
    if docker ps &> /dev/null; then
        echo "   ✅ Docker daemon 正在运行"
        docker version --format '   Docker 版本: {{.Server.Version}}'
    else
        echo "   ❌ Docker daemon 未运行"
        echo "   💡 请执行: open -a Docker"
        exit 1
    fi
else
    echo "   ❌ Docker 未安装"
    exit 1
fi

echo ""

# 2. 检查 GitHub Actions Runner
echo "2. 检查 GitHub Actions Runner..."
RUNNER_DIR="$HOME/actions-runner"

if [ -d "$RUNNER_DIR" ]; then
    echo "   ✅ Runner 目录存在: $RUNNER_DIR"
    
    if [ -f "$RUNNER_DIR/.runner" ]; then
        echo "   📋 Runner 配置:"
        cat "$RUNNER_DIR/.runner" | grep -E "(agentName|poolId|serverUrl)" | sed 's/^/      /'
    fi
    
    if [ -f "$RUNNER_DIR/.credentials" ]; then
        echo "   ✅ Runner 凭证已配置"
    fi
    
    # 检查 runner 是否在运行
    if pgrep -f "Runner.Listener" > /dev/null; then
        echo "   ✅ Runner 进程正在运行"
        ps aux | grep "Runner.Listener" | grep -v grep | sed 's/^/      /'
    else
        echo "   ⚠️  Runner 进程未运行"
        echo "   💡 请执行: cd $RUNNER_DIR && ./run.sh"
    fi
else
    echo "   ❌ Runner 目录不存在: $RUNNER_DIR"
    echo "   💡 请先配置 GitHub Actions Self-hosted Runner"
fi

echo ""

# 3. 检查 Runner 标签（如果配置文件存在）
echo "3. 检查 Runner 标签..."
if [ -f "$RUNNER_DIR/.runner" ]; then
    LABELS=$(cat "$RUNNER_DIR/.runner" | grep "agentName" || echo "未找到标签信息")
    echo "   $LABELS"
    echo ""
    echo "   💡 CI/CD 配置中使用的标签:"
    echo "      - test & build-and-push: [self-hosted, macmini, prod]"
    echo "      - deploy: [self-hosted, macOS, ARM64]"
fi

echo ""

# 4. 网络连接检查
echo "4. 检查网络连接..."
if ping -c 1 ghcr.io &> /dev/null; then
    echo "   ✅ 可以连接到 ghcr.io"
else
    echo "   ⚠️  无法连接到 ghcr.io"
fi

if ping -c 1 github.com &> /dev/null; then
    echo "   ✅ 可以连接到 github.com"
else
    echo "   ⚠️  无法连接到 github.com"
fi

echo ""
echo "=== 检查完成 ==="
