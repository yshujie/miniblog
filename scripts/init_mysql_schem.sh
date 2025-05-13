
#!/bin/bash

# 初始化 mysql 数据库

# 检查 mysql 是否启动
if ! mysqladmin -u root -p ping; then
    echo "mysql 未启动"
    exit 1
fi

# 导入数据库
mysql -u root -p miniblog < /data/mysql/data/miniblog.sql

# 验证数据库，查看 miniblog 数据库是否创建成功
echo "验证数据库，查看 miniblog 数据库是否创建成功"
mysql -u root -p -e "show databases;"

# 验证 miniblog 数据库是否导入成功
echo "验证 miniblog 数据库是否导入成功"
mysql -u root -p miniblog -e "show tables;"

