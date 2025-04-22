## 这是一个用gin框架和vue框架搭建一个简易博客前台展示和后台管理系统

### 一.技术选型

-前端：用基于vue的``ant design vue``构建后台管理页面和基于vue的``vuetify``构建前台展示页面。

-后端：用``Gin``快速搭建基础restful风格API，Gin是一个go语言编写的Web框架。

-数据库：采用``MySql``，使用gorm实现对数据库的基本操作。

### 二.项目结构

```bash
├── api # Swagger / OpenAPI 文档存放目录
│   └── openapi
│       └── openapi.yaml # OpenAPI 3.0 API 接口文档
├── build # 构建文件存放目录
│   └──  docker # docker 构建文件存放目录
├── cmd # main 文件存放目录
│   └── miniblog
│       └── main.go
├── configs # 配置文件存放目录
│   ├── miniblog.sql # 数据库初始化 SQL
│   ├── miniblog.yaml # miniblog 配置文件
│   └── nginx.conf # Nginx 配置
├── docs # 项目文档
│   ├── devel # 开发文档
│   │   └── zh-CN # 中文文档
│   │       ├── architecture.md # miniblog 架构介绍
│   │       ├── conversions # 规范文档存放目录
│   │       │   ├── api.md # 接口规范
│   │       │   ├── commit.md # Commit 规范
│   │       │   ├── directory.md # 目录结构规范
│   │       │   ├── error_code.md # 错误码规范
│   │       │   ├── go_code.md # 代码规范
│   │       │   ├── log.md # 日志规范
│   │       │   └── version.md # 版本规范
│   │       └── README.md
│   ├── guide # 用户文档
│   │   └── zh-CN # 中文文档
│   │       ├── operation-guide # 操作指南
│   │       ├── quickstart # 快速入门
│   │       └── README.md
│   └── images # 项目图片存放目录
├── internal # 内部代码保存目录，这里面的代码不能被外部程序引用
│   ├── miniblog # miniblog 代码实现目录
│   │   ├── biz # biz 层代码
│   │   ├── controller # controller 层代码
│   │   │   └── v1 # API 接口版本
│   │   │       ├── post # 博客相关代码实现
│   │   │       └── user
│   │   ├── store # store 层代码
│   │   │       ├── post # 博客相关代码实现
│   │   │       └── user
│   │   ├── helper.go # 工具类代码存放文件
│   │   ├── router.go # Gin 路由加载代码
│   │   └── miniblog.go # miniblog 主业务逻辑实现代码
│   └── pkg # 内部包保存目录
│       ├── core # core 包，用来保存一些核心的函数
│       ├── errno # errno 包，实现了 miniblog 的错误码功能
│       │   ├── code.go # 错误码定义文件
│       │   └── errno.go # errno 包功能函数文件
│       ├── known # 存放项目级的常量定义
│       ├── log # miniblog 自定义 log 包
│       ├── middleware # Gin 中间件包
│       │   ├── authn.go # 认证中间件
│       │   ├── authz.go # 授权中间件
│       │   ├── header.go # 指定 HTTP Response Header
│       │   └── requestid.go # 请求 / 返回头中添加 X-Request-ID
│       └── model # GORM Model，model 层代码
├── pkg # 可供外部程序直接使用的 Go 包存放目录
│   ├── api # REST API 接口定义存放目录
│   ├── proto # Protobuf 接口定义存放目录
│   ├── auth # auth 包，用来完成认证、授权功能
│   │   ├── authn.go # 认证功能
│   │   └── authz.go # 授权功能
│   ├── db # db 包，用来完成 MySQL 数据库连接
│   ├── token # JWT Token 的签发和解析
│   ├── util # 工具类包存放目录
│   │   └── id # id 包，用来生成唯一短 ID
│   └── version # version 包，用来保存 / 输出版本信息
├── scripts # 脚本文件
├── third_party # 第三方 Go 包存放目录
├── _output # 临时文件存放目录
├── Makefile # Makefile 文件，一般大型软件系统都是采用 make 来作为编译工具
├── go.mod
├── go.sum
└── README.md # 中文 README
```

### 三.使用说明

1-需要把config/config.ini里的文件配置(特别是数据库mysql的配置)修改成自己需要的配置，本项目使用七牛云对象存储上传的文件，你可以自己在七牛云注册一个账号，可以免费获赠10G的存储空间

2-本项目可以直接在windows上运行，建议使用goland，可以将blog.sql导入数据库中，初始登录用户为admin,密码为123456

3-[前台展示]（<http://localhost:8080/front）和> [后台管理] (<http://localhost:8080/admin>) 的切换需要修改routes/router.go,打开相关注释即可

4-如果需要修改前端部分，可以修改web下的文件，修改完后运行`npm run build`，把dist文件覆盖掉static里的文件

5-本项目是前后端分离项目，可以注释掉routes/router.go中加载静态资源的代码，把后端代码运行起来，然后在web/admin(或者web/front)目录下，运行`npm run serve`即可

### 四.部署项目（Linux系统下）

方式一：使用dockerfile部署(确保服务器上有下载docker)

```shell
#克隆项目
git clone git@github.com:tmnhs/gin-blog.git
#在gin-blog目录下
cd gin-blog
#docker编译镜像
docker build -t myblog .
#运行docker
docker run -p 8080:8080 -t --name myblog myblog
#注意此时的数据库配置应该改为服务器上的配置
```

方式二：使用脚本部署

```shell
#克隆项目
git clone git@github.com:tmnhs/gin-blog.git
#在gin-blog目录下
cd gin-blog
#必须在linux系统下
#启动项目
./serve.sh start
#暂停项目
./serve.sh stop
#重启服务
./serve.sh restart
#注意，脚本运行的端口号必须大于1024，不然可能会没有权限
```
