# 网络架构快速参考

> 本文档曾记录 `infra_shared`、`qs_net` 和 `jenkins_net` 的旧迁移方案。
> 该方案已经失效，相关迁移脚本已删除，请勿继续执行历史命令。

## 当前网络契约

MiniBlog 的当前 `docker-compose.yml` 使用两个外部网络：

| 网络 | 责任 |
| --- | --- |
| `miniblog-network` | MiniBlog 后端、博客前端和管理后台之间的业务通信 |
| `infra-network` | MiniBlog 服务访问由基础设施侧提供的依赖 |

GitHub Actions 部署会在 ServerD 上按需创建 `miniblog-network`，但要求
`infra-network` 已经由基础设施侧创建。仓库内不再提供修改基础设施网络拓扑的一键脚本。

## 部署前检查

```bash
docker network inspect miniblog-network >/dev/null 2>&1 || \
  docker network create miniblog-network

docker network inspect infra-network >/dev/null

docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet
```

如果 `infra-network` 不存在，应回到基础设施配置确认网络名称、创建责任和依赖服务，
不要临时创建同名空网络来掩盖配置错误。

## 网络状态检查

```bash
docker network inspect miniblog-network
docker network inspect infra-network

docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=100
```

检查某个 MiniBlog 容器实际加入的网络：

```bash
docker inspect miniblog-backend \
  --format='{{range $name, $config := .NetworkSettings.Networks}}{{$name}} {{end}}'
```

## 502 排查边界

出现 Nginx 502 时，按请求链路检查：

1. ServerA Nginx upstream 配置指向的 ServerD 地址和端口是否正确。
2. ServerD 上 `miniblog-backend` 是否运行并监听发布端口。
3. ServerA 到 ServerD 的网络连通性是否正常。
4. 后端日志中是否存在启动失败、数据库或 Redis 连接错误。

不要通过把 ServerA Nginx 随意加入 ServerD 的 Docker 网络来修复跨主机链路。

## 事实来源

- 网络声明：`docker-compose.yml`
- 生产差异：`docker-compose.prod.yml`
- 部署预检：`.github/workflows/cicd.yml`
- 脚本边界与执行方式：`scripts/README.md`
- 基础设施网络的创建和维护：infra 仓库中的 `docs/network/architecture.md`
