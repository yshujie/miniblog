# MiniBlog 在 serverB 部署

三个生产服务 `miniblog-backend`、`miniblog-frontend-admin`、`miniblog-frontend-blog` 均由 serverB 的 `/opt/miniblog` 管理。serverA 继续终止三个原域名的 TLS，并通过共享 `infra-network` overlay 访问服务名。serverD 保留 MQ 和 QS worker，不再运行 MiniBlog。

## 发布与运维目标

CI/CD、db-ops、Notion read operations 和 reviewed maintenance 统一使用仓库级 `SVRB_HOST`、`SVRB_USERNAME`（兼容 `SVRB_USER`）、`SVRB_SSH_KEY`、`SVRB_SSH_PORT`、`SVRB_HOST_FINGERPRINT`。不回退到 `SVRD_*`。发布和数据库脚本执行前验证远端 hostname 为 serverB，SSH/SCP 与 Notion transport 固定校验其主机指纹。

部署用户为 deploy，拥有应用目录和日志目录，可直接运行 Docker。部署公钥保留既有 authorized_keys 并单独追加，使用 restrict 禁止端口/代理转发及 PTY。私钥仅存于仓库部署 Secret，不提交 Git。四类工作流共享 miniblog-prod 并发组，部署和 Notion 运维继续使用服务器端 `.operations.lock`，数据库操作默认全部跳过。

## 数据与切换

迁移采用原运行镜像标签，保持外部 MySQL、Redis 配置、JWT、Notion 只读凭据和 feature switches。先保留 serverB 旧目录与卷，再迁入 serverD 的整个应用目录，包括 configs、ops/notion/reviews、维护回执和历史备份。上传卷在停止旧后台后完成最终同步；前端 dist 卷使用同版镜像中的资源。

预验证使用隔离名称、禁止服务 aliases 和禁用 Notion scheduler 的候选容器；数据库 preflight 仅检查兼容性，不执行初始化、迁移或 seed。切换时先停止旧端三容器，再在 serverB 启动正式名称并重载 serverA Nginx，避免 overlay 名称冲突和两个定时同步器同时运行。`MINIBLOG_NOTION_SYNC_ENABLED`、config revision、暂停状态及 publication hold 均保持原契约，不借迁移更改内容或发布状态。

迁移后合并本次部署目标变更，恢复工作流并核对 main CI/CD 实际在 serverB 部署同一 SHA 的三个镜像。确认功能正常后移除 serverD 原容器，保留原卷、私有配置和切换归档作为回滚副本；不执行全局 Docker prune。

## 验证和恢复边界

检查三个域名、后端 health、公共文章目录与文章详情、前端静态资源、Notion 单实例运行、镜像版本和环境标志。核对 serverB 原 IAM/运营后台以及 serverD MQ/worker 的 ID、启动时间和挂载未变。短时资源快照不替代完整峰值压测或业务账号人工验收。

回滚前先停新后台的定时任务和写入口，确认共享数据库兼容及迁移后 uploads 的最新写入；不能只启动旧卷而丢失新文件。恢复 serverD 原配置、服务名和 serverA upstream 后再恢复入口。后续 GitHub 工作流仍以 serverB 为默认目标，回滚主机需要独立调整，不得通过误配 `SVRB_HOST` 指向其他主机绕过 hostname/fingerprint 检查。
