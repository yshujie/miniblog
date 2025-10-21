# CI/CD 说明（miniblog）

## 概述

----
本项目的 CI/CD 使用 GitHub Actions 执行构建、打包并将容器镜像推送到 GitHub Container Registry (GHCR)。部署阶段在自托管 macOS Runner 上通过 SSH 登录到服务器 A（生产）并拉取镜像，使用 docker compose 进行无停机覆盖更新。

## 为什么使用 GHCR 而不是 Docker Hub

- 认证与权限集中： GHCR 与 GitHub 仓库相结合，能够使用组织或仓库级别的访问控制和 PAT（Personal Access Token）。
- 集成体验更好：actions 与 GHCR 无缝集成（例：actions/setup-go、docker/build-push-action 与 GHCR）。
- 速率限制与私有仓储：Docker Hub 对匿名拉取有更严格的速率限制。GHCR 对私有镜像/组织仓库管理更友好。
- 统一管理：使用 GHCR 可以把代码和镜像托管在同一平台上，降低凭据管理复杂度。

## 如何部署（高层）

1. CI 构建 Pipeline（`build-and-push`）
   - 在自托管 runner 上使用 docker buildx 构建多架构镜像（目前 target linux/amd64）并推送到 GHCR。
   - 使用 `docker/build-push-action`，并登录 ghcr.io：
     - 登录命令示例：

  ```bash
  echo "${{ secrets.GHCR_TOKEN }}" | docker login ghcr.io -u "${{ github.repository_owner }}" --password-stdin
  ```

- 镜像标签（tags）：

- `ghcr.io/OWNER/miniblog-backend:${TAG}`

- `ghcr.io/OWNER/miniblog-frontend-blog:${TAG}`

- `ghcr.io/OWNER/miniblog-frontend-admin:${TAG}`

## 2. 部署阶段（`deploy` job）

- 使用 SSH 私钥登录到服务器 A（在 Actions 的 Secrets 中配置：`SVRA_HOST`, `SVRA_USER`, `SVRA_SSH_KEY` 等）。

  - 在服务器 A 上执行：

    - `docker login ghcr.io`（使用 `GHCR_TOKEN`）

    - `docker compose -f docker-compose.yml -f docker-compose.prod.yml pull` 拉取新镜像

    - `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d` 无停机更新容器

- 配置 nginx 反向代理（在宿主机上运行的 nginx 容器）以代理到对应容器。前端通常不暴露宿主端口，而由 nginx 反向代理到容器内部 8080 端口。

## 为什么不需要把镜像推到 Docker Hub 也能部署？

-----------------------------------

- 部署服务器 A 会直接从 GHCR 拉取镜像（`docker pull` / `docker compose pull`），不依赖 Docker Hub。
- CI 已经将镜像推送到 GHCR（工作流里的 `docker/build-push-action` 使用 ghcr.io 标签）。服务器上只要能访问 GHCR（并登录）即可拉取并启动镜像。

## 关键 Secrets 与权限

------------------

- `GHCR_TOKEN`：让服务器或 runner 使用 `docker login ghcr.io` 拉取镜像。建议使用具有最小权限的 token（packages: read / write as needed）。
- `SVRA_SSH_KEY`：Actions 用于登录服务器 A 的私钥（提前把公钥加入服务器 `~/.ssh/authorized_keys`）。
- 服务器上应确保 Docker 与 nginx 有适当权限，并在必要时为 deploy 用户提供非交互 sudo 或在容器以外创建目录。

## 注意点与常见问题

-----------------

- nginx 找不到新镜像服务：

- 原因：nginx 可能没有连接到内部 `miniblog_net` 或 upstream 缓存未刷新。

- 解决方法：在部署脚本中自动 `docker network connect miniblog_net nginx` 并在确认后端健康后执行 `nginx -s reload`。
- 端口与代理：
- 后端容器映射了宿主 8090（由 compose 指定），以便健康检查或直接访问；前端容器不直接映射端口（通过 nginx 代理访问），这是设计所致。
- 如果想推到 Docker Hub：
- 可以在 workflow 中额外登录 Docker Hub 并在 tags 中添加 `docker.io/<user>/<repo>:tag`。但这会增加推送时间与带宽。

## 如何手动触发数据库操作（db-ops）

---------------------------------

- DB init/migrate/seed 被移到了单独的 workflow：`.github/workflows/db-ops.yml`，仅可手动触发（`workflow_dispatch`），避免在每次 push 时执行潜在的破坏性操作。

## 常用调试命令（在服务器 A）

-------------------------

- 查看容器与端口：

  ```bash
  docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Ports}}"
  ```

- 查看 miniblog 网络内容器：

  ```bash
  docker network inspect miniblog_net --format '{{range .Containers}}{{.Name}} {{end}}' ; echo
  ```

- 从网络内部访问后端健康：

  ```bash
  docker run --rm --network=miniblog_net curlimages/curl:8.1.2 -I http://miniblog-backend:8080/health
  ```

## 结语

----
如果你希望我把镜像同时也推到 Docker Hub，或把部署改为优先从 Docker Hub 拉取（替代 GHCR），我可以帮你把对应的登录、tag 和策略加到 workflow，并保留开关参数。也可以为部署加入备份/滚回步骤。
