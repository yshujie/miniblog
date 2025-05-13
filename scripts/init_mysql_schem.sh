
#!/bin/bash

# 初始化 mysql 数据库

# 检查 mysql 是否启动
if ! mysqladmin -u root -p ping; then
    echo "mysql 未启动"
    exit 1
fi

# 如果 miniblog 数据库存在，则不再导入
if mysql -u root -p -e "show databases;" | grep -q "miniblog"; then
    echo "miniblog 数据库已存在，不再导入"
    exit 0
fi

# 导入数据库
mysql -u root -p miniblog < /data/mysql/data/miniblog.sql

# 验证数据库，查看 miniblog 数据库是否创建成功
echo "验证数据库，查看 miniblog 数据库是否创建成功"
if ! mysql -u root -p -e "show databases;" | grep -q "miniblog"; then
    echo "miniblog 数据库创建失败"
    exit 1
fi

# 验证 miniblog 数据库是否导入成功
echo "验证 miniblog 数据库是否导入成功"
if ! mysql -u root -p miniblog -e "show tables;" | grep -q "miniblog"; then
    echo "miniblog 数据库导入失败"
    exit 1
fi