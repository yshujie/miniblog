#!/bin/bash

# 使用固定的容器名称
MYSQL_CONTAINER_NAME="miniblog-mysql-1"

# 获取 mysql 密码
MYSQL_PASSWORD="miniblog123"

# 检查 docker 中的 mysql 是否启动
if ! docker exec ${MYSQL_CONTAINER_NAME} mysqladmin -u root -p${MYSQL_PASSWORD} ping; then
    echo "mysql 未启动"
    exit 1
fi

# 如果 miniblog 数据库存在，则不再导入
if docker exec ${MYSQL_CONTAINER_NAME} mysql -u root -p${MYSQL_PASSWORD} -e "show databases;" | grep -q "miniblog"; then
    echo "miniblog 数据库已存在，不再导入"
    exit 0
fi

# 导入数据库
docker exec -i ${MYSQL_CONTAINER_NAME} mysql -u root -p${MYSQL_PASSWORD} < /data/mysql/data/miniblog.sql

# 验证数据库，查看 miniblog 数据库是否创建成功
echo "验证数据库，查看 miniblog 数据库是否创建成功"
if ! docker exec ${MYSQL_CONTAINER_NAME} mysql -u root -p${MYSQL_PASSWORD} -e "show databases;" | grep -q "miniblog"; then
    echo "miniblog 数据库创建失败"
    exit 1
fi

echo "miniblog 数据库创建成功"