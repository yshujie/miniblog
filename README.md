# miniblog

## 一个基于gin框架和vue框架搭建的简易博客前台展示和后台管理系统

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
│   │   ├── store # store 层代码
│   │   ├── helper.go # 工具类代码存放文件
│   │   ├── router.go # Gin 路由加载代码
│   │   └── miniblog.go # miniblog 主业务逻辑实现代码
│   └── pkg # 内部包保存目录
│       ├── core # core 包，用来保存一些核心的函数
│       ├── errno # errno 包，实现了 miniblog 的错误码功能
│       ├── known # 存放项目级的常量定义
│       ├── log # miniblog 自定义 log 包
│       ├── middleware # Gin 中间件包
│       └── model # GORM Model，model 层代码
├── pkg # 可供外部程序直接使用的 Go 包存放目录
│   ├── api # REST API 接口定义存放目录
│   ├── proto # Protobuf 接口定义存放目录
│   ├── auth # auth 包，用来完成认证、授权功能
│   ├── db # db 包，用来完成 MySQL 数据库连接
│   ├── token # JWT Token 的签发和解析
│   ├── util # 工具类包存放目录
│   └── version # version 包，用来保存 / 输出版本信息
├── scripts # 脚本文件
├── third_party # 第三方 Go 包存放目录
├── Makefile # Makefile 文件，一般大型软件系统都是采用 make 来作为编译工具
├── go.mod
├── go.sum
└── README.md # 中文 README
```
