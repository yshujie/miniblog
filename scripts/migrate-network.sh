#!/usr/bin/env bash
#
# 网络架构迁移脚本
# 用途: 从当前 infra_shared 迁移到新的多应用网络架构
#

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running"
        exit 1
    fi
    log_info "Docker is running"
}

# 创建网络（如果不存在）
create_network_if_not_exists() {
    local network_name=$1
    local subnet=$2
    local gateway=$3
    shift 3
    local extra_opts=("$@")
    
    if docker network ls --format '{{.Name}}' | grep -w "$network_name" >/dev/null 2>&1; then
        log_warn "Network $network_name already exists, skipping..."
        return 0
    fi
    
    log_info "Creating network: $network_name ($subnet)"
    docker network create \
        --driver bridge \
        --subnet "$subnet" \
        --gateway "$gateway" \
        "${extra_opts[@]}" \
        "$network_name"
}

# 桥接容器到网络
bridge_container_to_network() {
    local container=$1
    local network=$2
    
    # 检查容器是否存在
    if ! docker ps -a --format '{{.Names}}' | grep -w "$container" >/dev/null 2>&1; then
        log_warn "Container $container not found, skipping..."
        return 0
    fi
    
    # 检查是否已连接
    if docker inspect "$container" 2>/dev/null | jq -e ".[0].NetworkSettings.Networks.\"$network\"" >/dev/null 2>&1; then
        log_warn "Container $container already connected to $network, skipping..."
        return 0
    fi
    
    log_info "Connecting $container to $network"
    docker network connect "$network" "$container"
}

# 主流程
main() {
    log_info "Starting network architecture migration..."
    
    check_docker
    
    echo ""
    log_info "=== Step 1: Creating application networks ==="
    
    # MiniBlog 网络（保留现有的 infra_shared 作为别名）
    # 注意: 这里我们不创建新的 miniblog_net，而是继续使用 infra_shared
    # 如果要重命名，需要停止所有服务
    log_info "Using existing infra_shared as miniblog_net (alias)"
    
    # QS 应用网络
    create_network_if_not_exists \
        "qs_net" \
        "172.22.0.0/16" \
        "172.22.0.1" \
        --label "app.name=qs" \
        --label "network.type=application"
    
    # Jenkins CI/CD 网络
    create_network_if_not_exists \
        "jenkins_net" \
        "172.23.0.0/16" \
        "172.23.0.1" \
        --label "service.name=jenkins" \
        --label "network.type=cicd"
    
    echo ""
    log_info "=== Step 2: Bridging infrastructure services to MiniBlog network ==="
    
    # MiniBlog 依赖（infra_shared）
    bridge_container_to_network "mysql" "infra_shared"
    bridge_container_to_network "redis" "infra_shared"
    bridge_container_to_network "nginx" "infra_shared"
    
    echo ""
    log_info "=== Step 3: Bridging infrastructure services to QS network ==="
    
    # QS 依赖
    bridge_container_to_network "mysql" "qs_net"
    bridge_container_to_network "redis" "qs_net"
    bridge_container_to_network "mongo" "qs_net"
    bridge_container_to_network "nginx" "qs_net"
    
    echo ""
    log_info "=== Step 4: Bridging Jenkins to CI/CD network ==="
    
    # Jenkins 依赖
    bridge_container_to_network "jenkins" "jenkins_net"
    
    echo ""
    log_info "=== Step 5: Verification ==="
    
    echo ""
    log_info "Network topology:"
    docker network ls --format '{{.Name}}' | while read net; do
        if [[ "$net" =~ (infra_shared|qs_net|jenkins_net|infra-backend|infra-frontend) ]]; then
            echo "  === Network: $net ==="
            docker network inspect "$net" --format '{{range .Containers}}    - {{.Name}} ({{.IPv4Address}}){{println}}{{end}}'
        fi
    done
    
    echo ""
    log_info "Migration completed successfully!"
    
    echo ""
    log_warn "Next steps:"
    echo "  1. Update Jenkins credentials to use correct hostnames (mysql, redis)"
    echo "  2. Restart MiniBlog services: docker-compose restart"
    echo "  3. Verify connectivity: docker exec miniblog-backend ping -c 3 mysql"
    echo "  4. Check application logs for any connection errors"
}

# 执行主流程
main "$@"
