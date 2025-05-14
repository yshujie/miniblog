#!/bin/sh

# 等待 MySQL 服务启动
echo "Waiting for MySQL to start..."
while ! mysqladmin ping -h localhost -u root -p${MYSQL_ROOT_PASSWORD} --silent; do
    sleep 1
done

# 检查数据库是否存在
if ! mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "USE miniblog" 2>/dev/null; then
    echo "Initializing miniblog database..."
    # 创建数据库
    mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "CREATE DATABASE IF NOT EXISTS miniblog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
    
    # 创建用户并授权
    mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "CREATE USER IF NOT EXISTS '${MYSQL_USER}'@'%' IDENTIFIED BY '${MYSQL_PASSWORD}';"
    mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "GRANT ALL PRIVILEGES ON miniblog.* TO '${MYSQL_USER}'@'%';"
    mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "FLUSH PRIVILEGES;"
    
    # 导入初始化数据
    mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} miniblog < /docker-entrypoint-initdb.d/miniblog.sql
    
    echo "Database initialization completed."
else
    echo "Database miniblog already exists."
fi 