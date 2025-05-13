#!/bin/bash
# 系统初始化，创建系统依赖的目录、文件等

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "当前工作目录: $(pwd)"
echo "脚本目录: ${SCRIPT_DIR}"
echo "项目根目录: ${PROJECT_ROOT}"

# 创建 nginx 相关目录
echo "创建 nginx 相关目录"
mkdir -p /data/nginx/conf.d  # nginx 配置文件目录
mkdir -p /data/nginx/ssl     # nginx ssl 证书目录
mkdir -p /data/logs/nginx    # nginx 日志目录

# 复制 nginx 配置文件
echo "复制 nginx 配置文件"
cp ${PROJECT_ROOT}/configs/nginx/nginx.conf /data/nginx/nginx.conf
cp ${PROJECT_ROOT}/configs/nginx/conf.d/default.conf /data/nginx/conf.d/default.conf
cp /data/ssl/* /data/nginx/ssl/

echo "查看 nginx 配置文件"
ls -l /data/nginx/nginx.conf
ls -l /data/nginx/conf.d/default.conf
ls -l /data/nginx/ssl/

# 创建 mysql 相关目录
echo "创建 mysql 相关目录"
mkdir -p /data/mysql/data    # mysql 数据目录
mkdir -p /data/logs/mysql    # mysql 日志目录

# 复制 mysql 初始化数据
cp ${PROJECT_ROOT}/configs/mysql/miniblog.sql /data/mysql/miniblog.sql

# 创建 redis 相关目录
echo "创建 redis 相关目录"
mkdir -p /data/redis/data    # redis 数据目录
mkdir -p /data/logs/redis    # redis 日志目录

# 创建 miniblog 相关目录
echo "创建 miniblog 相关目录"
mkdir -p /etc/miniblog       # miniblog 安装目录
mkdir -p /data/logs/miniblog # miniblog 日志目录

# 复制 miniblog 配置文件
echo "复制 miniblog 配置文件"
cp ${PROJECT_ROOT}/configs/miniblog.yaml /etc/miniblog/config.yaml

# 权限设置
echo "权限设置"
chmod 755 /data/logs/nginx
chmod 755 /data/logs/mysql
chmod 755 /data/logs/redis
chmod 755 /data/logs/miniblog
chmod 644 /data/nginx/conf.d/*.conf