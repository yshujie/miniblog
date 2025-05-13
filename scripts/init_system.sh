#!/bin/bash
# 系统初始化，创建系统依赖的目录、文件等

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "当前工作目录: $(pwd)"
echo "脚本目录: ${SCRIPT_DIR}"
echo "项目根目录: ${PROJECT_ROOT}"

# 创建 nginx 相关目录
mkdir -p /etc/nginx/        # nginx 配置目录
mkdir -p /etc/nginx/conf.d/ # nginx 配置文件目录
mkdir -p /etc/nginx/ssl/    # nginx ssl 证书目录
mkdir -p /data/logs/nginx/  # nginx 日志目录

# 将 nginx 配置文件复制到 /etc/nginx/
cp -r ${PROJECT_ROOT}/configs/nginx/ /etc/nginx/
cp ${PROJECT_ROOT}/configs/nginx/conf.d/* /etc/nginx/conf.d/

# 检查 nginx 配置文件是否存在
if [ ! -f /etc/nginx/nginx.conf ]; then
    echo "nginx 配置文件 nginx.conf 不存在"
    exit 1
fi
# 检查 /etc/nginx/conf.d/default.conf 文件是否存在
if [ ! -f /etc/nginx/conf.d/default.conf ]; then
    echo "nginx 配置文件 default.conf 不存在"
    exit 1
fi

# 创建 mysql 相关目录
mkdir -p /var/lib/mysql/  # mysql 安装目录
mkdir -p /data/mysql/data/ # mysql 数据目录
mkdir -p /data/logs/mysql/ # mysql 日志目录

# 将 mysql 初始化数据复制到 /data/mysql/data/
echo "复制 miniblog.sql..."
cp ${PROJECT_ROOT}/configs/mysql/miniblog.sql /data/mysql/data/miniblog.sql

# 创建 redis 相关目录
mkdir -p /var/lib/redis/ # redis 安装目录
mkdir -p /data/redis/data/ # redis 数据目录
mkdir -p /data/logs/redis/ # redis 日志目录

# 创建 miniblog 相关目录
mkdir -p /etc/miniblog/ # miniblog 安装目录
mkdir -p /data/logs/miniblog/ # miniblog 日志目录

# 将 miniblog 配置文件复制到 /etc/miniblog/config.yaml
echo "复制 miniblog.yaml..."
cp ${PROJECT_ROOT}/configs/miniblog.yaml /etc/miniblog/config.yaml

# 权限设置
chmod 755 /data/logs/nginx/
chmod 755 /data/logs/mysql/
chmod 755 /data/logs/redis/
chmod 755 /data/logs/miniblog/
chmod 644 /etc/nginx/nginx.conf
chmod 644 /etc/nginx/conf.d/yangshujie.com.conf